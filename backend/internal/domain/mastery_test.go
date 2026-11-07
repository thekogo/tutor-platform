package domain

import (
	"math"
	"reflect"
	"testing"
)

// plainConfig has no recency decay and flat difficulty weights so expected
// values are easy to compute by hand.
func plainConfig() MasteryConfig {
	return MasteryConfig{
		PrimaryWeight:    1,
		SecondaryWeight:  0.5,
		RepeatWeight:     0.3,
		RecencyDecay:     1,
		DifficultyWeight: map[int]float64{1: 1, 2: 1, 3: 1},
		MinEvidence:      3,
		WeakBelow:        0.5,
		MasteredAtLeast:  0.8,
	}
}

func ans(primary ConceptID, correct bool, minute int, others ...ConceptID) AnswerEvidence {
	return AnswerEvidence{
		QuestionID:     "q",
		Correct:        correct,
		AnsweredAt:     at(minute),
		Difficulty:     2,
		PrimaryConcept: primary,
		OtherConcepts:  others,
	}
}

// results builds n Answers on concept a at increasing times, the first `correct` of them correct.
func results(n, correct int) []AnswerEvidence {
	out := make([]AnswerEvidence, n)
	for i := range out {
		out[i] = ans("a", i < correct, i)
	}
	return out
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestDefaultMasteryConfig(t *testing.T) {
	want := MasteryConfig{
		PrimaryWeight: 1.0, SecondaryWeight: 0.5, RepeatWeight: 0.3, PlacementWeight: 4.0, RecencyDecay: 0.85,
		DifficultyWeight: map[int]float64{1: 0.8, 2: 1.0, 3: 1.25},
		MinEvidence:      3.0, WeakBelow: 0.5, MasteredAtLeast: 0.8,
	}
	if got := DefaultMasteryConfig(); !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultMasteryConfig() = %+v", got)
	}
}

