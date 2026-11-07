package app

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/domain"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Session kinds.
const (
	KindQuiz       = "quiz"
	KindDiagnostic = "diagnostic"
	KindPlacement  = "placement"
	KindExam       = "exam"
)

// CreateSessionInput starts a Quiz, Diagnostic Quiz, Placement Test, or Exam.
type CreateSessionInput struct {
	Kind      string
	CourseID  uuid.UUID
	ConceptID *uuid.UUID
	TopicID   *uuid.UUID
	ExamID    *uuid.UUID
}

// SessionView is a study session's state.
type SessionView struct {
	ID             uuid.UUID
	Kind           string
	CourseID       uuid.UUID
	ConceptID      *uuid.UUID
	TopicID        *uuid.UUID
	ExamID         *uuid.UUID
	TotalQuestions *int // nil for adaptive Placement Tests
	AnsweredCount  int
	StartedAt      time.Time
	Deadline       *time.Time
	FinishedAt     *time.Time
}

// ServedChoice is a choice without its correctness.
type ServedChoice struct {
	ID     uuid.UUID
	Label  string
	BodyMd string
}

// ServedQuestion is a Question as shown before answering: no key, no Explanation.
type ServedQuestion struct {
	ID         uuid.UUID
	StemMd     string
	Format     string
	Difficulty int
	Position   int
	IsRepeat   bool
	Choices    []ServedChoice
}

// MasteryChangeView is a level change caused by an Answer.
type MasteryChangeView struct {
	ConceptID   uuid.UUID
	ConceptName string
	Before      domain.MasteryLevel
	After       domain.MasteryLevel
}

// AnswerResult is the feedback for one Answer. Feedback fields are nil for Exams.
type AnswerResult struct {
	Recorded        bool
	Correct         *bool
	CorrectChoiceID *uuid.UUID
	ExplanationMd   *string
	MasteryChanges  []MasteryChangeView
}

// TopicStat is per-Topic accuracy within a session.
type TopicStat struct {
	TopicID   uuid.UUID
	TopicName string
	Answered  int
	Correct   int
}

// ReviewItem reveals the key and Explanation after a session is finished.
type ReviewItem struct {
	QuestionID      uuid.UUID
	StemMd          string
	ChosenChoiceID  *uuid.UUID
	CorrectChoiceID uuid.UUID
	Correct         bool
	ExplanationMd   string
}

// SessionResult summarizes a finished session.
type SessionResult struct {
	SessionID      uuid.UUID
	Kind           string
	Answered       int
	Correct        int
	Total          int
	TopicBreakdown []TopicStat
	RootCauses     []RecommendationView
	Review         []ReviewItem
}

