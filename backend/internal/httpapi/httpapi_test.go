package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/auth"
	"github.com/thekogo/tutor-platform/backend/internal/httpapi"
	"github.com/thekogo/tutor-platform/backend/internal/testutil"
)

const origin = "http://localhost:3000"

// fakeIdP authenticates whoever's email is passed as the authorization code.
type fakeIdP struct{}

func (fakeIdP) AuthCodeURL(_ context.Context, state, nonce, _ string) string {
	return "https://idp.test/auth?state=" + url.QueryEscape(state) + "&nonce=" + url.QueryEscape(nonce)
}

func (fakeIdP) Exchange(_ context.Context, code, _, _ string) (auth.Identity, error) {
	if code == "bad" {
		return auth.Identity{}, fmt.Errorf("bad code")
	}
	return auth.Identity{Email: code, EmailVerified: !strings.HasPrefix(code, "unverified"), Name: "Google " + code}, nil
}

type harness struct {
	t   *testing.T
	env *testutil.Env
	srv *httptest.Server
}

func newHarness(t *testing.T, devLogin bool, allowedOrigins ...string) *harness {
	t.Helper()
	env := testutil.NewEnv(t)
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	authSvc := auth.New(env.Pool, auth.Config{AppBaseURL: origin, DevLogin: devLogin}, fakeIdP{})
	mux := http.NewServeMux()
	mux.Handle("/api/", httpapi.New(env.App, authSvc, httpapi.Config{AppBaseURL: origin, AllowedOrigins: allowedOrigins}, log))
	srv := httptest.NewServer(httpapi.Common(mux, log))
	t.Cleanup(srv.Close)
	return &harness{t: t, env: env, srv: srv}
}

type client struct {
	h    *harness
	http *http.Client
}

