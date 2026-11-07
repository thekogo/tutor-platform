-- name: CreateQuestion :one
INSERT INTO questions (course_id, stem_md, explanation_md, difficulty, format, source, source_ref)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CreateChoice :one
INSERT INTO question_choices (question_id, label, body_md, is_correct, position)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateChoice :exec
UPDATE question_choices SET body_md = $2, is_correct = $3, position = $4 WHERE id = $1;

-- name: ClearCorrectChoices :exec
UPDATE question_choices SET is_correct = false WHERE question_id = $1 AND is_correct;

-- name: DeleteChoice :exec
DELETE FROM question_choices WHERE id = $1;

-- name: ChoiceHasAnswers :one
SELECT EXISTS (SELECT 1 FROM answers WHERE choice_id = $1)::boolean;

-- name: AddQuestionConcept :exec
INSERT INTO question_concepts (question_id, concept_id, is_primary) VALUES ($1, $2, $3);

-- name: DeleteQuestionConcepts :exec
DELETE FROM question_concepts WHERE question_id = $1;

-- name: GetQuestion :one
SELECT * FROM questions WHERE id = $1;

-- name: GetQuestionForUpdate :one
SELECT * FROM questions WHERE id = $1 FOR UPDATE;

-- name: ListChoicesForQuestions :many
SELECT * FROM question_choices WHERE question_id = ANY(@question_ids::uuid[])
ORDER BY question_id, position, label;

-- name: ListConceptsForQuestions :many
SELECT * FROM question_concepts WHERE question_id = ANY(@question_ids::uuid[])
ORDER BY question_id, is_primary DESC, concept_id;

-- name: ListQuestionsByIDs :many
SELECT * FROM questions WHERE id = ANY(@ids::uuid[]);

-- name: SetQuestionStatus :exec
UPDATE questions SET status = $2, verification_note = $3, updated_at = now() WHERE id = $1;

-- name: UpdateQuestionContent :exec
UPDATE questions
SET stem_md = $2, explanation_md = $3, difficulty = $4, format = $5, source = $6, source_ref = $7,
    status = $8, verification_note = $9, updated_at = now()
WHERE id = $1;

-- name: ListDraftQuestions :many
SELECT * FROM questions
WHERE course_id = $1 AND status = 'draft'
ORDER BY created_at, id
LIMIT $2;

-- name: IsExamQuestion :one
SELECT EXISTS (SELECT 1 FROM exam_questions WHERE question_id = $1)::boolean;

-- name: ServingCandidates :many
-- Active, non-Exam Questions tagged with any of the given Concepts, with the
-- Learner's history on each. Ordering/selection happens in the app layer.
SELECT q.id,
       q.difficulty,
       bool_or(qc.is_primary)::boolean           AS is_primary,
       coalesce(h.times_answered, 0)::bigint     AS times_answered,
       coalesce(h.last_answered_at, 'epoch'::timestamptz)::timestamptz AS last_answered_at
FROM questions q
JOIN question_concepts qc ON qc.question_id = q.id AND qc.concept_id = ANY(@concept_ids::uuid[])
LEFT JOIN (
    SELECT a.question_id, count(*) AS times_answered, max(a.answered_at) AS last_answered_at
    FROM answers a WHERE a.learner_id = @learner_id
    GROUP BY a.question_id
) h ON h.question_id = q.id
WHERE q.status = 'active'
  AND q.difficulty BETWEEN @min_difficulty::smallint AND @max_difficulty::smallint
  AND NOT EXISTS (SELECT 1 FROM exam_questions eq WHERE eq.question_id = q.id)
GROUP BY q.id, q.difficulty, h.times_answered, h.last_answered_at;

-- name: PrimaryCandidates :many
-- Active, non-Exam Questions whose Primary Concept is the given Concept (Placement probes).
SELECT q.id,
       q.difficulty,
       coalesce(h.times_answered, 0)::bigint AS times_answered
FROM questions q
JOIN question_concepts qc ON qc.question_id = q.id AND qc.is_primary AND qc.concept_id = @concept_id
LEFT JOIN (
    SELECT a.question_id, count(*) AS times_answered
    FROM answers a WHERE a.learner_id = @learner_id
    GROUP BY a.question_id
) h ON h.question_id = q.id
WHERE q.status = 'active'
  AND NOT EXISTS (SELECT 1 FROM exam_questions eq WHERE eq.question_id = q.id);

-- name: ConceptsWithPrimaryQuestions :many
SELECT DISTINCT qc.concept_id
FROM question_concepts qc JOIN questions q ON q.id = qc.question_id
WHERE q.course_id = $1 AND q.status = 'active' AND qc.is_primary
  AND NOT EXISTS (SELECT 1 FROM exam_questions eq WHERE eq.question_id = q.id);

-- name: CountServableQuestionsForConcept :one
SELECT count(DISTINCT q.id)
FROM questions q JOIN question_concepts qc ON qc.question_id = q.id
WHERE qc.concept_id = $1 AND q.status = 'active'
  AND NOT EXISTS (SELECT 1 FROM exam_questions eq WHERE eq.question_id = q.id);

-- name: PoolStats :many
SELECT qc.concept_id,
       count(*) FILTER (WHERE q.status = 'draft')::bigint   AS draft,
       count(*) FILTER (WHERE q.status = 'active')::bigint  AS active,
       count(*) FILTER (WHERE q.status = 'flagged')::bigint AS flagged,
       count(*) FILTER (WHERE q.status = 'retired')::bigint AS retired,
       count(*) FILTER (WHERE q.status = 'active' AND qc.is_primary)::bigint AS active_primary,
       count(*) FILTER (WHERE q.status = 'active' AND q.difficulty = 1)::bigint AS active_d1,
       count(*) FILTER (WHERE q.status = 'active' AND q.difficulty = 2)::bigint AS active_d2,
       count(*) FILTER (WHERE q.status = 'active' AND q.difficulty = 3)::bigint AS active_d3,
       count(*) FILTER (WHERE q.status = 'active'
           AND EXISTS (SELECT 1 FROM exam_questions eq WHERE eq.question_id = q.id))::bigint AS active_in_exams
FROM question_concepts qc JOIN questions q ON q.id = qc.question_id
WHERE q.course_id = $1
GROUP BY qc.concept_id;
