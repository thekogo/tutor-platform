package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/domain"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// ListCourses returns every Course.
func (s *Service) ListCourses(ctx context.Context) ([]store.Course, error) {
	return s.q.ListCourses(ctx)
}

// ConceptView is a live Concept with its effective Prerequisites.
type ConceptView struct {
	ID              uuid.UUID
	Slug            string
	Name            string
	TopicID         uuid.UUID
	PrerequisiteIDs []uuid.UUID
	HasLesson       bool
}

// CourseGraphView is the knowledge graph for the web app.
type CourseGraphView struct {
	Course   store.Course
	Topics   []store.Topic
	Concepts []ConceptView
	Formats  []store.QuestionFormat
}

// CourseGraph returns a Course's Topics, live Concepts, and Prerequisites.
func (s *Service) CourseGraph(ctx context.Context, courseID uuid.UUID) (CourseGraphView, error) {
	g, err := s.Graph(ctx, courseID)
	if err != nil {
		return CourseGraphView{}, err
	}
	v := CourseGraphView{Course: g.Course, Topics: g.Topics, Formats: g.Formats, Concepts: []ConceptView{}}
	for _, c := range g.LiveConcepts() {
		v.Concepts = append(v.Concepts, conceptView(g, c))
	}
	return v, nil
}

func conceptView(g *CourseGraph, c store.Concept) ConceptView {
	pre := g.Prereqs[c.ID]
	if pre == nil {
		pre = []uuid.UUID{}
	}
	return ConceptView{ID: c.ID, Slug: c.Slug, Name: c.Name, TopicID: c.TopicID, PrerequisiteIDs: pre, HasLesson: g.HasLesson[c.ID]}
}

// MasteryView is a Learner's Mastery of one Concept.
type MasteryView struct {
	ConceptID uuid.UUID
	Score     float64
	Evidence  float64
	Level     domain.MasteryLevel
}

func masteryView(id uuid.UUID, m map[domain.ConceptID]domain.Mastery) MasteryView {
	mm := domain.LevelOf(m, cid(id))
	return MasteryView{ConceptID: id, Score: mm.Score, Evidence: mm.Evidence, Level: mm.Level}
}

// ConceptDetail is a Concept with Lesson, neighbors, and the Learner's Mastery.
type ConceptDetail struct {
	ConceptView
	TopicName           string
	LessonMd            *string
	DependentIDs        []uuid.UUID
	Mastery             MasteryView
	ActiveQuestionCount int
}

// findConcept locates the Course of a Concept.
func (s *Service) findConcept(ctx context.Context, conceptID uuid.UUID) (*CourseGraph, store.Concept, error) {
	c, err := s.q.GetConcept(ctx, conceptID)
	if err != nil {
		return nil, c, mapNotFound(err, "concept")
	}
	g, err := s.Graph(ctx, c.CourseID)
	if err != nil {
		return nil, c, err
	}
	if !g.Live(conceptID) {
		return nil, c, notFound("concept")
	}
	return g, g.ConceptByID[conceptID], nil
}

// ConceptDetail returns one Concept for the Actor.
func (s *Service) ConceptDetail(ctx context.Context, a Actor, conceptID uuid.UUID) (ConceptDetail, error) {
	g, c, err := s.findConcept(ctx, conceptID)
	if err != nil {
		return ConceptDetail{}, err
	}
	d := ConceptDetail{ConceptView: conceptView(g, c), TopicName: g.TopicByID[c.TopicID].Name, DependentIDs: []uuid.UUID{}}
	for _, dep := range g.Graph.Dependents(cid(c.ID)) {
		d.DependentIDs = append(d.DependentIDs, ucid(dep))
	}
	if l, err := s.q.GetLesson(ctx, c.ID); err == nil {
		d.LessonMd = &l.BodyMd
	} else if !store.IsNotFound(err) {
		return ConceptDetail{}, err
	}
	m, err := masteryMap(ctx, s.q, a.LearnerID, g.Course.ID)
	if err != nil {
		return ConceptDetail{}, err
	}
	d.Mastery = masteryView(c.ID, m)
	n, err := s.q.CountServableQuestionsForConcept(ctx, c.ID)
	if err != nil {
		return ConceptDetail{}, err
	}
	d.ActiveQuestionCount = int(n)
	return d, nil
}

// CourseMastery returns the Actor's Mastery for every live Concept.
func (s *Service) CourseMastery(ctx context.Context, a Actor, courseID uuid.UUID) ([]MasteryView, error) {
	g, err := s.Graph(ctx, courseID)
	if err != nil {
		return nil, err
	}
	m, err := masteryMap(ctx, s.q, a.LearnerID, courseID)
	if err != nil {
		return nil, err
	}
	out := []MasteryView{}
	for _, c := range g.LiveConcepts() {
		out = append(out, masteryView(c.ID, m))
	}
	return out, nil
}

// ConceptRef names a Concept with the Learner's level.
type ConceptRef struct {
	ID    uuid.UUID
	Name  string
	Level domain.MasteryLevel
}

