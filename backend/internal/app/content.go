package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Content Admin use cases (graph authoring). Called by the MCP server and the
// seed command. Every write is validated (DAG via the domain engine, references
// to existing Concepts) and runs in one transaction; the Course's cached graph
// is invalidated afterwards.

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,99}$`)

func checkSlug(kind, slug string) error {
	if !slugRe.MatchString(slug) {
		return E(KindInvalid, "invalid_slug", "%s id %q must match %s", kind, slug, slugRe.String())
	}
	return nil
}

// GraphDoc is the graph.yaml / import_course_graph shape. IDs are slugs.
type GraphDoc struct {
	Course          CourseDoc    `json:"course" yaml:"course" jsonschema:"the Course"`
	QuestionFormats []FormatDoc  `json:"question_formats" yaml:"question_formats" jsonschema:"Question Formats (presentation styles, not graph nodes)"`
	Topics          []TopicDoc   `json:"topics" yaml:"topics" jsonschema:"Topics in display order"`
	Concepts        []ConceptDoc `json:"concepts" yaml:"concepts" jsonschema:"Concepts in display order"`
}

// CourseDoc identifies a Course.
type CourseDoc struct {
	ID   string `json:"id" yaml:"id" jsonschema:"Course slug, e.g. cu-best-math"`
	Name string `json:"name" yaml:"name"`
	Goal string `json:"goal,omitempty" yaml:"goal"`
}

// FormatDoc is a Question Format.
type FormatDoc struct {
	ID   string `json:"id" yaml:"id" jsonschema:"Question Format slug"`
	Name string `json:"name" yaml:"name"`
}

// TopicDoc is a Topic.
type TopicDoc struct {
	ID         string   `json:"id" yaml:"id" jsonschema:"Topic slug"`
	Name       string   `json:"name" yaml:"name"`
	ExamWeight *float64 `json:"exam_weight,omitempty" yaml:"exam_weight" jsonschema:"Exam Weight; null when unknown"`
}

// ConceptDoc is a Concept with its Prerequisites (slugs).
type ConceptDoc struct {
	ID            string   `json:"id" yaml:"id" jsonschema:"Concept slug"`
	Topic         string   `json:"topic" yaml:"topic" jsonschema:"Topic slug"`
	Name          string   `json:"name" yaml:"name"`
	Prerequisites []string `json:"prerequisites" yaml:"prerequisites" jsonschema:"slugs of Concepts that must be understood first"`
	Archived      bool     `json:"archived,omitempty" yaml:"-" jsonschema:"output only: the Concept is archived"`
	HasLesson     bool     `json:"has_lesson,omitempty" yaml:"-" jsonschema:"output only: a Lesson exists"`
}

// ImportResult summarizes an import.
type ImportResult struct {
	CourseID      string   `json:"course_id"`
	Formats       int      `json:"formats"`
	Topics        int      `json:"topics"`
	Concepts      int      `json:"concepts"`
	Prerequisites int      `json:"prerequisites"`
	Archived      []string `json:"archived"`
	NotInDocument []string `json:"not_in_document" jsonschema:"live Concepts in the database absent from the document (left untouched unless archive_missing)"`
}

// courseBySlug loads a Course graph by slug.
func (s *Service) courseBySlug(ctx context.Context, slug string) (*CourseGraph, error) {
	c, err := s.q.GetCourseBySlug(ctx, slug)
	if err != nil {
		return nil, mapNotFound(err, fmt.Sprintf("course %q", slug))
	}
	return s.Graph(ctx, c.ID)
}

// CourseGraphBySlug returns a Course's graph by slug.
func (s *Service) CourseGraphBySlug(ctx context.Context, slug string) (*CourseGraph, error) {
	return s.courseBySlug(ctx, slug)
}

// ExportGraph renders a Course graph as a GraphDoc (including archived Concepts).
func ExportGraph(g *CourseGraph) GraphDoc {
	doc := GraphDoc{
		Course:          CourseDoc{ID: g.Course.Slug, Name: g.Course.Name, Goal: g.Course.Goal},
		QuestionFormats: []FormatDoc{}, Topics: []TopicDoc{}, Concepts: []ConceptDoc{},
	}
	for _, f := range g.Formats {
		doc.QuestionFormats = append(doc.QuestionFormats, FormatDoc{ID: f.Slug, Name: f.Name})
	}
	for _, t := range g.Topics {
		doc.Topics = append(doc.Topics, TopicDoc{ID: t.Slug, Name: t.Name, ExamWeight: t.ExamWeight})
	}
	for _, c := range g.Concepts {
		cd := ConceptDoc{ID: c.Slug, Topic: g.TopicByID[c.TopicID].Slug, Name: c.Name, Prerequisites: []string{},
			Archived: c.ArchivedAt != nil, HasLesson: g.HasLesson[c.ID]}
		src := g.Prereqs[c.ID]
		if c.ArchivedAt != nil {
			src = g.RawPrereqs[c.ID]
		}
		for _, p := range src {
			cd.Prerequisites = append(cd.Prerequisites, g.ConceptByID[p].Slug)
		}
		doc.Concepts = append(doc.Concepts, cd)
	}
	return doc
}

// ImportCourseGraph upserts a whole graph by slug (idempotent). Concepts in
// the database but absent from the document are archived only when
// archiveMissing is set.
func (s *Service) ImportCourseGraph(ctx context.Context, doc GraphDoc, archiveMissing bool) (ImportResult, error) {
	if err := validateDoc(doc); err != nil {
		return ImportResult{}, err
	}
	var res ImportResult
	var courseID uuid.UUID
	err := s.tx(ctx, func(q *store.Queries) error {
		var existing *CourseGraph
		if c, err := q.GetCourseBySlug(ctx, doc.Course.ID); err == nil {
			existing, err = graphTx(ctx, q, c.ID)
			if err != nil {
				return err
			}
		} else if !store.IsNotFound(err) {
			return err
		}
		// Validate the resulting graph before writing.
		ch := graphChange{addConcepts: map[string]string{}, setPrereqs: map[string][]string{},
			archive: map[string]bool{}, knownTopicIDs: map[string]bool{}}
		inDoc := map[string]bool{}
		for _, t := range doc.Topics {
			ch.knownTopicIDs[t.ID] = true
		}
		for _, c := range doc.Concepts {
			inDoc[c.ID] = true
			ch.addConcepts[c.ID] = c.Topic
			ch.setPrereqs[c.ID] = c.Prerequisites
		}
		if existing != nil {
			for _, c := range existing.LiveConcepts() {
				if !inDoc[c.Slug] {
					res.NotInDocument = append(res.NotInDocument, c.Slug)
					if archiveMissing {
						ch.archive[c.Slug] = true
					}
				}
			}
		}
		if err := validateChange(existing, ch); err != nil {
			return err
		}

		course, err := q.UpsertCourse(ctx, store.UpsertCourseParams{Slug: doc.Course.ID, Name: doc.Course.Name, Goal: doc.Course.Goal})
		if err != nil {
			return err
		}
		courseID = course.ID
		res.CourseID = course.ID.String()
		for i, f := range doc.QuestionFormats {
			if err := q.UpsertQuestionFormat(ctx, store.UpsertQuestionFormatParams{
				CourseID: course.ID, Slug: f.ID, Name: f.Name, Position: int32(i + 1),
			}); err != nil {
				return err
			}
			res.Formats++
		}
		topicIDs := map[string]uuid.UUID{}
		if existing != nil {
			for slug, t := range existing.TopicBySlug {
				topicIDs[slug] = t.ID
			}
		}
		for i, t := range doc.Topics {
			row, err := q.UpsertTopic(ctx, store.UpsertTopicParams{
				CourseID: course.ID, Slug: t.ID, Name: t.Name, ExamWeight: t.ExamWeight, Position: int32(i + 1),
			})
			if err != nil {
				return err
			}
			topicIDs[t.ID] = row.ID
			res.Topics++
		}
		conceptIDs := map[string]uuid.UUID{}
		if existing != nil {
			for slug, c := range existing.ConceptBySlug {
				conceptIDs[slug] = c.ID
			}
		}
		for i, c := range doc.Concepts {
			row, err := q.UpsertConcept(ctx, store.UpsertConceptParams{
				CourseID: course.ID, TopicID: topicIDs[c.Topic], Slug: c.ID, Name: c.Name, Position: int32(i + 1),
			})
			if err != nil {
				return err
			}
			conceptIDs[c.ID] = row.ID
			res.Concepts++
		}
		for _, c := range doc.Concepts {
			id := conceptIDs[c.ID]
			if err := q.DeletePrerequisitesOf(ctx, id); err != nil {
				return err
			}
			for _, p := range c.Prerequisites {
				if err := q.InsertPrerequisite(ctx, store.InsertPrerequisiteParams{ConceptID: id, PrerequisiteID: conceptIDs[p]}); err != nil {
					return err
				}
				res.Prerequisites++
			}
		}
		for slug := range ch.archive {
			if err := q.ArchiveConcept(ctx, conceptIDs[slug]); err != nil {
				return err
			}
			res.Archived = append(res.Archived, slug)
		}
		// Final check: the stored graph loads and validates.
		_, err = graphTx(ctx, q, course.ID)
		return err
	})
	if err != nil {
		return ImportResult{}, err
	}
	s.invalidate(courseID)
	if res.Archived == nil {
		res.Archived = []string{}
	}
	if res.NotInDocument == nil {
		res.NotInDocument = []string{}
	}
	return res, nil
}

func validateDoc(doc GraphDoc) error {
	var errs []string
	add := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	if !slugRe.MatchString(doc.Course.ID) {
		add("course.id %q is not a valid slug", doc.Course.ID)
	}
	if strings.TrimSpace(doc.Course.Name) == "" {
		add("course.name is required")
	}
	seen := map[string]bool{}
	for i, f := range doc.QuestionFormats {
		if !slugRe.MatchString(f.ID) || strings.TrimSpace(f.Name) == "" || seen[f.ID] {
			add("question_formats[%d]: needs a unique valid id and a name", i)
		}
		seen[f.ID] = true
	}
	topics := map[string]bool{}
	for i, t := range doc.Topics {
		if !slugRe.MatchString(t.ID) || strings.TrimSpace(t.Name) == "" || topics[t.ID] {
			add("topics[%d]: needs a unique valid id and a name", i)
		}
		if t.ExamWeight != nil && *t.ExamWeight < 0 {
			add("topics[%d]: exam_weight must be >= 0", i)
		}
		topics[t.ID] = true
	}
	concepts := map[string]bool{}
	for i, c := range doc.Concepts {
		if !slugRe.MatchString(c.ID) || strings.TrimSpace(c.Name) == "" || concepts[c.ID] {
			add("concepts[%d] (%s): needs a unique valid id and a name", i, c.ID)
		}
		if !topics[c.Topic] {
			add("concepts[%d] (%s): unknown topic %q", i, c.ID, c.Topic)
		}
		concepts[c.ID] = true
	}
	for i, c := range doc.Concepts {
		for _, p := range c.Prerequisites {
			if !concepts[p] {
				add("concepts[%d] (%s): unknown prerequisite %q", i, c.ID, p)
			}
		}
	}
	if len(errs) > 0 {
		return &Error{Kind: KindInvalid, Code: "invalid_graph", Message: "graph document is invalid", Details: errs}
	}
	return nil
}

// UpsertTopicInput creates or updates a Topic.
type UpsertTopicInput struct {
	Course     string
	Slug       string
	Name       string
	ExamWeight *float64
}

// UpsertTopic creates or updates a Topic.
func (s *Service) UpsertTopic(ctx context.Context, in UpsertTopicInput) (store.Topic, error) {
	g, err := s.courseBySlug(ctx, in.Course)
	if err != nil {
		return store.Topic{}, err
	}
	if err := checkSlug("topic", in.Slug); err != nil {
		return store.Topic{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return store.Topic{}, E(KindInvalid, "name_required", "name is required")
	}
	if in.ExamWeight != nil && *in.ExamWeight < 0 {
		return store.Topic{}, E(KindInvalid, "invalid_exam_weight", "exam_weight must be >= 0")
	}
	var pos int32
	if t, ok := g.TopicBySlug[in.Slug]; ok {
		pos = t.Position
	} else if pos, err = s.q.NextTopicPosition(ctx, g.Course.ID); err != nil {
		return store.Topic{}, err
	}
	t, err := s.q.UpsertTopic(ctx, store.UpsertTopicParams{
		CourseID: g.Course.ID, Slug: in.Slug, Name: in.Name, ExamWeight: in.ExamWeight, Position: pos,
	})
	s.invalidate(g.Course.ID)
	return t, err
}

// UpsertConceptInput creates or updates a Concept. Prerequisites, when
// non-nil, replace the Concept's Prerequisites.
type UpsertConceptInput struct {
	Course        string
	Slug          string
	Topic         string
	Name          string
	Prerequisites *[]string
}

// UpsertConcept creates, updates, or un-archives a Concept.
func (s *Service) UpsertConcept(ctx context.Context, in UpsertConceptInput) (store.Concept, error) {
	g, err := s.courseBySlug(ctx, in.Course)
	if err != nil {
		return store.Concept{}, err
	}
	if err := checkSlug("concept", in.Slug); err != nil {
		return store.Concept{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return store.Concept{}, E(KindInvalid, "name_required", "name is required")
	}
	topic, ok := g.TopicBySlug[in.Topic]
	if !ok {
		return store.Concept{}, E(KindInvalid, "unknown_topic", "unknown topic %q", in.Topic)
	}
	ch := graphChange{addConcepts: map[string]string{in.Slug: in.Topic}, setPrereqs: map[string][]string{}, unarchive: map[string]bool{in.Slug: true}}
	if in.Prerequisites != nil {
		ch.setPrereqs[in.Slug] = *in.Prerequisites
	}
	if err := validateChange(g, ch); err != nil {
		return store.Concept{}, err
	}
	var out store.Concept
	err = s.tx(ctx, func(q *store.Queries) error {
		var pos int32
		if c, ok := g.ConceptBySlug[in.Slug]; ok {
			pos = c.Position
		} else if pos, err = q.NextConceptPosition(ctx, g.Course.ID); err != nil {
			return err
		}
		out, err = q.UpsertConcept(ctx, store.UpsertConceptParams{
			CourseID: g.Course.ID, TopicID: topic.ID, Slug: in.Slug, Name: in.Name, Position: pos,
		})
		if err != nil {
			return err
		}
		if in.Prerequisites != nil {
			if err := replacePrereqs(ctx, q, g, out.ID, *in.Prerequisites); err != nil {
				return err
			}
		}
		_, err = graphTx(ctx, q, g.Course.ID)
		return err
	})
	s.invalidate(g.Course.ID)
	return out, err
}

func replacePrereqs(ctx context.Context, q *store.Queries, g *CourseGraph, conceptID uuid.UUID, slugs []string) error {
	if err := q.DeletePrerequisitesOf(ctx, conceptID); err != nil {
		return err
	}
	for _, p := range slugs {
		pc, ok := g.LiveBySlug(p)
		if !ok {
			return E(KindInvalid, "unknown_concept", "unknown or archived prerequisite %q", p)
		}
		if err := q.InsertPrerequisite(ctx, store.InsertPrerequisiteParams{ConceptID: conceptID, PrerequisiteID: pc.ID}); err != nil {
			return err
		}
	}
	return nil
}

// ArchiveConcept archives a Concept (Concepts are never deleted). Its
// Dependents keep their chains: they inherit its Prerequisites.
func (s *Service) ArchiveConcept(ctx context.Context, course, slug string) error {
	g, err := s.courseBySlug(ctx, course)
	if err != nil {
		return err
	}
	c, ok := g.ConceptBySlug[slug]
	if !ok {
		return notFound(fmt.Sprintf("concept %q", slug))
	}
	if err := s.q.ArchiveConcept(ctx, c.ID); err != nil {
		return err
	}
	s.invalidate(g.Course.ID)
	return nil
}

// SetPrerequisites replaces a Concept's Prerequisites; cycles are rejected.
func (s *Service) SetPrerequisites(ctx context.Context, course, slug string, prereqs []string) error {
	g, err := s.courseBySlug(ctx, course)
	if err != nil {
		return err
	}
	c, ok := g.LiveBySlug(slug)
	if !ok {
		return notFound(fmt.Sprintf("concept %q", slug))
	}
	if err := validateChange(g, graphChange{setPrereqs: map[string][]string{slug: prereqs}}); err != nil {
		return err
	}
	err = s.tx(ctx, func(q *store.Queries) error {
		if err := replacePrereqs(ctx, q, g, c.ID, prereqs); err != nil {
			return err
		}
		_, err := graphTx(ctx, q, g.Course.ID)
		return err
	})
	s.invalidate(g.Course.ID)
	return err
}

// UpsertLesson sets the Markdown Lesson of a Concept.
func (s *Service) UpsertLesson(ctx context.Context, course, slug, body string) error {
	g, err := s.courseBySlug(ctx, course)
	if err != nil {
		return err
	}
	c, ok := g.LiveBySlug(slug)
	if !ok {
		return notFound(fmt.Sprintf("concept %q", slug))
	}
	if strings.TrimSpace(body) == "" {
		return E(KindInvalid, "body_required", "body_md is required")
	}
	if err := s.q.UpsertLesson(ctx, store.UpsertLessonParams{ConceptID: c.ID, BodyMd: body}); err != nil {
		return err
	}
	s.invalidate(g.Course.ID)
	return nil
}
