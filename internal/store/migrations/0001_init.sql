-- 0001_init.sql -- the initial schema.
--
-- Migrations are append-only. Never edit a file that has shipped: a database
-- already at version 1 will not re-run this, so an edit here only affects
-- fresh installs and the two would silently diverge. Add 0002_*.sql instead.
--
-- Every table is STRICT. SQLite's default is type *affinity*, meaning a column
-- declared INTEGER will still accept the string 'hello'. STRICT turns the
-- declared type into a real constraint, which is the behaviour you would
-- expect coming from H2 or Postgres.

------------------------------------------------------------------------------
-- CONTENT -- owned by the catalog (seed data today, the API later).
--
-- A refresh replaces these wholesale. Nothing the user typed lives here, so
-- there is nothing to merge and nothing to lose.
------------------------------------------------------------------------------

CREATE TABLE problems (
    id          INTEGER PRIMARY KEY,
    title       TEXT    NOT NULL,
    platform    TEXT    NOT NULL DEFAULT '',
    difficulty  TEXT    NOT NULL DEFAULT '',
    description TEXT    NOT NULL DEFAULT '',
    url         TEXT    NOT NULL DEFAULT '',
    -- Unix seconds. 0 means "seeded locally, never fetched", which is what
    -- the TTL check will look at once there is a server to refresh from.
    fetched_at  INTEGER NOT NULL DEFAULT 0
) STRICT;

-- The child tables below all carry `ord` because JSON arrays are ordered and
-- SQL rows are not. Without it, examples and constraints would come back in
-- whatever order the query planner felt like.

CREATE TABLE problem_tags (
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    ord        INTEGER NOT NULL,
    tag        TEXT    NOT NULL,
    PRIMARY KEY (problem_id, ord)
) STRICT;

CREATE TABLE examples (
    problem_id  INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    ord         INTEGER NOT NULL,
    input       TEXT    NOT NULL DEFAULT '',
    output      TEXT    NOT NULL DEFAULT '',
    explanation TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (problem_id, ord)
) STRICT;

-- Named problem_constraints, not constraints: CONSTRAINT is a SQL keyword and
-- the plural is close enough to be a trap in a hand-written query.
CREATE TABLE problem_constraints (
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    ord        INTEGER NOT NULL,
    body       TEXT    NOT NULL,
    PRIMARY KEY (problem_id, ord)
) STRICT;

CREATE TABLE test_cases (
    problem_id      INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    ord             INTEGER NOT NULL,
    input           TEXT    NOT NULL DEFAULT '',
    expected_output TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (problem_id, ord)
) STRICT;

-- language is a free-text id, never an enum -- the server may know languages
-- this build has never heard of. internal/languages resolves unknown ids to a
-- usable default rather than rejecting them.
CREATE TABLE code_stubs (
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    language   TEXT    NOT NULL,
    stub       TEXT    NOT NULL,
    PRIMARY KEY (problem_id, language)
) STRICT;

------------------------------------------------------------------------------
-- LOCAL -- owned by the user. Never written by a content refresh.
--
-- Deliberately NO foreign key to problems. If a problem is withdrawn from the
-- catalog upstream, ON DELETE CASCADE would take the user's code with it.
-- An orphaned solution row is a far better outcome than a deleted one.
------------------------------------------------------------------------------

CREATE TABLE solutions (
    problem_id INTEGER NOT NULL,
    language   TEXT    NOT NULL,
    code       TEXT    NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (problem_id, language)
) STRICT;

CREATE TABLE progress (
    problem_id         INTEGER PRIMARY KEY,
    solved             INTEGER NOT NULL DEFAULT 0 CHECK (solved IN (0, 1)),
    review             INTEGER NOT NULL DEFAULT 0 CHECK (review IN (0, 1)),
    attempts           INTEGER NOT NULL DEFAULT 0,
    time_taken_minutes INTEGER NOT NULL DEFAULT 0,
    date_solved        TEXT    NOT NULL DEFAULT '',
    notes              TEXT    NOT NULL DEFAULT '',
    -- Which language the editor was last in for this problem.
    last_language      TEXT    NOT NULL DEFAULT '',
    -- Unix seconds. This replaces session.json: "continue where you left off"
    -- is the row with the highest value here, not a separately stored id that
    -- can disagree with it.
    last_opened_at     INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE INDEX progress_last_opened ON progress(last_opened_at DESC);

-- timestamp stays TEXT because that is what the payload carries and nothing
-- parses it yet. When submissions become real it earns a sortable column via
-- a later migration -- which is exactly what migrations are for.
CREATE TABLE submissions (
    id         TEXT    PRIMARY KEY,
    problem_id INTEGER NOT NULL,
    language   TEXT    NOT NULL DEFAULT '',
    status     TEXT    NOT NULL DEFAULT '',
    runtime    TEXT    NOT NULL DEFAULT '',
    memory     TEXT    NOT NULL DEFAULT '',
    timestamp  TEXT    NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX submissions_problem ON submissions(problem_id);
