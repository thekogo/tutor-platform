package domain

import (
	"fmt"
	"slices"
	"sort"
)

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
	// Every step after Concept is Weak. When several chains exist, the longest is
	// chosen (ties: lexicographically smallest sequence of IDs).
	Path []ConceptID
	// Unblocks is the number of Weak Concepts among Descendants(Concept), plus
	// Concept itself if Weak. For KindDiagnose the Concept is Unassessed, so this
	// counts the Weak Concepts it may be holding back — at least the Target's
	// Weak chain.
	Unblocks int
	Priority float64 // Unblocks × ExamWeight of Concept's Topic (nil weight = 1)
}

// Recommend returns Recommendations ordered by Priority (desc), ties broken by
// Concept ID (then Kind) for determinism, de-duplicated by (Kind, Concept).
//
// Only Concepts in g are considered; mastery entries for other IDs (e.g.
// archived Concepts) are ignored, and Concepts absent from mastery are Unassessed.
//
// Root Cause search, for each Weak Concept T:
//   - walk Prerequisites from T;
//   - never descend below a Mastered Concept (its Prerequisites are assumed fine);
//   - never descend below a Developing Concept either: it is not Weak, so it
//     cannot be a Root Cause and does not block one;
//   - a Weak Prerequisite is descended into;
//   - an Unassessed Prerequisite on the way yields a KindDiagnose Recommendation for it
//     (and the search does not assume it is fine); the search does not descend
//     below it — diagnose first, then recommend again;
//   - a Weak Concept none of whose Prerequisites are Weak or Unassessed is a Root Cause → KindStudy.
//
// If T itself has no Weak/Unassessed Prerequisites, T is its own Root Cause.
// When the same Root Cause serves several Targets, keep the one with the longest
// Path (ties: smaller Target ID, then lexicographically smaller Path).
func Recommend(g *Graph, mastery map[ConceptID]Mastery) []Recommendation {
	level := func(id ConceptID) MasteryLevel {
		l := LevelOf(mastery, id).Level
		if l == "" {
			return Unassessed
		}
		return l
	}

	type key struct {
		kind    RecommendationKind
		concept ConceptID
	}
	best := make(map[key]Recommendation)
	offer := func(r Recommendation) {
		k := key{r.Kind, r.Concept}
		cur, ok := best[k]
		if !ok || betterPath(r, cur) {
			best[k] = r
		}
	}

	for _, target := range g.order { // deterministic iteration
		if level(target) != Weak {
			continue
		}
		// region: Weak Concepts reachable from target through Weak Prerequisites.
		region := map[ConceptID]bool{target: true}
		queue := []ConceptID{target}
		for len(queue) > 0 {
			n := queue[0]
			queue = queue[1:]
			for _, p := range g.prereqs[n] {
				if !region[p] && level(p) == Weak {
					region[p] = true
					queue = append(queue, p)
				}
			}
		}
		// up(n): longest chain n → … → target inside region.
		memo := make(map[ConceptID][]ConceptID)
		var up func(n ConceptID) []ConceptID
		up = func(n ConceptID) []ConceptID {
			if n == target {
				return []ConceptID{target}
			}
			if p, ok := memo[n]; ok {
				return p
			}
			var bestPath []ConceptID
			for _, d := range g.dependents[n] {
				if !region[d] {
					continue
				}
				cand := append([]ConceptID{n}, up(d)...)
				if bestPath == nil || longerOrLexLess(cand, bestPath) {
					bestPath = cand
				}
			}
			memo[n] = bestPath
			return bestPath
		}
		for _, n := range g.order {
			if !region[n] {
				continue
			}
			blocked := false
			for _, p := range g.prereqs[n] {
				switch level(p) {
				case Weak:
					blocked = true
				case Unassessed:
					blocked = true
					offer(Recommendation{
						Kind:    KindDiagnose,
						Concept: p,
						Target:  target,
						Path:    append([]ConceptID{p}, up(n)...),
					})
				case Developing, Mastered:
					// not a gap: the search stops here
				}
			}
			if !blocked {
				offer(Recommendation{Kind: KindStudy, Concept: n, Target: target, Path: slices.Clone(up(n))})
			}
		}
	}

	out := make([]Recommendation, 0, len(best))
	for _, r := range best {
		r.Unblocks = 0
		if level(r.Concept) == Weak {
			r.Unblocks++
		}
		for _, d := range g.Descendants(r.Concept) {
			if level(d) == Weak {
				r.Unblocks++
			}
		}
		r.Priority = float64(r.Unblocks) * g.examWeight(r.Concept)
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		if a.Concept != b.Concept {
			return a.Concept < b.Concept
		}
		return a.Kind < b.Kind
	})
	return out
}

// betterPath reports whether candidate a should replace b for the same (Kind, Concept).
func betterPath(a, b Recommendation) bool {
	if len(a.Path) != len(b.Path) {
		return len(a.Path) > len(b.Path)
	}
	if a.Target != b.Target {
		return a.Target < b.Target
	}
	return slices.Compare(a.Path, b.Path) < 0
}

