package app

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Question serving rules (docs/architecture.md):
//   - only Active Questions are served;
//   - Exam Questions never appear in Quizzes (or Diagnostic/Placement);
//   - Questions the Learner has not answered are preferred;
//   - when a Concept's pool is exhausted, repeats are served (flagged is_repeat)
//     and a Content Request is raised for the Concept (one open request per Concept).

type pickedQuestion struct {
	ID       uuid.UUID
	IsRepeat bool
}

// pickQuestions selects up to n Questions tagged with any of concepts within
// the difficulty range. It returns the picks and the number of unanswered
// candidates that were available.
func pickQuestions(ctx context.Context, q *store.Queries, learnerID uuid.UUID, concepts []uuid.UUID, minD, maxD int16, n int) ([]pickedQuestion, int, error) {
	cands, err := q.ServingCandidates(ctx, store.ServingCandidatesParams{
		ConceptIds: concepts, LearnerID: learnerID, MinDifficulty: minD, MaxDifficulty: maxD,
	})
	if err != nil {
		return nil, 0, err
	}
	var fresh, seen []store.ServingCandidatesRow
	for _, c := range cands {
		if c.TimesAnswered == 0 {
			fresh = append(fresh, c)
		} else {
			seen = append(seen, c)
		}
	}
	// Fresh: random order, Questions whose Primary Concept is requested first.
	//nolint:gosec // non-security shuffle of serving order
	rand.Shuffle(len(fresh), func(i, j int) { fresh[i], fresh[j] = fresh[j], fresh[i] })
	sort.SliceStable(fresh, func(i, j int) bool { return fresh[i].IsPrimary && !fresh[j].IsPrimary })
	// Repeats: least recently answered first, then least often answered.
	sort.SliceStable(seen, func(i, j int) bool {
		if !seen[i].LastAnsweredAt.Equal(seen[j].LastAnsweredAt) {
			return seen[i].LastAnsweredAt.Before(seen[j].LastAnsweredAt)
		}
		return seen[i].TimesAnswered < seen[j].TimesAnswered
	})
	var out []pickedQuestion
	for _, c := range fresh {
		if len(out) == n {
			break
		}
		out = append(out, pickedQuestion{ID: c.ID})
	}
	for _, c := range seen {
		if len(out) == n {
			break
		}
		out = append(out, pickedQuestion{ID: c.ID, IsRepeat: true})
	}
	return out, len(fresh), nil
}

// raiseExhausted raises Content Requests for Concepts whose pool could not
// supply n fresh Questions to this Learner. Duplicates (an open request for
// the same Concept) are silently ignored.
func raiseExhausted(ctx context.Context, q *store.Queries, learnerID uuid.UUID, concepts []uuid.UUID, minD, maxD int16, n int, what string) error {
	for _, c := range concepts {
		_, fresh, err := pickQuestions(ctx, q, learnerID, []uuid.UUID{c}, minD, maxD, 0)
		if err != nil {
			return err
		}
		need := n
		if len(concepts) > 1 {
			need = 1 // Topic Quiz: only Concepts with no fresh Questions left are exhausted
		}
		if fresh >= need {
			continue
		}
		reason := fmt.Sprintf("Question pool exhausted: a Learner had %d unanswered Active Questions (difficulty %d-%d) for a %s needing %d",
			fresh, minD, maxD, what, n)
		if _, err := q.RaiseContentRequest(ctx, store.RaiseContentRequestParams{ConceptID: c, Reason: reason}); err != nil {
			return err
		}
	}
	return nil
}
