-- +goose Up

-- River v0.44.0 migrations 002-007. Version 001 is River's own
-- migration ledger and is omitted because Goose remains the migration authority.

-- River 002
CREATE TYPE /* TEMPLATE: schema */river_job_state AS ENUM(
  'available',
  'cancelled',
  'completed',
  'discarded',
  'retryable',
  'running',
  'scheduled'
);

CREATE TABLE /* TEMPLATE: schema */river_job(
  -- 8 bytes
  id bigserial PRIMARY KEY,

  -- 8 bytes (4 bytes + 2 bytes + 2 bytes)
  --
  -- `state` is kept near the top of the table for operator convenience -- when
  -- looking at jobs with `SELECT *` it'll appear first after ID. The other two
  -- fields aren't as important but are kept adjacent to `state` for alignment
  -- to get an 8-byte block.
  state /* TEMPLATE: schema */river_job_state NOT NULL DEFAULT 'available',
  attempt smallint NOT NULL DEFAULT 0,
  max_attempts smallint NOT NULL,

  -- 8 bytes each (no alignment needed)
  attempted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  finalized_at timestamptz,
  scheduled_at timestamptz NOT NULL DEFAULT NOW(),

  -- 2 bytes (some wasted padding probably)
  priority smallint NOT NULL DEFAULT 1,

  -- types stored out-of-band
  args jsonb,
  attempted_by text[],
  errors jsonb[],
  kind text NOT NULL,
  metadata jsonb NOT NULL DEFAULT '{}',
  queue text NOT NULL DEFAULT 'default',
  tags varchar(255)[],

  CONSTRAINT finalized_or_finalized_at_null CHECK ((state IN ('cancelled', 'completed', 'discarded') AND finalized_at IS NOT NULL) OR finalized_at IS NULL),
  CONSTRAINT max_attempts_is_positive CHECK (max_attempts > 0),
  CONSTRAINT priority_in_range CHECK (priority >= 1 AND priority <= 4),
  CONSTRAINT queue_length CHECK (char_length(queue) > 0 AND char_length(queue) < 128),
  CONSTRAINT kind_length CHECK (char_length(kind) > 0 AND char_length(kind) < 128)
);

-- We may want to consider adding another property here after `kind` if it seems
-- like it'd be useful for something.
CREATE INDEX river_job_kind ON /* TEMPLATE: schema */river_job USING btree(kind);

CREATE INDEX river_job_state_and_finalized_at_index ON /* TEMPLATE: schema */river_job USING btree(state, finalized_at) WHERE finalized_at IS NOT NULL;

CREATE INDEX river_job_prioritized_fetching_index ON /* TEMPLATE: schema */river_job USING btree(state, queue, priority, scheduled_at, id);

CREATE INDEX river_job_args_index ON /* TEMPLATE: schema */river_job USING GIN(args);

CREATE INDEX river_job_metadata_index ON /* TEMPLATE: schema */river_job USING GIN(metadata);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION /* TEMPLATE: schema */river_job_notify()
  RETURNS TRIGGER
  AS $$
DECLARE
  payload json;
BEGIN
  IF NEW.state = 'available' THEN
    -- Notify will coalesce duplicate notifications within a transaction, so
    -- keep these payloads generalized:
    payload = json_build_object('queue', NEW.queue);
    PERFORM
      pg_notify('river_insert', payload::text);
  END IF;
  RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER river_notify
  AFTER INSERT ON /* TEMPLATE: schema */river_job
  FOR EACH ROW
  EXECUTE PROCEDURE /* TEMPLATE: schema */river_job_notify();

CREATE UNLOGGED TABLE /* TEMPLATE: schema */river_leader(
    -- 8 bytes each (no alignment needed)
    elected_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,

    -- types stored out-of-band
    leader_id text NOT NULL,
    name text PRIMARY KEY,

    CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128),
    CONSTRAINT leader_id_length CHECK (char_length(leader_id) > 0 AND char_length(leader_id) < 128)
);

-- River 003
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN tags SET DEFAULT '{}';
UPDATE /* TEMPLATE: schema */river_job SET tags = '{}' WHERE tags IS NULL;
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN tags SET NOT NULL;

