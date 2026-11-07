package domain

import (
	"reflect"
	"testing"
)

// cuBestSlice is a hand-written slice of content/cu-best-math/graph.yaml
// (the real file needs a YAML dependency, which the domain package avoids).
func cuBestSlice(t testing.TB) *Graph {
	t.Helper()
	g, err := NewGraph(
		[]Topic{
			{ID: "numbers", Name: "จำนวน"},
			{ID: "ratio", Name: "อัตราส่วน สัดส่วน", ExamWeight: ptr(2)},
			{ID: "percent", Name: "ร้อยละ เปอร์เซ็น", ExamWeight: ptr(3)},
			{ID: "applications", Name: "โจทย์ประยุกต์", ExamWeight: ptr(2)},
		},
		[]Concept{
			{ID: "integer-operations", Topic: "numbers"},
			{ID: "fractions", Topic: "numbers"},
			{ID: "word-problem-translation", Topic: "applications"},
			{ID: "ratio-basics", Topic: "ratio"},
			{ID: "proportion", Topic: "ratio"},
			{ID: "percent-basics", Topic: "percent"},
			{ID: "percent-of", Topic: "percent"},
			{ID: "percent-change", Topic: "percent"},
			{ID: "concentration", Topic: "applications"},
			{ID: "mixing-solutions", Topic: "applications"},
		},
		[]Prerequisite{
			{Concept: "fractions", Prerequisite: "integer-operations"},
			{Concept: "ratio-basics", Prerequisite: "fractions"},
			{Concept: "proportion", Prerequisite: "ratio-basics"},
			{Concept: "percent-basics", Prerequisite: "fractions"},
			{Concept: "percent-of", Prerequisite: "percent-basics"},
			{Concept: "percent-of", Prerequisite: "proportion"},
			{Concept: "percent-change", Prerequisite: "percent-of"},
			{Concept: "concentration", Prerequisite: "percent-of"},
			{Concept: "concentration", Prerequisite: "ratio-basics"},
			{Concept: "mixing-solutions", Prerequisite: "concentration"},
			{Concept: "mixing-solutions", Prerequisite: "proportion"},
			{Concept: "mixing-solutions", Prerequisite: "word-problem-translation"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// The design's recommendation story: a Learner struggles with mixing
// solutions; the real gap is ratio basics, several Prerequisite steps down.
func TestScenarioMixingSolutionsRootCauseIsRatioBasics(t *testing.T) {
	g := cuBestSlice(t)
	m := levels(map[ConceptID]MasteryLevel{
		"mixing-solutions":         Weak,
		"concentration":            Weak,
		"percent-of":               Weak,
		"proportion":               Weak,
		"ratio-basics":             Weak,
		"percent-basics":           Mastered,
		"word-problem-translation": Mastered,
		"fractions":                Mastered,
		"integer-operations":       Mastered,
		// percent-change never answered → Unassessed, but it is a Dependent, not on the way down.
	})

	recs := Recommend(g, m)
	checkRecs(t, recs, []recWant{{
		kind:     KindStudy,
		concept:  "ratio-basics",
		target:   "mixing-solutions",
		path:     ids("ratio-basics", "proportion", "percent-of", "concentration", "mixing-solutions"),
		unblocks: 5, // ratio-basics + proportion, percent-of, concentration, mixing-solutions
		priority: 10,
	}})

	// Learning Path: the Learner works up the reason path one Concept at a time.
	path := recs[0].Path
	if step, ok := NextPathStep(path, m); !ok || step != "ratio-basics" {
		t.Fatalf("first step = %q,%v", step, ok)
	}
	m["ratio-basics"] = Mastery{Concept: "ratio-basics", Score: 0.65, Evidence: 4, Level: Developing}
	if step, ok := NextPathStep(path, m); !ok || step != "proportion" {
		t.Fatalf("second step = %q,%v", step, ok)
	}
	// Re-recommending now points at proportion.
	if recs := Recommend(g, m); len(recs) != 1 || recs[0].Concept != "proportion" ||
		!reflect.DeepEqual(recs[0].Path, ids("proportion", "percent-of", "concentration", "mixing-solutions")) {
		t.Fatalf("after ratio-basics improved: %+v", recs)
	}
	for _, id := range path[1:] {
		m[id] = Mastery{Concept: id, Score: 0.9, Evidence: 4, Level: Mastered}
	}
	if step, ok := NextPathStep(path, m); ok {
		t.Fatalf("path should be complete, got %q", step)
	}
	if recs := Recommend(g, m); len(recs) != 0 {
		t.Fatalf("expected no recommendations, got %+v", recs)
	}

	// The shortest Prerequisite chain is available separately.
	short, err := LearningPathSteps(g, "ratio-basics", "mixing-solutions")
	if err != nil || !reflect.DeepEqual(short, ids("ratio-basics", "concentration", "mixing-solutions")) {
		t.Fatalf("LearningPathSteps = %v, %v", short, err)
	}
}

func TestScenarioUnassessedWordProblemsNeedsDiagnosis(t *testing.T) {
	g := cuBestSlice(t)
	m := levels(map[ConceptID]MasteryLevel{
		"mixing-solutions": Weak,
		"concentration":    Weak,
		"percent-of":       Mastered,
		"proportion":       Developing,
		"ratio-basics":     Weak,
		"fractions":        Mastered,
		// word-problem-translation Unassessed
	})
	checkRecs(t, Recommend(g, m), []recWant{
		{KindStudy, "ratio-basics", "mixing-solutions", ids("ratio-basics", "concentration", "mixing-solutions"), 3, 6},
		{KindDiagnose, "word-problem-translation", "mixing-solutions", ids("word-problem-translation", "mixing-solutions"), 1, 2},
	})
}

func TestScenarioPlacementWalk(t *testing.T) {
	g := cuBestSlice(t)
	available := map[ConceptID]bool{}
	for _, c := range g.Concepts() {
		available[c.ID] = true
	}
	asked := map[ConceptID]bool{}
	m := map[ConceptID]Mastery{}
	// Simulated Learner: weak at ratio-basics and everything depending on it.
	weak := map[ConceptID]bool{"ratio-basics": true}
	for _, d := range g.Descendants("ratio-basics") {
		weak[d] = true
	}
	var probed []ConceptID
	for {
		c, ok := NextPlacementConcept(g, m, asked, available, 6)
		if !ok {
			break
		}
		probed = append(probed, c)
		asked[c] = true
		l := Mastered
		if weak[c] {
			l = Weak
		}
		m[c] = Mastery{Concept: c, Level: l, Evidence: 3}
	}
	// percent-basics (Mastered) prunes fractions and integer-operations, so the
	// six probes end exactly at the Root Cause.
	want := ids("mixing-solutions", "concentration", "percent-of", "proportion", "percent-basics", "ratio-basics")
	if !reflect.DeepEqual(probed, want) {
		t.Fatalf("probed %v, want %v", probed, want)
	}
}