// CreateSession starts a study session for the Actor.
func (s *Service) CreateSession(ctx context.Context, a Actor, in CreateSessionInput) (SessionView, error) {
	g, err := s.Graph(ctx, in.CourseID)
	if err != nil {
		return SessionView{}, err
	}
	now := s.now()
	params := store.CreateStudySessionParams{LearnerID: a.LearnerID, CourseID: in.CourseID, Kind: in.Kind}
	var picks []pickedQuestion
	var noQuestions *Error
	var exhausted func(q *store.Queries) error

	switch in.Kind {
	case KindQuiz, KindDiagnostic:
		var concepts []uuid.UUID
		minD, maxD, size, what := int16(1), int16(3), QuizSize, "Quiz"
		if in.Kind == KindDiagnostic {
			minD, maxD, size, what = 1, 2, DiagnosticSize, "Diagnostic Quiz"
		}
		switch {
		case in.ConceptID != nil:
			if !g.Live(*in.ConceptID) {
				return SessionView{}, E(KindInvalid, "unknown_concept", "concept is not part of this course")
			}
			concepts = []uuid.UUID{*in.ConceptID}
			params.ConceptID = in.ConceptID
		case in.TopicID != nil && in.Kind == KindQuiz:
			if _, ok := g.TopicByID[*in.TopicID]; !ok {
				return SessionView{}, E(KindInvalid, "unknown_topic", "topic is not part of this course")
			}
			for _, c := range g.LiveConcepts() {
				if c.TopicID == *in.TopicID {
					concepts = append(concepts, c.ID)
				}
			}
			params.TopicID = in.TopicID
		case in.Kind == KindDiagnostic:
			return SessionView{}, E(KindInvalid, "concept_required", "a Diagnostic Quiz needs conceptId")
		default:
			return SessionView{}, E(KindInvalid, "concept_or_topic_required", "a Quiz needs conceptId or topicId")
		}
		var fresh int
		picks, fresh, err = pickQuestions(ctx, s.q, a.LearnerID, concepts, minD, maxD, size)
		if err != nil {
			return SessionView{}, err
		}
		if fresh < size && len(concepts) > 0 {
			exhausted = func(q *store.Queries) error {
				return raiseExhausted(ctx, q, a.LearnerID, concepts, minD, maxD, size, what)
			}
		}
		if len(picks) == 0 {
			noQuestions = E(KindInvalid, "no_questions", "no Active Questions are available yet; a Content Request was raised")
		}
		params.MaxQuestions = ptr(int32(len(picks))) //nolint:gosec // at most QuizSize
	case KindPlacement:
		params.MaxQuestions = ptr(int32(PlacementMaxProbes))
	case KindExam:
		if in.ExamID == nil {
			return SessionView{}, E(KindInvalid, "exam_required", "an Exam session needs examId")
		}
		exam, err := s.q.GetExam(ctx, *in.ExamID)
		if err != nil || exam.CourseID != in.CourseID {
			return SessionView{}, E(KindInvalid, "unknown_exam", "exam is not part of this course")
		}
		ids, err := s.q.ListActiveExamQuestions(ctx, exam.ID)
		if err != nil {
			return SessionView{}, err
		}
		if len(ids) == 0 {
			return SessionView{}, E(KindInvalid, "no_questions", "this Exam has no Active Questions")
		}
		for _, id := range ids {
			picks = append(picks, pickedQuestion{ID: id})
		}
		params.ExamID = &exam.ID
		params.MaxQuestions = ptr(int32(len(ids))) //nolint:gosec // exam length, small
		params.Deadline = ptr(now.Add(time.Duration(exam.TimeLimitS) * time.Second))
	default:
		return SessionView{}, E(KindInvalid, "invalid_kind", "unknown session kind %q", in.Kind)
	}

	if noQuestions != nil {
		if exhausted != nil {
			if err := exhausted(s.q); err != nil {
				return SessionView{}, err
			}
		}
		return SessionView{}, noQuestions
	}

	var sess store.StudySession
	err = s.tx(ctx, func(q *store.Queries) error {
		var err error
		sess, err = q.CreateStudySession(ctx, params)
		if err != nil {
			return err
		}
		for i, p := range picks {
			if err := q.AddSessionQuestion(ctx, store.AddSessionQuestionParams{
				SessionID: sess.ID, QuestionID: p.ID, Position: int32(i + 1), IsRepeat: p.IsRepeat,
			}); err != nil {
				return err
			}
		}
		if exhausted != nil {
			return exhausted(q)
		}
		return nil
	})
	if err != nil {
		return SessionView{}, err
	}
	return sessionView(sess, 0), nil
}

func sessionView(s store.StudySession, answered int) SessionView {
	v := SessionView{
		ID: s.ID, Kind: s.Kind, CourseID: s.CourseID, ConceptID: s.ConceptID, TopicID: s.TopicID, ExamID: s.ExamID,
		AnsweredCount: answered, StartedAt: s.StartedAt, Deadline: s.Deadline, FinishedAt: s.FinishedAt,
	}
	if s.Kind != KindPlacement && s.MaxQuestions != nil {
		v.TotalQuestions = ptr(int(*s.MaxQuestions))
	}
	return v
}

// loadSession fetches a session the Actor may read (own, or any for the Owner).
func (s *Service) loadSession(ctx context.Context, q *store.Queries, a Actor, id uuid.UUID, forUpdate, mutate bool) (store.StudySession, error) {
	var sess store.StudySession
	var err error
	if forUpdate {
		sess, err = q.GetStudySessionForUpdate(ctx, id)
	} else {
		sess, err = q.GetStudySession(ctx, id)
	}
	if err != nil {
		return sess, mapNotFound(err, "session")
	}
	if sess.LearnerID != a.LearnerID && (mutate || !a.IsOwner) {
		return sess, notFound("session") // never reveal other Learners' sessions
	}
	return sess, nil
}