-- River 004
-- The args column never had a NOT NULL constraint or default value at the
-- database level, though we tried to ensure one at the application level.
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN args SET DEFAULT '{}';
UPDATE /* TEMPLATE: schema */river_job SET args = '{}' WHERE args IS NULL;
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN args SET NOT NULL;
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN args DROP DEFAULT;

-- The metadata column never had a NOT NULL constraint or default value at the
-- database level, though we tried to ensure one at the application level.
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN metadata SET DEFAULT '{}';
UPDATE /* TEMPLATE: schema */river_job SET metadata = '{}' WHERE metadata IS NULL;
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN metadata SET NOT NULL;

-- The 'pending' job state will be used for upcoming functionality:
ALTER TYPE /* TEMPLATE: schema */river_job_state ADD VALUE IF NOT EXISTS 'pending' AFTER 'discarded';

ALTER TABLE /* TEMPLATE: schema */river_job DROP CONSTRAINT finalized_or_finalized_at_null;
ALTER TABLE /* TEMPLATE: schema */river_job ADD CONSTRAINT finalized_or_finalized_at_null CHECK (
    (finalized_at IS NULL AND state NOT IN ('cancelled', 'completed', 'discarded')) OR
    (finalized_at IS NOT NULL AND state IN ('cancelled', 'completed', 'discarded'))
);

DROP TRIGGER river_notify ON /* TEMPLATE: schema */river_job;
DROP FUNCTION /* TEMPLATE: schema */river_job_notify;

--
-- Create table `river_queue`.
--

CREATE TABLE /* TEMPLATE: schema */river_queue (
    name text PRIMARY KEY NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}' ::jsonb,
    paused_at timestamptz,
    updated_at timestamptz NOT NULL
);

--
-- Alter `river_leader` to add a default value of 'default` to `name`.
--

ALTER TABLE /* TEMPLATE: schema */river_leader
    ALTER COLUMN name SET DEFAULT 'default',
    DROP CONSTRAINT name_length,
    ADD CONSTRAINT name_length CHECK (name = 'default');

-- River 005
-- River's migration-ledger rewrite is intentionally omitted. Goose owns the
-- only migration ledger in this repository.

--
-- Add `river_job.unique_key` and bring up an index on it.
--

-- These statements use `IF NOT EXISTS` to allow users with a `river_job` table
-- of non-trivial size to build the index `CONCURRENTLY` out of band of this
-- migration, then follow by completing the migration.
ALTER TABLE /* TEMPLATE: schema */river_job
    ADD COLUMN IF NOT EXISTS unique_key bytea;

CREATE UNIQUE INDEX IF NOT EXISTS river_job_kind_unique_key_idx ON /* TEMPLATE: schema */river_job (kind, unique_key) WHERE unique_key IS NOT NULL;

--
-- Create `river_client` and derivative.
--
-- This feature hasn't quite yet been implemented, but we're taking advantage of
-- the migration to add the schema early so that we can add it later without an
-- additional migration.
--

CREATE UNLOGGED TABLE /* TEMPLATE: schema */river_client (
    id text PRIMARY KEY NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}',
    paused_at timestamptz,
    updated_at timestamptz NOT NULL,
    CONSTRAINT name_length CHECK (char_length(id) > 0 AND char_length(id) < 128)
);

-- Differs from `river_queue` in that it tracks the queue state for a particular
-- active client.
CREATE UNLOGGED TABLE /* TEMPLATE: schema */river_client_queue (
    river_client_id text NOT NULL REFERENCES /* TEMPLATE: schema */river_client (id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    max_workers bigint NOT NULL DEFAULT 0,
    metadata jsonb NOT NULL DEFAULT '{}',
    num_jobs_completed bigint NOT NULL DEFAULT 0,
    num_jobs_running bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (river_client_id, name),
    CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128),
    CONSTRAINT num_jobs_completed_zero_or_positive CHECK (num_jobs_completed >= 0),
    CONSTRAINT num_jobs_running_zero_or_positive CHECK (num_jobs_running >= 0)
);

