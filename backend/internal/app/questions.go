package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Question sources.
const (
	SourceAIGenerated = "ai_generated"
	SourceImported    = "imported"
)

// ChoiceInput is one choice of a Question.
type ChoiceInput struct {
	Label     string `json:"label" jsonschema:"short label shown to the Learner, e.g. A"`
	BodyMd    string `json:"body_md" jsonschema:"Markdown (KaTeX math allowed)"`
	IsCorrect bool   `json:"is_correct" jsonschema:"true for exactly one choice"`
}

// QuestionInput is a full Question definition. Concept identities are slugs.
type QuestionInput struct {
	StemMd         string        `json:"stem_md" jsonschema:"the Question text (Markdown)"`
	ExplanationMd  string        `json:"explanation_md" jsonschema:"worked solution shown after answering (required)"`
	Choices        []ChoiceInput `json:"choices" jsonschema:"2-8 choices, exactly one with is_correct=true"`
	Concepts       []string      `json:"concepts" jsonschema:"slugs of every Concept the Question tests"`
	PrimaryConcept string        `json:"primary_concept" jsonschema:"slug of the Primary Concept; must be among concepts"`
	Difficulty     int           `json:"difficulty" jsonschema:"1 (easy) to 3 (hard)"`
	Format         string        `json:"format" jsonschema:"Question Format slug of the Course, e.g. standard"`
	Source         string        `json:"source" jsonschema:"ai_generated or imported"`
	SourceRef      *string       `json:"source_ref,omitempty" jsonschema:"where an imported Question came from"`
}

func validateQuestion(g *CourseGraph, in *QuestionInput) []string {
	var errs []string
	add := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	if strings.TrimSpace(in.StemMd) == "" {
		add("stem_md is required")
	}
	if strings.TrimSpace(in.ExplanationMd) == "" {
		add("explanation_md (Explanation) is required")
	}
	if in.Difficulty < 1 || in.Difficulty > 3 {
		add("difficulty must be 1, 2, or 3")
	}
	formatOK := false
	for _, f := range g.Formats {
		if f.Slug == in.Format {
			formatOK = true
		}
	}
	if !formatOK {
		add("unknown format %q", in.Format)
	}
	if in.Source != SourceAIGenerated && in.Source != SourceImported {
		add("source must be %q or %q", SourceAIGenerated, SourceImported)
	}
	if len(in.Choices) < 2 || len(in.Choices) > 8 {
		add("a Question needs 2-8 choices")
	}
	labels := map[string]bool{}
	correct := 0
	for i, c := range in.Choices {
		if strings.TrimSpace(c.Label) == "" || labels[c.Label] {
			add("choices[%d]: label must be non-empty and unique", i)
		}
		labels[c.Label] = true
		if strings.TrimSpace(c.BodyMd) == "" {
			add("choices[%d]: body_md is required", i)
		}
		if c.IsCorrect {
			correct++
		}
	}
	if correct != 1 {
		add("exactly one choice must be correct (got %d)", correct)
	}
	if len(in.Concepts) == 0 {
		add("concepts must list at least one Concept")
	}
	seen := map[string]bool{}
	primaryOK := false
	for _, slug := range in.Concepts {
		if seen[slug] {
			add("concept %q listed twice", slug)
		}
		seen[slug] = true
		if _, ok := g.LiveBySlug(slug); !ok {
			add("unknown or archived concept %q", slug)
		}
		if slug == in.PrimaryConcept {
			primaryOK = true
		}
	}
	if !primaryOK {
		add("primary_concept %q must be one of concepts", in.PrimaryConcept)
	}
	return errs
}

func insertTags(ctx context.Context, q *store.Queries, g *CourseGraph, qid uuid.UUID, in *QuestionInput) error {
	for _, slug := range in.Concepts {
		c, _ := g.LiveBySlug(slug)
		if err := q.AddQuestionConcept(ctx, store.AddQuestionConceptParams{
			QuestionID: qid, ConceptID: c.ID, IsPrimary: slug == in.PrimaryConcept,
		}); err != nil {
			return err
		}
	}
	return nil
}

