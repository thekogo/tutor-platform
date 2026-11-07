package domain

import (
	"math"
	"sort"
	"time"
)

// MasteryLevel is the band a Mastery falls into.
type MasteryLevel string

const (
	Unassessed MasteryLevel = "unassessed"
	Weak       MasteryLevel = "weak"
	Developing MasteryLevel = "developing"
	Mastered   MasteryLevel = "mastered"
)

// AnswerEvidence is one counted Answer, already joined with its Question's tags.
// Excluded Answers (Flagged or key-changed Questions) must be filtered out by the caller.
type AnswerEvidence struct {
	QuestionID     string
	Correct        bool
	AnsweredAt     time.Time
	Difficulty     int // 1..3
	PrimaryConcept ConceptID
	OtherConcepts  []ConceptID
	IsRepeat       bool // the Learner had already answered this Question before
	IsPlacement    bool // given during a Placement Test (weighted by PlacementWeight)
}

// MasteryConfig holds the tunable parameters of the Mastery calculation.
type MasteryConfig struct {
	PrimaryWeight    float64         // weight of evidence for the Primary Concept (default 1.0)
	SecondaryWeight  float64         // weight for other tagged Concepts (default 0.5)
	RepeatWeight     float64         // multiplier for repeat Answers (default 0.3)
	PlacementWeight  float64         // multiplier for Placement Test Answers (default 4.0: one Primary probe at any Difficulty reaches MinEvidence)
	RecencyDecay     float64         // per-newer-Answer multiplier on older evidence for the same Concept (default 0.85)
	DifficultyWeight map[int]float64 // default {1: 0.8, 2: 1.0, 3: 1.25}; a Difficulty missing from the map weighs 1.0
	MinEvidence      float64         // below this total weight the level is Unassessed (default 3.0, before recency decay)
	WeakBelow        float64         // score < WeakBelow → Weak (default 0.5)
	MasteredAtLeast  float64         // score >= MasteredAtLeast → Mastered (default 0.8)
}

// DefaultMasteryConfig returns the defaults documented on MasteryConfig.
func DefaultMasteryConfig() MasteryConfig {
	return MasteryConfig{
		PrimaryWeight:    1.0,
		SecondaryWeight:  0.5,
		RepeatWeight:     0.3,
		PlacementWeight:  4.0,
		RecencyDecay:     0.85,
		DifficultyWeight: map[int]float64{1: 0.8, 2: 1.0, 3: 1.25},
		MinEvidence:      3.0,
		WeakBelow:        0.5,
		MasteredAtLeast:  0.8,
	}
}

// Mastery is a Learner's current estimated command of one Concept.
type Mastery struct {
	Concept  ConceptID
	Score    float64 // 0..1; 0 when there is no evidence
	Evidence float64 // total weight of evidence (before recency decay)
	Level    MasteryLevel
}

// floatTolerance absorbs rounding error in threshold comparisons so that,
// e.g., three default-weight Answers (0.8+1.0+1.25 style sums) or a score of
// exactly 4/5 land on the intended side of a boundary.
const floatTolerance = 1e-9

// ComputeMastery derives Mastery for every Concept touched by the evidence.
//
// For each Concept, every Answer contributes weight
//
//	w = roleWeight (Primary/Secondary) × DifficultyWeight[d] × (RepeatWeight if repeat) × (PlacementWeight if placement) × RecencyDecay^k
//
// where k is the number of Answers touching that Concept with a strictly later
// AnsweredAt (Answers with equal timestamps do not decay each other, so the
// result never depends on slice order). An Answer touches a Concept at most
// once: if the Primary Concept is also listed in OtherConcepts, or a Concept is
// listed twice, the Primary role (or a single Secondary contribution) counts.
// An empty PrimaryConcept contributes nothing.
//
// Score is the weighted share of correct Answers. Level is Unassessed while the
// undecayed weight sum is below MinEvidence, otherwise Weak if Score <
// WeakBelow, Mastered if Score >= MasteredAtLeast, else Developing. Threshold
// comparisons allow a 1e-9 tolerance for floating-point rounding.
// Concepts with no evidence are absent from the result (callers treat them as Unassessed).
func ComputeMastery(evidence []AnswerEvidence, cfg MasteryConfig) map[ConceptID]Mastery {
	type contribution struct {
		at      time.Time
		weight  float64 // undecayed
		correct bool
	}
	byConcept := make(map[ConceptID][]contribution)
	for _, a := range evidence {
		base := 1.0
		if dw, ok := cfg.DifficultyWeight[a.Difficulty]; ok {
			base = dw
		}
		if a.IsRepeat {
			base *= cfg.RepeatWeight
		}
		if a.IsPlacement && cfg.PlacementWeight > 0 {
			base *= cfg.PlacementWeight
		}
		touched := make(map[ConceptID]bool, 1+len(a.OtherConcepts))
		if a.PrimaryConcept != "" {
			touched[a.PrimaryConcept] = true
			byConcept[a.PrimaryConcept] = append(byConcept[a.PrimaryConcept],
				contribution{a.AnsweredAt, cfg.PrimaryWeight * base, a.Correct})
		}
		for _, c := range a.OtherConcepts {
			if c == "" || touched[c] {
				continue
			}
			touched[c] = true
			byConcept[c] = append(byConcept[c], contribution{a.AnsweredAt, cfg.SecondaryWeight * base, a.Correct})
		}
	}

	out := make(map[ConceptID]Mastery, len(byConcept))
	for id, cs := range byConcept {
		// Sort by time so k (the count of strictly newer Answers) is a binary search.
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].at.Before(cs[j].at) })
		var evidenceSum, total, correct float64
		for _, c := range cs {
			firstNewer := sort.Search(len(cs), func(i int) bool { return cs[i].at.After(c.at) })
			k := len(cs) - firstNewer
			w := c.weight * math.Pow(cfg.RecencyDecay, float64(k))
			evidenceSum += c.weight
			total += w
			if c.correct {
				correct += w
			}
		}
		score := 0.0
		if total > 0 {
			score = math.Min(1, math.Max(0, correct/total))
		}
		out[id] = Mastery{
			Concept:  id,
			Score:    score,
			Evidence: evidenceSum,
			Level:    levelFor(score, evidenceSum, cfg),
		}
	}
	return out
}

func levelFor(score, evidence float64, cfg MasteryConfig) MasteryLevel {
	switch {
	case evidence < cfg.MinEvidence-floatTolerance:
		return Unassessed
	case score < cfg.WeakBelow-floatTolerance:
		return Weak
	case score >= cfg.MasteredAtLeast-floatTolerance:
		return Mastered
	default:
		return Developing
	}
}

// LevelOf returns the Mastery for id from m, or an Unassessed zero Mastery
// (with Concept set to id) when m has no entry.
func LevelOf(m map[ConceptID]Mastery, id ConceptID) Mastery {
	if ms, ok := m[id]; ok {
		return ms
	}
	return Mastery{Concept: id, Level: Unassessed}
}
