package domain

// RecommendationKind distinguishes studying a Root Cause from diagnosing an Unassessed Concept.
type RecommendationKind string

const (
	KindStudy    RecommendationKind = "study"
	KindDiagnose RecommendationKind = "diagnose"
)

// Recommendation suggests which Concept to study (or diagnose) next, and why.
type Recommendation struct {
	Kind    RecommendationKind
	Concept ConceptID // Root Cause (study) or Concept to diagnose
	Target  ConceptID // the struggling (Weak) Concept this serves
	// Path is the reason: Concept, then each Prerequisite-chain step up to Target (inclusive).
	Path     []ConceptID
	Unblocks int     // number of Weak Concepts among Descendants(Concept), plus Concept itself if Weak
	Priority float64 // Unblocks × ExamWeight of Concept's Topic (nil weight = 1); diagnose ranks as if Unblocks counted the Target's Weak chain
}

// Recommend returns Recommendations ordered by Priority (desc), ties broken by
// Concept ID for determinism, de-duplicated by (Kind, Concept).
//
// Root Cause search, for each Weak Concept T:
//   - walk Prerequisites from T;
//   - never descend below a Mastered Concept (its Prerequisites are assumed fine);
//   - a Weak Prerequisite is descended into;
//   - an Unassessed Prerequisite on the way yields a KindDiagnose Recommendation for it
//     (and the search does not assume it is fine);
//   - a Weak Concept none of whose Prerequisites are Weak or Unassessed is a Root Cause → KindStudy.
//
// If T itself has no Weak/Unassessed Prerequisites, T is its own Root Cause.
// When the same Root Cause serves several Targets, keep the one with the longest Path.
func Recommend(g *Graph, mastery map[ConceptID]Mastery) []Recommendation {
	panic("TODO")
}

// LearningPathSteps returns the Prerequisite chain from root up to target
// (inclusive both ends), choosing the shortest chain; ErrUnknownConcept if
// either is missing, or an error if target does not depend on root (unless root == target).
func LearningPathSteps(g *Graph, root, target ConceptID) ([]ConceptID, error) {
	panic("TODO")
}

// NextPathStep returns the first step that is not yet Developing or Mastered,
// and false when the Learning Path is complete.
func NextPathStep(steps []ConceptID, mastery map[ConceptID]Mastery) (ConceptID, bool) {
	panic("TODO")
}

// NextPlacementConcept picks the next Concept to probe in an adaptive Placement Test.
// asked holds Concepts already probed. Strategy: start from Concepts with the most
// Ancestors (deep, high-information); if a probed Concept is Mastered, its Ancestors
// are skipped; if Weak, probe its direct Prerequisites next. Returns false when
// maxProbes is reached or nothing informative remains. Only Concepts in
// `available` (those with Active Questions) may be chosen.
func NextPlacementConcept(g *Graph, mastery map[ConceptID]Mastery, asked map[ConceptID]bool, available map[ConceptID]bool, maxProbes int) (ConceptID, bool) {
	panic("TODO")
}
