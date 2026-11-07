-- name: UpsertCourse :one
INSERT INTO courses (slug, name, goal) VALUES ($1, $2, $3)
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, goal = EXCLUDED.goal
RETURNING *;

-- name: ListCourses :many
SELECT * FROM courses ORDER BY name, id;

-- name: GetCourse :one
SELECT * FROM courses WHERE id = $1;

-- name: GetCourseBySlug :one
SELECT * FROM courses WHERE slug = $1;

-- name: UpsertQuestionFormat :exec
INSERT INTO question_formats (course_id, slug, name, position) VALUES ($1, $2, $3, $4)
ON CONFLICT (course_id, slug) DO UPDATE SET name = EXCLUDED.name, position = EXCLUDED.position;

-- name: ListQuestionFormats :many
SELECT * FROM question_formats WHERE course_id = $1 ORDER BY position, slug;

-- name: UpsertTopic :one
INSERT INTO topics (course_id, slug, name, exam_weight, position) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (course_id, slug) DO UPDATE
    SET name = EXCLUDED.name, exam_weight = EXCLUDED.exam_weight, position = EXCLUDED.position
RETURNING *;

-- name: NextTopicPosition :one
SELECT (coalesce(max(position), 0) + 1)::integer FROM topics WHERE course_id = $1;

-- name: ListTopics :many
SELECT * FROM topics WHERE course_id = $1 ORDER BY position, slug;

-- name: UpsertConcept :one
INSERT INTO concepts (course_id, topic_id, slug, name, position) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (course_id, slug) DO UPDATE
    SET topic_id = EXCLUDED.topic_id, name = EXCLUDED.name, position = EXCLUDED.position, archived_at = NULL
RETURNING *;

-- name: NextConceptPosition :one
SELECT (coalesce(max(position), 0) + 1)::integer FROM concepts WHERE course_id = $1;

-- name: ListConcepts :many
SELECT * FROM concepts WHERE course_id = $1 ORDER BY position, slug;

-- name: GetConcept :one
SELECT * FROM concepts WHERE id = $1;

-- name: ArchiveConcept :exec
UPDATE concepts SET archived_at = now() WHERE id = $1 AND archived_at IS NULL;

-- name: ListPrerequisites :many
SELECT p.concept_id, p.prerequisite_id
FROM prerequisites p JOIN concepts c ON c.id = p.concept_id
WHERE c.course_id = $1
ORDER BY p.concept_id, p.prerequisite_id;

-- name: DeletePrerequisitesOf :exec
DELETE FROM prerequisites WHERE concept_id = $1;

-- name: InsertPrerequisite :exec
INSERT INTO prerequisites (concept_id, prerequisite_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UpsertLesson :exec
INSERT INTO lessons (concept_id, body_md, updated_at) VALUES ($1, $2, now())
ON CONFLICT (concept_id) DO UPDATE SET body_md = EXCLUDED.body_md, updated_at = now();

-- name: GetLesson :one
SELECT * FROM lessons WHERE concept_id = $1;

-- name: ListLessonConceptIDs :many
SELECT l.concept_id FROM lessons l JOIN concepts c ON c.id = l.concept_id WHERE c.course_id = $1;
