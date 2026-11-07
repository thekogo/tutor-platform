package domain

import (
	"errors"
	"testing"
)

// FuzzNewGraph decodes bytes into a small graph: data[0] = concept count (mod 12),
// then pairs of bytes are (concept, prerequisite) indices, with out-of-range
// indices producing unknown IDs. NewGraph must never panic, must return only
// documented errors, and any accepted graph must be acyclic.
func FuzzNewGraph(f *testing.F) {
	f.Add([]byte{3, 1, 0, 2, 1})
	f.Add([]byte{2, 0, 1, 1, 0})
	f.Add([]byte{1, 0, 0})
	f.Add([]byte{4, 3, 1, 3, 2, 1, 0, 2, 0})
	f.Add([]byte{2, 0, 9})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		n := int(data[0]) % 12
		name := func(b byte) ConceptID { return ConceptID(rune('a' + int(b)%16)) }
		var concepts []Concept
		for i := 0; i < n; i++ {
			concepts = append(concepts, Concept{ID: name(byte(i)), Topic: "t"})
		}
		var prereqs []Prerequisite
		for i := 1; i+1 < len(data); i += 2 {
			prereqs = append(prereqs, Prerequisite{Concept: name(data[i]), Prerequisite: name(data[i+1])})
		}
		g, err := NewGraph([]Topic{{ID: "t"}}, concepts, prereqs)
		if err != nil {
			if !errors.Is(err, ErrUnknownConcept) && !errors.Is(err, ErrSelfReference) && !errors.Is(err, ErrCycle) {
				t.Fatalf("undocumented error: %v", err)
			}
			return
		}
		order := g.Concepts()
		if len(order) != n {
			t.Fatalf("Concepts() has %d, want %d", len(order), n)
		}
		pos := map[ConceptID]int{}
		for i, c := range order {
			pos[c.ID] = i
		}
		for _, c := range order {
			for _, a := range g.Ancestors(c.ID) {
				if a == c.ID {
					t.Fatalf("%q is its own ancestor", c.ID)
				}
			}
			for _, p := range g.Prerequisites(c.ID) {
				if pos[p] >= pos[c.ID] {
					t.Fatalf("prerequisite %q not before %q in topological order", p, c.ID)
				}
			}
		}
	})
}
