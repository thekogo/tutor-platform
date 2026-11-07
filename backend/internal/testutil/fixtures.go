package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// CourseSlug is the slug of the fixture Course.
const CourseSlug = "test-course"

// Graph is the fixture knowledge graph:
//
//	topic t1: a ← b ← c   (c needs b, b needs a)
//	topic t2: d           (d needs a)
func Graph() app.GraphDoc {
	return app.GraphDoc{
		Course:          app.CourseDoc{ID: CourseSlug, Name: "Test Course", Goal: "pass"},
		QuestionFormats: []app.FormatDoc{{ID: "standard", Name: "Standard"}, {ID: "data-sufficiency", Name: "DS"}},
		Topics: []app.TopicDoc{
			{ID: "t1", Name: "Topic 1"},
			{ID: "t2", Name: "Topic 2"},
		},
		Concepts: []app.ConceptDoc{
			{ID: "a", Topic: "t1", Name: "Concept A", Prerequisites: []string{}},
			{ID: "b", Topic: "t1", Name: "Concept B", Prerequisites: []string{"a"}},
			{ID: "c", Topic: "t1", Name: "Concept C", Prerequisites: []string{"b"}},
			{ID: "d", Topic: "t2", Name: "Concept D", Prerequisites: []string{"a"}},
		},
	}
}

// Env is a migrated database with the fixture Course imported.
type Env struct {
	Pool     *pgxpool.Pool
	App      *app.Service
	Q        *store.Queries
	CourseID uuid.UUID
	Concepts map[string]uuid.UUID // slug → id
}

// NewEnv creates a database and imports the fixture graph.
func NewEnv(t testing.TB) *Env {
	t.Helper()
	pool := NewDB(t)
	svc := app.New(pool, nil)
	ctx := context.Background()
	res, err := svc.ImportCourseGraph(ctx, Graph(), false)
	if err != nil {
		t.Fatalf("import graph: %v", err)
	}
	e := &Env{Pool: pool, App: svc, Q: store.New(pool), CourseID: uuid.MustParse(res.CourseID), Concepts: map[string]uuid.UUID{}}
	g, err := svc.Graph(ctx, e.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range g.Concepts {
		e.Concepts[c.Slug] = c.ID
	}
	return e
}

// QuestionSpec builds a valid QuestionInput: choices A-D, A correct.
func QuestionSpec(primary string, difficulty int, others ...string) app.QuestionInput {
	return app.QuestionInput{
		StemMd:        fmt.Sprintf("What is %s? (d%d)", primary, difficulty),
		ExplanationMd: "Because A.",
		Choices: []app.ChoiceInput{
			{Label: "A", BodyMd: "right", IsCorrect: true},
			{Label: "B", BodyMd: "wrong 1"},
			{Label: "C", BodyMd: "wrong 2"},
			{Label: "D", BodyMd: "wrong 3"},
		},
		Concepts:       append([]string{primary}, others...),
		PrimaryConcept: primary,
		Difficulty:     difficulty,
		Format:         "standard",
		Source:         app.SourceAIGenerated,
	}
}

// ActiveQuestions creates n verified (Active) Questions on a Concept.
func (e *Env) ActiveQuestions(t testing.TB, n, difficulty int, primary string, others ...string) []uuid.UUID {
	t.Helper()
	ctx := context.Background()
	specs := make([]app.QuestionInput, n)
	for i := range specs {
		specs[i] = QuestionSpec(primary, difficulty, others...)
		specs[i].StemMd = fmt.Sprintf("%s #%d", specs[i].StemMd, i)
	}
	ids, err := e.App.CreateQuestions(ctx, CourseSlug, specs)
	if err != nil {
		t.Fatalf("create questions: %v", err)
	}
	for _, id := range ids {
		if _, err := e.App.SubmitVerification(ctx, id, "A", ""); err != nil {
			t.Fatalf("verify: %v", err)
		}
	}
	return ids
}

// Learner creates a Learner and returns its Actor.
func (e *Env) Learner(t testing.TB, email string, owner bool) app.Actor {
	t.Helper()
	l, err := e.Q.CreateLearner(context.Background(), store.CreateLearnerParams{Email: email, Name: email, IsOwner: owner})
	if err != nil {
		t.Fatal(err)
	}
	return app.Actor{LearnerID: l.ID, IsOwner: owner}
}

// Choice returns the id of the choice with the given label.
func (e *Env) Choice(t testing.TB, questionID uuid.UUID, label string) uuid.UUID {
	t.Helper()
	cs, err := e.Q.ListChoicesForQuestions(context.Background(), []uuid.UUID{questionID})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if c.Label == label {
			return c.ID
		}
	}
	t.Fatalf("no choice %s", label)
	return uuid.Nil
}
