package app

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/domain"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// CourseGraph is one Course's knowledge graph loaded from the database, with
// lookups and the validated domain.Graph over live (non-archived) Concepts.
// It is cached per Course and treated as immutable; graph writes invalidate it.
type CourseGraph struct {
	Course        store.Course
	Formats       []store.QuestionFormat
	Topics        []store.Topic
	TopicByID     map[uuid.UUID]store.Topic
	TopicBySlug   map[string]store.Topic
	Concepts      []store.Concept // all, including archived, in display order
	ConceptByID   map[uuid.UUID]store.Concept
	ConceptBySlug map[string]store.Concept
	// RawPrereqs are the stored Prerequisite edges (may touch archived Concepts).
	RawPrereqs map[uuid.UUID][]uuid.UUID
	// Prereqs are the effective Prerequisites between live Concepts: edges
	// through archived Concepts are bridged so chains stay intact.
	Prereqs   map[uuid.UUID][]uuid.UUID
	HasLesson map[uuid.UUID]bool
	Graph     *domain.Graph
}

// Live reports whether id is a non-archived Concept of this Course.
func (g *CourseGraph) Live(id uuid.UUID) bool {
	c, ok := g.ConceptByID[id]
	return ok && c.ArchivedAt == nil
}

// LiveConcepts returns the non-archived Concepts in display order.
func (g *CourseGraph) LiveConcepts() []store.Concept {
	out := make([]store.Concept, 0, len(g.Concepts))
	for _, c := range g.Concepts {
		if c.ArchivedAt == nil {
			out = append(out, c)
		}
	}
	return out
}

// Name returns the Concept name (empty when unknown).
func (g *CourseGraph) Name(id uuid.UUID) string { return g.ConceptByID[id].Name }

// LiveBySlug returns a live Concept by slug.
func (g *CourseGraph) LiveBySlug(slug string) (store.Concept, bool) {
	c, ok := g.ConceptBySlug[slug]
	return c, ok && c.ArchivedAt == nil
}

