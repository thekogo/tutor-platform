package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/domain"
	"github.com/thekogo/tutor-platform/backend/internal/store"
	"github.com/thekogo/tutor-platform/backend/internal/testutil"
)

var ctx = context.Background()

func wantKind(t *testing.T, err error, kind app.Kind, code string) {
	t.Helper()
	e, ok := app.AsError(err)
	if !ok {
		t.Fatalf("want app error %s, got %v", code, err)
	}
	if e.Kind != kind || (code != "" && e.Code != code) {
		t.Fatalf("want kind %d code %s, got kind %d code %s (%s)", kind, code, e.Kind, e.Code, e.Message)
	}
}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func TestImportIsIdempotentAndValidates(t *testing.T) {
	e := testutil.NewEnv(t)
	res := must[app.ImportResult](t)(e.App.ImportCourseGraph(ctx, testutil.Graph(), false))
	if res.Concepts != 4 || res.Prerequisites != 3 || res.CourseID != e.CourseID.String() {
		t.Fatalf("unexpected re-import result %+v", res)
	}
	g := must[*app.CourseGraph](t)(e.App.Graph(ctx, e.CourseID))
	if len(g.LiveConcepts()) != 4 {
		t.Fatalf("want 4 concepts, got %d", len(g.LiveConcepts()))
	}

	// A cycle in the document is rejected and nothing changes.
	bad := testutil.Graph()
	bad.Concepts[0].Prerequisites = []string{"c"}
	_, err := e.App.ImportCourseGraph(ctx, bad, false)
	wantKind(t, err, app.KindInvalid, "prerequisite_cycle")

	// Unknown references are reported.
	bad = testutil.Graph()
	bad.Concepts[1].Prerequisites = []string{"nope"}
	bad.Concepts[2].Topic = "zzz"
	_, err = e.App.ImportCourseGraph(ctx, bad, false)
	wantKind(t, err, app.KindInvalid, "invalid_graph")
	if ae, _ := app.AsError(err); len(ae.Details) != 2 {
		t.Fatalf("want 2 details, got %v", ae.Details)
	}
}

func TestGraphEditsRejectCyclesAndBridgeArchived(t *testing.T) {
	e := testutil.NewEnv(t)
	err := e.App.SetPrerequisites(ctx, testutil.CourseSlug, "a", []string{"c"})
	wantKind(t, err, app.KindInvalid, "prerequisite_cycle")
	err = e.App.SetPrerequisites(ctx, testutil.CourseSlug, "a", []string{"a"})
	if err == nil {
		t.Fatal("self prerequisite accepted")
	}
	_, err = e.App.UpsertConcept(ctx, app.UpsertConceptInput{Course: testutil.CourseSlug, Slug: "e", Topic: "t2", Name: "E", Prerequisites: &[]string{"ghost"}})
	wantKind(t, err, app.KindInvalid, "unknown_concept")

	// New concept e needs c; archive b: c should now (effectively) need a.
	must[store.Concept](t)(e.App.UpsertConcept(ctx, app.UpsertConceptInput{Course: testutil.CourseSlug, Slug: "e", Topic: "t2", Name: "E", Prerequisites: &[]string{"c"}}))
	if err := e.App.ArchiveConcept(ctx, testutil.CourseSlug, "b"); err != nil {
		t.Fatal(err)
	}
	g := must[*app.CourseGraph](t)(e.App.Graph(ctx, e.CourseID))
	if g.Live(e.Concepts["b"]) {
		t.Fatal("b still live")
	}
	pre := g.Prereqs[e.Concepts["c"]]
	if len(pre) != 1 || pre[0] != e.Concepts["a"] {
		t.Fatalf("c should inherit a through archived b, got %v", pre)
	}
	anc := g.Graph.Ancestors(domain.ConceptID(g.ConceptBySlug["e"].ID.String()))
	if len(anc) != 2 {
		t.Fatalf("e ancestors: want c,a got %v", anc)
	}
	// Re-import un-archives b.
	must[app.ImportResult](t)(e.App.ImportCourseGraph(ctx, testutil.Graph(), false))
	g = must[*app.CourseGraph](t)(e.App.Graph(ctx, e.CourseID))
	if !g.Live(e.Concepts["b"]) {
		t.Fatal("b not un-archived by import")
	}
}