-- River 006
CREATE OR REPLACE FUNCTION /* TEMPLATE: schema */river_job_state_in_bitmask(bitmask BIT(8), state /* TEMPLATE: schema */river_job_state)
RETURNS boolean
LANGUAGE SQL
IMMUTABLE
RETURN CASE state
    WHEN 'available' THEN get_bit(bitmask, 7)
    WHEN 'cancelled' THEN get_bit(bitmask, 6)
    WHEN 'completed' THEN get_bit(bitmask, 5)
    WHEN 'discarded' THEN get_bit(bitmask, 4)
    WHEN 'pending'   THEN get_bit(bitmask, 3)
    WHEN 'retryable' THEN get_bit(bitmask, 2)
    WHEN 'running'   THEN get_bit(bitmask, 1)
    WHEN 'scheduled' THEN get_bit(bitmask, 0)
    ELSE 0
END = 1;

--
-- Add `river_job.unique_states` and bring up an index on it.
--
-- This column may exist already if users manually created the column and index
-- as instructed in the changelog so the index could be created `CONCURRENTLY`.
--
ALTER TABLE /* TEMPLATE: schema */river_job ADD COLUMN IF NOT EXISTS unique_states BIT(8);

-- This statement uses `IF NOT EXISTS` to allow users with a `river_job` table
-- of non-trivial size to build the index `CONCURRENTLY` out of band of this
-- migration, then follow by completing the migration.
CREATE UNIQUE INDEX IF NOT EXISTS river_job_unique_idx ON /* TEMPLATE: schema */river_job (unique_key)
    WHERE unique_key IS NOT NULL
      AND unique_states IS NOT NULL
      AND /* TEMPLATE: schema */river_job_state_in_bitmask(unique_states, state);

-- Remove the old unique index. Users who are actively using the unique jobs
-- feature and who wish to avoid deploy downtime may want od drop this in a
-- subsequent migration once all jobs using the old unique system have been
-- completed (i.e. no more rows with non-null unique_key and null
-- unique_states).
DROP INDEX /* TEMPLATE: schema */river_job_kind_unique_key_idx;

-- River 007
--
-- Notification outbox.
--

CREATE TABLE /* TEMPLATE: schema */river_notification (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    payload text NOT NULL,
    topic text NOT NULL,
    CONSTRAINT topic_length CHECK (length(topic) > 0 AND length(topic) < 128)
);

CREATE INDEX river_notification_created_at_idx ON /* TEMPLATE: schema */river_notification (created_at);
CREATE INDEX river_notification_topic_id_idx ON /* TEMPLATE: schema */river_notification (topic, id);

--
-- SQLite JSONB conversion.
--
-- No-op. PostgreSQL already stores River JSON columns as jsonb.

--
-- SQL cleanup.
--

--
-- Drop unused tables `river_client` and `river_client_queue`.
--

DROP TABLE /* TEMPLATE: schema */river_client_queue;
DROP TABLE /* TEMPLATE: schema */river_client;

--
-- Adds `DEFAULT 25` to `river_job.max_attempts`.
--

ALTER TABLE /* TEMPLATE: schema */river_job
    ALTER COLUMN max_attempts SET DEFAULT 25;

--
-- Changes `river_queue.updated_at` to have a default of `CURRENT_TIMESTAMP`.
--

ALTER TABLE /* TEMPLATE: schema */river_queue
    ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP;

-- +goose Down

-- River 007
--
-- SQL cleanup rollback.
--

--
-- Add back unused tables `river_client` and `river_client_queue`.
--

CREATE UNLOGGED TABLE /* TEMPLATE: schema */river_client (
    id text PRIMARY KEY NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}',
    paused_at timestamptz,
    updated_at timestamptz NOT NULL,
    CONSTRAINT name_length CHECK (char_length(id) > 0 AND char_length(id) < 128)
);

CREATE UNLOGGED TABLE /* TEMPLATE: schema */river_client_queue (
    river_client_id text NOT NULL REFERENCES /* TEMPLATE: schema */river_client (id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    max_workers bigint NOT NULL DEFAULT 0,
    metadata jsonb NOT NULL DEFAULT '{}',
    num_jobs_completed bigint NOT NULL DEFAULT 0,
    num_jobs_running bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (river_client_id, name),
    CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128),
    CONSTRAINT num_jobs_completed_zero_or_positive CHECK (num_jobs_completed >= 0),
    CONSTRAINT num_jobs_running_zero_or_positive CHECK (num_jobs_running >= 0)
);