func TestComputeMastery(t *testing.T) {
	type want struct {
		score, evidence float64
		level           MasteryLevel
	}
	repeat := func(a AnswerEvidence) AnswerEvidence { a.IsRepeat = true; return a }
	diff := func(a AnswerEvidence, d int) AnswerEvidence { a.Difficulty = d; return a }

	tests := []struct {
		name     string
		cfg      MasteryConfig
		evidence []AnswerEvidence
		want     map[ConceptID]want
	}{
		{name: "nil evidence", cfg: plainConfig(), want: map[ConceptID]want{}},
		{name: "empty evidence", cfg: plainConfig(), evidence: []AnswerEvidence{}, want: map[ConceptID]want{}},
		{
			name: "below MinEvidence is Unassessed even if all wrong", cfg: plainConfig(),
			evidence: results(2, 0),
			want:     map[ConceptID]want{"a": {0, 2, Unassessed}},
		},
		{
			name: "exactly MinEvidence is assessed", cfg: plainConfig(),
			evidence: results(3, 0),
			want:     map[ConceptID]want{"a": {0, 3, Weak}},
		},
		{
			name: "MinEvidence reached by six secondary halves (float sum)", cfg: plainConfig(),
			evidence: []AnswerEvidence{
				ans("x", true, 0, "a"), ans("x", true, 1, "a"), ans("x", true, 2, "a"),
				ans("x", true, 3, "a"), ans("x", true, 4, "a"), ans("x", true, 5, "a"),
			},
			want: map[ConceptID]want{"a": {1, 3, Mastered}, "x": {1, 6, Mastered}},
		},
		{
			name: "default difficulty weights: three easy answers stay Unassessed", cfg: DefaultMasteryConfig(),
			evidence: []AnswerEvidence{diff(ans("a", true, 0), 1), diff(ans("a", true, 0), 1), diff(ans("a", true, 0), 1)},
			want:     map[ConceptID]want{"a": {1, 2.4, Unassessed}},
		},
		{
			name: "default difficulty weights: easy+medium+hard is assessed", cfg: DefaultMasteryConfig(),
			evidence: []AnswerEvidence{diff(ans("a", true, 0), 1), diff(ans("a", false, 0), 2), diff(ans("a", true, 0), 3)},
			want:     map[ConceptID]want{"a": {2.05 / 3.05, 3.05, Developing}},
		},
		{
			name: "difficulty missing from map weighs 1", cfg: DefaultMasteryConfig(),
			evidence: []AnswerEvidence{diff(ans("a", true, 0), 0), diff(ans("a", true, 0), 7)},
			want:     map[ConceptID]want{"a": {1, 2, Unassessed}},
		},
		{
			name: "score exactly 0.5 is Developing", cfg: plainConfig(),
			evidence: results(4, 2),
			want:     map[ConceptID]want{"a": {0.5, 4, Developing}},
		},
		{
			name: "score just under 0.5 is Weak", cfg: plainConfig(),
			evidence: results(5, 2),
			want:     map[ConceptID]want{"a": {0.4, 5, Weak}},
		},
		{
			name: "score exactly 0.8 is Mastered", cfg: plainConfig(),
			evidence: results(5, 4),
			want:     map[ConceptID]want{"a": {0.8, 5, Mastered}},
		},
		{
			name: "score just under 0.8 is Developing", cfg: plainConfig(),
			evidence: results(4, 3),
			want:     map[ConceptID]want{"a": {0.75, 4, Developing}},
		},
		{
			name: "repeat Answers weigh RepeatWeight", cfg: plainConfig(),
			evidence: []AnswerEvidence{ans("a", true, 0), repeat(ans("a", false, 1))},
			want:     map[ConceptID]want{"a": {1 / 1.3, 1.3, Unassessed}},
		},
		{
			name: "secondary Concepts get SecondaryWeight", cfg: plainConfig(),
			evidence: []AnswerEvidence{ans("a", true, 0, "b"), ans("b", false, 1)},
			want: map[ConceptID]want{
				"a": {1, 1, Unassessed},
				"b": {0.5 / 1.5, 1.5, Unassessed},
			},
		},
		{
			name: "primary listed again in others counts once as primary", cfg: plainConfig(),
			evidence: []AnswerEvidence{ans("a", true, 0, "a", "b", "b", "")},
			want: map[ConceptID]want{
				"a": {1, 1, Unassessed},
				"b": {1, 0.5, Unassessed},
			},
		},
		{
			name: "empty primary contributes only secondaries", cfg: plainConfig(),
			evidence: []AnswerEvidence{ans("", false, 0, "b")},
			want:     map[ConceptID]want{"b": {0, 0.5, Unassessed}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeMastery(tt.evidence, tt.cfg)
			if got == nil {
				t.Fatal("nil map")
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d concepts (%v), want %d", len(got), got, len(tt.want))
			}
			for id, w := range tt.want {
				m, ok := got[id]
				if !ok {
					t.Fatalf("missing %q", id)
				}
				if m.Concept != id || !approx(m.Score, w.score) || !approx(m.Evidence, w.evidence) || m.Level != w.level {
					t.Errorf("%s = %+v, want score=%v evidence=%v level=%v", id, m, w.score, w.evidence, w.level)
				}
			}
		})
	}
}