func TestCreateQuestionsValidation(t *testing.T) {
	e := testutil.NewEnv(t)
	good := testutil.QuestionSpec("b", 2, "a")
	twoCorrect := testutil.QuestionSpec("b", 2)
	twoCorrect.Choices[1].IsCorrect = true
	noExplanation := testutil.QuestionSpec("b", 2)
	noExplanation.ExplanationMd = "  "
	primaryNotTagged := testutil.QuestionSpec("b", 2)
	primaryNotTagged.PrimaryConcept = "c"
	unknownTag := testutil.QuestionSpec("b", 4, "zzz")
	unknownTag.Format = "nope"

	_, err := e.App.CreateQuestions(ctx, testutil.CourseSlug, []app.QuestionInput{good, twoCorrect, noExplanation, primaryNotTagged, unknownTag})
	wantKind(t, err, app.KindInvalid, "invalid_questions")
	ae, _ := app.AsError(err)
	if len(ae.Details) < 6 {
		t.Fatalf("expected detailed errors, got %v", ae.Details)
	}
	// All-or-nothing: the good one was not created either.
	stats := must[[]app.PoolStat](t)(e.App.PoolStats(ctx, testutil.CourseSlug))
	for _, s := range stats {
		if s.Draft+s.Active != 0 {
			t.Fatalf("questions were written despite validation failure: %+v", s)
		}
	}
	ids := must[[]uuid.UUID](t)(e.App.CreateQuestions(ctx, testutil.CourseSlug, []app.QuestionInput{good}))
	q := must[store.Question](t)(e.Q.GetQuestion(ctx, ids[0]))
	if q.Status != app.StatusDraft {
		t.Fatalf("new questions must be draft, got %s", q.Status)
	}
}

func TestVerificationFlow(t *testing.T) {
	e := testutil.NewEnv(t)
	ids := must[[]uuid.UUID](t)(e.App.CreateQuestions(ctx, testutil.CourseSlug, []app.QuestionInput{
		testutil.QuestionSpec("a", 1), testutil.QuestionSpec("b", 1),
	}))
	items := must[[]app.VerificationItem](t)(e.App.ListQuestionsForVerification(ctx, testutil.CourseSlug, 10))
	if len(items) != 2 || len(items[0].Choices) != 4 {
		t.Fatalf("unexpected verification list %+v", items)
	}
	// Mismatch: stays draft with note.
	res := must[app.VerificationResult](t)(e.App.SubmitVerification(ctx, ids[0], "B", "I computed B"))
	if res.Matched || res.Status != app.StatusDraft {
		t.Fatalf("mismatch result %+v", res)
	}
	q := must[store.Question](t)(e.Q.GetQuestion(ctx, ids[0]))
	if q.Status != app.StatusDraft || q.VerificationNote == nil {
		t.Fatalf("mismatch must keep Draft with note, got %s %v", q.Status, q.VerificationNote)
	}
	// Match: active.
	res = must[app.VerificationResult](t)(e.App.SubmitVerification(ctx, ids[1], "A", ""))
	if !res.Matched || res.Status != app.StatusActive {
		t.Fatalf("match result %+v", res)
	}
	_, err := e.App.SubmitVerification(ctx, ids[1], "A", "")
	wantKind(t, err, app.KindConflict, "not_draft")
	_, err = e.App.SubmitVerification(ctx, ids[0], "Z", "")
	wantKind(t, err, app.KindInvalid, "unknown_label")
	items = must[[]app.VerificationItem](t)(e.App.ListQuestionsForVerification(ctx, testutil.CourseSlug, 10))
	if len(items) != 1 {
		t.Fatalf("active question still listed for verification")
	}
}

