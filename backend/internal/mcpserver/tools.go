package mcpserver

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thekogo/tutor-platform/backend/internal/app"
)

type tools struct{ app *app.Service }

// add registers a typed tool.
func add[In, Out any](s *mcp.Server, name, desc string, h func(context.Context, In) (Out, error)) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: desc}, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		out, err := h(ctx, in)
		return nil, out, err
	})
}

func parseID(field, s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, app.E(app.KindInvalid, "invalid_id", "%s must be a UUID", field)
	}
	return id, nil
}

// ── inputs / outputs ────────────────────────────────────────────────────

type courseIn struct {
	Course string `json:"course" jsonschema:"Course slug, e.g. cu-best-math"`
}

type okOut struct {
	OK bool `json:"ok"`
}

type importIn struct {
	Graph          app.GraphDoc `json:"graph" jsonschema:"the whole graph, same shape as content/<course>/graph.yaml"`
	ArchiveMissing bool         `json:"archive_missing,omitempty" jsonschema:"archive live Concepts that are absent from the document (default false)"`
}

type upsertTopicIn struct {
	Course     string   `json:"course" jsonschema:"Course slug"`
	ID         string   `json:"id" jsonschema:"Topic slug"`
	Name       string   `json:"name"`
	ExamWeight *float64 `json:"exam_weight,omitempty" jsonschema:"Exam Weight; omit or null when unknown"`
}

type idOut struct {
	ID   string `json:"id" jsonschema:"internal UUID"`
	Slug string `json:"slug,omitempty"`
}

type upsertConceptIn struct {
	Course        string    `json:"course" jsonschema:"Course slug"`
	ID            string    `json:"id" jsonschema:"Concept slug"`
	Topic         string    `json:"topic" jsonschema:"Topic slug"`
	Name          string    `json:"name"`
	Prerequisites *[]string `json:"prerequisites,omitempty" jsonschema:"if present, replaces the Concept's Prerequisites (slugs); cycles are rejected"`
}

type conceptRefIn struct {
	Course  string `json:"course" jsonschema:"Course slug"`
	Concept string `json:"concept" jsonschema:"Concept slug"`
}

type setPrereqsIn struct {
	Course        string   `json:"course" jsonschema:"Course slug"`
	Concept       string   `json:"concept" jsonschema:"Concept slug"`
	Prerequisites []string `json:"prerequisites" jsonschema:"full replacement list of Prerequisite slugs"`
}

type lessonIn struct {
	Course  string `json:"course" jsonschema:"Course slug"`
	Concept string `json:"concept" jsonschema:"Concept slug"`
	BodyMd  string `json:"body_md" jsonschema:"Markdown Lesson: explanation and worked examples (KaTeX math allowed)"`
}

type createQuestionsIn struct {
	Course    string              `json:"course" jsonschema:"Course slug"`
	Questions []app.QuestionInput `json:"questions" jsonschema:"1-200 Questions; all are validated and created together as Draft"`
}

type createQuestionsOut struct {
	QuestionIDs []string `json:"question_ids"`
}

type listVerificationIn struct {
	Course string `json:"course" jsonschema:"Course slug"`
	Limit  int    `json:"limit,omitempty" jsonschema:"max Questions (default 50, max 200)"`
}

type listVerificationOut struct {
	Questions []app.VerificationItem `json:"questions" jsonschema:"Draft Questions WITHOUT answer keys or Explanations"`
}

type submitVerificationIn struct {
	QuestionID  string `json:"question_id"`
	ChosenLabel string `json:"chosen_label" jsonschema:"the label the independent verifier believes is correct"`
	Note        string `json:"note,omitempty" jsonschema:"verifier's reasoning (kept as verification_note)"`
}

type questionIDIn struct {
	QuestionID string `json:"question_id"`
}

type updateQuestionIn struct {
	QuestionID string `json:"question_id"`
	app.QuestionPatch
}