// Graph returns the (cached) knowledge graph of a Course.
func (s *Service) Graph(ctx context.Context, courseID uuid.UUID) (*CourseGraph, error) {
	s.mu.RLock()
	g, ok := s.graphs[courseID]
	s.mu.RUnlock()
	if ok {
		return g, nil
	}
	g, err := loadGraph(ctx, s.q, courseID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.graphs[courseID] = g
	s.mu.Unlock()
	return g, nil
}

// graphTx loads a fresh, uncached graph inside a transaction.
func graphTx(ctx context.Context, q *store.Queries, courseID uuid.UUID) (*CourseGraph, error) {
	return loadGraph(ctx, q, courseID)
}

// invalidate drops cached graphs after a graph write.
func (s *Service) invalidate(courseID uuid.UUID) {
	s.mu.Lock()
	delete(s.graphs, courseID)
	s.mu.Unlock()
}

func loadGraph(ctx context.Context, q *store.Queries, courseID uuid.UUID) (*CourseGraph, error) {
	course, err := q.GetCourse(ctx, courseID)
	if err != nil {
		return nil, mapNotFound(err, "course")
	}
	formats, err := q.ListQuestionFormats(ctx, courseID)
	if err != nil {
		return nil, err
	}
	topics, err := q.ListTopics(ctx, courseID)
	if err != nil {
		return nil, err
	}
	concepts, err := q.ListConcepts(ctx, courseID)
	if err != nil {
		return nil, err
	}
	edges, err := q.ListPrerequisites(ctx, courseID)
	if err != nil {
		return nil, err
	}
	lessons, err := q.ListLessonConceptIDs(ctx, courseID)
	if err != nil {
		return nil, err
	}
	g := &CourseGraph{
		Course:        course,
		Formats:       formats,
		Topics:        topics,
		TopicByID:     map[uuid.UUID]store.Topic{},
		TopicBySlug:   map[string]store.Topic{},
		Concepts:      concepts,
		ConceptByID:   map[uuid.UUID]store.Concept{},
		ConceptBySlug: map[string]store.Concept{},
		RawPrereqs:    map[uuid.UUID][]uuid.UUID{},
		HasLesson:     map[uuid.UUID]bool{},
	}
	for _, t := range topics {
		g.TopicByID[t.ID] = t
		g.TopicBySlug[t.Slug] = t
	}
	for _, c := range concepts {
		g.ConceptByID[c.ID] = c
		g.ConceptBySlug[c.Slug] = c
	}
	for _, e := range edges {
		g.RawPrereqs[e.ConceptID] = append(g.RawPrereqs[e.ConceptID], e.PrerequisiteID)
	}
	for _, id := range lessons {
		g.HasLesson[id] = true
	}
	g.Prereqs = bridgePrereqs(g)

	dt := make([]domain.Topic, 0, len(topics))
	for _, t := range topics {
		dt = append(dt, domain.Topic{ID: tid(t.ID), Name: t.Name, ExamWeight: t.ExamWeight})
	}
	var dc []domain.Concept
	var dp []domain.Prerequisite
	for _, c := range g.LiveConcepts() {
		dc = append(dc, domain.Concept{ID: cid(c.ID), Topic: tid(c.TopicID), Name: c.Name})
		for _, p := range g.Prereqs[c.ID] {
			dp = append(dp, domain.Prerequisite{Concept: cid(c.ID), Prerequisite: cid(p)})
		}
	}
	dg, err := domain.NewGraph(dt, dc, dp)
	if err != nil {
		return nil, fmt.Errorf("app: course %s graph invalid: %w", course.Slug, err)
	}
	g.Graph = dg
	return g, nil
}

// bridgePrereqs computes effective Prerequisites between live Concepts,
// replacing an archived Prerequisite by its own (effective) Prerequisites.
func bridgePrereqs(g *CourseGraph) map[uuid.UUID][]uuid.UUID {
	out := map[uuid.UUID][]uuid.UUID{}
	for _, c := range g.Concepts {
		if c.ArchivedAt != nil {
			continue
		}
		seen := map[uuid.UUID]bool{}
		set := map[uuid.UUID]bool{}
		var walk func(id uuid.UUID)
		walk = func(id uuid.UUID) {
			for _, p := range g.RawPrereqs[id] {
				if seen[p] {
					continue
				}
				seen[p] = true
				if g.Live(p) {
					set[p] = true
				} else if _, ok := g.ConceptByID[p]; ok {
					walk(p)
				}
			}
		}
		walk(c.ID)
		delete(set, c.ID)
		ids := make([]uuid.UUID, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
		out[c.ID] = ids
	}
	return out
}

// graphChange describes a hypothetical edit, validated before writing.
// Concept identities are slugs.
type graphChange struct {
	addConcepts   map[string]string   // slug → topic slug (new or moved Concepts)
	setPrereqs    map[string][]string // slug → full Prerequisite slug list
	unarchive     map[string]bool
	archive       map[string]bool
	knownTopicIDs map[string]bool // extra Topic slugs that will exist
}

// validateChange builds the would-be live graph (by slug) and runs the domain
// DAG validation on it. It returns a KindInvalid error on failure.
func validateChange(g *CourseGraph, ch graphChange) error {
	if g == nil {
		g = &CourseGraph{TopicByID: map[uuid.UUID]store.Topic{}, TopicBySlug: map[string]store.Topic{}}
	}
	topicOK := map[string]bool{}
	for slug := range g.TopicBySlug {
		topicOK[slug] = true
	}
	for slug := range ch.knownTopicIDs {
		topicOK[slug] = true
	}
	live := map[string]string{} // slug → topic slug
	for _, c := range g.Concepts {
		if c.ArchivedAt == nil || ch.unarchive[c.Slug] {
			live[c.Slug] = g.TopicByID[c.TopicID].Slug
		}
	}
	for slug, topic := range ch.addConcepts {
		live[slug] = topic
	}
	for slug := range ch.archive {
		delete(live, slug)
	}
	prereqs := map[string][]string{}
	for _, c := range g.Concepts {
		if _, ok := live[c.Slug]; !ok {
			continue
		}
		// Raw edges to live Concepts plus bridged edges (so unarchiving is accounted for).
		for _, p := range g.Prereqs[c.ID] {
			prereqs[c.Slug] = append(prereqs[c.Slug], g.ConceptByID[p].Slug)
		}
		if c.ArchivedAt != nil {
			for _, p := range g.RawPrereqs[c.ID] {
				if pc, ok := g.ConceptByID[p]; ok {
					if _, isLive := live[pc.Slug]; isLive {
						prereqs[c.Slug] = append(prereqs[c.Slug], pc.Slug)
					}
				}
			}
		}
	}
	for slug, ps := range ch.setPrereqs {
		prereqs[slug] = ps
	}
	topicSet := map[string]bool{}
	var dt []domain.Topic
	var dc []domain.Concept
	var dp []domain.Prerequisite
	for slug, topic := range live {
		if !topicOK[topic] {
			return E(KindInvalid, "unknown_topic", "concept %q references unknown topic %q", slug, topic)
		}
		if !topicSet[topic] {
			topicSet[topic] = true
			dt = append(dt, domain.Topic{ID: domain.TopicID(topic), Name: topic})
		}
		dc = append(dc, domain.Concept{ID: domain.ConceptID(slug), Topic: domain.TopicID(topic), Name: slug})
	}
	for slug, ps := range prereqs {
		if _, ok := live[slug]; !ok {
			continue
		}
		for _, p := range ps {
			if _, ok := live[p]; !ok {
				return E(KindInvalid, "unknown_concept", "concept %q lists unknown or archived prerequisite %q", slug, p)
			}
			dp = append(dp, domain.Prerequisite{Concept: domain.ConceptID(slug), Prerequisite: domain.ConceptID(p)})
		}
	}
	if _, err := domain.NewGraph(dt, dc, dp); err != nil {
		code := "invalid_graph"
		switch {
		case errors.Is(err, domain.ErrCycle):
			code = "prerequisite_cycle"
		case errors.Is(err, domain.ErrSelfReference):
			code = "self_prerequisite"
		}
		return &Error{Kind: KindInvalid, Code: code, Message: err.Error()}
	}
	return nil
}