--
-- Revert addition of `DEFAULT 25` to `river_job.max_attempts`.
--

ALTER TABLE /* TEMPLATE: schema */river_job
    ALTER COLUMN max_attempts DROP DEFAULT;

--
-- Changes `river_queue.updated_at` to revert the default of `CURRENT_TIMESTAMP`.
--

ALTER TABLE /* TEMPLATE: schema */river_queue
    ALTER COLUMN updated_at DROP DEFAULT;

--
-- SQLite JSONB conversion rollback.
--
-- No-op. PostgreSQL already stores River JSON columns as jsonb.

--
-- Notification outbox rollback.
--

DROP TABLE /* TEMPLATE: schema */river_notification;

-- River 006

--
-- Drop `river_job.unique_states` and its index.
--

DROP INDEX /* TEMPLATE: schema */river_job_unique_idx;

ALTER TABLE /* TEMPLATE: schema */river_job
    DROP COLUMN unique_states;

CREATE UNIQUE INDEX IF NOT EXISTS river_job_kind_unique_key_idx ON /* TEMPLATE: schema */river_job (kind, unique_key) WHERE unique_key IS NOT NULL;

--
-- Drop `river_job_state_in_bitmask` function.
--
DROP FUNCTION /* TEMPLATE: schema */river_job_state_in_bitmask;

-- River 005
-- No River migration ledger was created, so there is nothing to revert here.

--
-- Drop `river_job.unique_key`.
--

ALTER TABLE /* TEMPLATE: schema */river_job
    DROP COLUMN unique_key;

--
-- Drop `river_client` and derivative.
--

DROP TABLE /* TEMPLATE: schema */river_client_queue;
DROP TABLE /* TEMPLATE: schema */river_client;

-- River 004
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN args DROP NOT NULL;

ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN metadata DROP NOT NULL;
ALTER TABLE /* TEMPLATE: schema */river_job ALTER COLUMN metadata DROP DEFAULT;

-- It is not possible to safely remove 'pending' from the river_job_state enum,
-- so leave it in place.

ALTER TABLE /* TEMPLATE: schema */river_job DROP CONSTRAINT finalized_or_finalized_at_null;
ALTER TABLE /* TEMPLATE: schema */river_job ADD CONSTRAINT finalized_or_finalized_at_null CHECK (
  (state IN ('cancelled', 'completed', 'discarded') AND finalized_at IS NOT NULL) OR finalized_at IS NULL
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION /* TEMPLATE: schema */river_job_notify()
  RETURNS TRIGGER
  AS $$
DECLARE
  payload json;
BEGIN
  IF NEW.state = 'available' THEN
    -- Notify will coalesce duplicate notifications within a transaction, so
    -- keep these payloads generalized:
    payload = json_build_object('queue', NEW.queue);
    PERFORM
      pg_notify('river_insert', payload::text);
  END IF;
  RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER river_notify
  AFTER INSERT ON /* TEMPLATE: schema */river_job
  FOR EACH ROW
  EXECUTE PROCEDURE /* TEMPLATE: schema */river_job_notify();

DROP TABLE /* TEMPLATE: schema */river_queue;

ALTER TABLE /* TEMPLATE: schema */river_leader
    ALTER COLUMN name DROP DEFAULT,
    DROP CONSTRAINT name_length,
    ADD CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128);

-- River 003
ALTER TABLE /* TEMPLATE: schema */river_job
    ALTER COLUMN tags DROP NOT NULL,
    ALTER COLUMN tags DROP DEFAULT;

-- River 002
DROP TABLE /* TEMPLATE: schema */river_job;
DROP FUNCTION /* TEMPLATE: schema */river_job_notify;
DROP TYPE /* TEMPLATE: schema */river_job_state;

DROP TABLE /* TEMPLATE: schema */river_leader;