type createExamIn struct {
	Course      string   `json:"course" jsonschema:"Course slug"`
	Name        string   `json:"name"`
	TimeLimitS  int      `json:"time_limit_s" jsonschema:"time limit in seconds"`
	QuestionIDs []string `json:"question_ids" jsonschema:"Active Question ids in paper order; they will no longer appear in Quizzes"`
}

type openOnlyIn struct {
	OpenOnly *bool `json:"open_only,omitempty" jsonschema:"only unresolved items (default true)"`
}

func openOnly(p *bool) bool { return p == nil || *p }

type flagOut struct {
	ID             string             `json:"id"`
	QuestionID     string             `json:"question_id"`
	Reason         string             `json:"reason"`
	LearnerName    string             `json:"learner_name"`
	CreatedAt      time.Time          `json:"created_at"`
	ResolvedAt     *time.Time         `json:"resolved_at,omitempty"`
	ResolutionNote *string            `json:"resolution_note,omitempty"`
	Question       app.QuestionDetail `json:"question" jsonschema:"the flagged Question including key and Explanation"`
}

type listFlagsOut struct {
	Flags []flagOut `json:"flags"`
}

type resolveFlagIn struct {
	FlagID     string `json:"flag_id"`
	KeyChanged bool   `json:"key_changed" jsonschema:"true if the answer key was wrong and has been fixed: all prior Answers are excluded from Mastery"`
	Resolution string `json:"resolution,omitempty" jsonschema:"reactivate (default), retire, or redraft"`
	Note       string `json:"note,omitempty"`
}

type contentRequestOut struct {
	ID          string     `json:"id"`
	Concept     string     `json:"concept" jsonschema:"Concept slug"`
	ConceptName string     `json:"concept_name"`
	CourseID    string     `json:"course_id"`
	Reason      string     `json:"reason"`
	CreatedAt   time.Time  `json:"created_at"`
	FulfilledAt *time.Time `json:"fulfilled_at,omitempty"`
	Note        *string    `json:"note,omitempty"`
}

type listContentRequestsOut struct {
	Requests []contentRequestOut `json:"requests"`
}

type fulfillIn struct {
	ID   string `json:"id" jsonschema:"Content Request id"`
	Note string `json:"note,omitempty"`
}

type poolStatsOut struct {
	Concepts []app.PoolStat `json:"concepts"`
}

type masteryOverviewOut struct {
	Learners int                       `json:"learners"`
	Concepts []app.MasteryOverviewItem `json:"concepts"`
}

// ── registration ────────────────────────────────────────────────────────

