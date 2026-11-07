package domain

import (
	"errors"
	"reflect"
	"testing"
)

type recWant struct {
	kind     RecommendationKind
	concept  ConceptID
	target   ConceptID
	path     []ConceptID
	unblocks int
	priority float64
}

func checkRecs(t *testing.T, got []Recommendation, want []recWant) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d recommendations %+v, want %d", len(got), got, len(want))
	}
	for i, w := range want {
		g := got[i]
		if g.Kind != w.kind || g.Concept != w.concept || g.Target != w.target ||
			!reflect.DeepEqual(g.Path, w.path) || g.Unblocks != w.unblocks || !approx(g.Priority, w.priority) {
			t.Errorf("rec[%d] = %+v\n want %+v", i, g, w)
		}
	}
}

func TestRecommend(t *testing.T) {
	type L = map[ConceptID]MasteryLevel
	tests := []struct {
		name  string
		ids   []ConceptID
		edges []edge
		lv    L
		want  []recWant
	}{
		{
			name: "no mastery, nothing Weak",
			ids:  ids("a", "b"), edges: []edge{{"b", "a"}},
			lv:   L{},
			want: nil,
		},
		{
			name: "Weak concept with no prerequisites is its own Root Cause",
			ids:  ids("a"),
			lv:   L{"a": Weak},
			want: []recWant{{KindStudy, "a", "a", ids("a"), 1, 1}},
		},
		{
			name: "Weak chain: deepest Weak is Root Cause, longest Path kept across Targets",
			ids:  ids("a", "b", "c"), edges: []edge{{"c", "b"}, {"b", "a"}},
			lv:   L{"a": Weak, "b": Weak, "c": Weak},
			want: []recWant{{KindStudy, "a", "c", ids("a", "b", "c"), 3, 3}},
		},
		{
			name: "Mastered prerequisite stops descent",
			ids:  ids("a", "b", "c"), edges: []edge{{"c", "b"}, {"b", "a"}},
			lv: L{"a": Weak, "b": Mastered, "c": Weak},
			want: []recWant{
				{KindStudy, "a", "a", ids("a"), 2, 2}, // a itself + Weak descendant c
				{KindStudy, "c", "c", ids("c"), 1, 1},
			},
		},
		{
			name: "Developing prerequisite stops descent",
			ids:  ids("a", "b", "c"), edges: []edge{{"c", "b"}, {"b", "a"}},
			lv:   L{"b": Developing, "c": Weak},
			want: []recWant{{KindStudy, "c", "c", ids("c"), 1, 1}},
		},
		{
			name: "Unassessed prerequisite yields diagnose and blocks the Root Cause",
			ids:  ids("a", "b", "c"), edges: []edge{{"c", "b"}, {"b", "a"}},
			lv: L{"a": Weak, "c": Weak}, // b absent → Unassessed
			want: []recWant{
				{KindStudy, "a", "a", ids("a"), 2, 2},
				{KindDiagnose, "b", "c", ids("b", "c"), 1, 1},
			},
		},
		{
			name: "explicit Unassessed and empty level both mean Unassessed",
			ids:  ids("u", "v", "t"), edges: []edge{{"t", "u"}, {"t", "v"}},
			lv: L{"t": Weak, "u": Unassessed, "v": ""},
			want: []recWant{
				{KindDiagnose, "u", "t", ids("u", "t"), 1, 1},
				{KindDiagnose, "v", "t", ids("v", "t"), 1, 1},
			},
		},
		{
			name: "deep Unassessed below a Weak chain",
			ids:  ids("a", "b", "c"), edges: []edge{{"c", "b"}, {"b", "a"}},
			lv:   L{"b": Weak, "c": Weak},
			want: []recWant{{KindDiagnose, "a", "c", ids("a", "b", "c"), 2, 2}},
		},
		{
			name: "Unassessed prerequisite is not descended below",
			ids:  ids("a", "b", "c"), edges: []edge{{"c", "b"}, {"b", "a"}},
			lv:   L{"c": Weak, "a": Unassessed}, // b absent
			want: []recWant{{KindDiagnose, "b", "c", ids("b", "c"), 1, 1}},
		},
		{
			name: "Weak and Unassessed siblings: diagnose plus deeper study",
			ids:  ids("w", "u", "t"), edges: []edge{{"t", "w"}, {"t", "u"}},
			lv: L{"t": Weak, "w": Weak},
			want: []recWant{
				{KindStudy, "w", "t", ids("w", "t"), 2, 2},
				{KindDiagnose, "u", "t", ids("u", "t"), 1, 1},
			},
		},
		{
			name: "diamond: one Root Cause, longest then lexicographically smallest Path",
			ids:  ids("a", "b", "c", "d"), edges: []edge{{"d", "c"}, {"d", "b"}, {"b", "a"}, {"c", "a"}},
			lv:   L{"a": Weak, "b": Weak, "c": Weak, "d": Weak},
			want: []recWant{{KindStudy, "a", "d", ids("a", "b", "d"), 4, 4}},
		},
		{
			name: "diamond with uneven arms picks the longer chain",
			ids:  ids("a", "b", "c", "d", "e"), edges: []edge{{"e", "a"}, {"e", "d"}, {"d", "c"}, {"c", "a"}, {"b", "a"}},
			lv:   L{"a": Weak, "c": Weak, "d": Weak, "e": Weak},
			want: []recWant{{KindStudy, "a", "e", ids("a", "c", "d", "e"), 4, 4}},
		},
		{
			name: "multiple Root Causes tie on Priority, broken by Concept ID",
			ids:  ids("q", "p", "x"), edges: []edge{{"x", "p"}, {"x", "q"}},
			lv: L{"p": Weak, "q": Weak, "x": Weak},
			want: []recWant{
				{KindStudy, "p", "x", ids("p", "x"), 2, 2},
				{KindStudy, "q", "x", ids("q", "x"), 2, 2},
			},
		},
		{
			name: "de-duplicated across Targets keeps longest Path",
			ids:  ids("r", "m", "t1", "t2"), edges: []edge{{"t1", "r"}, {"t2", "m"}, {"m", "r"}},
			lv:   L{"r": Weak, "m": Weak, "t1": Weak, "t2": Weak},
			want: []recWant{{KindStudy, "r", "t2", ids("r", "m", "t2"), 4, 4}},
		},
		{
			name: "equal-length Paths across Targets: smaller Target ID wins",
			ids:  ids("r", "tb", "ta"), edges: []edge{{"tb", "r"}, {"ta", "r"}},
			lv:   L{"r": Weak, "ta": Weak, "tb": Weak},
			want: []recWant{{KindStudy, "r", "ta", ids("r", "ta"), 3, 3}},
		},
		{
			name: "diagnose de-duplicated across Targets",
			ids:  ids("u", "t1", "m", "t2"), edges: []edge{{"t1", "u"}, {"m", "u"}, {"t2", "m"}},
			lv:   L{"t1": Weak, "m": Weak, "t2": Weak},
			want: []recWant{{KindDiagnose, "u", "t2", ids("u", "m", "t2"), 3, 3}},
		},
		{
			name: "mastery for Concepts outside the graph (archived) is ignored",
			ids:  ids("a"),
			lv:   L{"ghost": Weak, "a": Mastered},
			want: nil,
		},
		{
			name: "unrelated Weak descendants count toward Unblocks",
			ids:  ids("a", "b", "c"), edges: []edge{{"b", "a"}, {"c", "b"}},
			lv: L{"a": Weak, "b": Developing, "c": Weak},
			want: []recWant{
				{KindStudy, "a", "a", ids("a"), 2, 2},
				{KindStudy, "c", "c", ids("c"), 1, 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := mkGraph(t, tt.ids, tt.edges)
			checkRecs(t, Recommend(g, levels(tt.lv)), tt.want)
		})
	}
}

func TestRecommendPriorityExamWeight(t *testing.T) {
	// x (nil weight topic) needs p (nil weight) and q (weight 3) and h (weight 0.5).
	g, err := NewGraph(
		[]Topic{{ID: "plain"}, {ID: "heavy", ExamWeight: ptr(3)}, {ID: "light", ExamWeight: ptr(0.5)}},
		[]Concept{{ID: "x", Topic: "plain"}, {ID: "p", Topic: "plain"}, {ID: "q", Topic: "heavy"}, {ID: "h", Topic: "light"}},
		[]Prerequisite{{Concept: "x", Prerequisite: "p"}, {Concept: "x", Prerequisite: "q"}, {Concept: "x", Prerequisite: "h"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	got := Recommend(g, levels(map[ConceptID]MasteryLevel{"x": Weak, "p": Weak, "q": Weak, "h": Weak}))
	checkRecs(t, got, []recWant{
		{KindStudy, "q", "x", ids("q", "x"), 2, 6},
		{KindStudy, "p", "x", ids("p", "x"), 2, 2},
		{KindStudy, "h", "x", ids("h", "x"), 2, 1},
	})
}

func TestRecommendDeterministic(t *testing.T) {
	g := diamond(t)
	m := levels(map[ConceptID]MasteryLevel{"a": Weak, "b": Weak, "c": Weak, "d": Weak, "e": Weak})
	first := Recommend(g, m)
	for i := 0; i < 50; i++ {
		if got := Recommend(g, m); !reflect.DeepEqual(got, first) {
			t.Fatalf("run %d differs: %+v vs %+v", i, got, first)
		}
	}
}

func TestRecommendPathIsNotAliased(t *testing.T) {
	g := mkGraph(t, ids("a", "b"), []edge{{"b", "a"}})
	m := levels(map[ConceptID]MasteryLevel{"a": Weak, "b": Weak})
	r := Recommend(g, m)
	r[0].Path[0] = "mutated"
	if got := Recommend(g, m)[0].Path[0]; got != "a" {
		t.Fatalf("path aliased: %v", got)
	}
}

func TestLearningPathSteps(t *testing.T) {
	// c needs b and a; b needs a (shortcut a→c); d needs b and c; x isolated.
	g := mkGraph(t, ids("a", "b", "c", "d", "x"), []edge{{"c", "b"}, {"c", "a"}, {"b", "a"}, {"d", "b"}, {"d", "c"}})
	tests := []struct {
		name         string
		root, target ConceptID
		want         []ConceptID
		wantErr      error
	}{
		{"root equals target", "a", "a", ids("a"), nil},
		{"direct", "a", "b", ids("a", "b"), nil},
		{"shortest chain uses shortcut", "a", "c", ids("a", "c"), nil},
		{"tie broken by ID order of dependents", "a", "d", ids("a", "b", "d"), nil},
		{"unknown root", "zz", "a", nil, ErrUnknownConcept},
		{"unknown target", "a", "zz", nil, ErrUnknownConcept},
		{"wrong direction", "c", "a", nil, ErrNotPrerequisite},
		{"unrelated", "x", "d", nil, ErrNotPrerequisite},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LearningPathSteps(g, tt.root, tt.target)
			if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil && err != nil) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNextPathStep(t *testing.T) {
	steps := ids("a", "b", "c")
	type L = map[ConceptID]MasteryLevel
	tests := []struct {
		name   string
		steps  []ConceptID
		lv     L
		want   ConceptID
		wantOK bool
	}{
		{"empty path is complete", nil, L{}, "", false},
		{"first step Weak", steps, L{"a": Weak}, "a", true},
		{"skips Developing and Mastered", steps, L{"a": Mastered, "b": Developing, "c": Weak}, "c", true},
		{"absent Concept is Unassessed, not done", steps, L{"a": Mastered}, "b", true},
		{"explicit Unassessed is not done", steps, L{"a": Unassessed}, "a", true},
		{"all Developing or better is complete", steps, L{"a": Developing, "b": Mastered, "c": Developing}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NextPathStep(tt.steps, levels(tt.lv))
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("got %q,%v want %q,%v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNextPlacementConcept(t *testing.T) {
	// chain d→c→b→a (d deepest), e needs a, x isolated.
	g := mkGraph(t, ids("a", "b", "c", "d", "e", "x"), []edge{{"d", "c"}, {"c", "b"}, {"b", "a"}, {"e", "a"}})
	all := map[ConceptID]bool{"a": true, "b": true, "c": true, "d": true, "e": true, "x": true}
	without := func(drop ...ConceptID) map[ConceptID]bool {
		m := map[ConceptID]bool{}
		for k, v := range all {
			m[k] = v
		}
		for _, d := range drop {
			delete(m, d)
		}
		return m
	}
	type L = map[ConceptID]MasteryLevel
	type A = map[ConceptID]bool
	tests := []struct {
		name      string
		lv        L
		asked     A
		available map[ConceptID]bool
		max       int
		want      ConceptID
		wantOK    bool
	}{
		{"start from deepest", L{}, A{}, all, 10, "d", true},
		{"nil asked map", L{}, nil, all, 10, "d", true},
		{"maxProbes reached", L{}, A{"d": true}, all, 1, "", false},
		{"maxProbes zero", L{}, A{}, all, 0, "", false},
		{"false entries in asked are not probes", L{}, A{"d": false}, all, 1, "d", true},
		{"Mastered probe skips its ancestors", L{"d": Mastered}, A{"d": true}, all, 10, "e", true},
		{"Weak probe → direct prerequisite next", L{"d": Weak}, A{"d": true}, all, 10, "c", true},
		{"Weak probe with unavailable prerequisite falls back to deepest", L{"d": Weak}, A{"d": true}, without("c"), 10, "b", true},
		{"Weak chain keeps descending", L{"d": Weak, "c": Weak}, A{"d": true, "c": true}, all, 10, "b", true},
		{"Mastered elsewhere prunes Weak probe's prerequisite", L{"d": Weak, "e": Mastered}, A{"d": true, "e": true}, without("c", "b"), 10, "x", true},
		{"only available Concepts", L{}, A{}, A{"x": true}, 10, "x", true},
		{"nothing available", L{}, A{}, A{}, 10, "", false},
		{"already-assessed Concepts are skipped", L{"d": Developing, "c": Weak, "e": Developing}, A{}, all, 10, "b", true},
		{"available but unknown to graph is ignored", L{}, A{}, A{"ghost": true}, 10, "", false},
		{"everything asked → nothing informative remains", L{}, all, all, 100, "", false},
		{
			"Mastered at the bottom prunes everything below",
			L{"d": Mastered, "e": Mastered}, A{"d": true, "e": true}, all, 10, "x", true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NextPlacementConcept(g, levels(tt.lv), tt.asked, tt.available, tt.max)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("got %q,%v want %q,%v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNextPlacementConceptTieByID(t *testing.T) {
	g := diamond(t) // d needs b,c; both need a
	all := map[ConceptID]bool{"a": true, "b": true, "c": true, "d": true, "e": true}
	got, ok := NextPlacementConcept(g, levels(map[ConceptID]MasteryLevel{"d": Weak}), map[ConceptID]bool{"d": true}, all, 5)
	if !ok || got != "b" {
		t.Fatalf("got %q,%v want b", got, ok)
	}
	// With no Weak probe, equal-depth b and c tie → b.
	got, ok = NextPlacementConcept(g, nil, nil, map[ConceptID]bool{"b": true, "c": true}, 5)
	if !ok || got != "b" {
		t.Fatalf("got %q,%v want b", got, ok)
	}
}
