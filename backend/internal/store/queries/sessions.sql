-- name: CreateExam :one
INSERT INTO exams (course_id, name, time_limit_s) VALUES ($1, $2, $3) RETURNING *;

-- name: AddExamQuestion :exec
INSERT INTO exam_questions (exam_id, question_id, position) VALUES ($1, $2, $3);

-- name: GetExam :one
SELECT * FROM exams WHERE id = $1;

-- name: ListExams :many
SELECT e.*,
       (SELECT count(*) FROM exam_questions eq JOIN questions q ON q.id = eq.question_id
        WHERE eq.exam_id = e.id AND q.status = 'active')::bigint AS question_count
FROM exams e WHERE e.course_id = $1
ORDER BY e.created_at, e.id;

-- name: ListActiveExamQuestions :many
SELECT eq.question_id
FROM exam_questions eq JOIN questions q ON q.id = eq.question_id
WHERE eq.exam_id = $1 AND q.status = 'active'
ORDER BY eq.position;

-- name: LastExamResults :many
-- Most recent finished session per Exam for a Learner, with its score.
SELECT DISTINCT ON (s.exam_id)
       s.exam_id::uuid AS exam_id,
       (SELECT count(*) FROM session_questions sq WHERE sq.session_id = s.id)::bigint AS total,
       (SELECT count(*) FROM answers a WHERE a.session_id = s.id AND a.correct)::bigint AS correct
FROM study_sessions s
WHERE s.learner_id = $1 AND s.course_id = $2 AND s.kind = 'exam' AND s.finished_at IS NOT NULL
ORDER BY s.exam_id, s.finished_at DESC;

-- name: CreateStudySession :one
INSERT INTO study_sessions (learner_id, course_id, kind, concept_id, topic_id, exam_id, max_questions, deadline)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetStudySession :one
SELECT * FROM study_sessions WHERE id = $1;

-- name: GetStudySessionForUpdate :one
SELECT * FROM study_sessions WHERE id = $1 FOR UPDATE;

-- name: FinishStudySession :one
UPDATE study_sessions SET finished_at = coalesce(finished_at, now()) WHERE id = $1 RETURNING *;

-- name: AddSessionQuestion :exec
INSERT INTO session_questions (session_id, question_id, position, concept_id, is_repeat)
VALUES ($1, $2, $3, $4, $5);

-- name: ListSessionQuestions :many
SELECT sq.session_id, sq.question_id, sq.position, sq.concept_id, sq.is_repeat,
       a.id AS answer_id, a.choice_id, a.correct, a.answered_at
FROM session_questions sq
LEFT JOIN answers a ON a.session_id = sq.session_id AND a.question_id = sq.question_id
WHERE sq.session_id = $1
ORDER BY sq.position;

-- name: CountSessionQuestions :one
SELECT count(*) FROM session_questions WHERE session_id = $1;

-- name: CountSessionAnswers :one
SELECT count(*) FROM answers WHERE session_id = $1;

-- name: InsertAnswer :one
INSERT INTO answers (learner_id, question_id, session_id, choice_id, correct, is_repeat, time_spent_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: HasAnswered :one
SELECT EXISTS (SELECT 1 FROM answers WHERE learner_id = $1 AND question_id = $2)::boolean;

-- name: ListLearnersWhoAnswered :many
SELECT DISTINCT learner_id FROM answers WHERE question_id = $1;

-- name: ExcludeAnswersForQuestion :execrows
UPDATE answers SET excluded = true WHERE question_id = $1 AND NOT excluded;

-- name: FormatReport :many
SELECT f.slug, f.name,
       count(a.id)::bigint AS answered,
       count(a.id) FILTER (WHERE a.correct)::bigint AS correct
FROM question_formats f
LEFT JOIN questions q ON q.course_id = f.course_id AND q.format = f.slug
LEFT JOIN answers a ON a.question_id = q.id AND a.learner_id = @learner_id AND NOT a.excluded
WHERE f.course_id = @course_id
GROUP BY f.slug, f.name, f.position
ORDER BY f.position, f.slug;
