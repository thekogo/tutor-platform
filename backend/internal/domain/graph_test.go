package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestNewGraphValidation(t *testing.T) {
	tests := []struct {
		name     string
		topics   []Topic
		concepts []Concept
		prereqs  []Prerequisite
		wantErr  error
		detail   string
	}{
		{name: "empty graph is valid"},
		{
			name:     "valid chain",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "t"}, {ID: "b", Topic: "t"}},
			prereqs:  []Prerequisite{{Concept: "b", Prerequisite: "a"}},
		},
		{
			name:     "unknown topic",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "missing"}},
			wantErr:  ErrUnknownTopic,
			detail:   "missing",
		},
		{
			name:     "unknown prerequisite (e.g. archived concept left out)",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "t"}},
			prereqs:  []Prerequisite{{Concept: "a", Prerequisite: "archived"}},
			wantErr:  ErrUnknownConcept,
			detail:   "archived",
		},
		{
			name:     "unknown dependent concept",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "t"}},
			prereqs:  []Prerequisite{{Concept: "ghost", Prerequisite: "a"}},
			wantErr:  ErrUnknownConcept,
			detail:   "ghost",
		},
		{
			name:     "self reference",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "t"}},
			prereqs:  []Prerequisite{{Concept: "a", Prerequisite: "a"}},
			wantErr:  ErrSelfReference,
		},
		{
			name:     "direct cycle",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "t"}, {ID: "b", Topic: "t"}},
			prereqs:  []Prerequisite{{Concept: "a", Prerequisite: "b"}, {Concept: "b", Prerequisite: "a"}},
			wantErr:  ErrCycle,
			detail:   "a -> b -> a",
		},
		{
			name:     "indirect cycle",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "t"}, {ID: "b", Topic: "t"}, {ID: "c", Topic: "t"}, {ID: "z", Topic: "t"}},
			prereqs: []Prerequisite{
				{Concept: "z", Prerequisite: "a"},
				{Concept: "a", Prerequisite: "b"},
				{Concept: "b", Prerequisite: "c"},
				{Concept: "c", Prerequisite: "a"},
			},
			wantErr: ErrCycle,
			detail:  "a -> b -> c -> a",
		},
		{
			name:    "duplicate topic",
			topics:  []Topic{{ID: "t"}, {ID: "t"}},
			wantErr: ErrDuplicateID,
		},
		{
			name:     "duplicate concept",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "t"}, {ID: "a", Topic: "t"}},
			wantErr:  ErrDuplicateID,
		},
		{
			name:     "duplicate prerequisite pair is collapsed",
			topics:   []Topic{{ID: "t"}},
			concepts: []Concept{{ID: "a", Topic: "t"}, {ID: "b", Topic: "t"}},
			prereqs:  []Prerequisite{{Concept: "b", Prerequisite: "a"}, {Concept: "b", Prerequisite: "a"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := NewGraph(tt.topics, tt.concepts, tt.prereqs)
			if tt.wantErr == nil {
				if err != nil || g == nil {
					t.Fatalf("NewGraph() = %v, %v; want graph", g, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if g != nil {
				t.Fatalf("graph returned alongside error")
			}
			if tt.detail != "" && !strings.Contains(err.Error(), tt.detail) {
				t.Fatalf("err %q does not contain %q", err, tt.detail)
			}
		})
	}
}

func TestGraphDuplicatePrerequisiteCollapsed(t *testing.T) {
	g := mkGraph(t, ids("a", "b"), []edge{{"b", "a"}, {"b", "a"}})
	if got := g.Prerequisites("b"); !reflect.DeepEqual(got, ids("a")) {
		t.Fatalf("Prerequisites(b) = %v", got)
	}
	if got := g.Dependents("a"); !reflect.DeepEqual(got, ids("b")) {
		t.Fatalf("Dependents(a) = %v", got)
	}
}

// diamond: d needs b and c; b and c need a; e is isolated.
func diamond(t testing.TB) *Graph {
	return mkGraph(t, ids("e", "d", "c", "b", "a"), []edge{{"d", "c"}, {"d", "b"}, {"b", "a"}, {"c", "a"}})
}

func TestGraphLookups(t *testing.T) {
	w := 2.0
	g, err := NewGraph(
		[]Topic{{ID: "t1", Name: "One", ExamWeight: &w}},
		[]Concept{{ID: "a", Topic: "t1", Name: "A"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := g.Concept("a"); !ok || c.Name != "A" || c.Topic != "t1" {
		t.Errorf("Concept(a) = %+v, %v", c, ok)
	}
	if _, ok := g.Concept("zz"); ok {
		t.Errorf("Concept(zz) found")
	}
	if tp, ok := g.Topic("t1"); !ok || tp.Name != "One" || *tp.ExamWeight != 2 {
		t.Errorf("Topic(t1) = %+v, %v", tp, ok)
	}
	if _, ok := g.Topic("zz"); ok {
		t.Errorf("Topic(zz) found")
	}
}

func TestGraphConceptsOrder(t *testing.T) {
	tests := []struct {
		name  string
		ids   []ConceptID
		edges []edge
		want  []ConceptID
	}{
		{"empty", nil, nil, []ConceptID{}},
		{"independent sorted by ID", ids("c", "a", "b"), nil, ids("a", "b", "c")},
		{"topological beats ID", ids("a", "z"), []edge{{"a", "z"}}, ids("z", "a")},
		{"diamond", ids("e", "d", "c", "b", "a"), []edge{{"d", "c"}, {"d", "b"}, {"b", "a"}, {"c", "a"}}, ids("a", "b", "c", "d", "e")},
		{"ready set re-sorted", ids("m", "b", "y", "x"), []edge{{"b", "m"}, {"y", "x"}}, ids("m", "b", "x", "y")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := mkGraph(t, tt.ids, tt.edges)
			got := []ConceptID{}
			for _, c := range g.Concepts() {
				got = append(got, c.ID)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Concepts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGraphNeighbours(t *testing.T) {
	g := diamond(t)
	tests := []struct {
		name string
		fn   func(ConceptID) []ConceptID
		id   ConceptID
		want []ConceptID
	}{
		{"prereqs of d", g.Prerequisites, "d", ids("b", "c")},
		{"prereqs of root", g.Prerequisites, "a", nil},
		{"prereqs of unknown", g.Prerequisites, "zz", nil},
		{"dependents of a", g.Dependents, "a", ids("b", "c")},
		{"dependents of leaf", g.Dependents, "d", nil},
		{"dependents of unknown", g.Dependents, "zz", nil},
		{"ancestors of d (diamond counted once)", g.Ancestors, "d", ids("a", "b", "c")},
		{"ancestors of b", g.Ancestors, "b", ids("a")},
		{"ancestors of a", g.Ancestors, "a", nil},
		{"ancestors of isolated", g.Ancestors, "e", nil},
		{"ancestors of unknown", g.Ancestors, "zz", nil},
		{"descendants of a", g.Descendants, "a", ids("b", "c", "d")},
		{"descendants of d", g.Descendants, "d", nil},
		{"descendants of unknown", g.Descendants, "zz", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.id); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGraphReturnedSlicesAreCopies(t *testing.T) {
	g := diamond(t)
	p := g.Prerequisites("d")
	p[0] = "mutated"
	d := g.Dependents("a")
	d[0] = "mutated"
	if g.Prerequisites("d")[0] != "b" || g.Dependents("a")[0] != "b" {
		t.Fatal("graph mutated through returned slice")
	}
}

func TestCheckPrerequisites(t *testing.T) {
	// chain: c needs b, b needs a; x isolated.
	g := mkGraph(t, ids("a", "b", "c", "x"), []edge{{"c", "b"}, {"b", "a"}})
	tests := []struct {
		name    string
		concept ConceptID
		prereqs []ConceptID
		wantErr error
	}{
		{"no prereqs", "b", nil, nil},
		{"replace with same", "c", ids("b"), nil},
		{"add transitive shortcut", "c", ids("a", "b"), nil},
		{"add isolated", "a", ids("x"), nil},
		{"unknown concept", "zz", ids("a"), ErrUnknownConcept},
		{"unknown prerequisite", "a", ids("zz"), ErrUnknownConcept},
		{"self reference", "a", ids("a"), ErrSelfReference},
		{"direct cycle", "b", ids("c"), ErrCycle},
		{"indirect cycle", "a", ids("c"), ErrCycle},
		{"replacing existing edge removes old cycle risk", "b", ids("x"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.CheckPrerequisites(tt.concept, tt.prereqs)
			if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil && err != nil) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
	// g unchanged
	if got := g.Prerequisites("b"); !reflect.DeepEqual(got, ids("a")) {
		t.Fatalf("CheckPrerequisites mutated graph: %v", got)
	}
}
