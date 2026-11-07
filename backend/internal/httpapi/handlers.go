package httpapi

import (
	"context"
	"strings"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/auth"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// ── auth ────────────────────────────────────────────────────────────────

// GoogleLogin redirects to Google. The spec declares no Location header, so
// it is written directly on the ResponseWriter.
func (s *Server) GoogleLogin(ctx context.Context, req GoogleLoginRequestObject) (GoogleLoginResponseObject, error) {
	invite := ""
	if req.Params.Invite != nil {
		invite = *req.Params.Invite
	}
	u, cookie, err := s.auth.BeginLogin(ctx, invite)
	if err != nil {
		return nil, err
	}
	setCookie(ctx, cookie)
	setHeader(ctx, "Location", u)
	return GoogleLogin302Response{}, nil
}

// GoogleCallback completes login, sets the session cookie, and redirects to the web app.
func (s *Server) GoogleCallback(ctx context.Context, req GoogleCallbackRequestObject) (GoogleCallbackResponseObject, error) {
	stateCookie := ""
	if r := requestFrom(ctx); r != nil {
		if c, err := r.Cookie(auth.StateCookie); err == nil {
			stateCookie = c.Value
		}
	}
	setCookie(ctx, s.auth.ClearStateCookie())
	l, err := s.auth.CompleteLogin(ctx, req.Params.Code, req.Params.State, stateCookie)
	if err != nil {
		return nil, err
	}
	c, err := s.auth.StartSession(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	setCookie(ctx, c)
	setHeader(ctx, "Location", strings.TrimRight(s.cfg.AppBaseURL, "/")+"/")
	return GoogleCallback302Response{}, nil
}

// DevLogin logs in without Google when AUTH_DEV_LOGIN=true.
func (s *Server) DevLogin(ctx context.Context, req DevLoginRequestObject) (DevLoginResponseObject, error) {
	if req.Body == nil {
		return nil, app.E(app.KindBadRequest, "bad_request", "body required")
	}
	name := ""
	if req.Body.Name != nil {
		name = *req.Body.Name
	}
	l, err := s.auth.DevLogin(ctx, string(req.Body.Email), name)
	if err != nil {
		return nil, err
	}
	c, err := s.auth.StartSession(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	setCookie(ctx, c)
	return DevLogin204Response{}, nil
}

// Logout ends the session.
func (s *Server) Logout(ctx context.Context, _ LogoutRequestObject) (LogoutResponseObject, error) {
	token := ""
	if r := requestFrom(ctx); r != nil {
		if c, err := r.Cookie(sessionCookieName); err == nil {
			token = c.Value
		}
	}
	c, err := s.auth.EndSession(ctx, token)
	if err != nil {
		return nil, err
	}
	setCookie(ctx, c)
	return Logout204Response{}, nil
}

func learnerJSON(l store.Learner, placementDone *bool) Learner {
	return Learner{Id: l.ID, Email: openapi_types.Email(l.Email), Name: l.Name, IsOwner: l.IsOwner, PlacementDone: placementDone}
}

// GetMe returns the current Learner.
func (s *Server) GetMe(ctx context.Context, _ GetMeRequestObject) (GetMeResponseObject, error) {
	l, ok := learnerFrom(ctx)
	if !ok {
		return nil, app.E(app.KindUnauthorized, "unauthenticated", "login required")
	}
	done, err := s.app.LearnerPlacementDone(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	return GetMe200JSONResponse(learnerJSON(l, &done)), nil
}

// ── courses ─────────────────────────────────────────────────────────────

func courseSummary(c store.Course) CourseSummary {
	return CourseSummary{Id: c.ID, Slug: c.Slug, Name: c.Name, Goal: &c.Goal}
}

// ListCourses lists Courses.
func (s *Server) ListCourses(ctx context.Context, _ ListCoursesRequestObject) (ListCoursesResponseObject, error) {
	cs, err := s.app.ListCourses(ctx)
	if err != nil {
		return nil, err
	}
	out := ListCourses200JSONResponse{}
	for _, c := range cs {
		out = append(out, courseSummary(c))
	}
	return out, nil
}

func conceptJSON(c app.ConceptView) Concept {
	return Concept{Id: c.ID, Slug: c.Slug, Name: c.Name, TopicId: c.TopicID, PrerequisiteIds: c.PrerequisiteIDs, HasLesson: &c.HasLesson}
}

func f32p(v *float64) *float32 {
	if v == nil {
		return nil
	}
	f := float32(*v)
	return &f
}

// GetCourse returns the Course graph.
func (s *Server) GetCourse(ctx context.Context, req GetCourseRequestObject) (GetCourseResponseObject, error) {
	g, err := s.app.CourseGraph(ctx, req.CourseId)
	if err != nil {
		return nil, err
	}
	out := GetCourse200JSONResponse{
		Id: g.Course.ID, Slug: g.Course.Slug, Name: g.Course.Name, Goal: &g.Course.Goal,
		Topics: []Topic{}, Concepts: []Concept{}, QuestionFormats: []QuestionFormat{},
	}
	for _, t := range g.Topics {
		out.Topics = append(out.Topics, Topic{Id: t.ID, Slug: t.Slug, Name: t.Name, ExamWeight: f32p(t.ExamWeight)})
	}
	for _, c := range g.Concepts {
		out.Concepts = append(out.Concepts, conceptJSON(c))
	}
	for _, f := range g.Formats {
		out.QuestionFormats = append(out.QuestionFormats, QuestionFormat{Slug: f.Slug, Name: f.Name})
	}
	return out, nil
}

// ListExams lists a Course's Exams.
func (s *Server) ListExams(ctx context.Context, req ListExamsRequestObject) (ListExamsResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	exams, err := s.app.ListExams(ctx, a, req.CourseId)
	if err != nil {
		return nil, err
	}
	out := ListExams200JSONResponse{}
	for _, e := range exams {
		out = append(out, ExamSummary{Id: e.ID, Name: e.Name, TimeLimitSeconds: e.TimeLimitSeconds, QuestionCount: e.QuestionCount, LastScore: f32p(e.LastScore)})
	}
	return out, nil
}

func masteryJSON(m app.MasteryView) Mastery {
	return Mastery{ConceptId: m.ConceptID, Score: float32(m.Score), Evidence: float32(m.Evidence), Level: MasteryLevel(m.Level)}
}

// GetConcept returns a Concept with Lesson and Mastery.
func (s *Server) GetConcept(ctx context.Context, req GetConceptRequestObject) (GetConceptResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.app.ConceptDetail(ctx, a, req.ConceptId)
	if err != nil {
		return nil, err
	}
	return GetConcept200JSONResponse{
		Id: d.ID, Slug: d.Slug, Name: d.Name, TopicId: d.TopicID, PrerequisiteIds: d.PrerequisiteIDs, HasLesson: &d.HasLesson,
		TopicName: &d.TopicName, LessonMd: d.LessonMd, DependentIds: d.DependentIDs, Mastery: masteryJSON(d.Mastery),
		ActiveQuestionCount: &d.ActiveQuestionCount,
	}, nil
}

// ── progress ────────────────────────────────────────────────────────────

// GetMastery returns the Learner's Mastery Map data.
func (s *Server) GetMastery(ctx context.Context, req GetMasteryRequestObject) (GetMasteryResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	ms, err := s.app.CourseMastery(ctx, a, req.CourseId)
	if err != nil {
		return nil, err
	}
	out := GetMastery200JSONResponse{}
	for _, m := range ms {
		out = append(out, masteryJSON(m))
	}
	return out, nil
}

func refsJSON(rs []app.ConceptRef) []ConceptRef {
	out := make([]ConceptRef, 0, len(rs))
	for _, r := range rs {
		lvl := MasteryLevel(r.Level)
		out = append(out, ConceptRef{Id: r.ID, Name: r.Name, Level: &lvl})
	}
	return out
}

func recommendationJSON(r app.RecommendationView) Recommendation {
	target := r.TargetID
	return Recommendation{
		Kind: RecommendationKind(r.Kind), ConceptId: r.ConceptID, ConceptName: r.ConceptName, TargetConceptId: &target,
		Path: refsJSON(r.Path), Unblocks: r.Unblocks, Priority: float32(r.Priority),
	}
}

// GetRecommendations returns Recommendations by Priority.
func (s *Server) GetRecommendations(ctx context.Context, req GetRecommendationsRequestObject) (GetRecommendationsResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	recs, err := s.app.Recommendations(ctx, a, req.CourseId)
	if err != nil {
		return nil, err
	}
	out := GetRecommendations200JSONResponse{}
	for _, r := range recs {
		out = append(out, recommendationJSON(r))
	}
	return out, nil
}

// GetFormatReport returns accuracy per Question Format.
func (s *Server) GetFormatReport(ctx context.Context, req GetFormatReportRequestObject) (GetFormatReportResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := s.app.FormatReport(ctx, a, req.CourseId)
	if err != nil {
		return nil, err
	}
	out := GetFormatReport200JSONResponse{}
	for _, st := range stats {
		out = append(out, FormatStat{Format: st.Format, FormatName: st.FormatName, Answered: st.Answered, Correct: st.Correct, Accuracy: float32(st.Accuracy)})
	}
	return out, nil
}

func pathJSON(p app.LearningPathView) LearningPath {
	return LearningPath{Id: p.ID, CourseId: p.CourseID, Steps: refsJSON(p.Steps), CurrentStepConceptId: p.CurrentStep, StartedAt: p.StartedAt, CompletedAt: p.CompletedAt}
}

// GetLearningPath returns the active Learning Path.
func (s *Server) GetLearningPath(ctx context.Context, req GetLearningPathRequestObject) (GetLearningPathResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.app.LearningPath(ctx, a, req.CourseId)
	if err != nil {
		return nil, err
	}
	return GetLearningPath200JSONResponse(pathJSON(p)), nil
}

// StartLearningPath starts a Learning Path.
func (s *Server) StartLearningPath(ctx context.Context, req StartLearningPathRequestObject) (StartLearningPathResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, app.E(app.KindBadRequest, "bad_request", "body required")
	}
	p, err := s.app.StartLearningPath(ctx, a, req.CourseId, req.Body.RootConceptId, req.Body.TargetConceptId)
	if err != nil {
		return nil, err
	}
	return StartLearningPath201JSONResponse(pathJSON(p)), nil
}

// AbandonLearningPath abandons the active Learning Path.
func (s *Server) AbandonLearningPath(ctx context.Context, req AbandonLearningPathRequestObject) (AbandonLearningPathResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.app.AbandonLearningPath(ctx, a, req.CourseId); err != nil {
		return nil, err
	}
	return AbandonLearningPath204Response{}, nil
}

// ── sessions ────────────────────────────────────────────────────────────

func sessionJSON(v app.SessionView) StudySession {
	return StudySession{
		Id: v.ID, Kind: SessionKind(v.Kind), CourseId: v.CourseID, ConceptId: v.ConceptID, TopicId: v.TopicID, ExamId: v.ExamID,
		TotalQuestions: v.TotalQuestions, AnsweredCount: v.AnsweredCount, StartedAt: v.StartedAt, Deadline: v.Deadline, FinishedAt: v.FinishedAt,
	}
}

// CreateSession starts a session.
func (s *Server) CreateSession(ctx context.Context, req CreateSessionRequestObject) (CreateSessionResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, app.E(app.KindBadRequest, "bad_request", "body required")
	}
	v, err := s.app.CreateSession(ctx, a, app.CreateSessionInput{
		Kind: string(req.Body.Kind), CourseID: req.Body.CourseId, ConceptID: req.Body.ConceptId, TopicID: req.Body.TopicId, ExamID: req.Body.ExamId,
	})
	if err != nil {
		return nil, err
	}
	return CreateSession201JSONResponse(sessionJSON(v)), nil
}

// GetSession returns session state.
func (s *Server) GetSession(ctx context.Context, req GetSessionRequestObject) (GetSessionResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.app.GetSession(ctx, a, req.SessionId)
	if err != nil {
		return nil, err
	}
	return GetSession200JSONResponse(sessionJSON(v)), nil
}

// GetNextQuestion serves the next Question without its key.
func (s *Server) GetNextQuestion(ctx context.Context, req GetNextQuestionRequestObject) (GetNextQuestionResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	q, err := s.app.NextQuestion(ctx, a, req.SessionId)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return GetNextQuestion204Response{}, nil
	}
	out := GetNextQuestion200JSONResponse{Id: q.ID, StemMd: q.StemMd, Format: q.Format, Difficulty: &q.Difficulty, Position: q.Position}
	out.Choices = make([]struct {
		BodyMd string             `json:"bodyMd"`
		Id     openapi_types.UUID `json:"id"`
		Label  string             `json:"label"`
	}, 0, len(q.Choices))
	for _, c := range q.Choices {
		out.Choices = append(out.Choices, struct {
			BodyMd string             `json:"bodyMd"`
			Id     openapi_types.UUID `json:"id"`
			Label  string             `json:"label"`
		}{BodyMd: c.BodyMd, Id: c.ID, Label: c.Label})
	}
	return out, nil
}

// SubmitAnswer records an Answer.
func (s *Server) SubmitAnswer(ctx context.Context, req SubmitAnswerRequestObject) (SubmitAnswerResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, app.E(app.KindBadRequest, "bad_request", "body required")
	}
	res, err := s.app.SubmitAnswer(ctx, a, app.SubmitAnswerInput{
		SessionID: req.SessionId, QuestionID: req.Body.QuestionId, ChoiceID: req.Body.ChoiceId, TimeSpentMs: req.Body.TimeSpentMs,
	})
	if err != nil {
		return nil, err
	}
	out := SubmitAnswer200JSONResponse{Recorded: res.Recorded, Correct: res.Correct, CorrectChoiceId: res.CorrectChoiceID, ExplanationMd: res.ExplanationMd}
	if res.MasteryChanges != nil {
		changes := make([]struct {
			After       MasteryLevel       `json:"after"`
			Before      MasteryLevel       `json:"before"`
			ConceptId   openapi_types.UUID `json:"conceptId"`
			ConceptName string             `json:"conceptName"`
		}, 0, len(res.MasteryChanges))
		for _, c := range res.MasteryChanges {
			changes = append(changes, struct {
				After       MasteryLevel       `json:"after"`
				Before      MasteryLevel       `json:"before"`
				ConceptId   openapi_types.UUID `json:"conceptId"`
				ConceptName string             `json:"conceptName"`
			}{After: MasteryLevel(c.After), Before: MasteryLevel(c.Before), ConceptId: c.ConceptID, ConceptName: c.ConceptName})
		}
		out.MasteryChanges = &changes
	}
	return out, nil
}

