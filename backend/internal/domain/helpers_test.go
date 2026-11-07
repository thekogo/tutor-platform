package domain

import (
	"testing"
	"time"
)

// edge is {Concept, Prerequisite}.
type edge [2]ConceptID

// mkGraph builds a single-Topic Graph from Concept IDs and edges, failing the test on error.
func mkGraph(t testing.TB, ids []ConceptID, edges []edge) *Graph {
	t.Helper()
	g, err := buildGraph(ids, edges, nil)
	if err != nil {
		t.Fatalf("NewGraph: %v", err)
	}
	return g
}

func buildGraph(ids []ConceptID, edges []edge, weight *float64) (*Graph, error) {
	topics := []Topic{{ID: "t", Name: "T", ExamWeight: weight}}
	concepts := make([]Concept, len(ids))
	for i, id := range ids {
		concepts[i] = Concept{ID: id, Topic: "t", Name: string(id)}
	}
	prereqs := make([]Prerequisite, len(edges))
	for i, e := range edges {
		prereqs[i] = Prerequisite{Concept: e[0], Prerequisite: e[1]}
	}
	return NewGraph(topics, concepts, prereqs)
}

// levels builds a Mastery map with the given levels and plausible scores.
func levels(m map[ConceptID]MasteryLevel) map[ConceptID]Mastery {
	out := make(map[ConceptID]Mastery, len(m))
	for id, l := range m {
		score := map[MasteryLevel]float64{Weak: 0.2, Developing: 0.6, Mastered: 0.9, Unassessed: 0}[l]
		out[id] = Mastery{Concept: id, Score: score, Evidence: 5, Level: l}
	}
	return out
}

func ids(s ...string) []ConceptID {
	out := make([]ConceptID, len(s))
	for i, x := range s {
		out[i] = ConceptID(x)
	}
	return out
}

func ptr(f float64) *float64 { return &f }

var t0 = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

func at(minutes int) time.Time { return t0.Add(time.Duration(minutes) * time.Minute) }
