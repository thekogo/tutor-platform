-- Course Access (the Owner implicitly has every Course; see app/access.go).

-- name: HasCourseAccess :one
SELECT EXISTS (
    SELECT 1 FROM course_access WHERE learner_id = $1 AND course_id = $2
)::boolean AS ok;

-- name: ListAccessibleCourses :many
SELECT c.* FROM courses c
JOIN course_access a ON a.course_id = c.id
WHERE a.learner_id = $1
ORDER BY c.name, c.id;

-- name: ListCourseAccess :many
SELECT learner_id, course_id FROM course_access ORDER BY learner_id, course_id;

-- name: DeleteLearnerCourseAccess :exec
DELETE FROM course_access WHERE learner_id = $1;

-- name: GrantCourseAccess :exec
INSERT INTO course_access (learner_id, course_id)
SELECT $1, unnest(@course_ids::uuid[])
ON CONFLICT DO NOTHING;

-- name: GrantAllCourses :exec
INSERT INTO course_access (learner_id, course_id)
SELECT $1, c.id FROM courses c
ON CONFLICT DO NOTHING;

-- name: CountCoursesByIDs :one
SELECT count(*) FROM courses WHERE id = ANY(@course_ids::uuid[]);

-- name: AddInviteCourses :exec
INSERT INTO invite_courses (invite_id, course_id)
SELECT $1, unnest(@course_ids::uuid[])
ON CONFLICT DO NOTHING;

-- name: ListInviteCourses :many
SELECT invite_id, course_id FROM invite_courses ORDER BY invite_id, course_id;

-- name: GrantInviteCourses :exec
INSERT INTO course_access (learner_id, course_id)
SELECT $1, ic.course_id FROM invite_courses ic WHERE ic.invite_id = $2
ON CONFLICT DO NOTHING;