func resultJSON(r app.SessionResult) SessionResult {
	out := SessionResult{
		SessionId: r.SessionID, Kind: SessionKind(r.Kind), Answered: r.Answered, Correct: r.Correct, Total: &r.Total,
		RootCauses: []Recommendation{},
	}
	for _, rc := range r.RootCauses {
		out.RootCauses = append(out.RootCauses, recommendationJSON(rc))
	}
	out.TopicBreakdown = make([]struct {
		Answered  int                `json:"answered"`
		Correct   int                `json:"correct"`
		TopicId   openapi_types.UUID `json:"topicId"`
		TopicName string             `json:"topicName"`
	}, 0, len(r.TopicBreakdown))
	for _, t := range r.TopicBreakdown {
		out.TopicBreakdown = append(out.TopicBreakdown, struct {
			Answered  int                `json:"answered"`
			Correct   int                `json:"correct"`
			TopicId   openapi_types.UUID `json:"topicId"`
			TopicName string             `json:"topicName"`
		}{Answered: t.Answered, Correct: t.Correct, TopicId: t.TopicID, TopicName: t.TopicName})
	}
	out.Review = make([]struct {
		ChosenChoiceId  *openapi_types.UUID `json:"chosenChoiceId"`
		Correct         bool                `json:"correct"`
		CorrectChoiceId openapi_types.UUID  `json:"correctChoiceId"`
		ExplanationMd   string              `json:"explanationMd"`
		QuestionId      openapi_types.UUID  `json:"questionId"`
		StemMd          string              `json:"stemMd"`
	}, 0, len(r.Review))
	for _, it := range r.Review {
		out.Review = append(out.Review, struct {
			ChosenChoiceId  *openapi_types.UUID `json:"chosenChoiceId"`
			Correct         bool                `json:"correct"`
			CorrectChoiceId openapi_types.UUID  `json:"correctChoiceId"`
			ExplanationMd   string              `json:"explanationMd"`
			QuestionId      openapi_types.UUID  `json:"questionId"`
			StemMd          string              `json:"stemMd"`
		}{ChosenChoiceId: it.ChosenChoiceID, Correct: it.Correct, CorrectChoiceId: it.CorrectChoiceID, ExplanationMd: it.ExplanationMd, QuestionId: it.QuestionID, StemMd: it.StemMd})
	}
	return out
}