func (s *Service) deadlinePassed(sess store.StudySession) bool {
	return sess.Deadline != nil && !s.now().Before(*sess.Deadline)
}

// GetSession returns a session's state.
func (s *Service) GetSession(ctx context.Context, a Actor, id uuid.UUID) (SessionView, error) {
	sess, err := s.loadSession(ctx, s.q, a, id, false, false)
	if err != nil {
		return SessionView{}, err
	}
	n, err := s.q.CountSessionAnswers(ctx, id)
	if err != nil {
		return SessionView{}, err
	}
	return sessionView(sess, int(n)), nil
}

func (s *Service) checkOpen(sess store.StudySession) error {
	if sess.FinishedAt != nil {
		return E(KindGone, "session_finished", "this session is finished")
	}
	if sess.Kind == KindExam && s.deadlinePassed(sess) {
		return E(KindGone, "deadline_passed", "the Exam time limit has passed")
	}
	return nil
}

// NextQuestion returns the next unanswered Question, or nil when there are no more.
func (s *Service) NextQuestion(ctx context.Context, a Actor, id uuid.UUID) (*ServedQuestion, error) {
	var served *ServedQuestion
	err := s.tx(ctx, func(q *store.Queries) error {
		sess, err := s.loadSession(ctx, q, a, id, true, true)
		if err != nil {
			return err
		}
		if err := s.checkOpen(sess); err != nil {
			return err
		}
		sqs, err := q.ListSessionQuestions(ctx, id)
		if err != nil {
			return err
		}
		ids := make([]uuid.UUID, 0, len(sqs))
		for _, sq := range sqs {
			ids = append(ids, sq.QuestionID)
		}
		qs, err := questionsByID(ctx, q, ids)
		if err != nil {
			return err
		}
		for _, sq := range sqs {
			if sq.AnswerID != nil {
				continue
			}
			if qs[sq.QuestionID].Status != "active" {
				continue // pulled (Flagged/Retired) after the session started
			}
			served, err = serveQuestion(ctx, q, qs[sq.QuestionID], int(sq.Position), sq.IsRepeat)
			return err
		}
		if sess.Kind == KindPlacement {
			p, err := s.nextPlacementProbe(ctx, q, sess, sqs)
			if err != nil || p == nil {
				return err
			}
			qq, err := q.GetQuestion(ctx, p.ID)
			if err != nil {
				return err
			}
			served, err = serveQuestion(ctx, q, qq, len(sqs)+1, p.IsRepeat)
			return err
		}
		return nil
	})
	return served, err
}

func questionsByID(ctx context.Context, q *store.Queries, ids []uuid.UUID) (map[uuid.UUID]store.Question, error) {
	rows, err := q.ListQuestionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	m := make(map[uuid.UUID]store.Question, len(rows))
	for i := range rows {
		r := &rows[i]
		m[r.ID] = *r
	}
	return m, nil
}

func serveQuestion(ctx context.Context, q *store.Queries, qq store.Question, position int, repeat bool) (*ServedQuestion, error) {
	choices, err := q.ListChoicesForQuestions(ctx, []uuid.UUID{qq.ID})
	if err != nil {
		return nil, err
	}
	sq := &ServedQuestion{
		ID: qq.ID, StemMd: qq.StemMd, Format: qq.Format, Difficulty: int(qq.Difficulty), Position: position, IsRepeat: repeat,
	}
	for _, c := range choices {
		sq.Choices = append(sq.Choices, ServedChoice{ID: c.ID, Label: c.Label, BodyMd: c.BodyMd})
	}
	return sq, nil
}

// SubmitAnswerInput is one Answer.
type SubmitAnswerInput struct {
	SessionID   uuid.UUID
	QuestionID  uuid.UUID
	ChoiceID    uuid.UUID
	TimeSpentMs *int
}