// RecommendationView is a Recommendation with names and levels resolved.
type RecommendationView struct {
	Kind        domain.RecommendationKind
	ConceptID   uuid.UUID
	ConceptName string
	TargetID    uuid.UUID
	Path        []ConceptRef
	Unblocks    int
	Priority    float64
}

func (s *Service) recommendations(ctx context.Context, q *store.Queries, learnerID uuid.UUID, g *CourseGraph) ([]RecommendationView, error) {
	m, err := masteryMap(ctx, q, learnerID, g.Course.ID)
	if err != nil {
		return nil, err
	}
	recs := domain.Recommend(g.Graph, liveOnly(g, m))
	out := make([]RecommendationView, 0, len(recs))
	for _, r := range recs {
		out = append(out, RecommendationView{
			Kind: r.Kind, ConceptID: ucid(r.Concept), ConceptName: g.Name(ucid(r.Concept)), TargetID: ucid(r.Target),
			Path: refs(g, r.Path, m), Unblocks: r.Unblocks, Priority: r.Priority,
		})
	}
	return out, nil
}

func liveOnly(g *CourseGraph, m map[domain.ConceptID]domain.Mastery) map[domain.ConceptID]domain.Mastery {
	out := make(map[domain.ConceptID]domain.Mastery, len(m))
	for k, v := range m {
		if g.Live(ucid(k)) {
			out[k] = v
		}
	}
	return out
}

func refs(g *CourseGraph, ids []domain.ConceptID, m map[domain.ConceptID]domain.Mastery) []ConceptRef {
	out := make([]ConceptRef, 0, len(ids))
	for _, id := range ids {
		out = append(out, ConceptRef{ID: ucid(id), Name: g.Name(ucid(id)), Level: domain.LevelOf(m, id).Level})
	}
	return out
}

// Recommendations returns the Actor's Recommendations, ordered by Priority.
func (s *Service) Recommendations(ctx context.Context, a Actor, courseID uuid.UUID) ([]RecommendationView, error) {
	g, err := s.Graph(ctx, courseID)
	if err != nil {
		return nil, err
	}
	return s.recommendations(ctx, s.q, a.LearnerID, g)
}

// FormatStat is accuracy per Question Format.
type FormatStat struct {
	Format, FormatName string
	Answered, Correct  int
	Accuracy           float64
}

// FormatReport returns the Actor's accuracy per Question Format.
func (s *Service) FormatReport(ctx context.Context, a Actor, courseID uuid.UUID) ([]FormatStat, error) {
	if _, err := s.Graph(ctx, courseID); err != nil {
		return nil, err
	}
	rows, err := s.q.FormatReport(ctx, store.FormatReportParams{LearnerID: a.LearnerID, CourseID: courseID})
	if err != nil {
		return nil, err
	}
	out := []FormatStat{}
	for _, r := range rows {
		st := FormatStat{Format: r.Slug, FormatName: r.Name, Answered: int(r.Answered), Correct: int(r.Correct)}
		if r.Answered > 0 {
			st.Accuracy = float64(r.Correct) / float64(r.Answered)
		}
		out = append(out, st)
	}
	return out, nil
}

// ExamSummary describes an Exam with the Actor's last score.
type ExamSummary struct {
	ID               uuid.UUID
	Name             string
	TimeLimitSeconds int
	QuestionCount    int
	LastScore        *float64
}

// ListExams returns a Course's Exams.
func (s *Service) ListExams(ctx context.Context, a Actor, courseID uuid.UUID) ([]ExamSummary, error) {
	if _, err := s.Graph(ctx, courseID); err != nil {
		return nil, err
	}
	exams, err := s.q.ListExams(ctx, courseID)
	if err != nil {
		return nil, err
	}
	last, err := s.q.LastExamResults(ctx, store.LastExamResultsParams{LearnerID: a.LearnerID, CourseID: courseID})
	if err != nil {
		return nil, err
	}
	scores := map[uuid.UUID]float64{}
	for _, l := range last {
		if l.Total > 0 {
			scores[l.ExamID] = float64(l.Correct) / float64(l.Total)
		}
	}
	out := []ExamSummary{}
	for _, e := range exams {
		es := ExamSummary{ID: e.ID, Name: e.Name, TimeLimitSeconds: int(e.TimeLimitS), QuestionCount: int(e.QuestionCount)}
		if sc, ok := scores[e.ID]; ok {
			es.LastScore = ptr(sc)
		}
		out = append(out, es)
	}
	return out, nil
}

// ── Learning Paths ──────────────────────────────────────────────────────

// LearningPathView is a Learning Path with its current step.
type LearningPathView struct {
	ID          uuid.UUID
	CourseID    uuid.UUID
	Steps       []ConceptRef
	CurrentStep *uuid.UUID // nil when complete
	StartedAt   time.Time
	CompletedAt *time.Time
}

