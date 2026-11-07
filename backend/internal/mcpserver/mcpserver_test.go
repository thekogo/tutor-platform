package mcpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thekogo/tutor-platform/backend/internal/mcpserver"
	"github.com/thekogo/tutor-platform/backend/internal/testutil"
)

var ctx = context.Background()

func connect(t *testing.T, e *testutil.Env) *mcp.ClientSession {
	t.Helper()
	server := mcpserver.NewServer(e.App)
	st, ct := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// call invokes a tool and decodes its structured output; it returns the error text for tool errors.
func call(t *testing.T, cs *mcp.ClientSession, name string, args any, out any) string {
	t.Helper()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return "protocol: " + err.Error()
	}
	if res.IsError {
		var sb strings.Builder
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				sb.WriteString(tc.Text)
			}
		}
		return sb.String()
	}
	if out != nil {
		b, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("decode %s output: %v", name, err)
		}
	}
	return ""
}

func question(primary string, correct ...string) map[string]any {
	choices := []map[string]any{}
	for _, l := range []string{"A", "B", "C"} {
		isC := false
		for _, c := range correct {
			isC = isC || c == l
		}
		choices = append(choices, map[string]any{"label": l, "body_md": "choice " + l, "is_correct": isC})
	}
	return map[string]any{
		"stem_md": "2 + 2 = ?", "explanation_md": "It is 4.", "choices": choices,
		"primary_concept": primary, "concepts": []string{primary}, "difficulty": 1, "format": "standard", "source": "ai_generated",
	}
}

func TestToolsListed(t *testing.T) {
	e := testutil.NewEnv(t)
	cs := connect(t, e)
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, tool := range res.Tools {
		have[tool.Name] = true
	}
	for _, name := range []string{"get_course_graph", "import_course_graph", "upsert_topic", "upsert_concept", "archive_concept",
		"set_prerequisites", "upsert_lesson", "create_questions", "list_questions_for_verification", "submit_verification",
		"update_question", "retire_question", "create_exam", "list_flags", "resolve_flag", "list_content_requests",
		"fulfill_content_request", "get_pool_stats", "get_mastery_overview"} {
		if !have[name] {
			t.Errorf("missing tool %s", name)
		}
	}
}

func TestGraphToolsValidate(t *testing.T) {
	e := testutil.NewEnv(t)
	cs := connect(t, e)
	var g struct {
		Course   struct{ ID string }
		Concepts []struct {
			ID            string
			Prerequisites []string
		}
	}
	if msg := call(t, cs, "get_course_graph", map[string]any{"course": testutil.CourseSlug}, &g); msg != "" {
		t.Fatal(msg)
	}
	if g.Course.ID != testutil.CourseSlug || len(g.Concepts) != 4 {
		t.Fatalf("graph %+v", g)
	}
	if msg := call(t, cs, "set_prerequisites", map[string]any{"course": testutil.CourseSlug, "concept": "a", "prerequisites": []string{"c"}}, nil); !strings.Contains(msg, "prerequisite_cycle") {
		t.Fatalf("cycle not rejected: %q", msg)
	}
	if msg := call(t, cs, "upsert_concept", map[string]any{"course": testutil.CourseSlug, "id": "e", "topic": "nope", "name": "E"}, nil); !strings.Contains(msg, "unknown_topic") {
		t.Fatalf("unknown topic not rejected: %q", msg)
	}
	if msg := call(t, cs, "upsert_concept", map[string]any{"course": testutil.CourseSlug, "id": "e", "topic": "t2", "name": "E", "prerequisites": []string{"d"}}, nil); msg != "" {
		t.Fatal(msg)
	}
	if msg := call(t, cs, "upsert_lesson", map[string]any{"course": testutil.CourseSlug, "concept": "e", "body_md": "# E"}, nil); msg != "" {
		t.Fatal(msg)
	}
	if msg := call(t, cs, "upsert_topic", map[string]any{"course": testutil.CourseSlug, "id": "t3", "name": "T3", "exam_weight": 2.5}, nil); msg != "" {
		t.Fatal(msg)
	}
	if msg := call(t, cs, "archive_concept", map[string]any{"course": testutil.CourseSlug, "concept": "e"}, nil); msg != "" {
		t.Fatal(msg)
	}
	doc := testutil.Graph()
	doc.Concepts[0].Prerequisites = []string{"d"} // a ← d ← a: cycle
	if msg := call(t, cs, "import_course_graph", map[string]any{"graph": doc}, nil); !strings.Contains(msg, "cycle") {
		t.Fatalf("import cycle not rejected: %q", msg)
	}
	var imp struct {
		Concepts      int      `json:"concepts"`
		NotInDocument []string `json:"not_in_document"`
	}
	if msg := call(t, cs, "import_course_graph", map[string]any{"graph": testutil.Graph()}, &imp); msg != "" {
		t.Fatal(msg)
	}
	if imp.Concepts != 4 {
		t.Fatalf("import %+v", imp)
	}
}

