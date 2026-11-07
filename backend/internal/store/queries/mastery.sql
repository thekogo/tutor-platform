-- name: MasteryEvidence :many
-- Counted Answers (not excluded, Question Active) of one Learner in one Course,
-- optionally restricted to Answers whose Question is tagged with any of concept_ids.
SELECT a.id,
       a.question_id,
       a.correct,
       a.answered_at,
       a.is_repeat,
       (ss.kind = 'placement')::boolean AS is_placement,
       q.difficulty,
       pc.concept_id AS primary_concept_id,
       ARRAY(SELECT oc.concept_id FROM question_concepts oc
             WHERE oc.question_id = q.id AND NOT oc.is_primary
             ORDER BY oc.concept_id)::uuid[] AS other_concept_ids
FROM answers a
JOIN study_sessions ss ON ss.id = a.session_id
JOIN questions q ON q.id = a.question_id
JOIN question_concepts pc ON pc.question_id = q.id AND pc.is_primary
WHERE a.learner_id = @learner_id
  AND q.course_id = @course_id
  AND NOT a.excluded
  AND q.status = 'active'
  AND (sqlc.narg(concept_ids)::uuid[] IS NULL OR EXISTS (
        SELECT 1 FROM question_concepts tc
        WHERE tc.question_id = q.id AND tc.concept_id = ANY(sqlc.narg(concept_ids)::uuid[])))
ORDER BY a.answered_at, a.id;

-- name: SessionEvidence :many
-- All Answers in one session (used for Placement-local estimates).
SELECT a.id,
       a.question_id,
       a.correct,
       a.answered_at,
       a.is_repeat,
       q.difficulty,
       pc.concept_id AS primary_concept_id
FROM answers a
JOIN questions q ON q.id = a.question_id
JOIN question_concepts pc ON pc.question_id = q.id AND pc.is_primary
WHERE a.session_id = $1
ORDER BY a.answered_at, a.id;

-- name: ListMasteryForCourse :many
SELECT m.*
FROM mastery m JOIN concepts c ON c.id = m.concept_id
WHERE m.learner_id = $1 AND c.course_id = $2;

-- name: UpsertMastery :exec
INSERT INTO mastery (learner_id, concept_id, score, evidence, level, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (learner_id, concept_id) DO UPDATE
    SET score = EXCLUDED.score, evidence = EXCLUDED.evidence, level = EXCLUDED.level, updated_at = now();

-- name: DeleteMastery :exec
DELETE FROM mastery WHERE learner_id = $1 AND concept_id = ANY(@concept_ids::uuid[]);

-- name: DeleteMasteryForCourse :exec
DELETE FROM mastery m USING concepts c
WHERE m.concept_id = c.id AND m.learner_id = $1 AND c.course_id = $2;

-- name: ListLearnerCoursePairs :many
-- Every (Learner, Course) with at least one Answer, for full Mastery rebuilds.
SELECT DISTINCT a.learner_id, q.course_id
FROM answers a JOIN questions q ON q.id = a.question_id
ORDER BY a.learner_id, q.course_id;

-- name: MasteryOverview :many
SELECT m.concept_id, m.level, count(*)::bigint AS learners
FROM mastery m JOIN concepts c ON c.id = m.concept_id
WHERE c.course_id = $1
GROUP BY m.concept_id, m.level;

-- name: GetActivePath :one
SELECT * FROM learning_paths
WHERE learner_id = $1 AND course_id = $2 AND abandoned_at IS NULL
ORDER BY started_at DESC, id DESC
LIMIT 1;

-- name: AbandonActivePaths :exec
UPDATE learning_paths SET abandoned_at = now()
WHERE learner_id = $1 AND course_id = $2 AND abandoned_at IS NULL AND completed_at IS NULL;

-- name: AbandonAllPaths :exec
UPDATE learning_paths SET abandoned_at = now()
WHERE learner_id = $1 AND course_id = $2 AND abandoned_at IS NULL;

-- name: CreateLearningPath :one
INSERT INTO learning_paths (learner_id, course_id, root_concept_id, target_concept_id, path)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CompleteLearningPath :exec
UPDATE learning_paths SET completed_at = now() WHERE id = $1 AND completed_at IS NULL;

-- name: CreateFlag :one
INSERT INTO flags (question_id, learner_id, reason) VALUES ($1, $2, $3) RETURNING *;

-- name: ListFlags :many
SELECT f.*, l.name AS learner_name, q.course_id, q.status AS question_status
FROM flags f
JOIN learners l ON l.id = f.learner_id
JOIN questions q ON q.id = f.question_id
WHERE (NOT @open_only::boolean OR f.resolved_at IS NULL)
ORDER BY f.created_at DESC, f.id;

-- name: GetFlag :one
SELECT * FROM flags WHERE id = $1;

-- name: ResolveFlagsForQuestion :execrows
UPDATE flags SET resolved_at = now(), resolution_note = $2
WHERE question_id = $1 AND resolved_at IS NULL;

-- name: RaiseContentRequest :execrows
INSERT INTO content_requests (concept_id, reason) VALUES ($1, $2)
ON CONFLICT (concept_id) WHERE fulfilled_at IS NULL DO NOTHING;

-- name: ListContentRequests :many
SELECT r.*, c.name AS concept_name, c.slug AS concept_slug, c.course_id
FROM content_requests r JOIN concepts c ON c.id = r.concept_id
WHERE (NOT @open_only::boolean OR r.fulfilled_at IS NULL)
ORDER BY r.created_at, r.id;

-- name: FulfillContentRequest :execrows
UPDATE content_requests SET fulfilled_at = now(), note = $2 WHERE id = $1 AND fulfilled_at IS NULL;