func TestDatabaseEnforcesQuestionIntegrity(t *testing.T) {
	e := testutil.NewEnv(t)
	ids := e.ActiveQuestions(t, 1, 1, "a")
	// Second correct choice violates the partial unique index.
	_, err := e.Pool.Exec(ctx, `UPDATE question_choices SET is_correct = true WHERE question_id = $1 AND label = 'B'`, ids[0])
	if err == nil || !store.IsUniqueViolation(err) {
		t.Fatalf("want unique violation, got %v", err)
	}
	// No correct choice fails at commit (deferred trigger).
	_, err = e.Pool.Exec(ctx, `UPDATE question_choices SET is_correct = false WHERE question_id = $1`, ids[0])
	if err == nil {
		t.Fatal("question without a correct choice accepted")
	}
	// Second primary concept violates the partial unique index.
	_, err = e.Pool.Exec(ctx, `INSERT INTO question_concepts (question_id, concept_id, is_primary) VALUES ($1, $2, true)`, ids[0], e.Concepts["b"])
	if err == nil {
		t.Fatal("second primary accepted")
	}
	// Difficulty CHECK.
	_, err = e.Pool.Exec(ctx, `UPDATE questions SET difficulty = 4 WHERE id = $1`, ids[0])
	if err == nil {
		t.Fatal("difficulty 4 accepted")
	}
	// Answers are append-only.
	a := e.Learner(t, "x@example.com", false)
	sess := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["a"])}))
	must[app.AnswerResult](t)(e.App.SubmitAnswer(ctx, a, app.SubmitAnswerInput{SessionID: sess.ID, QuestionID: ids[0], ChoiceID: e.Choice(t, ids[0], "B")}))
	if _, err := e.Pool.Exec(ctx, `UPDATE answers SET correct = true`); err == nil {
		t.Fatal("answer mutated")
	}
	if _, err := e.Pool.Exec(ctx, `DELETE FROM answers`); err == nil {
		t.Fatal("answer deleted")
	}
	if _, err := e.Pool.Exec(ctx, `UPDATE answers SET excluded = true`); err != nil {
		t.Fatalf("excluded must be updatable: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }

func TestQuizServingRules(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	active := e.ActiveQuestions(t, 3, 2, "b")
	examOnly := e.ActiveQuestions(t, 1, 2, "b")
	// A draft question is never served.
	must[[]uuid.UUID](t)(e.App.CreateQuestions(ctx, testutil.CourseSlug, []app.QuestionInput{testutil.QuestionSpec("b", 2)}))
	must[store.Exam](t)(e.App.CreateExam(ctx, testutil.CourseSlug, "Mock", 600, examOnly))

	sess := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["b"])}))
	if *sess.TotalQuestions != 3 {
		t.Fatalf("quiz should hold the 3 servable questions, got %d", *sess.TotalQuestions)
	}
	served := map[uuid.UUID]bool{}
	for {
		q, err := e.App.NextQuestion(ctx, a, sess.ID)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			break
		}
		if q.IsRepeat {
			t.Fatal("first quiz must not contain repeats")
		}
		served[q.ID] = true
		must[app.AnswerResult](t)(e.App.SubmitAnswer(ctx, a, app.SubmitAnswerInput{SessionID: sess.ID, QuestionID: q.ID, ChoiceID: e.Choice(t, q.ID, "A")}))
	}
	if len(served) != 3 || served[examOnly[0]] {
		t.Fatalf("served %v; want exactly the active non-exam questions %v", served, active)
	}
	// Pool exhausted → exactly one open Content Request for b (deduped).
	reqs := must[[]app.ContentRequestView](t)(e.App.ListContentRequestsAdmin(ctx, true))
	if len(reqs) != 1 || reqs[0].ConceptSlug != "b" {
		t.Fatalf("want one content request for b, got %+v", reqs)
	}
	// Second quiz: all repeats, flagged as such; still one open request.
	sess2 := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["b"])}))
	q := must[*app.ServedQuestion](t)(e.App.NextQuestion(ctx, a, sess2.ID))
	if !q.IsRepeat {
		t.Fatal("repeat not flagged")
	}
	must[app.AnswerResult](t)(e.App.SubmitAnswer(ctx, a, app.SubmitAnswerInput{SessionID: sess2.ID, QuestionID: q.ID, ChoiceID: e.Choice(t, q.ID, "A")}))
	var repeat bool
	if err := e.Pool.QueryRow(ctx, `SELECT is_repeat FROM answers WHERE session_id = $1`, sess2.ID).Scan(&repeat); err != nil || !repeat {
		t.Fatalf("answer is_repeat = %v (%v)", repeat, err)
	}
	reqs = must[[]app.ContentRequestView](t)(e.App.ListContentRequestsAdmin(ctx, true))
	if len(reqs) != 1 {
		t.Fatalf("content requests must be deduped per concept, got %d", len(reqs))
	}
	// No questions at all → 422 and a request for that concept.
	_, err := e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["d"])})
	wantKind(t, err, app.KindInvalid, "no_questions")
	reqs = must[[]app.ContentRequestView](t)(e.App.ListContentRequestsAdmin(ctx, true))
	if len(reqs) != 2 {
		t.Fatalf("want request for d too, got %+v", reqs)
	}
	// Topic quiz pulls from all concepts of the topic.
	e.ActiveQuestions(t, 2, 1, "a")
	tq := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, TopicID: ptr(must[*app.CourseGraph](t)(e.App.Graph(ctx, e.CourseID)).TopicBySlug["t1"].ID)}))
	if *tq.TotalQuestions != 5 {
		t.Fatalf("topic quiz: want 5 questions (2 fresh a + 3 repeats b), got %d", *tq.TotalQuestions)
	}
}