func pathIDs(p store.LearningPath) []domain.ConceptID {
	ids := make([]domain.ConceptID, 0, len(p.Path))
	for _, id := range p.Path {
		ids = append(ids, cid(id))
	}
	return ids
}

// checkPathCompletion marks the Learner's active Learning Path complete once
// every step is Developing or better.
func (s *Service) checkPathCompletion(ctx context.Context, q *store.Queries, learnerID, courseID uuid.UUID) error {
	p, err := q.GetActivePath(ctx, store.GetActivePathParams{LearnerID: learnerID, CourseID: courseID})
	if store.IsNotFound(err) {
		return nil
	}
	if err != nil || p.CompletedAt != nil {
		return err
	}
	m, err := masteryMap(ctx, q, learnerID, courseID)
	if err != nil {
		return err
	}
	if _, more := domain.NextPathStep(pathIDs(p), m); !more {
		return q.CompleteLearningPath(ctx, p.ID)
	}
	return nil
}

// LearningPath returns the Actor's current (or most recently completed) Learning Path.
func (s *Service) LearningPath(ctx context.Context, a Actor, courseID uuid.UUID) (LearningPathView, error) {
	g, err := s.Graph(ctx, courseID)
	if err != nil {
		return LearningPathView{}, err
	}
	if err := s.checkPathCompletion(ctx, s.q, a.LearnerID, courseID); err != nil {
		return LearningPathView{}, err
	}
	p, err := s.q.GetActivePath(ctx, store.GetActivePathParams{LearnerID: a.LearnerID, CourseID: courseID})
	if err != nil {
		return LearningPathView{}, mapNotFound(err, "learning path")
	}
	m, err := masteryMap(ctx, s.q, a.LearnerID, courseID)
	if err != nil {
		return LearningPathView{}, err
	}
	return pathView(g, p, m), nil
}

func pathView(g *CourseGraph, p store.LearningPath, m map[domain.ConceptID]domain.Mastery) LearningPathView {
	v := LearningPathView{ID: p.ID, CourseID: p.CourseID, Steps: refs(g, pathIDs(p), m), StartedAt: p.StartedAt, CompletedAt: p.CompletedAt}
	if p.CompletedAt == nil {
		if next, ok := domain.NextPathStep(pathIDs(p), m); ok {
			v.CurrentStep = ptr(ucid(next))
		}
	}
	return v
}

// StartLearningPath starts a Learning Path from root to target, replacing any
// active one. When a matching Recommendation exists its reason path is used,
// so the walk matches what the Learner was shown; otherwise the shortest
// Prerequisite chain.
func (s *Service) StartLearningPath(ctx context.Context, a Actor, courseID, root, target uuid.UUID) (LearningPathView, error) {
	g, err := s.Graph(ctx, courseID)
	if err != nil {
		return LearningPathView{}, err
	}
	if !g.Live(root) || !g.Live(target) {
		return LearningPathView{}, E(KindInvalid, "unknown_concept", "root and target must be Concepts of this course")
	}
	var steps []domain.ConceptID
	recs, err := s.recommendations(ctx, s.q, a.LearnerID, g)
	if err != nil {
		return LearningPathView{}, err
	}
	for _, r := range recs {
		if r.ConceptID == root && r.TargetID == target {
			for _, p := range r.Path {
				steps = append(steps, cid(p.ID))
			}
			break
		}
	}
	if steps == nil {
		steps, err = domain.LearningPathSteps(g.Graph, cid(root), cid(target))
		if err != nil {
			if errors.Is(err, domain.ErrNotPrerequisite) {
				return LearningPathView{}, E(KindInvalid, "not_prerequisite", "target does not depend on root")
			}
			return LearningPathView{}, E(KindInvalid, "invalid_path", "%v", err)
		}
	}
	path := make([]uuid.UUID, 0, len(steps))
	for _, st := range steps {
		path = append(path, ucid(st))
	}
	var p store.LearningPath
	err = s.tx(ctx, func(q *store.Queries) error {
		if err := q.AbandonAllPaths(ctx, store.AbandonAllPathsParams{LearnerID: a.LearnerID, CourseID: courseID}); err != nil {
			return err
		}
		var err error
		p, err = q.CreateLearningPath(ctx, store.CreateLearningPathParams{
			LearnerID: a.LearnerID, CourseID: courseID, RootConceptID: root, TargetConceptID: target, Path: path,
		})
		return err
	})
	if err != nil {
		return LearningPathView{}, err
	}
	m, err := masteryMap(ctx, s.q, a.LearnerID, courseID)
	if err != nil {
		return LearningPathView{}, err
	}
	return pathView(g, p, m), nil
}

// AbandonLearningPath abandons the Actor's Learning Path in a Course.
func (s *Service) AbandonLearningPath(ctx context.Context, a Actor, courseID uuid.UUID) error {
	if _, err := s.Graph(ctx, courseID); err != nil {
		return err
	}
	return s.q.AbandonAllPaths(ctx, store.AbandonAllPathsParams{LearnerID: a.LearnerID, CourseID: courseID})
}