func TestComputeMasteryRecencyDecay(t *testing.T) {
	cfg := plainConfig()
	cfg.RecencyDecay = 0.5
	wrongOld := ans("a", false, 0)
	rightNew := ans("a", true, 10)

	tests := []struct {
		name     string
		evidence []AnswerEvidence
		score    float64
	}{
		// wrong decayed once (0.5), correct full (1) → 1/1.5
		{"chronological slice", []AnswerEvidence{wrongOld, rightNew}, 1 / 1.5},
		{"reverse slice gives same result (ordered by AnsweredAt)", []AnswerEvidence{rightNew, wrongOld}, 1 / 1.5},
		{"newer wrong answer dominates", []AnswerEvidence{ans("a", true, 0), ans("a", false, 10)}, 0.5 / 1.5},
		{"equal timestamps do not decay each other", []AnswerEvidence{ans("a", true, 5), ans("a", false, 5)}, 0.5},
		{
			"k counts every strictly newer Answer",
			// wrong@0 decays by 0.5^2, correct@1 and correct@2... ordering: w(0)=0.25, w(1)=0.5, w(2)=1
			[]AnswerEvidence{ans("a", true, 2), ans("a", false, 0), ans("a", true, 1)},
			1.5 / 1.75,
		},
		{
			"newer Answers on other Concepts do not decay",
			[]AnswerEvidence{ans("a", true, 0), ans("a", false, 1), ans("b", false, 99)},
			0.5 / 1.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ComputeMastery(tt.evidence, cfg)["a"]
			if !approx(m.Score, tt.score) {
				t.Fatalf("score = %v, want %v", m.Score, tt.score)
			}
			if !approx(m.Evidence, float64(countFor("a", tt.evidence))) {
				t.Fatalf("evidence = %v should be undecayed", m.Evidence)
			}
		})
	}
}

func countFor(id ConceptID, ev []AnswerEvidence) int {
	n := 0
	for _, e := range ev {
		if e.PrimaryConcept == id {
			n++
		}
	}
	return n
}

func TestComputeMasteryDecayAffectsLevelNotEvidence(t *testing.T) {
	// Old failures, recent successes: decay lifts the level.
	ev := []AnswerEvidence{ans("a", false, 0), ans("a", false, 1), ans("a", true, 2), ans("a", true, 3)}
	flat := ComputeMastery(ev, plainConfig())["a"]
	cfg := plainConfig()
	cfg.RecencyDecay = 0.5
	decayed := ComputeMastery(ev, cfg)["a"]
	if flat.Level != Developing || decayed.Level != Mastered {
		t.Fatalf("flat=%v decayed=%v", flat, decayed)
	}
	if flat.Evidence != decayed.Evidence {
		t.Fatalf("evidence differs: %v vs %v", flat.Evidence, decayed.Evidence)
	}
}

func TestLevelOf(t *testing.T) {
	m := map[ConceptID]Mastery{"a": {Concept: "a", Score: 0.9, Evidence: 4, Level: Mastered}}
	if got := LevelOf(m, "a"); got != m["a"] {
		t.Errorf("LevelOf(a) = %+v", got)
	}
	if got := LevelOf(m, "b"); got != (Mastery{Concept: "b", Level: Unassessed}) {
		t.Errorf("LevelOf(b) = %+v", got)
	}
	if got := LevelOf(nil, "c"); got.Level != Unassessed || got.Concept != "c" {
		t.Errorf("LevelOf(nil, c) = %+v", got)
	}
}

func TestComputeMasteryPlacementWeight(t *testing.T) {
	cfg := plainConfig()
	cfg.PlacementWeight = 4
	probe := ans("a", true, 0)
	probe.IsPlacement = true

	m := ComputeMastery([]AnswerEvidence{probe}, cfg)
	if got := m["a"]; !approx(got.Evidence, 4) || got.Level != Mastered {
		t.Fatalf("one placement probe should reach MinEvidence: %+v", got)
	}
	if got := ComputeMastery([]AnswerEvidence{ans("a", true, 0)}, cfg)["a"]; got.Level != Unassessed {
		t.Fatalf("one ordinary Answer must stay Unassessed: %+v", got)
	}

	// With the defaults, a probe at the lowest Difficulty is still enough.
	easy := probe
	easy.Difficulty = 1
	if got := ComputeMastery([]AnswerEvidence{easy}, DefaultMasteryConfig())["a"]; got.Level == Unassessed {
		t.Fatalf("default PlacementWeight too low for Difficulty 1: %+v", got)
	}
}