// FinishSession finishes a session and returns its result.
func (s *Server) FinishSession(ctx context.Context, req FinishSessionRequestObject) (FinishSessionResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.app.FinishSession(ctx, a, req.SessionId)
	if err != nil {
		return nil, err
	}
	return FinishSession200JSONResponse(resultJSON(r)), nil
}

// GetSessionResult returns a finished session's result.
func (s *Server) GetSessionResult(ctx context.Context, req GetSessionResultRequestObject) (GetSessionResultResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.app.SessionResult(ctx, a, req.SessionId)
	if err != nil {
		return nil, err
	}
	return GetSessionResult200JSONResponse(resultJSON(r)), nil
}

// FlagQuestion flags a Question.
func (s *Server) FlagQuestion(ctx context.Context, req FlagQuestionRequestObject) (FlagQuestionResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, app.E(app.KindBadRequest, "bad_request", "body required")
	}
	if err := s.app.FlagQuestion(ctx, a, req.QuestionId, req.Body.Reason); err != nil {
		return nil, err
	}
	return FlagQuestion201Response{}, nil
}

// ── admin (Owner only; enforced in app/auth) ────────────────────────────

// ListInvites lists invites.
func (s *Server) ListInvites(ctx context.Context, _ ListInvitesRequestObject) (ListInvitesResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	invs, err := s.auth.ListInvites(ctx, a)
	if err != nil {
		return nil, err
	}
	out := ListInvites200JSONResponse{}
	for _, i := range invs {
		out = append(out, Invite{Id: i.ID, CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt, UsedBy: i.UsedBy})
	}
	return out, nil
}