// SubmitAnswer records an Answer, recomputes Mastery for the Question's
// Concepts, and returns feedback (non-Exam sessions only).
func (s *Service) SubmitAnswer(ctx context.Context, a Actor, in SubmitAnswerInput) (AnswerResult, error) {
	var res AnswerResult
	var courseID uuid.UUID
	err := s.tx(ctx, func(q *store.Queries) error {
		sess, err := s.loadSession(ctx, q, a, in.SessionID, true, true)
		if err != nil {
			return err
		}
		if sess.FinishedAt != nil {
			return E(KindConflict, "session_finished", "this session is finished")
		}
		if sess.Kind == KindExam && s.deadlinePassed(sess) {
			return E(KindGone, "deadline_passed", "the Exam time limit has passed")
		}
		courseID = sess.CourseID
		sqs, err := q.ListSessionQuestions(ctx, sess.ID)
		if err != nil {
			return err
		}
		var found *store.ListSessionQuestionsRow
		for i := range sqs {
			if sqs[i].QuestionID == in.QuestionID {
				found = &sqs[i]
			}
		}
		if found == nil {
			return E(KindConflict, "not_in_session", "this Question is not part of the session")
		}
		if found.AnswerID != nil {
			return E(KindConflict, "already_answered", "this Question was already answered in this session")
		}
		question, err := q.GetQuestion(ctx, in.QuestionID)
		if err != nil {
			return err
		}
		choices, err := q.ListChoicesForQuestions(ctx, []uuid.UUID{question.ID})
		if err != nil {
			return err
		}
		var chosen, correct *store.QuestionChoice
		for i := range choices {
			if choices[i].ID == in.ChoiceID {
				chosen = &choices[i]
			}
			if choices[i].IsCorrect {
				correct = &choices[i]
			}
		}
		if chosen == nil {
			return E(KindInvalid, "invalid_choice", "the choice does not belong to this Question")
		}
		repeat, err := q.HasAnswered(ctx, store.HasAnsweredParams{LearnerID: a.LearnerID, QuestionID: question.ID})
		if err != nil {
			return err
		}
		var spent *int32
		if in.TimeSpentMs != nil {
			if *in.TimeSpentMs < 0 {
				return E(KindInvalid, "invalid_time", "timeSpentMs must be >= 0")
			}
			spent = ptr(int32(min(*in.TimeSpentMs, 1<<31-1)))
		}
		if _, err := q.InsertAnswer(ctx, store.InsertAnswerParams{
			LearnerID: a.LearnerID, QuestionID: question.ID, SessionID: sess.ID, ChoiceID: chosen.ID,
			Correct: chosen.IsCorrect, IsRepeat: repeat, TimeSpentMs: spent,
		}); err != nil {
			return err
		}
		tags, err := q.ListConceptsForQuestions(ctx, []uuid.UUID{question.ID})
		if err != nil {
			return err
		}
		concepts := make([]uuid.UUID, 0, len(tags))
		for _, t := range tags {
			concepts = append(concepts, t.ConceptID)
		}
		changes, err := s.recomputeMastery(ctx, q, a.LearnerID, sess.CourseID, concepts)
		if err != nil {
			return err
		}
		if err := s.checkPathCompletion(ctx, q, a.LearnerID, sess.CourseID); err != nil {
			return err
		}
		res.Recorded = true
		if sess.Kind == KindExam {
			return nil // no feedback until the Exam is finished
		}
		res.Correct = ptr(chosen.IsCorrect)
		if correct != nil {
			res.CorrectChoiceID = ptr(correct.ID)
		}
		res.ExplanationMd = ptr(question.ExplanationMd)
		res.MasteryChanges = []MasteryChangeView{}
		for _, c := range changes {
			res.MasteryChanges = append(res.MasteryChanges, MasteryChangeView{ConceptID: c.ConceptID, Before: c.Before, After: c.After})
		}
		return nil
	})
	if err != nil {
		return AnswerResult{}, err
	}
	if len(res.MasteryChanges) > 0 {
		g, err := s.Graph(ctx, courseID)
		if err != nil {
			return AnswerResult{}, err
		}
		for i := range res.MasteryChanges {
			res.MasteryChanges[i].ConceptName = g.Name(res.MasteryChanges[i].ConceptID)
		}
	}
	return res, nil
}

