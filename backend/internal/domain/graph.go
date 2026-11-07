package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ConceptID and TopicID are opaque identifiers (UUID strings in practice).
type (
	ConceptID string
	TopicID   string
)

// Concept is a node of the knowledge graph.
type Concept struct {
	ID    ConceptID
	Topic TopicID
	Name  string
}

// Topic groups Concepts. ExamWeight is nil when unknown (treated as 1).
type Topic struct {
	ID         TopicID
	Name       string
	ExamWeight *float64
}

// Prerequisite says Prerequisite must be understood before Concept.
type Prerequisite struct {
	Concept      ConceptID
	Prerequisite ConceptID
}

var (
	ErrUnknownConcept = errors.New("domain: unknown concept")
	ErrUnknownTopic   = errors.New("domain: unknown topic")
	ErrCycle          = errors.New("domain: prerequisite cycle")
	ErrSelfReference  = errors.New("domain: concept cannot be its own prerequisite")
	// ErrDuplicateID is returned by NewGraph when two Topics or two Concepts share an ID.
	ErrDuplicateID = errors.New("domain: duplicate id")
	// ErrNotPrerequisite is returned by LearningPathSteps when target does not
	// (transitively) depend on root.
	ErrNotPrerequisite = errors.New("domain: target does not depend on root")
)

// Graph is an immutable, validated DAG of Concepts.
type Graph struct {
	// unexported fields, filled by NewGraph
	topics     map[TopicID]Topic
	concepts   map[ConceptID]Concept
	prereqs    map[ConceptID][]ConceptID // sorted by ID, de-duplicated
	dependents map[ConceptID][]ConceptID // sorted by ID, de-duplicated
	order      []ConceptID               // topological (Prerequisites first), ties by ID
}

// NewGraph validates and builds a Graph. It returns ErrUnknownTopic,
// ErrUnknownConcept, ErrSelfReference, ErrDuplicateID, or ErrCycle (wrapped
// with detail). Duplicate Prerequisite pairs are collapsed silently.
//
// The Graph has no notion of archiving: callers pass only live (non-archived)
// Concepts and the Prerequisites between them. A Prerequisite that mentions a
// Concept not passed in (e.g. an archived one) is rejected with ErrUnknownConcept.
func NewGraph(topics []Topic, concepts []Concept, prereqs []Prerequisite) (*Graph, error) {
	g := &Graph{
		topics:     make(map[TopicID]Topic, len(topics)),
		concepts:   make(map[ConceptID]Concept, len(concepts)),
		prereqs:    make(map[ConceptID][]ConceptID),
		dependents: make(map[ConceptID][]ConceptID),
	}
	for _, t := range topics {
		if _, dup := g.topics[t.ID]; dup {
			return nil, fmt.Errorf("%w: topic %q", ErrDuplicateID, t.ID)
		}
		g.topics[t.ID] = t
	}
	for _, c := range concepts {
		if _, dup := g.concepts[c.ID]; dup {
			return nil, fmt.Errorf("%w: concept %q", ErrDuplicateID, c.ID)
		}
		if _, ok := g.topics[c.Topic]; !ok {
			return nil, fmt.Errorf("%w: concept %q references topic %q", ErrUnknownTopic, c.ID, c.Topic)
		}
		g.concepts[c.ID] = c
	}
	seen := make(map[Prerequisite]bool, len(prereqs))
	for _, p := range prereqs {
		if _, ok := g.concepts[p.Concept]; !ok {
			return nil, fmt.Errorf("%w: %q (prerequisite target)", ErrUnknownConcept, p.Concept)
		}
		if _, ok := g.concepts[p.Prerequisite]; !ok {
			return nil, fmt.Errorf("%w: %q (prerequisite of %q)", ErrUnknownConcept, p.Prerequisite, p.Concept)
		}
		if p.Concept == p.Prerequisite {
			return nil, fmt.Errorf("%w: %q", ErrSelfReference, p.Concept)
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		g.prereqs[p.Concept] = append(g.prereqs[p.Concept], p.Prerequisite)
		g.dependents[p.Prerequisite] = append(g.dependents[p.Prerequisite], p.Concept)
	}
	for _, m := range []map[ConceptID][]ConceptID{g.prereqs, g.dependents} {
		for _, ids := range m {
			sortIDs(ids)
		}
	}
	order, ok := topoOrder(g.concepts, g.prereqs, g.dependents)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCycle, describeCycle(g.concepts, g.prereqs))
	}
	g.order = order
	return g, nil
}

// topoOrder runs Kahn's algorithm, always taking the smallest ready ID.
// It reports false if the graph has a cycle.
func topoOrder(concepts map[ConceptID]Concept, prereqs, dependents map[ConceptID][]ConceptID) ([]ConceptID, bool) {
	indeg := make(map[ConceptID]int, len(concepts))
	var ready []ConceptID
	for id := range concepts {
		indeg[id] = len(prereqs[id])
		if indeg[id] == 0 {
			ready = append(ready, id)
		}
	}
	sortIDs(ready)
	order := make([]ConceptID, 0, len(concepts))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		var added bool
		for _, d := range dependents[id] {
			indeg[d]--
			if indeg[d] == 0 {
				ready = append(ready, d)
				added = true
			}
		}
		if added {
			sortIDs(ready)
		}
	}
	return order, len(order) == len(concepts)
}

