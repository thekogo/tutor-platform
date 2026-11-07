package domain

import "time"

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
}

// MasteryConfig holds the tunable parameters of the Mastery calculation.
type MasteryConfig struct {
	PrimaryWeight    float64         // weight of evidence for the Primary Concept (default 1.0)
	SecondaryWeight  float64         // weight for other tagged Concepts (default 0.5)
	RepeatWeight     float64         // multiplier for repeat Answers (default 0.3)
	RecencyDecay     float64         // per-newer-Answer multiplier on older evidence for the same Concept (default 0.85)
	DifficultyWeight map[int]float64 // default {1: 0.8, 2: 1.0, 3: 1.25}
	MinEvidence      float64         // below this total weight the level is Unassessed (default 3.0, before recency decay)
	WeakBelow        float64         // score < WeakBelow → Weak (default 0.5)
	MasteredAtLeast  float64         // score >= MasteredAtLeast → Mastered (default 0.8)
}

// DefaultMasteryConfig returns the defaults documented on MasteryConfig.
func DefaultMasteryConfig() MasteryConfig { panic("TODO") }

// Mastery is a Learner's current estimated command of one Concept.
type Mastery struct {
	Concept  ConceptID
	Score    float64 // 0..1; 0 when there is no evidence
	Evidence float64 // total weight of evidence (before recency decay)
	Level    MasteryLevel
}

// ComputeMastery derives Mastery for every Concept touched by the evidence.
//
// For each Concept, every Answer contributes weight
//
//	w = roleWeight (Primary/Secondary) × DifficultyWeight[d] × (RepeatWeight if repeat) × RecencyDecay^k
//
// where k is the number of newer Answers touching that Concept. Score is the
// weighted share of correct Answers. Level is Unassessed while the undecayed
// weight sum is below MinEvidence, otherwise banded by WeakBelow/MasteredAtLeast.
// Concepts with no evidence are absent from the result (callers treat them as Unassessed).
func ComputeMastery(evidence []AnswerEvidence, cfg MasteryConfig) map[ConceptID]Mastery {
	panic("TODO")
}

// LevelOf returns the Mastery for id from m, or an Unassessed zero Mastery.
func LevelOf(m map[ConceptID]Mastery, id ConceptID) Mastery { panic("TODO") }