// longerOrLexLess orders chains: longer first, then lexicographically smaller.
func longerOrLexLess(a, b []ConceptID) bool {
	if len(a) != len(b) {
		return len(a) > len(b)
	}
	return slices.Compare(a, b) < 0
}

// examWeight returns the Exam Weight of the Concept's Topic, 1 when unknown.
func (g *Graph) examWeight(id ConceptID) float64 {
	c, ok := g.concepts[id]
	if !ok {
		return 1
	}
	t, ok := g.topics[c.Topic]
	if !ok || t.ExamWeight == nil {
		return 1
	}
	return *t.ExamWeight
}

// LearningPathSteps returns the Prerequisite chain from root up to target
// (inclusive both ends), choosing the shortest chain (ties: the chain found by
// a breadth-first walk over Dependents in ID order); ErrUnknownConcept if
// either is missing, or ErrNotPrerequisite if target does not depend on root
// (unless root == target, which yields [root]).
func LearningPathSteps(g *Graph, root, target ConceptID) ([]ConceptID, error) {
	for _, id := range []ConceptID{root, target} {
		if _, ok := g.concepts[id]; !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownConcept, id)
		}
	}
	if root == target {
		return []ConceptID{root}, nil
	}
	parent := map[ConceptID]ConceptID{root: root}
	queue := []ConceptID{root}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, d := range g.dependents[n] {
			if _, seen := parent[d]; seen {
				continue
			}
			parent[d] = n
			if d == target {
				var path []ConceptID
				for c := target; c != root; c = parent[c] {
					path = append(path, c)
				}
				path = append(path, root)
				slices.Reverse(path)
				return path, nil
			}
			queue = append(queue, d)
		}
	}
	return nil, fmt.Errorf("%w: %q does not depend on %q", ErrNotPrerequisite, target, root)
}

// NextPathStep returns the first step that is not yet Developing or Mastered
// (i.e. Weak or Unassessed, including Concepts absent from mastery), and false
// when the Learning Path is complete (or empty).
func NextPathStep(steps []ConceptID, mastery map[ConceptID]Mastery) (ConceptID, bool) {
	for _, s := range steps {
		switch LevelOf(mastery, s).Level {
		case Developing, Mastered:
			continue
		default:
			return s, true
		}
	}
	return "", false
}

// NextPlacementConcept picks the next Concept to probe in an adaptive Placement Test.
// asked holds Concepts already probed (entries set to true). Strategy:
//
//   - stop (false) once len(asked) >= maxProbes;
//   - a Concept is a candidate if it is in g, in `available` (those with
//     Active Questions), not yet asked, still Unassessed in mastery, and not an
//     Ancestor of any Mastered Concept (Mastered implies its Prerequisites are fine);
//   - if some probed Concept is Weak, prefer the candidate direct Prerequisites
//     of Weak probed Concepts;
//   - otherwise (or if none qualify) pick the candidate with the most Ancestors
//     (deep, high-information).
//
// Ties are broken by Concept ID. Returns false when nothing informative remains.
func NextPlacementConcept(g *Graph, mastery map[ConceptID]Mastery, asked map[ConceptID]bool, available map[ConceptID]bool, maxProbes int) (ConceptID, bool) {
	probes := 0
	for _, v := range asked {
		if v {
			probes++
		}
	}
	if probes >= maxProbes {
		return "", false
	}
	level := func(id ConceptID) MasteryLevel { return LevelOf(mastery, id).Level }

	assumedFine := make(map[ConceptID]bool)
	for _, id := range g.order {
		if level(id) == Mastered {
			for _, a := range g.Ancestors(id) {
				assumedFine[a] = true
			}
		}
	}
	eligible := func(id ConceptID) bool {
		l := level(id)
		return available[id] && !asked[id] && !assumedFine[id] && (l == Unassessed || l == "")
	}
	depth := func(id ConceptID) int { return len(g.Ancestors(id)) }
	pick := func(cands []ConceptID) (ConceptID, bool) {
		var best ConceptID
		bestDepth := -1
		for _, c := range cands { // cands in ID order → first max wins ties
			if d := depth(c); d > bestDepth {
				best, bestDepth = c, d
			}
		}
		return best, bestDepth >= 0
	}

	// 1. Direct Prerequisites of Weak probed Concepts.
	seen := make(map[ConceptID]bool)
	var followUps []ConceptID
	for _, id := range g.order {
		if !asked[id] || level(id) != Weak {
			continue
		}
		for _, p := range g.prereqs[id] {
			if !seen[p] && eligible(p) {
				seen[p] = true
				followUps = append(followUps, p)
			}
		}
	}
	sortIDs(followUps)
	if c, ok := pick(followUps); ok {
		return c, true
	}

	// 2. Deepest remaining candidate.
	var all []ConceptID
	for _, id := range g.order {
		if eligible(id) {
			all = append(all, id)
		}
	}
	sortIDs(all)
	return pick(all)
}