func (h *harness) client() *client {
	jar, _ := cookiejar.New(nil)
	return &client{h: h, http: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type resp struct {
	status int
	body   []byte
	header http.Header
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (c *client) do(method, path string, body any, hdr ...string) resp {
	c.h.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, c.h.srv.URL+path, rd)
	if err != nil {
		c.h.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		req.Header.Set("Origin", origin)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		if hdr[i+1] == "" {
			req.Header.Del(hdr[i])
		} else {
			req.Header.Set(hdr[i], hdr[i+1])
		}
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.h.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{status: res.StatusCode, body: b, header: res.Header}
}

func (c *client) devLogin(email string) {
	c.h.t.Helper()
	r := c.do(http.MethodPost, "/api/auth/dev-login", map[string]string{"email": email})
	if r.status != http.StatusNoContent {
		c.h.t.Fatalf("dev login %d %s", r.status, r.body)
	}
}

func wantStatus(t *testing.T, r resp, status int, code string) {
	t.Helper()
	if r.status != status {
		t.Fatalf("want %d, got %d: %s", status, r.status, r.body)
	}
	if code != "" {
		var e httpapi.Error
		r.json(t, &e)
		if e.Code != code || e.Message == "" {
			t.Fatalf("want error code %s, got %+v", code, e)
		}
	}
}

func TestAuthRequiredAndOrigin(t *testing.T) {
	h := newHarness(t, true)
	c := h.client()
	for _, p := range []string{"/api/me", "/api/courses", "/api/courses/" + h.env.CourseID.String() + "/mastery", "/api/admin/learners"} {
		wantStatus(t, c.do(http.MethodGet, p, nil), http.StatusUnauthorized, "unauthenticated")
	}
	wantStatus(t, c.do(http.MethodPost, "/api/sessions", map[string]string{"kind": "quiz"}), http.StatusUnauthorized, "unauthenticated")
	wantStatus(t, c.do(http.MethodGet, "/api/nope", nil), http.StatusNotFound, "not_found")

	// Unsafe methods need a matching Origin, even for login.
	wantStatus(t, c.do(http.MethodPost, "/api/auth/dev-login", map[string]string{"email": "x@example.com"}, "Origin", ""), http.StatusForbidden, "bad_origin")
	wantStatus(t, c.do(http.MethodPost, "/api/auth/dev-login", map[string]string{"email": "x@example.com"}, "Origin", "https://evil.example"), http.StatusForbidden, "bad_origin")

	c.devLogin("Owner@Example.com")
	r := c.do(http.MethodGet, "/api/me", nil)
	wantStatus(t, r, http.StatusOK, "")
	var me httpapi.Learner
	r.json(t, &me)
	if !me.IsOwner || me.Email != "owner@example.com" || me.PlacementDone == nil || *me.PlacementDone {
		t.Fatalf("first dev login must be Owner: %+v", me)
	}
	if r.header.Get("X-Request-Id") == "" {
		t.Fatal("missing request id")
	}
	// Session cookie attributes.
	r = c.do(http.MethodPost, "/api/auth/dev-login", map[string]string{"email": "owner@example.com"})
	sc := r.header.Get("Set-Cookie")
	if !strings.Contains(sc, "tp_session=") || !strings.Contains(sc, "HttpOnly") || !strings.Contains(sc, "SameSite=Lax") {
		t.Fatalf("cookie %q", sc)
	}
	wantStatus(t, c.do(http.MethodPost, "/api/auth/logout", nil), http.StatusNoContent, "")
	wantStatus(t, c.do(http.MethodGet, "/api/me", nil), http.StatusUnauthorized, "unauthenticated")

	// Malformed body / bad UUID → 400 JSON.
	c.devLogin("owner@example.com")
	wantStatus(t, c.do(http.MethodGet, "/api/sessions/not-a-uuid", nil), http.StatusBadRequest, "bad_request")
	r = c.do(http.MethodPost, "/api/sessions", nil, "Content-Type", "application/json")
	wantStatus(t, r, http.StatusBadRequest, "bad_request")
}

func TestDevLoginDisabled(t *testing.T) {
	h := newHarness(t, false)
	wantStatus(t, h.client().do(http.MethodPost, "/api/auth/dev-login", map[string]string{"email": "x@example.com"}), http.StatusNotFound, "not_found")
}

func TestGoogleLoginInviteOnly(t *testing.T) {
	h := newHarness(t, false)
	login := func(c *client, email, invite string) resp {
		path := "/api/auth/google/login"
		if invite != "" {
			path += "?invite=" + url.QueryEscape(invite)
		}
		r := c.do(http.MethodGet, path, nil)
		wantStatus(t, r, http.StatusFound, "")
		loc, _ := url.Parse(r.header.Get("Location"))
		state := loc.Query().Get("state")
		return c.do(http.MethodGet, "/api/auth/google/callback?code="+url.QueryEscape(email)+"&state="+url.QueryEscape(state), nil)
	}
	owner := h.client()
	r := login(owner, "boss@example.com", "")
	wantStatus(t, r, http.StatusFound, "")
	if r.header.Get("Location") != origin+"/" {
		t.Fatalf("redirect %q", r.header.Get("Location"))
	}
	var me httpapi.Learner
	r = owner.do(http.MethodGet, "/api/me", nil)
	r.json(t, &me)
	if !me.IsOwner {
		t.Fatal("first Google login must become Owner")
	}

	// Not invited → 403.
	stranger := h.client()
	wantStatus(t, login(stranger, "stranger@example.com", ""), http.StatusForbidden, "invite_required")
	wantStatus(t, login(stranger, "stranger@example.com", "forged"), http.StatusForbidden, "invite_invalid")
	wantStatus(t, login(stranger, "unverified@example.com", ""), http.StatusForbidden, "email_unverified")

	// State mismatch → 403.
	wantStatus(t, stranger.do(http.MethodGet, "/api/auth/google/callback?code=x@example.com&state=wrong", nil), http.StatusForbidden, "invalid_state")

	// Owner invites; invite works once.
	r = owner.do(http.MethodPost, "/api/admin/invites", nil)
	wantStatus(t, r, http.StatusCreated, "")
	var inv httpapi.CreateInvite201JSONResponse
	r.json(t, &inv)
	u, _ := url.Parse(inv.Url)
	token := u.Query().Get("invite")
	if token == "" || !strings.HasPrefix(inv.Url, origin+"/login?invite=") {
		t.Fatalf("invite url %q", inv.Url)
	}
	friend := h.client()
	wantStatus(t, login(friend, "friend@example.com", token), http.StatusFound, "")
	r = friend.do(http.MethodGet, "/api/me", nil)
	r.json(t, &me)
	if me.IsOwner || me.Email != "friend@example.com" {
		t.Fatalf("invited learner %+v", me)
	}
	wantStatus(t, login(h.client(), "second@example.com", token), http.StatusForbidden, "invite_invalid")

	r = owner.do(http.MethodGet, "/api/admin/invites", nil)
	var invs []httpapi.Invite
	r.json(t, &invs)
	if len(invs) != 1 || invs[0].UsedBy == nil || *invs[0].UsedBy != "friend@example.com" {
		t.Fatalf("invites %+v", invs)
	}
	// Invite list never contains the token.
	if strings.Contains(string(r.body), token) {
		t.Fatal("invite token leaked in list")
	}
}

func TestOwnerOnlyEndpoints(t *testing.T) {
	h := newHarness(t, true)
	owner, learner := h.client(), h.client()
	owner.devLogin("owner@example.com")
	learner.devLogin("learner@example.com")
	qid := uuid.New()
	for _, ep := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/invites"},
		{http.MethodPost, "/api/admin/invites"},
		{http.MethodGet, "/api/admin/learners"},
		{http.MethodGet, "/api/admin/flags"},
		{http.MethodGet, "/api/admin/content-requests"},
		{http.MethodPost, "/api/admin/questions/" + qid.String() + "/retire"},
	} {
		wantStatus(t, learner.do(ep.method, ep.path, nil), http.StatusForbidden, "forbidden")
	}
	r := owner.do(http.MethodGet, "/api/admin/learners", nil)
	wantStatus(t, r, http.StatusOK, "")
	var ls []httpapi.Learner
	r.json(t, &ls)
	if len(ls) != 2 {
		t.Fatalf("learners %+v", ls)
	}
	wantStatus(t, owner.do(http.MethodPost, "/api/admin/questions/"+qid.String()+"/retire", nil), http.StatusNotFound, "not_found")
}

func createSession(t *testing.T, c *client, body map[string]any) httpapi.StudySession {
	t.Helper()
	r := c.do(http.MethodPost, "/api/sessions", body)
	wantStatus(t, r, http.StatusCreated, "")
	var s httpapi.StudySession
	r.json(t, &s)
	return s
}

func TestAnswerKeySecrecyAndFeedback(t *testing.T) {
	h := newHarness(t, true)
	c := h.client()
	c.devLogin("l@example.com")
	ids := h.env.ActiveQuestions(t, 2, 1, "a")
	s := createSession(t, c, map[string]any{"kind": "quiz", "courseId": h.env.CourseID, "conceptId": h.env.Concepts["a"]})

	r := c.do(http.MethodGet, "/api/sessions/"+s.Id.String()+"/next", nil)
	wantStatus(t, r, http.StatusOK, "")
	raw := string(r.body)
	for _, leak := range []string{"isCorrect", "is_correct", "explanation", "Because A", "correct"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("served question leaks %q: %s", leak, raw)
		}
	}
	var q httpapi.ServedQuestion
	r.json(t, &q)
	if len(q.Choices) != 4 || q.Position != 1 {
		t.Fatalf("served %+v", q)
	}
	// Result is unavailable before finishing.
	wantStatus(t, c.do(http.MethodGet, "/api/sessions/"+s.Id.String()+"/result", nil), http.StatusConflict, "not_finished")

	// Non-exam: feedback right after answering.
	r = c.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/answers", map[string]any{"questionId": q.Id, "choiceId": q.Choices[1].Id, "timeSpentMs": 1200})
	wantStatus(t, r, http.StatusOK, "")
	var ar httpapi.AnswerResult
	r.json(t, &ar)
	if !ar.Recorded || ar.Correct == nil || *ar.Correct || ar.CorrectChoiceId == nil || *ar.CorrectChoiceId != q.Choices[0].Id || ar.ExplanationMd == nil {
		t.Fatalf("feedback %+v", ar)
	}
	wantStatus(t, c.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/answers", map[string]any{"questionId": q.Id, "choiceId": q.Choices[0].Id}), http.StatusConflict, "already_answered")
	wantStatus(t, c.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/answers", map[string]any{"questionId": uuid.New(), "choiceId": q.Choices[0].Id}), http.StatusConflict, "not_in_session")

	// Another learner cannot see or use the session.
	other := h.client()
	other.devLogin("other@example.com")
	wantStatus(t, other.do(http.MethodGet, "/api/sessions/"+s.Id.String(), nil), http.StatusNotFound, "not_found")
	wantStatus(t, other.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/finish", nil), http.StatusNotFound, "not_found")

	r = c.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/finish", nil)
	wantStatus(t, r, http.StatusOK, "")
	var res httpapi.SessionResult
	r.json(t, &res)
	if res.Answered != 1 || *res.Total != 2 || len(res.Review) != 2 || res.Review[0].ExplanationMd == "" {
		t.Fatalf("result %+v", res)
	}
	wantStatus(t, c.do(http.MethodGet, "/api/sessions/"+s.Id.String()+"/next", nil), http.StatusGone, "session_finished")
	_ = ids
}

func TestExamSecrecyAndDeadline(t *testing.T) {
	h := newHarness(t, true)
	c := h.client()
	c.devLogin("l@example.com")
	ids := h.env.ActiveQuestions(t, 2, 2, "b")
	exam, err := h.env.App.CreateExam(context.Background(), testutil.CourseSlug, "Mock", 30, ids)
	if err != nil {
		t.Fatal(err)
	}
	r := c.do(http.MethodGet, "/api/courses/"+h.env.CourseID.String()+"/exams", nil)
	var exams []httpapi.ExamSummary
	r.json(t, &exams)
	if len(exams) != 1 || exams[0].QuestionCount != 2 || exams[0].TimeLimitSeconds != 30 {
		t.Fatalf("exams %+v", exams)
	}
	now := time.Now()
	h.env.App.SetClock(func() time.Time { return now })
	s := createSession(t, c, map[string]any{"kind": "exam", "courseId": h.env.CourseID, "examId": exam.ID})
	if s.Deadline == nil || s.TotalQuestions == nil || *s.TotalQuestions != 2 {
		t.Fatalf("exam session %+v", s)
	}
	var q httpapi.ServedQuestion
	c.do(http.MethodGet, "/api/sessions/"+s.Id.String()+"/next", nil).json(t, &q)
	r = c.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/answers", map[string]any{"questionId": q.Id, "choiceId": q.Choices[0].Id})
	wantStatus(t, r, http.StatusOK, "")
	if strings.TrimSpace(string(r.body)) != `{"recorded":true}` {
		t.Fatalf("exam answer must not reveal anything: %s", r.body)
	}
	// After the deadline the server refuses more Answers.
	h.env.App.SetClock(func() time.Time { return now.Add(31 * time.Second) })
	wantStatus(t, c.do(http.MethodGet, "/api/sessions/"+s.Id.String()+"/next", nil), http.StatusGone, "deadline_passed")
	wantStatus(t, c.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/answers", map[string]any{"questionId": ids[0], "choiceId": q.Choices[0].Id}), http.StatusGone, "deadline_passed")
	r = c.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/finish", nil)
	wantStatus(t, r, http.StatusOK, "")
	var res httpapi.SessionResult
	r.json(t, &res)
	if res.Answered != 1 || len(res.Review) != 2 {
		t.Fatalf("exam result %+v", res)
	}
	unanswered := 0
	for _, it := range res.Review {
		if it.ChosenChoiceId == nil {
			unanswered++
		}
	}
	if unanswered != 1 {
		t.Fatalf("review should include the unanswered question with null choice: %+v", res.Review)
	}
}

func TestFlagEffects(t *testing.T) {
	h := newHarness(t, true)
	owner, a, b := h.client(), h.client(), h.client()
	owner.devLogin("owner@example.com")
	a.devLogin("a@example.com")
	b.devLogin("b@example.com")
	ids := h.env.ActiveQuestions(t, 1, 1, "d")

	wantStatus(t, a.do(http.MethodPost, "/api/questions/"+ids[0].String()+"/flags", map[string]string{"reason": "wrong"}), http.StatusForbidden, "forbidden")

	s := createSession(t, a, map[string]any{"kind": "quiz", "courseId": h.env.CourseID, "conceptId": h.env.Concepts["d"]})
	var q httpapi.ServedQuestion
	a.do(http.MethodGet, "/api/sessions/"+s.Id.String()+"/next", nil).json(t, &q)
	wantStatus(t, a.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/answers", map[string]any{"questionId": q.Id, "choiceId": q.Choices[0].Id}), http.StatusOK, "")
	wantStatus(t, a.do(http.MethodPost, "/api/questions/"+q.Id.String()+"/flags", map[string]string{"reason": strings.Repeat("x", 2001)}), http.StatusUnprocessableEntity, "invalid_reason")
	wantStatus(t, a.do(http.MethodPost, "/api/questions/"+q.Id.String()+"/flags", map[string]string{"reason": "the key is B"}), http.StatusCreated, "")

	// Pulled for everyone: b cannot get it in a quiz.
	wantStatus(t, b.do(http.MethodPost, "/api/sessions", map[string]any{"kind": "quiz", "courseId": h.env.CourseID, "conceptId": h.env.Concepts["d"]}), http.StatusUnprocessableEntity, "no_questions")
	// Its Answers no longer count toward a's Mastery.
	var ms []httpapi.Mastery
	a.do(http.MethodGet, "/api/courses/"+h.env.CourseID.String()+"/mastery", nil).json(t, &ms)
	for _, m := range ms {
		if m.Evidence != 0 {
			t.Fatalf("flagged answer still counted: %+v", m)
		}
	}
	// Owner sees the Flag and the Content Request.
	var flags []httpapi.Flag
	owner.do(http.MethodGet, "/api/admin/flags", nil).json(t, &flags)
	if len(flags) != 1 || flags[0].Reason != "the key is B" || flags[0].LearnerName == nil {
		t.Fatalf("flags %+v", flags)
	}
	var reqs []httpapi.ContentRequest
	owner.do(http.MethodGet, "/api/admin/content-requests", nil).json(t, &reqs)
	if len(reqs) != 1 || reqs[0].ConceptName != "Concept D" {
		t.Fatalf("content requests %+v", reqs)
	}
	wantStatus(t, owner.do(http.MethodPost, "/api/admin/questions/"+q.Id.String()+"/retire", nil), http.StatusNoContent, "")
	owner.do(http.MethodGet, "/api/admin/flags", nil).json(t, &flags)
	if len(flags) != 0 {
		t.Fatal("retiring should resolve open flags")
	}
}

func TestCourseProgressEndpoints(t *testing.T) {
	h := newHarness(t, true)
	c := h.client()
	c.devLogin("l@example.com")
	cid := h.env.CourseID.String()

	var courses []httpapi.CourseSummary
	c.do(http.MethodGet, "/api/courses", nil).json(t, &courses)
	if len(courses) != 1 || courses[0].Slug != testutil.CourseSlug {
		t.Fatalf("courses %+v", courses)
	}
	var g httpapi.CourseGraph
	c.do(http.MethodGet, "/api/courses/"+cid, nil).json(t, &g)
	if len(g.Concepts) != 4 || len(g.Topics) != 2 || len(g.QuestionFormats) != 2 {
		t.Fatalf("graph %+v", g)
	}
	wantStatus(t, c.do(http.MethodGet, "/api/courses/"+uuid.NewString(), nil), http.StatusNotFound, "not_found")

	if err := h.env.App.UpsertLesson(context.Background(), testutil.CourseSlug, "b", "# Lesson B"); err != nil {
		t.Fatal(err)
	}
	var d httpapi.ConceptDetail
	c.do(http.MethodGet, "/api/concepts/"+h.env.Concepts["b"].String(), nil).json(t, &d)
	if d.LessonMd == nil || *d.LessonMd != "# Lesson B" || len(d.PrerequisiteIds) != 1 || len(d.DependentIds) != 1 || d.Mastery.Level != "unassessed" {
		t.Fatalf("concept detail %+v", d)
	}

	// Make b Weak with a Unassessed → a diagnose recommendation exists.
	h.env.ActiveQuestions(t, 5, 2, "b")
	s := createSession(t, c, map[string]any{"kind": "quiz", "courseId": h.env.CourseID, "conceptId": h.env.Concepts["b"]})
	for {
		r := c.do(http.MethodGet, "/api/sessions/"+s.Id.String()+"/next", nil)
		if r.status == http.StatusNoContent {
			break
		}
		var q httpapi.ServedQuestion
		r.json(t, &q)
		wantStatus(t, c.do(http.MethodPost, "/api/sessions/"+s.Id.String()+"/answers", map[string]any{"questionId": q.Id, "choiceId": q.Choices[3].Id}), http.StatusOK, "")
	}
	var recs []httpapi.Recommendation
	c.do(http.MethodGet, "/api/courses/"+cid+"/recommendations", nil).json(t, &recs)
	if len(recs) == 0 || len(recs[0].Path) == 0 {
		t.Fatalf("recommendations %+v", recs)
	}
	var fr []httpapi.FormatStat
	c.do(http.MethodGet, "/api/courses/"+cid+"/format-report", nil).json(t, &fr)
	if fr[0].Answered != 5 || fr[0].Correct != 0 {
		t.Fatalf("format report %+v", fr)
	}

	wantStatus(t, c.do(http.MethodGet, "/api/courses/"+cid+"/learning-path", nil), http.StatusNotFound, "not_found")
	r := c.do(http.MethodPost, "/api/courses/"+cid+"/learning-path", map[string]any{"rootConceptId": recs[0].ConceptId, "targetConceptId": recs[0].TargetConceptId})
	wantStatus(t, r, http.StatusCreated, "")
	var lp httpapi.LearningPath
	r.json(t, &lp)
	if len(lp.Steps) != len(recs[0].Path) || lp.CurrentStepConceptId == nil {
		t.Fatalf("learning path %+v vs rec %+v", lp, recs[0])
	}
	wantStatus(t, c.do(http.MethodGet, "/api/courses/"+cid+"/learning-path", nil), http.StatusOK, "")
	wantStatus(t, c.do(http.MethodDelete, "/api/courses/"+cid+"/learning-path", nil), http.StatusNoContent, "")
	wantStatus(t, c.do(http.MethodPost, "/api/courses/"+cid+"/learning-path", map[string]any{"rootConceptId": h.env.Concepts["c"], "targetConceptId": h.env.Concepts["a"]}), http.StatusUnprocessableEntity, "not_prerequisite")
}

func TestAllowedOriginsWildcard(t *testing.T) {
	c := newHarness(t, true, "*").client()
	// Any Origin (e.g. an ngrok tunnel) and a missing Origin are accepted.
	for _, o := range []string{"https://abc123.ngrok-free.app", ""} {
		r := c.do(http.MethodPost, "/api/auth/dev-login", map[string]string{"email": "x@example.com"}, "Origin", o)
		if r.status != http.StatusNoContent {
			t.Fatalf("origin %q: %d %s", o, r.status, r.body)
		}
	}
}

func TestAllowedOriginsList(t *testing.T) {
	c := newHarness(t, true, origin, "https://tunnel.example").client()
	wantStatus(t, c.do(http.MethodPost, "/api/auth/dev-login", map[string]string{"email": "x@example.com"}, "Origin", "https://tunnel.example/"), http.StatusNoContent, "")
	wantStatus(t, c.do(http.MethodPost, "/api/auth/dev-login", map[string]string{"email": "x@example.com"}, "Origin", "https://evil.example"), http.StatusForbidden, "bad_origin")
}