func TestCreateQuestionsValidation(t *testing.T) {
	e := testutil.NewEnv(t)
	cs := connect(t, e)
	noExpl := question("a", "A")
	noExpl["explanation_md"] = ""
	notAmongTags := question("a", "A")
	notAmongTags["primary_concept"] = "b"
	unknownTag := question("a", "A")
	unknownTag["concepts"] = []string{"a", "ghost"}
	cases := map[string]map[string]any{
		"exactly one choice must be correct": question("a", "A", "B"),
		"explanation_md":                     noExpl,
		"must be one of concepts":            notAmongTags,
		"unknown or archived concept":        unknownTag,
	}
	for want, q := range cases {
		msg := call(t, cs, "create_questions", map[string]any{"course": testutil.CourseSlug, "questions": []any{q}}, nil)
		if !strings.Contains(msg, want) {
			t.Errorf("want error containing %q, got %q", want, msg)
		}
	}
	// Missing required field is rejected by the input schema.
	bad := question("a", "A")
	delete(bad, "difficulty")
	if msg := call(t, cs, "create_questions", map[string]any{"course": testutil.CourseSlug, "questions": []any{bad}}, nil); msg == "" {
		t.Error("missing difficulty accepted")
	}
	var stats struct {
		Concepts []struct{ Draft int } `json:"concepts"`
	}
	call(t, cs, "get_pool_stats", map[string]any{"course": testutil.CourseSlug}, &stats)
	for _, s := range stats.Concepts {
		if s.Draft != 0 {
			t.Fatal("invalid questions were stored")
		}
	}
}

func TestVerificationFlow(t *testing.T) {
	e := testutil.NewEnv(t)
	cs := connect(t, e)
	var created struct {
		QuestionIDs []string `json:"question_ids"`
	}
	if msg := call(t, cs, "create_questions", map[string]any{"course": testutil.CourseSlug, "questions": []any{question("a", "B"), question("b", "C")}}, &created); msg != "" {
		t.Fatal(msg)
	}
	if len(created.QuestionIDs) != 2 {
		t.Fatalf("created %+v", created)
	}

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_questions_for_verification", Arguments: map[string]any{"course": testutil.CourseSlug, "limit": 10}})
	if err != nil || res.IsError {
		t.Fatalf("list: %v %+v", err, res)
	}
	raw, _ := json.Marshal(res)
	for _, leak := range []string{"is_correct", "explanation", "It is 4"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("verification list leaks %q: %s", leak, raw)
		}
	}
	var list struct {
		Questions []struct {
			ID      string `json:"id"`
			StemMd  string `json:"stem_md"`
			Choices []struct {
				Label  string `json:"label"`
				BodyMd string `json:"body_md"`
			} `json:"choices"`
		} `json:"questions"`
	}
	b, _ := json.Marshal(res.StructuredContent)
	_ = json.Unmarshal(b, &list)
	if len(list.Questions) != 2 || len(list.Questions[0].Choices) != 3 {
		t.Fatalf("list %+v", list)
	}

	var vr struct {
		Matched bool   `json:"matched"`
		Status  string `json:"status"`
	}
	// Mismatch keeps Draft with a note.
	if msg := call(t, cs, "submit_verification", map[string]any{"question_id": created.QuestionIDs[0], "chosen_label": "A", "note": "I got A"}, &vr); msg != "" {
		t.Fatal(msg)
	}
	if vr.Matched || vr.Status != "draft" {
		t.Fatalf("mismatch %+v", vr)
	}
	var detail struct {
		Status           string  `json:"status"`
		VerificationNote *string `json:"verification_note"`
	}
	call(t, cs, "get_question", map[string]any{"question_id": created.QuestionIDs[0]}, &detail)
	if detail.Status != "draft" || detail.VerificationNote == nil || !strings.Contains(*detail.VerificationNote, "I got A") {
		t.Fatalf("detail %+v", detail)
	}
	// Match activates.
	if msg := call(t, cs, "submit_verification", map[string]any{"question_id": created.QuestionIDs[1], "chosen_label": "C"}, &vr); msg != "" {
		t.Fatal(msg)
	}
	if !vr.Matched || vr.Status != "active" {
		t.Fatalf("match %+v", vr)
	}
	if msg := call(t, cs, "submit_verification", map[string]any{"question_id": "nope", "chosen_label": "C"}, nil); !strings.Contains(msg, "invalid_id") {
		t.Fatalf("bad id: %q", msg)
	}

	// Fix the mismatched question, re-verify, then build an Exam from Active Questions.
	if msg := call(t, cs, "update_question", map[string]any{"question_id": created.QuestionIDs[0], "explanation_md": "Clearer."}, nil); msg != "" {
		t.Fatal(msg)
	}
	call(t, cs, "submit_verification", map[string]any{"question_id": created.QuestionIDs[0], "chosen_label": "B"}, &vr)
	if !vr.Matched {
		t.Fatal("re-verification failed")
	}
	var exam struct {
		ID string `json:"id"`
	}
	if msg := call(t, cs, "create_exam", map[string]any{"course": testutil.CourseSlug, "name": "Mock", "time_limit_s": 600, "question_ids": created.QuestionIDs}, &exam); msg != "" || exam.ID == "" {
		t.Fatalf("create exam: %q %+v", msg, exam)
	}
}

