package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/domain"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Mastery is a projection of the Answer history (ADR 0002): these functions
// recompute it from Answers and upsert the `mastery` cache table.

func toEvidence(rows []store.MasteryEvidenceRow) []domain.AnswerEvidence {
	ev := make([]domain.AnswerEvidence, 0, len(rows))
	for _, r := range rows {
		others := make([]domain.ConceptID, 0, len(r.OtherConceptIds))
		for _, o := range r.OtherConceptIds {
			others = append(others, cid(o))
		}
		ev = append(ev, domain.AnswerEvidence{
			QuestionID:     r.QuestionID.String(),
			Correct:        r.Correct,
			AnsweredAt:     r.AnsweredAt,
			Difficulty:     int(r.Difficulty),
			PrimaryConcept: cid(r.PrimaryConceptID),
			OtherConcepts:  others,
			IsRepeat:       r.IsRepeat,
		})
	}
	return ev
}

// masteryMap reads the cached Mastery of a Learner in a Course.
func masteryMap(ctx context.Context, q *store.Queries, learnerID, courseID uuid.UUID) (map[domain.ConceptID]domain.Mastery, error) {
	rows, err := q.ListMasteryForCourse(ctx, store.ListMasteryForCourseParams{LearnerID: learnerID, CourseID: courseID})
	if err != nil {
		return nil, err
	}
	m := make(map[domain.ConceptID]domain.Mastery, len(rows))
	for _, r := range rows {
		m[cid(r.ConceptID)] = domain.Mastery{
			Concept: cid(r.ConceptID), Score: r.Score, Evidence: r.Evidence, Level: domain.MasteryLevel(r.Level),
		}
	}
	return m, nil
}

// MasteryChange is a Concept whose Mastery Level changed after a recompute.
type MasteryChange struct {
	ConceptID uuid.UUID
	Before    domain.MasteryLevel
	After     domain.MasteryLevel
}

// recomputeMastery recomputes a Learner's Mastery for the given Concepts of a
// Course (all Concepts when concepts is nil) and returns level changes.
func (s *Service) recomputeMastery(ctx context.Context, q *store.Queries, learnerID, courseID uuid.UUID, concepts []uuid.UUID) ([]MasteryChange, error) {
	before, err := masteryMap(ctx, q, learnerID, courseID)
	if err != nil {
		return nil, err
	}
	params := store.MasteryEvidenceParams{LearnerID: learnerID, CourseID: courseID}
	if concepts != nil {
		if len(concepts) == 0 {
			return nil, nil
		}
		params.ConceptIds = concepts
	}
	rows, err := q.MasteryEvidence(ctx, params)
	if err != nil {
		return nil, err
	}
	computed := domain.ComputeMastery(toEvidence(rows), s.masteryCfg)

	targets := concepts
	if targets == nil {
		set := map[uuid.UUID]bool{}
		for c := range before {
			set[ucid(c)] = true
		}
		for c := range computed {
			set[ucid(c)] = true
		}
		for c := range set {
			targets = append(targets, c)
		}
	}
	var changes []MasteryChange
	var drop []uuid.UUID
	for _, c := range targets {
		old := domain.LevelOf(before, cid(c))
		m, ok := computed[cid(c)]
		if !ok {
			drop = append(drop, c)
			m = domain.LevelOf(computed, cid(c))
		} else {
			score := min(max(m.Score, 0), 1)
			if err := q.UpsertMastery(ctx, store.UpsertMasteryParams{
				LearnerID: learnerID, ConceptID: c, Score: score, Evidence: m.Evidence, Level: string(m.Level),
			}); err != nil {
				return nil, fmt.Errorf("upsert mastery: %w", err)
			}
		}
		if old.Level != m.Level {
			changes = append(changes, MasteryChange{ConceptID: c, Before: old.Level, After: m.Level})
		}
	}
	if len(drop) > 0 {
		if err := q.DeleteMastery(ctx, store.DeleteMasteryParams{LearnerID: learnerID, ConceptIds: drop}); err != nil {
			return nil, err
		}
	}
	return changes, nil
}

// recomputeForQuestion recomputes Mastery of every Learner who answered a
// Question, for the Question's Concepts plus any extra Concepts (e.g. old tags).
func (s *Service) recomputeForQuestion(ctx context.Context, q *store.Queries, question store.Question, extra []uuid.UUID) error {
	tags, err := q.ListConceptsForQuestions(ctx, []uuid.UUID{question.ID})
	if err != nil {
		return err
	}
	set := map[uuid.UUID]bool{}
	for _, t := range tags {
		set[t.ConceptID] = true
	}
	for _, c := range extra {
		set[c] = true
	}
	concepts := make([]uuid.UUID, 0, len(set))
	for c := range set {
		concepts = append(concepts, c)
	}
	learners, err := q.ListLearnersWhoAnswered(ctx, question.ID)
	if err != nil {
		return err
	}
	for _, l := range learners {
		if _, err := s.recomputeMastery(ctx, q, l, question.CourseID, concepts); err != nil {
			return err
		}
	}
	return nil
}

// RebuildAllMastery recomputes every Learner's Mastery in every Course from
// scratch (used after graph edits or Mastery calculation changes).
func (s *Service) RebuildAllMastery(ctx context.Context) (int, error) {
	pairs, err := s.q.ListLearnerCoursePairs(ctx)
	if err != nil {
		return 0, err
	}
	for _, p := range pairs {
		err := s.tx(ctx, func(q *store.Queries) error {
			if err := q.DeleteMasteryForCourse(ctx, store.DeleteMasteryForCourseParams(p)); err != nil {
				return err
			}
			_, err := s.recomputeMastery(ctx, q, p.LearnerID, p.CourseID, nil)
			return err
		})
		if err != nil {
			return 0, fmt.Errorf("rebuild mastery for learner %s course %s: %w", p.LearnerID, p.CourseID, err)
		}
	}
	return len(pairs), nil
}