func TestDiagnosticUsesEasyQuestions(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	e.ActiveQuestions(t, 3, 1, "c")
	e.ActiveQuestions(t, 3, 2, "c")
	hard := e.ActiveQuestions(t, 3, 3, "c")
	s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindDiagnostic, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["c"])}))
	if *s.TotalQuestions != app.DiagnosticSize {
		t.Fatalf("diagnostic size %d", *s.TotalQuestions)
	}
	rows := must[[]store.ListSessionQuestionsRow](t)(e.Q.ListSessionQuestions(ctx, s.ID))
	for _, r := range rows {
		for _, h := range hard {
			if r.QuestionID == h {
				t.Fatal("difficulty 3 question in diagnostic")
			}
		}
	}
	_, err := e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindDiagnostic, CourseID: e.CourseID})
	wantKind(t, err, app.KindInvalid, "concept_required")
}

func answerAll(t *testing.T, e *testutil.Env, a app.Actor, sessionID uuid.UUID, label string) []app.AnswerResult {
	t.Helper()
	var out []app.AnswerResult
	for {
		q, err := e.App.NextQuestion(ctx, a, sessionID)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			return out
		}
		out = append(out, must[app.AnswerResult](t)(e.App.SubmitAnswer(ctx, a, app.SubmitAnswerInput{SessionID: sessionID, QuestionID: q.ID, ChoiceID: e.Choice(t, q.ID, label)})))
	}
}

