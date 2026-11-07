-- +goose Up
-- Data model: see docs/architecture.md. Vocabulary follows CONTEXT.md.

-- ── People ──────────────────────────────────────────────────────────────
CREATE TABLE learners (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email      text        NOT NULL UNIQUE CHECK (email = lower(email) AND email <> ''),
    name       text        NOT NULL DEFAULT '',
    is_owner   boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
-- At most one Owner (guards the "first-ever login becomes Owner" race).
CREATE UNIQUE INDEX learners_single_owner ON learners ((true)) WHERE is_owner;

CREATE TABLE invites (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash bytea       NOT NULL UNIQUE,
    created_by uuid        NOT NULL REFERENCES learners (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_by    uuid REFERENCES learners (id),
    used_at    timestamptz,
    CHECK ((used_by IS NULL) = (used_at IS NULL))
);

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    learner_id uuid        NOT NULL REFERENCES learners (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_learner ON sessions (learner_id);
CREATE INDEX sessions_expires ON sessions (expires_at);

-- ── Course structure ────────────────────────────────────────────────────
CREATE TABLE courses (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       text        NOT NULL UNIQUE CHECK (slug <> ''),
    name       text        NOT NULL CHECK (name <> ''),
    goal       text        NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE question_formats (
    course_id uuid NOT NULL REFERENCES courses (id),
    slug      text NOT NULL CHECK (slug <> ''),
    name      text NOT NULL CHECK (name <> ''),
    position  integer NOT NULL DEFAULT 0,
    PRIMARY KEY (course_id, slug)
);

CREATE TABLE topics (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   uuid    NOT NULL REFERENCES courses (id),
    slug        text    NOT NULL CHECK (slug <> ''),
    name        text    NOT NULL CHECK (name <> ''),
    exam_weight double precision CHECK (exam_weight IS NULL OR exam_weight >= 0),
    position    integer NOT NULL DEFAULT 0,
    UNIQUE (course_id, slug),
    UNIQUE (id, course_id)
);

CREATE TABLE concepts (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   uuid    NOT NULL REFERENCES courses (id),
    topic_id    uuid    NOT NULL,
    slug        text    NOT NULL CHECK (slug <> ''),
    name        text    NOT NULL CHECK (name <> ''),
    position    integer NOT NULL DEFAULT 0,
    archived_at timestamptz,
    UNIQUE (course_id, slug),
    -- a Concept's Topic belongs to the same Course
    FOREIGN KEY (topic_id, course_id) REFERENCES topics (id, course_id)
);
CREATE INDEX concepts_topic ON concepts (topic_id);

CREATE TABLE prerequisites (
    concept_id      uuid NOT NULL REFERENCES concepts (id),
    prerequisite_id uuid NOT NULL REFERENCES concepts (id),
    PRIMARY KEY (concept_id, prerequisite_id),
    CHECK (concept_id <> prerequisite_id)
);
CREATE INDEX prerequisites_prerequisite ON prerequisites (prerequisite_id);

CREATE TABLE lessons (
    concept_id uuid PRIMARY KEY REFERENCES concepts (id),
    body_md    text        NOT NULL CHECK (btrim(body_md) <> ''),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- ── Assessment ──────────────────────────────────────────────────────────
CREATE TABLE questions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id         uuid        NOT NULL REFERENCES courses (id),
    stem_md           text        NOT NULL CHECK (btrim(stem_md) <> ''),
    explanation_md    text        NOT NULL CHECK (btrim(explanation_md) <> ''),
    difficulty        smallint    NOT NULL CHECK (difficulty BETWEEN 1 AND 3),
    format            text        NOT NULL,
    source            text        NOT NULL CHECK (source IN ('ai_generated', 'imported')),
    source_ref        text,
    status            text        NOT NULL DEFAULT 'draft'
                                  CHECK (status IN ('draft', 'active', 'flagged', 'retired')),
    verification_note text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (course_id, format) REFERENCES question_formats (course_id, slug)
);
CREATE INDEX questions_course_status ON questions (course_id, status);
CREATE INDEX questions_status ON questions (status);

CREATE TABLE question_choices (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id uuid    NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    label       text    NOT NULL CHECK (label <> ''),
    body_md     text    NOT NULL CHECK (btrim(body_md) <> ''),
    is_correct  boolean NOT NULL DEFAULT false,
    position    integer NOT NULL DEFAULT 0,
    UNIQUE (question_id, label)
);
-- At most one correct choice per Question; "at least one" is enforced by the
-- deferred constraint trigger below (and validated in the app layer).
CREATE UNIQUE INDEX question_choices_one_correct ON question_choices (question_id) WHERE is_correct;

CREATE TABLE question_concepts (
    question_id uuid    NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    concept_id  uuid    NOT NULL REFERENCES concepts (id),
    is_primary  boolean NOT NULL DEFAULT false,
    PRIMARY KEY (question_id, concept_id)
);
-- Exactly one Primary Concept per Question: at most one here, at least one via trigger.
CREATE UNIQUE INDEX question_concepts_one_primary ON question_concepts (question_id) WHERE is_primary;
CREATE INDEX question_concepts_concept ON question_concepts (concept_id, question_id);

-- +goose StatementBegin
CREATE FUNCTION check_question_integrity() RETURNS trigger
    LANGUAGE plpgsql AS
$$
DECLARE
    qid       uuid;
    n_correct integer;
    n_choices integer;
    n_primary integer;
BEGIN
    IF TG_TABLE_NAME = 'questions' THEN
        qid := NEW.id;
    ELSIF TG_OP = 'DELETE' THEN
        qid := OLD.question_id;
    ELSE
        qid := NEW.question_id;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM questions WHERE id = qid) THEN
        RETURN NULL; -- question deleted in the same transaction
    END IF;
    SELECT count(*), count(*) FILTER (WHERE is_correct) INTO n_choices, n_correct
    FROM question_choices WHERE question_id = qid;
    IF n_choices < 2 OR n_correct <> 1 THEN
        RAISE EXCEPTION 'question % must have at least two choices and exactly one correct choice (has %, % correct)',
            qid, n_choices, n_correct USING ERRCODE = 'check_violation';
    END IF;
    SELECT count(*) FILTER (WHERE is_primary) INTO n_primary
    FROM question_concepts WHERE question_id = qid;
    IF n_primary <> 1 THEN
        RAISE EXCEPTION 'question % must have exactly one primary concept (has %)', qid, n_primary
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER questions_integrity
    AFTER INSERT OR UPDATE ON questions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_question_integrity();
CREATE CONSTRAINT TRIGGER question_choices_integrity
    AFTER INSERT OR UPDATE OR DELETE ON question_choices
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_question_integrity();
CREATE CONSTRAINT TRIGGER question_concepts_integrity
    AFTER INSERT OR UPDATE OR DELETE ON question_concepts
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_question_integrity();

CREATE TABLE exams (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id    uuid        NOT NULL REFERENCES courses (id),
    name         text        NOT NULL CHECK (name <> ''),
    time_limit_s integer     NOT NULL CHECK (time_limit_s > 0),
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX exams_course ON exams (course_id);

CREATE TABLE exam_questions (
    exam_id     uuid    NOT NULL REFERENCES exams (id),
    question_id uuid    NOT NULL REFERENCES questions (id),
    position    integer NOT NULL CHECK (position > 0),
    PRIMARY KEY (exam_id, question_id),
    UNIQUE (exam_id, position)
);
-- "Exam Questions never appear in Quizzes" is checked per candidate.
CREATE INDEX exam_questions_question ON exam_questions (question_id);

-- ── Sessions and Answers ────────────────────────────────────────────────
CREATE TABLE study_sessions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    learner_id    uuid        NOT NULL REFERENCES learners (id),
    course_id     uuid        NOT NULL REFERENCES courses (id),
    kind          text        NOT NULL CHECK (kind IN ('quiz', 'diagnostic', 'placement', 'exam')),
    concept_id    uuid REFERENCES concepts (id),
    topic_id      uuid REFERENCES topics (id),
    exam_id       uuid REFERENCES exams (id),
    max_questions integer CHECK (max_questions IS NULL OR max_questions > 0),
    started_at    timestamptz NOT NULL DEFAULT now(),
    finished_at   timestamptz,
    deadline      timestamptz,
    CHECK (kind <> 'exam' OR (exam_id IS NOT NULL AND deadline IS NOT NULL)),
    CHECK (kind <> 'diagnostic' OR concept_id IS NOT NULL),
    CHECK (kind <> 'quiz' OR concept_id IS NOT NULL OR topic_id IS NOT NULL)
);
CREATE INDEX study_sessions_learner ON study_sessions (learner_id, course_id, kind, started_at DESC);

CREATE TABLE session_questions (
    session_id  uuid    NOT NULL REFERENCES study_sessions (id),
    question_id uuid    NOT NULL REFERENCES questions (id),
    position    integer NOT NULL CHECK (position > 0),
    concept_id  uuid REFERENCES concepts (id), -- the probed Concept (Placement Tests)
    is_repeat   boolean NOT NULL DEFAULT false,
    PRIMARY KEY (session_id, position),
    UNIQUE (session_id, question_id)
);

CREATE TABLE answers (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    learner_id    uuid        NOT NULL REFERENCES learners (id),
    question_id   uuid        NOT NULL REFERENCES questions (id),
    session_id    uuid        NOT NULL REFERENCES study_sessions (id),
    choice_id     uuid        NOT NULL REFERENCES question_choices (id),
    correct       boolean     NOT NULL,
    is_repeat     boolean     NOT NULL DEFAULT false,
    time_spent_ms integer CHECK (time_spent_ms IS NULL OR time_spent_ms >= 0),
    answered_at   timestamptz NOT NULL DEFAULT now(),
    excluded      boolean     NOT NULL DEFAULT false,
    UNIQUE (session_id, question_id)
);
CREATE INDEX answers_learner_question ON answers (learner_id, question_id, answered_at);
CREATE INDEX answers_question ON answers (question_id);

-- Answers are append-only; only `excluded` may change.
-- +goose StatementBegin
CREATE FUNCTION answers_append_only() RETURNS trigger
    LANGUAGE plpgsql AS
$$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'answers are append-only' USING ERRCODE = 'restrict_violation';
    END IF;
    IF (NEW.id, NEW.learner_id, NEW.question_id, NEW.session_id, NEW.choice_id, NEW.correct,
        NEW.is_repeat, NEW.time_spent_ms, NEW.answered_at)
        IS DISTINCT FROM
       (OLD.id, OLD.learner_id, OLD.question_id, OLD.session_id, OLD.choice_id, OLD.correct,
        OLD.is_repeat, OLD.time_spent_ms, OLD.answered_at) THEN
        RAISE EXCEPTION 'answers are append-only; only excluded may change' USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER answers_append_only
    BEFORE UPDATE OR DELETE ON answers
    FOR EACH ROW EXECUTE FUNCTION answers_append_only();

-- Mastery is a cache derived from Answers (ADR 0002).
CREATE TABLE mastery (
    learner_id uuid             NOT NULL REFERENCES learners (id),
    concept_id uuid             NOT NULL REFERENCES concepts (id),
    score      double precision NOT NULL CHECK (score BETWEEN 0 AND 1),
    evidence   double precision NOT NULL CHECK (evidence >= 0),
    level      text             NOT NULL CHECK (level IN ('unassessed', 'weak', 'developing', 'mastered')),
    updated_at timestamptz      NOT NULL DEFAULT now(),
    PRIMARY KEY (learner_id, concept_id)
);
CREATE INDEX mastery_concept ON mastery (concept_id);

-- ── Learning ────────────────────────────────────────────────────────────
CREATE TABLE learning_paths (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    learner_id        uuid        NOT NULL REFERENCES learners (id),
    course_id         uuid        NOT NULL REFERENCES courses (id),
    root_concept_id   uuid        NOT NULL REFERENCES concepts (id),
    target_concept_id uuid        NOT NULL REFERENCES concepts (id),
    path              uuid[]      NOT NULL CHECK (cardinality(path) > 0),
    started_at        timestamptz NOT NULL DEFAULT now(),
    completed_at      timestamptz,
    abandoned_at      timestamptz
);
CREATE UNIQUE INDEX learning_paths_one_active ON learning_paths (learner_id, course_id)
    WHERE completed_at IS NULL AND abandoned_at IS NULL;

-- ── Quality loop ────────────────────────────────────────────────────────
CREATE TABLE flags (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id     uuid        NOT NULL REFERENCES questions (id),
    learner_id      uuid        NOT NULL REFERENCES learners (id),
    reason          text        NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 2000),
    created_at      timestamptz NOT NULL DEFAULT now(),
    resolved_at     timestamptz,
    resolution_note text
);
CREATE INDEX flags_open ON flags (created_at) WHERE resolved_at IS NULL;
CREATE INDEX flags_question ON flags (question_id);

CREATE TABLE content_requests (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    concept_id   uuid        NOT NULL REFERENCES concepts (id),
    reason       text        NOT NULL CHECK (reason <> ''),
    created_at   timestamptz NOT NULL DEFAULT now(),
    fulfilled_at timestamptz,
    note         text
);
-- Content Requests are raised per Concept: at most one open request per Concept.
CREATE UNIQUE INDEX content_requests_one_open ON content_requests (concept_id) WHERE fulfilled_at IS NULL;

-- +goose Down
DROP TABLE content_requests, flags, learning_paths, mastery, answers, session_questions, study_sessions,
    exam_questions, exams, question_concepts, question_choices, questions, lessons, prerequisites,
    concepts, topics, question_formats, courses, sessions, invites, learners;
DROP FUNCTION answers_append_only();
DROP FUNCTION check_question_integrity();
