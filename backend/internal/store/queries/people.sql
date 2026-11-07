-- name: GetLearner :one
SELECT * FROM learners WHERE id = $1;

-- name: GetLearnerByEmail :one
SELECT * FROM learners WHERE email = $1;

-- name: CountLearners :one
SELECT count(*) FROM learners;

-- name: CreateLearner :one
INSERT INTO learners (email, name, is_owner) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateLearnerName :exec
UPDATE learners SET name = $2 WHERE id = $1;

-- name: ListLearners :many
SELECT * FROM learners ORDER BY created_at, id;

-- name: LearnerPlacementDone :one
SELECT EXISTS (
    SELECT 1 FROM study_sessions
    WHERE learner_id = $1 AND kind = 'placement' AND finished_at IS NOT NULL
)::boolean AS done;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, learner_id, expires_at) VALUES ($1, $2, $3);

-- name: GetSessionLearner :one
SELECT l.*
FROM sessions s JOIN learners l ON l.id = s.learner_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at <= now();

-- name: CreateInvite :one
INSERT INTO invites (token_hash, created_by, expires_at) VALUES ($1, $2, $3) RETURNING *;

-- name: ListInvites :many
SELECT i.*, u.email AS used_by_email
FROM invites i LEFT JOIN learners u ON u.id = i.used_by
ORDER BY i.created_at DESC, i.id;

-- name: GetInviteByHashForUpdate :one
SELECT * FROM invites WHERE token_hash = $1 FOR UPDATE;

-- name: UseInvite :exec
UPDATE invites SET used_by = $2, used_at = now() WHERE id = $1 AND used_by IS NULL;
