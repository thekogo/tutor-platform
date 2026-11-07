package domain_test

import (
	"fmt"
	"time"

	"github.com/thekogo/tutor-platform/backend/internal/domain"
)

func ExampleRecommend() {
	g, err := domain.NewGraph(
		[]domain.Topic{{ID: "ratio", Name: "อัตราส่วน"}},
		[]domain.Concept{
			{ID: "fractions", Topic: "ratio"},
			{ID: "ratio-basics", Topic: "ratio"},
			{ID: "proportion", Topic: "ratio"},
		},
		[]domain.Prerequisite{
			{Concept: "ratio-basics", Prerequisite: "fractions"},
			{Concept: "proportion", Prerequisite: "ratio-basics"},
		},
	)
	if err != nil {
		panic(err)
	}
	mastery := map[domain.ConceptID]domain.Mastery{
		"fractions":    {Concept: "fractions", Score: 0.9, Evidence: 5, Level: domain.Mastered},
		"ratio-basics": {Concept: "ratio-basics", Score: 0.3, Evidence: 4, Level: domain.Weak},
		"proportion":   {Concept: "proportion", Score: 0.2, Evidence: 6, Level: domain.Weak},
	}
	for _, r := range domain.Recommend(g, mastery) {
		fmt.Printf("%s %s for %s via %v (unblocks %d)\n", r.Kind, r.Concept, r.Target, r.Path, r.Unblocks)
	}
	// Output:
	// study ratio-basics for proportion via [ratio-basics proportion] (unblocks 2)
}

func ExampleRecommend_diagnose() {
	g, _ := domain.NewGraph(
		[]domain.Topic{{ID: "t"}},
		[]domain.Concept{{ID: "percent-of", Topic: "t"}, {ID: "percent-change", Topic: "t"}},
		[]domain.Prerequisite{{Concept: "percent-change", Prerequisite: "percent-of"}},
	)
	// percent-of has no Answers yet: Unassessed means unknown, never weak.
	mastery := map[domain.ConceptID]domain.Mastery{
		"percent-change": {Concept: "percent-change", Score: 0.25, Evidence: 4, Level: domain.Weak},
	}
	r := domain.Recommend(g, mastery)[0]
	fmt.Println(r.Kind, r.Concept, r.Path)
	// Output:
	// diagnose percent-of [percent-of percent-change]
}

func ExampleComputeMastery() {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	evidence := []domain.AnswerEvidence{
		{QuestionID: "q1", Correct: false, AnsweredAt: t0, Difficulty: 2, PrimaryConcept: "percent-of"},
		{QuestionID: "q2", Correct: true, AnsweredAt: t0.Add(time.Hour), Difficulty: 2, PrimaryConcept: "percent-of"},
		{QuestionID: "q3", Correct: true, AnsweredAt: t0.Add(2 * time.Hour), Difficulty: 3, PrimaryConcept: "percent-of",
			OtherConcepts: []domain.ConceptID{"proportion"}},
	}
	m := domain.ComputeMastery(evidence, domain.DefaultMasteryConfig())
	for _, id := range []domain.ConceptID{"percent-of", "proportion"} {
		fmt.Printf("%s: score=%.2f evidence=%.3f level=%s\n", id, m[id].Score, m[id].Evidence, m[id].Level)
	}
	// Output:
	// percent-of: score=0.74 evidence=3.250 level=developing
	// proportion: score=1.00 evidence=0.625 level=unassessed
}

func ExampleLearningPathSteps() {
	g, _ := domain.NewGraph(
		[]domain.Topic{{ID: "t"}},
		[]domain.Concept{{ID: "a", Topic: "t"}, {ID: "b", Topic: "t"}, {ID: "c", Topic: "t"}},
		[]domain.Prerequisite{{Concept: "b", Prerequisite: "a"}, {Concept: "c", Prerequisite: "b"}},
	)
	steps, _ := domain.LearningPathSteps(g, "a", "c")
	mastery := map[domain.ConceptID]domain.Mastery{"a": {Concept: "a", Level: domain.Developing}}
	next, _ := domain.NextPathStep(steps, mastery)
	fmt.Println(steps, next)
	// Output:
	// [a b c] b
}
