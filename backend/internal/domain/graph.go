package domain

import "errors"

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
)

// Graph is an immutable, validated DAG of Concepts.
type Graph struct {
	// unexported fields, filled by NewGraph
	topics     map[TopicID]Topic
	concepts   map[ConceptID]Concept
	prereqs    map[ConceptID][]ConceptID
	dependents map[ConceptID][]ConceptID
}

// NewGraph validates and builds a Graph. It returns ErrUnknownTopic,
// ErrUnknownConcept, ErrSelfReference, or ErrCycle (wrapped with detail).
func NewGraph(topics []Topic, concepts []Concept, prereqs []Prerequisite) (*Graph, error) {
	panic("TODO")
}

// Concept returns the Concept and whether it exists.
func (g *Graph) Concept(id ConceptID) (Concept, bool) { panic("TODO") }

// Topic returns the Topic and whether it exists.
func (g *Graph) Topic(id TopicID) (Topic, bool) { panic("TODO") }

// Concepts returns all Concepts in a deterministic (topological, then ID) order.
func (g *Graph) Concepts() []Concept { panic("TODO") }

// Prerequisites returns the direct Prerequisites of a Concept.
func (g *Graph) Prerequisites(id ConceptID) []ConceptID { panic("TODO") }

// Dependents returns the Concepts that directly list id as a Prerequisite.
func (g *Graph) Dependents(id ConceptID) []ConceptID { panic("TODO") }

// Ancestors returns all transitive Prerequisites of a Concept.
func (g *Graph) Ancestors(id ConceptID) []ConceptID { panic("TODO") }

// Descendants returns all Concepts that transitively depend on id.
func (g *Graph) Descendants(id ConceptID) []ConceptID { panic("TODO") }

// CheckPrerequisites reports whether replacing the Prerequisites of concept
// with prereqs would keep the graph acyclic, without mutating g.
func (g *Graph) CheckPrerequisites(concept ConceptID, prereqs []ConceptID) error {
	panic("TODO")
}
