package app_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/llm"
	"github.com/thekogo/tutor-platform/backend/internal/testutil"
)

func TestConceptTutorChat(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	b := e.Concepts["b"]
	in := app.AskConceptHelpInput{ConceptID: b, Messages: ask("why does this work?")}

	// Disabled until configured.
	if d := must[app.ConceptDetail](t)(e.App.ConceptDetail(ctx, a, b)); d.HelpAvailable {
		t.Fatal("help available without a model")
	}
	_, err := e.App.AskConceptHelp(ctx, a, in)
	wantKind(t, err, app.KindUnavailable, "help_disabled")

	m := &fakeModel{}
	e.App.SetHelp(app.HelpConfig{Model: m, MaxPerHour: 3})
	if d := must[app.ConceptDetail](t)(e.App.ConceptDetail(ctx, a, b)); !d.HelpAvailable {
		t.Fatal("help should be available on a Concept page")
	}

	// Works without a Lesson.
	must[string](t)(e.App.AskConceptHelp(ctx, a, in))
	if !strings.Contains(m.last[0].Content, "No lesson has been written") {
		t.Error("missing-lesson note not in prompt")
	}

	// With a Lesson, the model gets it and the Concept's neighbors as system context.
	if err := e.App.UpsertLesson(ctx, testutil.CourseSlug, "b", "## Lesson B\nUnique lesson body 42."); err != nil {
		t.Fatal(err)
	}
	must[string](t)(e.App.AskConceptHelp(ctx, a, in))
	if len(m.last) != 2 || m.last[0].Role != llm.RoleSystem || m.last[1].Content != "why does this work?" {
		t.Fatalf("prompt shape %+v", m.last)
	}
	sys := m.last[0].Content
	for _, want := range []string{"Test Course", "Topic: Topic 1", "Concept: Concept B", "Prerequisites (concepts this builds on): Concept A", "Leads to: Concept C", "mastery of this concept: unassessed", "Unique lesson body 42."} {
		if !strings.Contains(sys, want) {
			t.Errorf("system prompt missing %q", want)
		}
	}

	// Validation and unknown Concepts.
	_, err = e.App.AskConceptHelp(ctx, a, app.AskConceptHelpInput{ConceptID: b, Messages: ask("  ")})
	wantKind(t, err, app.KindInvalid, "invalid_messages")
	_, err = e.App.AskConceptHelp(ctx, a, app.AskConceptHelpInput{ConceptID: uuid.New(), Messages: ask("hi")})
	wantKind(t, err, app.KindNotFound, "not_found")

	// Shares the hourly budget with the Quiz Tutor Chat (3 per hour, 2 used).
	must[string](t)(e.App.AskConceptHelp(ctx, a, in))
	_, err = e.App.AskConceptHelp(ctx, a, in)
	wantKind(t, err, app.KindTooManyRequests, "help_rate_limited")
}