// describeCycle finds one cycle (deterministically) and renders it as
// "a -> b -> a", where each arrow points from a Concept to one of its Prerequisites.
func describeCycle(concepts map[ConceptID]Concept, prereqs map[ConceptID][]ConceptID) string {
	const (
		white = iota
		grey
		black
	)
	color := make(map[ConceptID]int, len(concepts))
	var stack []ConceptID
	var found []ConceptID
	var visit func(id ConceptID) bool
	visit = func(id ConceptID) bool {
		color[id] = grey
		stack = append(stack, id)
		for _, p := range prereqs[id] {
			switch color[p] {
			case grey:
				for i, s := range stack {
					if s == p {
						found = append(append([]ConceptID{}, stack[i:]...), p)
						return true
					}
				}
			case white:
				if visit(p) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
		return false
	}
	ids := make([]ConceptID, 0, len(concepts))
	for id := range concepts {
		ids = append(ids, id)
	}
	sortIDs(ids)
	for _, id := range ids {
		if color[id] == white && visit(id) {
			break
		}
	}
	parts := make([]string, len(found))
	for i, id := range found {
		parts[i] = string(id)
	}
	return strings.Join(parts, " -> ")
}

// Concept returns the Concept and whether it exists.
func (g *Graph) Concept(id ConceptID) (Concept, bool) {
	c, ok := g.concepts[id]
	return c, ok
}

// Topic returns the Topic and whether it exists.
func (g *Graph) Topic(id TopicID) (Topic, bool) {
	t, ok := g.topics[id]
	return t, ok
}

// Concepts returns all Concepts in a deterministic order: topological
// (every Prerequisite before the Concepts that need it), ties broken by ID.
func (g *Graph) Concepts() []Concept {
	out := make([]Concept, len(g.order))
	for i, id := range g.order {
		out[i] = g.concepts[id]
	}
	return out
}

// Prerequisites returns the direct Prerequisites of a Concept, sorted by ID.
// It returns nil for an unknown Concept.
func (g *Graph) Prerequisites(id ConceptID) []ConceptID {
	return cloneIDs(g.prereqs[id])
}

// Dependents returns the Concepts that directly list id as a Prerequisite,
// sorted by ID. It returns nil for an unknown Concept.
func (g *Graph) Dependents(id ConceptID) []ConceptID {
	return cloneIDs(g.dependents[id])
}

// Ancestors returns all transitive Prerequisites of a Concept (excluding the
// Concept itself), sorted by ID. It returns nil for an unknown Concept.
func (g *Graph) Ancestors(id ConceptID) []ConceptID {
	return closure(id, g.prereqs)
}

// Descendants returns all Concepts that transitively depend on id (excluding
// id itself), sorted by ID. It returns nil for an unknown Concept.
func (g *Graph) Descendants(id ConceptID) []ConceptID {
	return closure(id, g.dependents)
}

// CheckPrerequisites reports whether replacing the Prerequisites of concept
// with prereqs would keep the graph acyclic, without mutating g. It returns
// ErrUnknownConcept if concept or any of prereqs is missing, ErrSelfReference
// if concept lists itself, ErrCycle if a proposed Prerequisite already
// (transitively) depends on concept, and nil otherwise.
func (g *Graph) CheckPrerequisites(concept ConceptID, prereqs []ConceptID) error {
	if _, ok := g.concepts[concept]; !ok {
		return fmt.Errorf("%w: %q", ErrUnknownConcept, concept)
	}
	for _, p := range prereqs {
		if _, ok := g.concepts[p]; !ok {
			return fmt.Errorf("%w: %q (proposed prerequisite of %q)", ErrUnknownConcept, p, concept)
		}
		if p == concept {
			return fmt.Errorf("%w: %q", ErrSelfReference, concept)
		}
	}
	// Replacing concept's own Prerequisites cannot create a cycle unless a new
	// Prerequisite is reachable from concept through dependents.
	desc := make(map[ConceptID]bool)
	for _, d := range g.Descendants(concept) {
		desc[d] = true
	}
	for _, p := range prereqs {
		if desc[p] {
			return fmt.Errorf("%w: %q already depends on %q", ErrCycle, p, concept)
		}
	}
	return nil
}

// closure returns every node reachable from start via adj (start excluded), sorted.
func closure(start ConceptID, adj map[ConceptID][]ConceptID) []ConceptID {
	seen := map[ConceptID]bool{start: true}
	queue := append([]ConceptID(nil), adj[start]...)
	var out []ConceptID
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
		queue = append(queue, adj[id]...)
	}
	sortIDs(out)
	return out
}

func sortIDs(ids []ConceptID) {
	slices.Sort(ids)
}

func cloneIDs(ids []ConceptID) []ConceptID {
	if len(ids) == 0 {
		return nil
	}
	return append([]ConceptID(nil), ids...)
}