// CreateQuestions batch-creates Draft Questions (all or nothing).
func (s *Service) CreateQuestions(ctx context.Context, course string, in []QuestionInput) ([]uuid.UUID, error) {
	g, err := s.courseBySlug(ctx, course)
	if err != nil {
		return nil, err
	}
	if len(in) == 0 || len(in) > 200 {
		return nil, E(KindInvalid, "invalid_batch", "send 1-200 questions per call")
	}
	var details []string
	for i := range in {
		for _, e := range validateQuestion(g, &in[i]) {
			details = append(details, fmt.Sprintf("questions[%d]: %s", i, e))
		}
	}
	if len(details) > 0 {
		return nil, &Error{Kind: KindInvalid, Code: "invalid_questions", Message: "no Questions were created", Details: details}
	}
	ids := make([]uuid.UUID, 0, len(in))
	err = s.tx(ctx, func(q *store.Queries) error {
		for i := range in {
			qi := &in[i]
			row, err := q.CreateQuestion(ctx, store.CreateQuestionParams{
				CourseID: g.Course.ID, StemMd: qi.StemMd, ExplanationMd: qi.ExplanationMd, Difficulty: int16(qi.Difficulty), //nolint:gosec // validated 1..3
				Format: qi.Format, Source: qi.Source, SourceRef: qi.SourceRef,
			})
			if err != nil {
				return err
			}
			for j, c := range qi.Choices {
				if _, err := q.CreateChoice(ctx, store.CreateChoiceParams{
					QuestionID: row.ID, Label: c.Label, BodyMd: c.BodyMd, IsCorrect: c.IsCorrect, Position: int32(j + 1),
				}); err != nil {
					return err
				}
			}
			if err := insertTags(ctx, q, g, row.ID, qi); err != nil {
				return err
			}
			ids = append(ids, row.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// VerificationItem is a Draft Question for the independent verifier:
// no answer key and no Explanation.
type VerificationItem struct {
	ID         string               `json:"id"`
	StemMd     string               `json:"stem_md"`
	Choices    []VerificationChoice `json:"choices"`
	Format     string               `json:"format"`
	Difficulty int                  `json:"difficulty"`
	Concepts   []string             `json:"concepts"`
}

// VerificationChoice is a choice without correctness.
type VerificationChoice struct {
	Label  string `json:"label"`
	BodyMd string `json:"body_md"`
}

// ListQuestionsForVerification lists Draft Questions without answer keys.
func (s *Service) ListQuestionsForVerification(ctx context.Context, course string, limit int) ([]VerificationItem, error) {
	g, err := s.courseBySlug(ctx, course)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	qs, err := s.q.ListDraftQuestions(ctx, store.ListDraftQuestionsParams{CourseID: g.Course.ID, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(qs))
	for i := range qs {
		q := &qs[i]
		ids = append(ids, q.ID)
	}
	choices, err := s.q.ListChoicesForQuestions(ctx, ids)
	if err != nil {
		return nil, err
	}
	tags, err := s.q.ListConceptsForQuestions(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]VerificationItem, 0, len(qs))
	idx := map[uuid.UUID]int{}
	for i := range qs {
		q := &qs[i]
		idx[q.ID] = i
		out = append(out, VerificationItem{ID: q.ID.String(), StemMd: q.StemMd, Format: q.Format, Difficulty: int(q.Difficulty),
			Choices: []VerificationChoice{}, Concepts: []string{}})
	}
	for _, c := range choices {
		it := &out[idx[c.QuestionID]]
		it.Choices = append(it.Choices, VerificationChoice{Label: c.Label, BodyMd: c.BodyMd})
	}
	for _, t := range tags {
		it := &out[idx[t.QuestionID]]
		it.Concepts = append(it.Concepts, g.ConceptByID[t.ConceptID].Slug)
	}
	return out, nil
}

// VerificationResult is the outcome of submit_verification.
type VerificationResult struct {
	QuestionID string `json:"question_id"`
	Matched    bool   `json:"matched"`
	Status     string `json:"status"`
}

// SubmitVerification records the independent verifier's answer: a match makes
// the Question Active; a mismatch keeps it Draft with a verification note.
func (s *Service) SubmitVerification(ctx context.Context, questionID uuid.UUID, chosenLabel, note string) (VerificationResult, error) {
	var res VerificationResult
	err := s.tx(ctx, func(q *store.Queries) error {
		question, err := q.GetQuestionForUpdate(ctx, questionID)
		if err != nil {
			return mapNotFound(err, "question")
		}
		if question.Status != StatusDraft {
			return E(KindConflict, "not_draft", "only Draft Questions can be verified (status is %s)", question.Status)
		}
		choices, err := q.ListChoicesForQuestions(ctx, []uuid.UUID{questionID})
		if err != nil {
			return err
		}
		known := false
		var key string
		for _, c := range choices {
			if c.Label == chosenLabel {
				known = true
			}
			if c.IsCorrect {
				key = c.Label
			}
		}
		if !known {
			return E(KindInvalid, "unknown_label", "no choice has label %q", chosenLabel)
		}
		res = VerificationResult{QuestionID: questionID.String(), Matched: chosenLabel == key}
		if res.Matched {
			res.Status = StatusActive
			var n *string
			if strings.TrimSpace(note) != "" {
				n = &note
			}
			return q.SetQuestionStatus(ctx, store.SetQuestionStatusParams{ID: questionID, Status: StatusActive, VerificationNote: n})
		}
		res.Status = StatusDraft
		msg := fmt.Sprintf("verification mismatch at %s: independent verifier chose %q", s.now().UTC().Format(time.RFC3339), chosenLabel)
		if strings.TrimSpace(note) != "" {
			msg += ": " + note
		}
		return q.SetQuestionStatus(ctx, store.SetQuestionStatusParams{ID: questionID, Status: StatusDraft, VerificationNote: &msg})
	})
	return res, err
}

// QuestionDetail is a full Question for the Content Admin (includes the key).
type QuestionDetail struct {
	ID               string  `json:"id"`
	Course           string  `json:"course"`
	Status           string  `json:"status"`
	VerificationNote *string `json:"verification_note,omitempty"`
	QuestionInput
}

// GetQuestionDetail returns a Question with key and Explanation (Content Admin only).
func (s *Service) GetQuestionDetail(ctx context.Context, id uuid.UUID) (QuestionDetail, error) {
	question, err := s.q.GetQuestion(ctx, id)
	if err != nil {
		return QuestionDetail{}, mapNotFound(err, "question")
	}
	g, err := s.Graph(ctx, question.CourseID)
	if err != nil {
		return QuestionDetail{}, err
	}
	in, err := loadQuestionInput(ctx, s.q, g, question)
	if err != nil {
		return QuestionDetail{}, err
	}
	return QuestionDetail{ID: id.String(), Course: g.Course.Slug, Status: question.Status, VerificationNote: question.VerificationNote, QuestionInput: in}, nil
}

func loadQuestionInput(ctx context.Context, q *store.Queries, g *CourseGraph, question store.Question) (QuestionInput, error) {
	in := QuestionInput{StemMd: question.StemMd, ExplanationMd: question.ExplanationMd, Difficulty: int(question.Difficulty),
		Format: question.Format, Source: question.Source, SourceRef: question.SourceRef, Choices: []ChoiceInput{}, Concepts: []string{}}
	choices, err := q.ListChoicesForQuestions(ctx, []uuid.UUID{question.ID})
	if err != nil {
		return in, err
	}
	for _, c := range choices {
		in.Choices = append(in.Choices, ChoiceInput{Label: c.Label, BodyMd: c.BodyMd, IsCorrect: c.IsCorrect})
	}
	tags, err := q.ListConceptsForQuestions(ctx, []uuid.UUID{question.ID})
	if err != nil {
		return in, err
	}
	for _, t := range tags {
		slug := g.ConceptByID[t.ConceptID].Slug
		in.Concepts = append(in.Concepts, slug)
		if t.IsPrimary {
			in.PrimaryConcept = slug
		}
	}
	return in, nil
}

// QuestionPatch holds optional changes to a Question. Choices, when present,
// replace the choice list matched by label (answered choices cannot be removed).
type QuestionPatch struct {
	StemMd         *string        `json:"stem_md,omitempty"`
	ExplanationMd  *string        `json:"explanation_md,omitempty"`
	Choices        *[]ChoiceInput `json:"choices,omitempty"`
	Concepts       *[]string      `json:"concepts,omitempty"`
	PrimaryConcept *string        `json:"primary_concept,omitempty"`
	Difficulty     *int           `json:"difficulty,omitempty"`
	Format         *string        `json:"format,omitempty"`
	Source         *string        `json:"source,omitempty"`
	SourceRef      *string        `json:"source_ref,omitempty"`
}

// UpdateResult reports what an update did.
type UpdateResult struct {
	Status          string `json:"status"`
	KeyChanged      bool   `json:"key_changed"`
	ExcludedAnswers int64  `json:"excluded_answers"`
}

// UpdateQuestion fixes a Question. An Active Question goes back to Draft for
// re-verification; a Flagged one stays Flagged until resolve_flag. If the
// correct choice changes, prior Answers are excluded from Mastery.
func (s *Service) UpdateQuestion(ctx context.Context, id uuid.UUID, p QuestionPatch) (UpdateResult, error) {
	var res UpdateResult
	err := s.tx(ctx, func(q *store.Queries) error {
		question, err := q.GetQuestionForUpdate(ctx, id)
		if err != nil {
			return mapNotFound(err, "question")
		}
		if question.Status == StatusRetired {
			return E(KindConflict, "retired", "retired Questions cannot be edited")
		}
		g, err := graphTx(ctx, q, question.CourseID)
		if err != nil {
			return err
		}
		old, err := loadQuestionInput(ctx, q, g, question)
		if err != nil {
			return err
		}
		next := old
		next.Choices = append([]ChoiceInput(nil), old.Choices...)
		applyPatch(&next, p)
		if errs := validateQuestion(g, &next); len(errs) > 0 {
			return &Error{Kind: KindInvalid, Code: "invalid_question", Message: "question not updated", Details: errs}
		}
		res.KeyChanged = correctLabel(old.Choices) != correctLabel(next.Choices)
		status := question.Status
		if status == StatusActive {
			status = StatusDraft
		}
		res.Status = status
		if err := q.UpdateQuestionContent(ctx, store.UpdateQuestionContentParams{
			ID: id, StemMd: next.StemMd, ExplanationMd: next.ExplanationMd, Difficulty: int16(next.Difficulty), //nolint:gosec // validated 1..3
			Format: next.Format, Source: next.Source, SourceRef: next.SourceRef, Status: status, VerificationNote: nil,
		}); err != nil {
			return err
		}
		if p.Choices != nil {
			if err := replaceChoices(ctx, q, id, next.Choices); err != nil {
				return err
			}
		}
		var oldConcepts []uuid.UUID
		if p.Concepts != nil || p.PrimaryConcept != nil {
			for _, slug := range old.Concepts {
				oldConcepts = append(oldConcepts, g.ConceptBySlug[slug].ID)
			}
			if err := q.DeleteQuestionConcepts(ctx, id); err != nil {
				return err
			}
			if err := insertTags(ctx, q, g, id, &next); err != nil {
				return err
			}
		}
		if res.KeyChanged {
			if res.ExcludedAnswers, err = q.ExcludeAnswersForQuestion(ctx, id); err != nil {
				return err
			}
		}
		question.Status = status
		return s.recomputeForQuestion(ctx, q, question, oldConcepts)
	})
	return res, err
}

func applyPatch(in *QuestionInput, p QuestionPatch) {
	if p.StemMd != nil {
		in.StemMd = *p.StemMd
	}
	if p.ExplanationMd != nil {
		in.ExplanationMd = *p.ExplanationMd
	}
	if p.Choices != nil {
		in.Choices = *p.Choices
	}
	if p.Concepts != nil {
		in.Concepts = *p.Concepts
	}
	if p.PrimaryConcept != nil {
		in.PrimaryConcept = *p.PrimaryConcept
	}
	if p.Difficulty != nil {
		in.Difficulty = *p.Difficulty
	}
	if p.Format != nil {
		in.Format = *p.Format
	}
	if p.Source != nil {
		in.Source = *p.Source
	}
	if p.SourceRef != nil {
		in.SourceRef = p.SourceRef
	}
}

func correctLabel(cs []ChoiceInput) string {
	for _, c := range cs {
		if c.IsCorrect {
			return c.Label
		}
	}
	return ""
}

func replaceChoices(ctx context.Context, q *store.Queries, qid uuid.UUID, next []ChoiceInput) error {
	existing, err := q.ListChoicesForQuestions(ctx, []uuid.UUID{qid})
	if err != nil {
		return err
	}
	byLabel := map[string]store.QuestionChoice{}
	for _, c := range existing {
		byLabel[c.Label] = c
	}
	keep := map[string]bool{}
	for _, c := range next {
		keep[c.Label] = true
	}
	for _, c := range existing {
		if keep[c.Label] {
			continue
		}
		answered, err := q.ChoiceHasAnswers(ctx, c.ID)
		if err != nil {
			return err
		}
		if answered {
			return E(KindInvalid, "choice_answered", "choice %q has Answers and cannot be removed", c.Label)
		}
		if err := q.DeleteChoice(ctx, c.ID); err != nil {
			return err
		}
	}
	// Clear correctness first so the one-correct unique index never trips mid-update.
	if err := q.ClearCorrectChoices(ctx, qid); err != nil {
		return err
	}
	for i, c := range next {
		if old, ok := byLabel[c.Label]; ok {
			if err := q.UpdateChoice(ctx, store.UpdateChoiceParams{ID: old.ID, BodyMd: c.BodyMd, IsCorrect: c.IsCorrect, Position: int32(i + 1)}); err != nil {
				return err
			}
			continue
		}
		if _, err := q.CreateChoice(ctx, store.CreateChoiceParams{
			QuestionID: qid, Label: c.Label, BodyMd: c.BodyMd, IsCorrect: c.IsCorrect, Position: int32(i + 1),
		}); err != nil {
			return err
		}
	}
	return nil
}

// CreateExam creates a fixed timed paper from Active Questions of a Course.
func (s *Service) CreateExam(ctx context.Context, course, name string, timeLimitS int, questionIDs []uuid.UUID) (store.Exam, error) {
	g, err := s.courseBySlug(ctx, course)
	if err != nil {
		return store.Exam{}, err
	}
	if strings.TrimSpace(name) == "" {
		return store.Exam{}, E(KindInvalid, "name_required", "name is required")
	}
	if timeLimitS <= 0 || timeLimitS > 24*3600 {
		return store.Exam{}, E(KindInvalid, "invalid_time_limit", "time_limit_s must be 1..86400")
	}
	if len(questionIDs) == 0 {
		return store.Exam{}, E(KindInvalid, "questions_required", "an Exam needs at least one Question")
	}
	qs, err := questionsByID(ctx, s.q, questionIDs)
	if err != nil {
		return store.Exam{}, err
	}
	var details []string
	seen := map[uuid.UUID]bool{}
	for _, id := range questionIDs {
		qq, ok := qs[id]
		switch {
		case seen[id]:
			details = append(details, fmt.Sprintf("%s listed twice", id))
		case !ok || qq.CourseID != g.Course.ID:
			details = append(details, fmt.Sprintf("%s is not a Question of this course", id))
		case qq.Status != StatusActive:
			details = append(details, fmt.Sprintf("%s is %s, not active", id, qq.Status))
		}
		seen[id] = true
	}
	if len(details) > 0 {
		return store.Exam{}, &Error{Kind: KindInvalid, Code: "invalid_exam", Message: "exam not created", Details: details}
	}
	var exam store.Exam
	err = s.tx(ctx, func(q *store.Queries) error {
		var err error
		exam, err = q.CreateExam(ctx, store.CreateExamParams{CourseID: g.Course.ID, Name: name, TimeLimitS: int32(timeLimitS)}) //nolint:gosec // validated <= 86400
		if err != nil {
			return err
		}
		for i, id := range questionIDs {
			if err := q.AddExamQuestion(ctx, store.AddExamQuestionParams{ExamID: exam.ID, QuestionID: id, Position: int32(i + 1)}); err != nil {
				return err
			}
		}
		return nil
	})
	return exam, err
}

// ListFlagsAdmin lists Flags for the Content Admin.
func (s *Service) ListFlagsAdmin(ctx context.Context, openOnly bool) ([]FlagView, error) {
	return s.listFlags(ctx, openOnly)
}

// Flag resolutions.
const (
	ResolveReactivate = "reactivate"
	ResolveRetire     = "retire"
	ResolveRedraft    = "redraft"
)

// ResolveFlagResult reports a Flag resolution.
type ResolveFlagResult struct {
	QuestionID      string `json:"question_id"`
	Status          string `json:"status"`
	FlagsResolved   int64  `json:"flags_resolved"`
	ExcludedAnswers int64  `json:"excluded_answers"`
}

// ResolveFlag resolves every open Flag on the flagged Question. keyChanged
// marks all prior Answers excluded (they were graded against a wrong key).
// resolution decides the Question's new status. Affected Learners' Mastery is
// recomputed.
func (s *Service) ResolveFlag(ctx context.Context, flagID uuid.UUID, keyChanged bool, resolution, note string) (ResolveFlagResult, error) {
	if resolution == "" {
		resolution = ResolveReactivate
	}
	status := map[string]string{ResolveReactivate: StatusActive, ResolveRetire: StatusRetired, ResolveRedraft: StatusDraft}[resolution]
	if status == "" {
		return ResolveFlagResult{}, E(KindInvalid, "invalid_resolution", "resolution must be reactivate, retire, or redraft")
	}
	var res ResolveFlagResult
	err := s.tx(ctx, func(q *store.Queries) error {
		flag, err := q.GetFlag(ctx, flagID)
		if err != nil {
			return mapNotFound(err, "flag")
		}
		if flag.ResolvedAt != nil {
			return E(KindConflict, "already_resolved", "this Flag is already resolved")
		}
		question, err := q.GetQuestionForUpdate(ctx, flag.QuestionID)
		if err != nil {
			return err
		}
		var n *string
		if strings.TrimSpace(note) != "" {
			n = &note
		}
		if res.FlagsResolved, err = q.ResolveFlagsForQuestion(ctx, store.ResolveFlagsForQuestionParams{QuestionID: question.ID, ResolutionNote: n}); err != nil {
			return err
		}
		if keyChanged {
			if res.ExcludedAnswers, err = q.ExcludeAnswersForQuestion(ctx, question.ID); err != nil {
				return err
			}
		}
		if err := q.SetQuestionStatus(ctx, store.SetQuestionStatusParams{ID: question.ID, Status: status, VerificationNote: question.VerificationNote}); err != nil {
			return err
		}
		question.Status = status
		res.QuestionID = question.ID.String()
		res.Status = status
		return s.recomputeForQuestion(ctx, q, question, nil)
	})
	return res, err
}

// ListContentRequestsAdmin lists Content Requests for the Content Admin.
func (s *Service) ListContentRequestsAdmin(ctx context.Context, openOnly bool) ([]ContentRequestView, error) {
	return s.listContentRequests(ctx, openOnly)
}

// FulfillContentRequest closes a Content Request.
func (s *Service) FulfillContentRequest(ctx context.Context, id uuid.UUID, note string) error {
	var n *string
	if strings.TrimSpace(note) != "" {
		n = &note
	}
	rows, err := s.q.FulfillContentRequest(ctx, store.FulfillContentRequestParams{ID: id, Note: n})
	if err != nil {
		return err
	}
	if rows == 0 {
		return E(KindNotFound, "not_found", "no open Content Request with that id")
	}
	return nil
}

// PoolStat is the Question pool of one Concept.
type PoolStat struct {
	Concept       string   `json:"concept"`
	Name          string   `json:"name"`
	HasLesson     bool     `json:"has_lesson"`
	Draft         int64    `json:"draft"`
	Active        int64    `json:"active"`
	Flagged       int64    `json:"flagged"`
	Retired       int64    `json:"retired"`
	ActivePrimary int64    `json:"active_primary"`
	ActiveByDiff  [3]int64 `json:"active_by_difficulty" jsonschema:"Active Questions at difficulty 1, 2, 3"`
	ActiveInExams int64    `json:"active_in_exams"`
}

// PoolStats returns per-Concept Question counts (no Question content).
func (s *Service) PoolStats(ctx context.Context, course string) ([]PoolStat, error) {
	g, err := s.courseBySlug(ctx, course)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.PoolStats(ctx, g.Course.ID)
	if err != nil {
		return nil, err
	}
	by := map[uuid.UUID]store.PoolStatsRow{}
	for _, r := range rows {
		by[r.ConceptID] = r
	}
	out := []PoolStat{}
	for _, c := range g.LiveConcepts() {
		r := by[c.ID]
		out = append(out, PoolStat{
			Concept: c.Slug, Name: c.Name, HasLesson: g.HasLesson[c.ID], Draft: r.Draft, Active: r.Active, Flagged: r.Flagged,
			Retired: r.Retired, ActivePrimary: r.ActivePrimary, ActiveByDiff: [3]int64{r.ActiveD1, r.ActiveD2, r.ActiveD3},
			ActiveInExams: r.ActiveInExams,
		})
	}
	return out, nil
}

// MasteryOverviewItem counts Learners per Mastery Level for one Concept.
type MasteryOverviewItem struct {
	Concept    string `json:"concept"`
	Name       string `json:"name"`
	Unassessed int64  `json:"unassessed" jsonschema:"Learners with some but too little evidence (Learners with none are not counted)"`
	Weak       int64  `json:"weak"`
	Developing int64  `json:"developing"`
	Mastered   int64  `json:"mastered"`
}

// MasteryOverview returns aggregate Mastery per Concept (no per-Learner data).
func (s *Service) MasteryOverview(ctx context.Context, course string) ([]MasteryOverviewItem, int, error) {
	g, err := s.courseBySlug(ctx, course)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.MasteryOverview(ctx, g.Course.ID)
	if err != nil {
		return nil, 0, err
	}
	learners, err := s.q.CountLearners(ctx)
	if err != nil {
		return nil, 0, err
	}
	by := map[uuid.UUID]*MasteryOverviewItem{}
	out := []MasteryOverviewItem{}
	for _, c := range g.LiveConcepts() {
		out = append(out, MasteryOverviewItem{Concept: c.Slug, Name: c.Name})
	}
	for i, c := range g.LiveConcepts() {
		by[c.ID] = &out[i]
	}
	for _, r := range rows {
		it := by[r.ConceptID]
		if it == nil {
			continue
		}
		switch r.Level {
		case "unassessed":
			it.Unassessed = r.Learners
		case "weak":
			it.Weak = r.Learners
		case "developing":
			it.Developing = r.Learners
		case "mastered":
			it.Mastered = r.Learners
		}
	}
	return out, int(learners), nil
}