func (t *tools) register(s *mcp.Server) {
	a := t.app
	add(s, "get_course_graph", "Get a Course's Topics, Concepts (with archived flag and has_lesson), Prerequisites, and Question Formats.",
		func(ctx context.Context, in courseIn) (app.GraphDoc, error) {
			g, err := a.CourseGraphBySlug(ctx, in.Course)
			if err != nil {
				return app.GraphDoc{}, err
			}
			return app.ExportGraph(g), nil
		})
	add(s, "import_course_graph", "Upsert a whole knowledge graph by slug (same shape as graph.yaml). Idempotent; validates the Prerequisite DAG.",
		func(ctx context.Context, in importIn) (app.ImportResult, error) {
			return a.ImportCourseGraph(ctx, in.Graph, in.ArchiveMissing)
		})
	add(s, "upsert_topic", "Create or update a Topic.",
		func(ctx context.Context, in upsertTopicIn) (idOut, error) {
			tp, err := a.UpsertTopic(ctx, app.UpsertTopicInput{Course: in.Course, Slug: in.ID, Name: in.Name, ExamWeight: in.ExamWeight})
			return idOut{ID: tp.ID.String(), Slug: tp.Slug}, err
		})
	add(s, "upsert_concept", "Create, update, or un-archive a Concept; optionally replace its Prerequisites (cycles rejected).",
		func(ctx context.Context, in upsertConceptIn) (idOut, error) {
			c, err := a.UpsertConcept(ctx, app.UpsertConceptInput{Course: in.Course, Slug: in.ID, Topic: in.Topic, Name: in.Name, Prerequisites: in.Prerequisites})
			return idOut{ID: c.ID.String(), Slug: c.Slug}, err
		})
	add(s, "archive_concept", "Archive a Concept (never deleted). Its Dependents inherit its Prerequisites.",
		func(ctx context.Context, in conceptRefIn) (okOut, error) {
			return okOut{OK: true}, a.ArchiveConcept(ctx, in.Course, in.Concept)
		})
	add(s, "set_prerequisites", "Replace a Concept's Prerequisites. Rejects cycles and unknown Concepts.",
		func(ctx context.Context, in setPrereqsIn) (okOut, error) {
			return okOut{OK: true}, a.SetPrerequisites(ctx, in.Course, in.Concept, in.Prerequisites)
		})
	add(s, "upsert_lesson", "Set the Markdown Lesson of a Concept.",
		func(ctx context.Context, in lessonIn) (okOut, error) {
			return okOut{OK: true}, a.UpsertLesson(ctx, in.Course, in.Concept, in.BodyMd)
		})
	add(s, "create_questions", "Batch-create Draft Questions. Each needs stem, 2-8 choices with exactly one correct, an Explanation, Concept tags (slugs) including the Primary Concept, difficulty 1-3, format, and source.",
		func(ctx context.Context, in createQuestionsIn) (createQuestionsOut, error) {
			ids, err := a.CreateQuestions(ctx, in.Course, in.Questions)
			out := createQuestionsOut{QuestionIDs: []string{}}
			for _, id := range ids {
				out.QuestionIDs = append(out.QuestionIDs, id.String())
			}
			return out, err
		})
	add(s, "list_questions_for_verification", "List Draft Questions for the independent verifier, WITHOUT answer keys or Explanations.",
		func(ctx context.Context, in listVerificationIn) (listVerificationOut, error) {
			qs, err := a.ListQuestionsForVerification(ctx, in.Course, in.Limit)
			return listVerificationOut{Questions: qs}, err
		})
	add(s, "submit_verification", "Submit the verifier's chosen answer for a Draft Question. Match: the Question becomes Active. Mismatch: it stays Draft with a verification_note.",
		func(ctx context.Context, in submitVerificationIn) (app.VerificationResult, error) {
			id, err := parseID("question_id", in.QuestionID)
			if err != nil {
				return app.VerificationResult{}, err
			}
			return a.SubmitVerification(ctx, id, in.ChosenLabel, in.Note)
		})
	add(s, "get_question", "Get one Question with its key, Explanation, tags, status, and verification_note (for fixing; never show to Learners).",
		func(ctx context.Context, in questionIDIn) (app.QuestionDetail, error) {
			id, err := parseID("question_id", in.QuestionID)
			if err != nil {
				return app.QuestionDetail{}, err
			}
			return a.GetQuestionDetail(ctx, id)
		})
	add(s, "update_question", "Fix a Question (only the fields given change; choices are matched by label). Active Questions return to Draft for re-verification; Flagged ones stay Flagged until resolve_flag. Changing the correct choice excludes prior Answers from Mastery.",
		func(ctx context.Context, in updateQuestionIn) (app.UpdateResult, error) {
			id, err := parseID("question_id", in.QuestionID)
			if err != nil {
				return app.UpdateResult{}, err
			}
			return a.UpdateQuestion(ctx, id, in.QuestionPatch)
		})
	add(s, "retire_question", "Retire a Question: it is no longer served and its Answers stop counting toward Mastery.",
		func(ctx context.Context, in questionIDIn) (okOut, error) {
			id, err := parseID("question_id", in.QuestionID)
			if err != nil {
				return okOut{}, err
			}
			return okOut{OK: true}, a.RetireQuestion(ctx, id)
		})
	add(s, "create_exam", "Create a fixed, timed Exam from Active Questions of the Course. Exam Questions are kept out of Quizzes.",
		func(ctx context.Context, in createExamIn) (idOut, error) {
			ids := make([]uuid.UUID, 0, len(in.QuestionIDs))
			for _, raw := range in.QuestionIDs {
				id, err := parseID("question_ids[]", raw)
				if err != nil {
					return idOut{}, err
				}
				ids = append(ids, id)
			}
			e, err := a.CreateExam(ctx, in.Course, in.Name, in.TimeLimitS, ids)
			return idOut{ID: e.ID.String()}, err
		})
	add(s, "list_flags", "List Learner Flags with the flagged Question (including key and Explanation) so it can be fixed.",
		func(ctx context.Context, in openOnlyIn) (listFlagsOut, error) {
			fs, err := a.ListFlagsAdmin(ctx, openOnly(in.OpenOnly))
			if err != nil {
				return listFlagsOut{}, err
			}
			out := listFlagsOut{Flags: []flagOut{}}
			for i := range fs {
				f := &fs[i]
				q, err := a.GetQuestionDetail(ctx, f.QuestionID)
				if err != nil {
					return listFlagsOut{}, err
				}
				out.Flags = append(out.Flags, flagOut{ID: f.ID.String(), QuestionID: f.QuestionID.String(), Reason: f.Reason, LearnerName: f.LearnerName,
					CreatedAt: f.CreatedAt, ResolvedAt: f.ResolvedAt, ResolutionNote: f.ResolutionNote, Question: q})
			}
			return out, nil
		})
	add(s, "resolve_flag", "Resolve all open Flags on the flagged Question. key_changed=true excludes all prior Answers from Mastery. resolution: reactivate (default), retire, or redraft. Affected Learners' Mastery is recomputed.",
		func(ctx context.Context, in resolveFlagIn) (app.ResolveFlagResult, error) {
			id, err := parseID("flag_id", in.FlagID)
			if err != nil {
				return app.ResolveFlagResult{}, err
			}
			return a.ResolveFlag(ctx, id, in.KeyChanged, in.Resolution, in.Note)
		})
	add(s, "list_content_requests", "List Content Requests (Concepts that need more Questions or Lesson material).",
		func(ctx context.Context, in openOnlyIn) (listContentRequestsOut, error) {
			rs, err := a.ListContentRequestsAdmin(ctx, openOnly(in.OpenOnly))
			out := listContentRequestsOut{Requests: []contentRequestOut{}}
			for i := range rs {
				r := &rs[i]
				out.Requests = append(out.Requests, contentRequestOut{ID: r.ID.String(), Concept: r.ConceptSlug, ConceptName: r.ConceptName,
					CourseID: r.CourseID.String(), Reason: r.Reason, CreatedAt: r.CreatedAt, FulfilledAt: r.FulfilledAt, Note: r.Note})
			}
			return out, err
		})
	add(s, "fulfill_content_request", "Mark a Content Request fulfilled (after adding Questions or a Lesson).",
		func(ctx context.Context, in fulfillIn) (okOut, error) {
			id, err := parseID("id", in.ID)
			if err != nil {
				return okOut{}, err
			}
			return okOut{OK: true}, a.FulfillContentRequest(ctx, id, in.Note)
		})
	add(s, "get_pool_stats", "Per-Concept Question counts by status and difficulty (no Question content).",
		func(ctx context.Context, in courseIn) (poolStatsOut, error) {
			st, err := a.PoolStats(ctx, in.Course)
			return poolStatsOut{Concepts: st}, err
		})
	add(s, "get_mastery_overview", "Aggregate Mastery Level counts per Concept across Learners (no per-Learner data, no Question content).",
		func(ctx context.Context, in courseIn) (masteryOverviewOut, error) {
			items, n, err := a.MasteryOverview(ctx, in.Course)
			return masteryOverviewOut{Learners: n, Concepts: items}, err
		})
}