// CreateInvite creates an invite.
func (s *Server) CreateInvite(ctx context.Context, _ CreateInviteRequestObject) (CreateInviteResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	i, err := s.auth.CreateInvite(ctx, a)
	if err != nil {
		return nil, err
	}
	return CreateInvite201JSONResponse{Id: i.ID, CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt, Url: i.URL}, nil
}

// ListLearners lists Learners.
func (s *Server) ListLearners(ctx context.Context, _ ListLearnersRequestObject) (ListLearnersResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	ls, err := s.app.ListLearners(ctx, a)
	if err != nil {
		return nil, err
	}
	out := ListLearners200JSONResponse{}
	for _, l := range ls {
		out = append(out, learnerJSON(l, nil))
	}
	return out, nil
}

// ListFlags lists Flags.
func (s *Server) ListFlags(ctx context.Context, req ListFlagsRequestObject) (ListFlagsResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	open := true
	if req.Params.Open != nil {
		open = *req.Params.Open
	}
	fs, err := s.app.ListFlags(ctx, a, open)
	if err != nil {
		return nil, err
	}
	out := ListFlags200JSONResponse{}
	for i := range fs {
		f := &fs[i]
		name := f.LearnerName
		out = append(out, Flag{Id: f.ID, QuestionId: f.QuestionID, LearnerName: &name, Reason: f.Reason, CreatedAt: f.CreatedAt, ResolvedAt: f.ResolvedAt})
	}
	return out, nil
}

// ListContentRequests lists open Content Requests.
func (s *Server) ListContentRequests(ctx context.Context, _ ListContentRequestsRequestObject) (ListContentRequestsResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	rs, err := s.app.ListContentRequests(ctx, a)
	if err != nil {
		return nil, err
	}
	out := ListContentRequests200JSONResponse{}
	for i := range rs {
		r := &rs[i]
		out = append(out, ContentRequest{Id: r.ID, ConceptId: r.ConceptID, ConceptName: r.ConceptName, Reason: r.Reason, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

// RetireQuestion retires a Question.
func (s *Server) RetireQuestion(ctx context.Context, req RetireQuestionRequestObject) (RetireQuestionResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.app.RetireQuestionAsOwner(ctx, a, req.QuestionId); err != nil {
		return nil, err
	}
	return RetireQuestion204Response{}, nil
}