func TestFlagResolutionWithKeyChange(t *testing.T) {
	e := testutil.NewEnv(t)
	cs := connect(t, e)
	a := e.Learner(t, "l@example.com", false)
	ids := e.ActiveQuestions(t, 1, 1, "a")
	sess, err := e.App.CreateSession(ctx, a, appQuiz(e))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.App.SubmitAnswer(ctx, a, appAnswer(sess.ID, ids[0], e.Choice(t, ids[0], "B"))); err != nil {
		t.Fatal(err)
	}
	if err := e.App.FlagQuestion(ctx, a, ids[0], "B is right"); err != nil {
		t.Fatal(err)
	}
	var flags struct {
		Flags []struct {
			ID       string `json:"id"`
			Question struct {
				Status  string `json:"status"`
				Choices []struct {
					IsCorrect bool `json:"is_correct"`
				} `json:"choices"`
			} `json:"question"`
		} `json:"flags"`
	}
	if msg := call(t, cs, "list_flags", map[string]any{}, &flags); msg != "" {
		t.Fatal(msg)
	}
	if len(flags.Flags) != 1 || flags.Flags[0].Question.Status != "flagged" {
		t.Fatalf("flags %+v", flags)
	}
	var rr struct {
		Status          string `json:"status"`
		ExcludedAnswers int    `json:"excluded_answers"`
	}
	if msg := call(t, cs, "resolve_flag", map[string]any{"flag_id": flags.Flags[0].ID, "key_changed": true, "resolution": "bogus"}, nil); !strings.Contains(msg, "invalid_resolution") {
		t.Fatalf("bad resolution: %q", msg)
	}
	if msg := call(t, cs, "resolve_flag", map[string]any{"flag_id": flags.Flags[0].ID, "key_changed": true, "note": "fixed key"}, &rr); msg != "" {
		t.Fatal(msg)
	}
	if rr.Status != "active" || rr.ExcludedAnswers != 1 {
		t.Fatalf("resolve %+v", rr)
	}
	var excluded bool
	if err := e.Pool.QueryRow(ctx, `SELECT excluded FROM answers WHERE question_id = $1`, ids[0]).Scan(&excluded); err != nil || !excluded {
		t.Fatalf("answer not excluded (%v)", err)
	}

	var reqs struct {
		Requests []struct {
			ID      string `json:"id"`
			Concept string `json:"concept"`
		} `json:"requests"`
	}
	call(t, cs, "list_content_requests", map[string]any{}, &reqs)
	if len(reqs.Requests) != 1 || reqs.Requests[0].Concept != "a" {
		t.Fatalf("requests %+v", reqs)
	}
	if msg := call(t, cs, "fulfill_content_request", map[string]any{"id": reqs.Requests[0].ID, "note": "added 10"}, nil); msg != "" {
		t.Fatal(msg)
	}
	if msg := call(t, cs, "fulfill_content_request", map[string]any{"id": reqs.Requests[0].ID}, nil); !strings.Contains(msg, "not_found") {
		t.Fatalf("double fulfill: %q", msg)
	}
	var ov struct {
		Learners int `json:"learners"`
	}
	if msg := call(t, cs, "get_mastery_overview", map[string]any{"course": testutil.CourseSlug}, &ov); msg != "" || ov.Learners != 1 {
		t.Fatalf("overview %q %+v", msg, ov)
	}
	if msg := call(t, cs, "retire_question", map[string]any{"question_id": ids[0]}, nil); msg != "" {
		t.Fatal(msg)
	}
}

func TestHTTPBearerAuth(t *testing.T) {
	e := testutil.NewEnv(t)
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := httptest.NewServer(mcpserver.Handler(e.App, "secret", log))
	defer srv.Close()
	post := func(token string) int {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, strings.NewReader(
			`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		return res.StatusCode
	}
	if got := post(""); got != http.StatusUnauthorized {
		t.Fatalf("no token: %d", got)
	}
	if got := post("wrong"); got != http.StatusUnauthorized {
		t.Fatalf("wrong token: %d", got)
	}
	if got := post("secret"); got != http.StatusOK {
		t.Fatalf("valid token: %d", got)
	}
	disabled := httptest.NewServer(mcpserver.Handler(e.App, "", log))
	defer disabled.Close()
	res, err := http.Post(disabled.URL, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("disabled: %d", res.StatusCode)
	}
}