// FinishSession marks a session finished and returns its result.
func (s *Service) FinishSession(ctx context.Context, a Actor, id uuid.UUID) (SessionResult, error) {
	err := s.tx(ctx, func(q *store.Queries) error {
		if _, err := s.loadSession(ctx, q, a, id, true, true); err != nil {
			return err
		}
		_, err := q.FinishStudySession(ctx, id)
		return err
	})
	if err != nil {
		return SessionResult{}, err
	}
	return s.SessionResult(ctx, a, id)
}

// SessionResult returns the result of a finished session, including the
// answer keys and Explanations.
func (s *Service) SessionResult(ctx context.Context, a Actor, id uuid.UUID) (SessionResult, error) {
	sess, err := s.loadSession(ctx, s.q, a, id, false, false)
	if err != nil {
		return SessionResult{}, err
	}
	if sess.FinishedAt == nil {
		return SessionResult{}, E(KindConflict, "not_finished", "the session is not finished yet")
	}
	g, err := s.Graph(ctx, sess.CourseID)
	if err != nil {
		return SessionResult{}, err
	}
	sqs, err := s.q.ListSessionQuestions(ctx, id)
	if err != nil {
		return SessionResult{}, err
	}
	ids := make([]uuid.UUID, 0, len(sqs))
	for _, sq := range sqs {
		ids = append(ids, sq.QuestionID)
	}
	qs, err := questionsByID(ctx, s.q, ids)
	if err != nil {
		return SessionResult{}, err
	}
	choices, err := s.q.ListChoicesForQuestions(ctx, ids)
	if err != nil {
		return SessionResult{}, err
	}
	correctChoice := map[uuid.UUID]uuid.UUID{}
	for _, c := range choices {
		if c.IsCorrect {
			correctChoice[c.QuestionID] = c.ID
		}
	}
	tags, err := s.q.ListConceptsForQuestions(ctx, ids)
	if err != nil {
		return SessionResult{}, err
	}
	primary := map[uuid.UUID]uuid.UUID{}
	touched := map[domain.ConceptID]bool{}
	for _, t := range tags {
		if t.IsPrimary {
			primary[t.QuestionID] = t.ConceptID
		}
	}

	res := SessionResult{SessionID: id, Kind: sess.Kind, Total: len(sqs), Review: []ReviewItem{}, TopicBreakdown: []TopicStat{}}
	topicStats := map[uuid.UUID]*TopicStat{}
	answeredQ := map[uuid.UUID]bool{}
	for _, sq := range sqs {
		qq := qs[sq.QuestionID]
		item := ReviewItem{
			QuestionID: qq.ID, StemMd: qq.StemMd, ChosenChoiceID: sq.ChoiceID,
			CorrectChoiceID: correctChoice[qq.ID], ExplanationMd: qq.ExplanationMd,
		}
		if sq.AnswerID != nil {
			answeredQ[qq.ID] = true
			res.Answered++
			item.Correct = sq.Correct != nil && *sq.Correct
			if item.Correct {
				res.Correct++
			}
			if c, ok := g.ConceptByID[primary[qq.ID]]; ok {
				ts := topicStats[c.TopicID]
				if ts == nil {
					ts = &TopicStat{TopicID: c.TopicID, TopicName: g.TopicByID[c.TopicID].Name}
					topicStats[c.TopicID] = ts
				}
				ts.Answered++
				if item.Correct {
					ts.Correct++
				}
			}
		}
		res.Review = append(res.Review, item)
	}
	for _, t := range tags {
		if answeredQ[t.QuestionID] {
			touched[cid(t.ConceptID)] = true
		}
	}
	for _, t := range g.Topics {
		if ts, ok := topicStats[t.ID]; ok {
			res.TopicBreakdown = append(res.TopicBreakdown, *ts)
		}
	}
	// Root Causes: Recommendations whose reason path touches this session's Concepts.
	recs, err := s.recommendations(ctx, s.q, sess.LearnerID, g)
	if err != nil {
		return SessionResult{}, err
	}
	res.RootCauses = []RecommendationView{}
	for _, r := range recs {
		for _, p := range r.Path {
			if touched[cid(p.ID)] {
				res.RootCauses = append(res.RootCauses, r)
				break
			}
		}
	}
	sort.SliceStable(res.RootCauses, func(i, j int) bool { return res.RootCauses[i].Priority > res.RootCauses[j].Priority })
	return res, nil
}