func TestMasteryRecomputeFlagAndResolve(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	other := e.Learner(t, "o@example.com", false)
	ids := e.ActiveQuestions(t, 5, 2, "b", "a")

	s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["b"])}))
	results := answerAll(t, e, a, s.ID, "B") // all wrong
	last := results[len(results)-1]
	if last.Correct == nil || *last.Correct || last.CorrectChoiceID == nil || last.ExplanationMd == nil {
		t.Fatalf("quiz feedback missing: %+v", last)
	}
	changed := false
	for _, r := range results {
		for _, c := range r.MasteryChanges {
			if c.ConceptID == e.Concepts["b"] && c.After == domain.Weak {
				changed = true
			}
		}
	}
	if !changed {
		t.Fatal("expected b to become Weak via masteryChanges")
	}
	m := must[[]app.MasteryView](t)(e.App.CourseMastery(ctx, a, e.CourseID))
	levels := map[uuid.UUID]domain.MasteryLevel{}
	for _, mv := range m {
		levels[mv.ConceptID] = mv.Level
	}
	if levels[e.Concepts["b"]] != domain.Weak {
		t.Fatalf("b level %s", levels[e.Concepts["b"]])
	}
	// Other learner's mastery is untouched (private).
	om := must[[]app.MasteryView](t)(e.App.CourseMastery(ctx, other, e.CourseID))
	for _, mv := range om {
		if mv.Level != domain.Unassessed {
			t.Fatal("other learner affected")
		}
	}
	// Recommendations: b weak, a unassessed → diagnose a.
	recs := must[[]app.RecommendationView](t)(e.App.Recommendations(ctx, a, e.CourseID))
	if len(recs) == 0 {
		t.Fatal("no recommendations")
	}

	// Rebuild equals incremental.
	before := must[[]app.MasteryView](t)(e.App.CourseMastery(ctx, a, e.CourseID))
	must[int](t)(e.App.RebuildAllMastery(ctx))
	after := must[[]app.MasteryView](t)(e.App.CourseMastery(ctx, a, e.CourseID))
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("rebuild differs: %+v vs %+v", before[i], after[i])
		}
	}

	// Only a Learner who answered may Flag.
	err := e.App.FlagQuestion(ctx, other, ids[0], "wrong key")
	wantKind(t, err, app.KindForbidden, "forbidden")
	err = e.App.FlagQuestion(ctx, a, ids[0], "")
	wantKind(t, err, app.KindInvalid, "invalid_reason")

	// Flag all five: they are pulled for everyone and Mastery drops back to Unassessed.
	for _, id := range ids {
		if err := e.App.FlagQuestion(ctx, a, id, "the key is wrong"); err != nil {
			t.Fatal(err)
		}
	}
	q := must[store.Question](t)(e.Q.GetQuestion(ctx, ids[0]))
	if q.Status != app.StatusFlagged {
		t.Fatalf("status %s", q.Status)
	}
	m = must[[]app.MasteryView](t)(e.App.CourseMastery(ctx, a, e.CourseID))
	for _, mv := range m {
		if mv.Level != domain.Unassessed || mv.Evidence != 0 {
			t.Fatalf("flagged answers still count: %+v", mv)
		}
	}
	_, err = e.App.CreateSession(ctx, other, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["b"])})
	wantKind(t, err, app.KindInvalid, "no_questions")

	// Resolve without key change: answers count again.
	flags := must[[]app.FlagView](t)(e.App.ListFlagsAdmin(ctx, true))
	if len(flags) != 5 {
		t.Fatalf("flags %d", len(flags))
	}
	byQ := map[uuid.UUID]uuid.UUID{}
	for _, f := range flags {
		byQ[f.QuestionID] = f.ID
	}
	for i, id := range ids {
		key := i == 0 // first question: key changed → its answers excluded forever
		r := must[app.ResolveFlagResult](t)(e.App.ResolveFlag(ctx, byQ[id], key, "", "fixed"))
		if r.Status != app.StatusActive || r.FlagsResolved != 1 || (key && r.ExcludedAnswers != 1) {
			t.Fatalf("resolve result %+v", r)
		}
	}
	var excluded int
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM answers WHERE excluded`).Scan(&excluded); err != nil || excluded != 1 {
		t.Fatalf("excluded answers %d (%v)", excluded, err)
	}
	m = must[[]app.MasteryView](t)(e.App.CourseMastery(ctx, a, e.CourseID))
	for _, mv := range m {
		if mv.ConceptID == e.Concepts["b"] && mv.Level != domain.Weak {
			t.Fatalf("after resolving, 4 wrong answers should make b Weak again: %+v", mv)
		}
	}
	_, err = e.App.ResolveFlag(ctx, byQ[ids[0]], false, "", "")
	wantKind(t, err, app.KindConflict, "already_resolved")
}

func TestUpdateQuestionKeyChangeExcludesAnswers(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	ids := e.ActiveQuestions(t, 1, 1, "a")
	s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["a"])}))
	answerAll(t, e, a, s.ID, "B")

	choices := []app.ChoiceInput{{Label: "A", BodyMd: "x"}, {Label: "B", BodyMd: "y", IsCorrect: true}, {Label: "C", BodyMd: "z"}}
	res := must[app.UpdateResult](t)(e.App.UpdateQuestion(ctx, ids[0], app.QuestionPatch{Choices: &choices}))
	if !res.KeyChanged || res.ExcludedAnswers != 1 || res.Status != app.StatusDraft {
		t.Fatalf("update result %+v", res)
	}
	// Removing an answered choice is refused.
	short := []app.ChoiceInput{{Label: "A", BodyMd: "x", IsCorrect: true}, {Label: "C", BodyMd: "z"}}
	_, err := e.App.UpdateQuestion(ctx, ids[0], app.QuestionPatch{Choices: &short})
	wantKind(t, err, app.KindInvalid, "choice_answered")
	// Invalid patch rejected.
	bad := 7
	_, err = e.App.UpdateQuestion(ctx, ids[0], app.QuestionPatch{Difficulty: &bad})
	wantKind(t, err, app.KindInvalid, "invalid_question")
}

func TestExamDeadlineAndSecrecy(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	ids := e.ActiveQuestions(t, 2, 2, "c")
	exam := must[store.Exam](t)(e.App.CreateExam(ctx, testutil.CourseSlug, "Mock 1", 60, ids))
	_, err := e.App.CreateExam(ctx, testutil.CourseSlug, "Bad", 60, []uuid.UUID{uuid.New()})
	wantKind(t, err, app.KindInvalid, "invalid_exam")

	now := time.Now().Truncate(time.Microsecond)
	e.App.SetClock(func() time.Time { return now })
	s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindExam, CourseID: e.CourseID, ExamID: &exam.ID}))
	if s.Deadline == nil || s.Deadline.Sub(now) != 60*time.Second {
		t.Fatalf("deadline %v", s.Deadline)
	}
	q := must[*app.ServedQuestion](t)(e.App.NextQuestion(ctx, a, s.ID))
	r := must[app.AnswerResult](t)(e.App.SubmitAnswer(ctx, a, app.SubmitAnswerInput{SessionID: s.ID, QuestionID: q.ID, ChoiceID: e.Choice(t, q.ID, "A")}))
	if !r.Recorded || r.Correct != nil || r.CorrectChoiceID != nil || r.ExplanationMd != nil || r.MasteryChanges != nil {
		t.Fatalf("exam answer leaked feedback: %+v", r)
	}
	_, err = e.App.SessionResult(ctx, a, s.ID)
	wantKind(t, err, app.KindConflict, "not_finished")

	e.App.SetClock(func() time.Time { return now.Add(61 * time.Second) })
	_, err = e.App.NextQuestion(ctx, a, s.ID)
	wantKind(t, err, app.KindGone, "deadline_passed")
	_, err = e.App.SubmitAnswer(ctx, a, app.SubmitAnswerInput{SessionID: s.ID, QuestionID: ids[0], ChoiceID: e.Choice(t, ids[0], "A")})
	wantKind(t, err, app.KindGone, "deadline_passed")

	res := must[app.SessionResult](t)(e.App.FinishSession(ctx, a, s.ID))
	if res.Total != 2 || res.Answered != 1 || res.Correct != 1 || len(res.Review) != 2 {
		t.Fatalf("result %+v", res)
	}
	if res.Review[0].ExplanationMd == "" || res.Review[0].CorrectChoiceID == uuid.Nil {
		t.Fatal("review missing key/explanation after finish")
	}
	if len(res.TopicBreakdown) != 1 || res.TopicBreakdown[0].Answered != 1 {
		t.Fatalf("topic breakdown %+v", res.TopicBreakdown)
	}
	exams := must[[]app.ExamSummary](t)(e.App.ListExams(ctx, a, e.CourseID))
	if len(exams) != 1 || exams[0].LastScore == nil || *exams[0].LastScore != 0.5 {
		t.Fatalf("exam summary %+v", exams)
	}
	// Exam questions never appear in quizzes.
	_, err = e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["c"])})
	wantKind(t, err, app.KindInvalid, "no_questions")
}

func TestSessionAuthorization(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	owner := e.Learner(t, "owner@example.com", true)
	other := e.Learner(t, "o@example.com", false)
	e.ActiveQuestions(t, 2, 1, "a")
	s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["a"])}))
	_, err := e.App.GetSession(ctx, other, s.ID)
	wantKind(t, err, app.KindNotFound, "not_found")
	_, err = e.App.NextQuestion(ctx, other, s.ID)
	wantKind(t, err, app.KindNotFound, "not_found")
	if _, err := e.App.GetSession(ctx, owner, s.ID); err != nil {
		t.Fatalf("owner should see any session: %v", err)
	}
	_, err = e.App.NextQuestion(ctx, owner, s.ID)
	wantKind(t, err, app.KindNotFound, "not_found") // but cannot take it
	_, err = e.App.ListLearners(ctx, a)
	wantKind(t, err, app.KindForbidden, "forbidden")
	if _, err := e.App.ListLearners(ctx, owner); err != nil {
		t.Fatal(err)
	}
}

func TestPlacementIsAdaptiveAndBounded(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	for _, c := range []string{"a", "b", "c", "d"} {
		e.ActiveQuestions(t, 2, 2, c)
	}
	s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindPlacement, CourseID: e.CourseID}))
	if s.TotalQuestions != nil {
		t.Fatal("placement total must be nil")
	}
	probes := 0
	seen := map[uuid.UUID]bool{}
	for {
		q, err := e.App.NextQuestion(ctx, a, s.ID)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			break
		}
		// Calling next again before answering returns the same question.
		again := must[*app.ServedQuestion](t)(e.App.NextQuestion(ctx, a, s.ID))
		if again.ID != q.ID {
			t.Fatal("next not idempotent")
		}
		if seen[q.ID] {
			t.Fatal("question repeated in placement")
		}
		seen[q.ID] = true
		probes++
		if probes > app.PlacementMaxProbes {
			t.Fatal("placement exceeded max probes")
		}
		must[app.AnswerResult](t)(e.App.SubmitAnswer(ctx, a, app.SubmitAnswerInput{SessionID: s.ID, QuestionID: q.ID, ChoiceID: e.Choice(t, q.ID, "B")}))
	}
	if probes == 0 {
		t.Fatal("no probes served")
	}
	must[app.SessionResult](t)(e.App.FinishSession(ctx, a, s.ID))
	done := must[bool](t)(e.App.LearnerPlacementDone(ctx, a.LearnerID))
	if !done {
		t.Fatal("placementDone not set")
	}
}

func TestLearningPath(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	_, err := e.App.LearningPath(ctx, a, e.CourseID)
	wantKind(t, err, app.KindNotFound, "not_found")
	_, err = e.App.StartLearningPath(ctx, a, e.CourseID, e.Concepts["c"], e.Concepts["a"])
	wantKind(t, err, app.KindInvalid, "not_prerequisite")

	p := must[app.LearningPathView](t)(e.App.StartLearningPath(ctx, a, e.CourseID, e.Concepts["a"], e.Concepts["c"]))
	if len(p.Steps) != 3 || p.Steps[0].ID != e.Concepts["a"] || p.CurrentStep == nil || *p.CurrentStep != e.Concepts["a"] {
		t.Fatalf("path %+v", p)
	}
	// Master a, b and c → path completes.
	for _, c := range []string{"a", "b", "c"} {
		e.ActiveQuestions(t, 5, 3, c)
		s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts[c])}))
		answerAll(t, e, a, s.ID, "A")
	}
	p = must[app.LearningPathView](t)(e.App.LearningPath(ctx, a, e.CourseID))
	if p.CompletedAt == nil || p.CurrentStep != nil {
		t.Fatalf("path should be complete: %+v", p)
	}
	if err := e.App.AbandonLearningPath(ctx, a, e.CourseID); err != nil {
		t.Fatal(err)
	}
	_, err = e.App.LearningPath(ctx, a, e.CourseID)
	wantKind(t, err, app.KindNotFound, "not_found")
}

func TestFormatReportAndOverview(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)
	e.ActiveQuestions(t, 2, 1, "a")
	s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindQuiz, CourseID: e.CourseID, ConceptID: ptr(e.Concepts["a"])}))
	answerAll(t, e, a, s.ID, "A")
	rep := must[[]app.FormatStat](t)(e.App.FormatReport(ctx, a, e.CourseID))
	if len(rep) != 2 || rep[0].Format != "standard" || rep[0].Answered != 2 || rep[0].Accuracy != 1 || rep[1].Answered != 0 {
		t.Fatalf("format report %+v", rep)
	}
	ov, n := must2(t)(e.App.MasteryOverview(ctx, testutil.CourseSlug))
	if n != 1 || len(ov) != 4 || ov[0].Unassessed != 1 {
		t.Fatalf("overview %d %+v", n, ov)
	}
}

func must2(t *testing.T) func([]app.MasteryOverviewItem, int, error) ([]app.MasteryOverviewItem, int) {
	return func(v []app.MasteryOverviewItem, n int, err error) ([]app.MasteryOverviewItem, int) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v, n
	}
}

func TestPlacementRequiresQuestionsAndSeedsMastery(t *testing.T) {
	e := testutil.NewEnv(t)
	a := e.Learner(t, "l@example.com", false)

	// No Active Questions: refuse to start, and nothing counts as done.
	_, err := e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindPlacement, CourseID: e.CourseID})
	wantKind(t, err, app.KindInvalid, "no_questions")
	if must[bool](t)(e.App.LearnerPlacementDone(ctx, a.LearnerID)) {
		t.Fatal("placementDone without any Answers")
	}

	e.ActiveQuestions(t, 2, 2, "c")
	s := must[app.SessionView](t)(e.App.CreateSession(ctx, a, app.CreateSessionInput{Kind: app.KindPlacement, CourseID: e.CourseID}))
	q := must[*app.ServedQuestion](t)(e.App.NextQuestion(ctx, a, s.ID))
	if q == nil {
		t.Fatal("no probe served")
	}
	must[app.AnswerResult](t)(e.App.SubmitAnswer(ctx, a, app.SubmitAnswerInput{SessionID: s.ID, QuestionID: q.ID, ChoiceID: e.Choice(t, q.ID, "A")}))
	must[app.SessionResult](t)(e.App.FinishSession(ctx, a, s.ID))

	// A single probe Answer is enough to leave Unassessed (PlacementWeight).
	assessed := false
	for _, m := range must[[]app.MasteryView](t)(e.App.CourseMastery(ctx, a, e.CourseID)) {
		if m.ConceptID == e.Concepts["c"] && m.Level != domain.Unassessed {
			assessed = true
		}
	}
	if !assessed {
		t.Fatal("placement probe did not seed Mastery")
	}
	if !must[bool](t)(e.App.LearnerPlacementDone(ctx, a.LearnerID)) {
		t.Fatal("placementDone not set after answering")
	}
}
