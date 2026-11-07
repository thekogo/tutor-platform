package app

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/domain"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Placement Tests are adaptive: each /next call picks the next Concept to probe
// with domain.NextPlacementConcept and serves one Question whose Primary
// Concept it is.
//
// Probe selection needs a level for each probed Concept after a single
// Answer, which the standard Mastery config (MinEvidence 3.0) would leave
// Unassessed. So probe selection uses a placement-local estimate computed only
// from this session's Answers with MinEvidence 0.5 (one primary Answer decides
// Weak vs Mastered). That estimate is NOT persisted: the stored Mastery stays a
// pure projection of the Answer history (ADR 0002) — placement Answers are
// ordinary Answers and count toward it like any other.

func (s *Service) nextPlacementProbe(ctx context.Context, q *store.Queries, sess store.StudySession, sqs []store.ListSessionQuestionsRow) (*pickedQuestion, error) {
	maxProbes := PlacementMaxProbes
	if sess.MaxQuestions != nil {
		maxProbes = int(*sess.MaxQuestions)
	}
	if len(sqs) >= maxProbes {
		return nil, nil
	}
	g, err := s.Graph(ctx, sess.CourseID)
	if err != nil {
		return nil, err
	}
	rows, err := q.SessionEvidence(ctx, sess.ID)
	if err != nil {
		return nil, err
	}
	ev := make([]domain.AnswerEvidence, 0, len(rows))
	for _, r := range rows {
		ev = append(ev, domain.AnswerEvidence{
			QuestionID: r.QuestionID.String(), Correct: r.Correct, AnsweredAt: r.AnsweredAt,
			Difficulty: int(r.Difficulty), PrimaryConcept: cid(r.PrimaryConceptID),
		})
	}
	local := domain.ComputeMastery(ev, s.placementCfg)
	asked := map[domain.ConceptID]bool{}
	for _, sq := range sqs {
		if sq.ConceptID != nil {
			asked[cid(*sq.ConceptID)] = true
		}
	}
	availIDs, err := q.ConceptsWithPrimaryQuestions(ctx, sess.CourseID)
	if err != nil {
		return nil, err
	}
	available := map[domain.ConceptID]bool{}
	for _, id := range availIDs {
		if g.Live(id) {
			available[cid(id)] = true
		}
	}
	for {
		next, ok := domain.NextPlacementConcept(g.Graph, local, asked, available, maxProbes)
		if !ok {
			return nil, nil
		}
		conceptID := ucid(next)
		pick, err := pickPlacementQuestion(ctx, q, sess.LearnerID, conceptID, sqs)
		if err != nil {
			return nil, err
		}
		if pick == nil { // every candidate already used in this session
			delete(available, next)
			continue
		}
		if err := q.AddSessionQuestion(ctx, store.AddSessionQuestionParams{
			SessionID: sess.ID, QuestionID: pick.ID, Position: int32(len(sqs) + 1), //nolint:gosec // bounded by PlacementMaxProbes
			ConceptID: &conceptID, IsRepeat: pick.IsRepeat,
		}); err != nil {
			return nil, err
		}
		return pick, nil
	}
}

// pickPlacementQuestion prefers unanswered, medium-difficulty Questions.
func pickPlacementQuestion(ctx context.Context, q *store.Queries, learnerID, conceptID uuid.UUID, sqs []store.ListSessionQuestionsRow) (*pickedQuestion, error) {
	cands, err := q.PrimaryCandidates(ctx, store.PrimaryCandidatesParams{ConceptID: conceptID, LearnerID: learnerID})
	if err != nil {
		return nil, err
	}
	used := map[uuid.UUID]bool{}
	for _, sq := range sqs {
		used[sq.QuestionID] = true
	}
	diffRank := map[int16]int{2: 0, 1: 1, 3: 2}
	var pool []store.PrimaryCandidatesRow
	for _, c := range cands {
		if !used[c.ID] {
			pool = append(pool, c)
		}
	}
	if len(pool) == 0 {
		return nil, nil
	}
	sort.Slice(pool, func(i, j int) bool {
		fi, fj := pool[i].TimesAnswered == 0, pool[j].TimesAnswered == 0
		if fi != fj {
			return fi
		}
		if diffRank[pool[i].Difficulty] != diffRank[pool[j].Difficulty] {
			return diffRank[pool[i].Difficulty] < diffRank[pool[j].Difficulty]
		}
		return pool[i].ID.String() < pool[j].ID.String()
	})
	return &pickedQuestion{ID: pool[0].ID, IsRepeat: pool[0].TimesAnswered > 0}, nil
}
