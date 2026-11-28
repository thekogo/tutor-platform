-- +goose Up
-- Course Access: which Courses a Learner may study. The Owner implicitly has
-- access to every Course and never has rows here.
CREATE TABLE course_access (
    learner_id uuid        NOT NULL REFERENCES learners (id) ON DELETE CASCADE,
    course_id  uuid        NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    granted_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (learner_id, course_id)
);
CREATE INDEX course_access_course ON course_access (course_id);

-- The Courses an invite grants when it is used.
CREATE TABLE invite_courses (
    invite_id uuid NOT NULL REFERENCES invites (id) ON DELETE CASCADE,
    course_id uuid NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    PRIMARY KEY (invite_id, course_id)
);

-- Before this migration every Learner could study every Course; keep that.
INSERT INTO course_access (learner_id, course_id)
SELECT l.id, c.id FROM learners l CROSS JOIN courses c WHERE NOT l.is_owner;

-- +goose Down
DROP TABLE invite_courses, course_access;
