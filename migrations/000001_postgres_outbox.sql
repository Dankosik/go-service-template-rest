-- +goose Up

-- One head per ordering key. last_sequence is the retained, monotonic authority
-- that rejects a reused sequence: Append advances it under a row lock in the
-- same transaction as the insert, and cleanup never deletes a head, so the
-- rejection survives the events it was derived from. current_sequence names the
-- key's earliest unpublished sequence, or NULL while the key is idle.
--
-- Every append and every publication for a key updates that key's single row,
-- and a batched ordered finalization advances one head per event in the lease,
-- so heads face whole-page updates. fillfactor leaves room for a second version
-- of every live row a page carries, which keeps those updates heap-only.
CREATE TABLE outbox_ordering_heads (
    ordering_key text PRIMARY KEY,
    last_sequence bigint NOT NULL,
    current_sequence bigint,
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT outbox_ordering_heads_key_check CHECK (
        octet_length(ordering_key) BETWEEN 1 AND 256
        AND ordering_key !~ '[[:cntrl:]]'
    ),
    CONSTRAINT outbox_ordering_heads_sequence_check CHECK (
        last_sequence > 0
        AND (
            current_sequence IS NULL
            OR current_sequence BETWEEN 1 AND last_sequence
        )
    )
) WITH (
    fillfactor = 45,
    autovacuum_vacuum_scale_factor = 0.01,
    autovacuum_vacuum_threshold = 2000,
    autovacuum_analyze_scale_factor = 0.01,
    autovacuum_analyze_threshold = 2000
);

-- The event envelope is immutable; every other column is delivery state.
-- ordering_ready materializes "this event is its key's claimable head" so the
-- claim never scans a hot key's predecessors.
--
-- Storage parameters are measured rather than conventional. A batch claim
-- updates most of one heap page at a time and touches no indexed column, so a
-- page with room for a second version of every live row it carries keeps every
-- claim heap-only: no index insert, and the dead version reclaimed by ordinary
-- page pruning instead of an index vacuum pass. Over insert, claim, and publish
-- of 2,000 events on the repository's PostgreSQL 17 fixture:
--
--   fillfactor  100     70     60     50     45     40     30
--   HOT claims  201    845   1168   1700   1996   2000   2000
--   cycle WAL  2.12M  1.53M  1.31M  1.06M  0.97M  0.94M  0.94M
--   heap       1.18M  1.07M  1.00M  0.94M  0.93M  0.95M  1.29M
--
-- 45 is the knee: every claim is heap-only and the heap is at its smallest,
-- because the version sprawl a packed page causes costs more than the reserve.
-- Below it the unused reserve grows the table again. With a 1 KiB payload the
-- same setting moved cycle WAL from 6.28M to 1.00M and the heap from 8.15M to
-- 5.50M, so this is a ratio rather than a value tuned to one payload size.
-- Re-measure if the claim stops updating whole pages at a time.
--
-- Autovacuum's default scale factors are proportional to table size, so a
-- retention window would make maintenance wait longer the more published rows
-- are kept even though the churn producing dead tuples is unrelated to that.
-- These values keep a floor that does not move with retention and a 1% slope
-- that still scales; they are a starting point, not a measured optimum. Raise
-- them when autovacuum runs back to back on this table, since each pass reads
-- its indexes, and lower them when n_dead_tup or the relation size keeps
-- growing between passes.
CREATE TABLE outbox_events (
    id text PRIMARY KEY,
    event_type text NOT NULL,
    source text NOT NULL,
    destination text NOT NULL,
    schema_name text NOT NULL,
    occurred_at timestamptz NOT NULL,
    payload bytea NOT NULL,
    metadata bytea NOT NULL DEFAULT '\x7b7d'::bytea,
    ordering_key text,
    ordering_sequence bigint,
    ordering_ready boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    available_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    cycle_attempt_count integer NOT NULL DEFAULT 0,
    total_attempt_count bigint NOT NULL DEFAULT 0,
    last_attempt_at timestamptz,
    lease_token text,
    lease_expires_at timestamptz,
    published_at timestamptz,
    poisoned_at timestamptz,
    last_error_class text,
    redrive_count integer NOT NULL DEFAULT 0,
    last_redrive_id text,
    last_redriven_at timestamptz,
    CONSTRAINT outbox_events_id_check CHECK (
        octet_length(id) BETWEEN 1 AND 256 AND id !~ '[[:cntrl:]]'
    ),
    CONSTRAINT outbox_events_type_check CHECK (
        octet_length(event_type) BETWEEN 1 AND 256 AND event_type !~ '[[:cntrl:]]'
    ),
    CONSTRAINT outbox_events_source_check CHECK (
        octet_length(source) BETWEEN 1 AND 256 AND source !~ '[[:cntrl:]]'
    ),
    CONSTRAINT outbox_events_destination_check CHECK (
        octet_length(destination) BETWEEN 1 AND 256 AND destination !~ '[[:cntrl:]]'
    ),
    CONSTRAINT outbox_events_schema_check CHECK (
        octet_length(schema_name) BETWEEN 1 AND 256 AND schema_name !~ '[[:cntrl:]]'
    ),
    CONSTRAINT outbox_events_occurred_at_check CHECK (
        isfinite(occurred_at)
        AND occurred_at <> TIMESTAMPTZ '0001-01-01 00:00:00+00'
    ),
    -- IS JSON holds the stored bytes to the same lenient grammar the json type
    -- accepts, rather than the jsonb one: the outbox stores and retries the
    -- exact bytes it was given, and jsonb would reject payloads that are valid
    -- JSON, such as a number outside PostgreSQL numeric range or a string with
    -- an escaped NUL. IS JSON OBJECT is also the whole metadata rule, so the
    -- object requirement no longer needs a second decode and a text pattern
    -- beside the parse. Bytes that are not valid UTF-8 are rejected as an
    -- encoding error rather than a check violation; both refuse the row, and
    -- the append path validates encoding before the statement is sent.
    CONSTRAINT outbox_events_payload_check CHECK (
        octet_length(payload) BETWEEN 1 AND 262144
        AND payload IS JSON
    ),
    CONSTRAINT outbox_events_metadata_check CHECK (
        octet_length(metadata) BETWEEN 2 AND 32768
        AND metadata IS JSON OBJECT
    ),
    CONSTRAINT outbox_events_ordering_check CHECK (
        (ordering_key IS NULL AND ordering_sequence IS NULL)
        OR (
            ordering_key IS NOT NULL
            AND octet_length(ordering_key) BETWEEN 1 AND 256
            AND ordering_key !~ '[[:cntrl:]]'
            AND ordering_sequence IS NOT NULL
            AND ordering_sequence > 0
        )
    ),
    CONSTRAINT outbox_events_envelope_size_check CHECK (
        octet_length(id)
        + octet_length(event_type)
        + octet_length(source)
        + octet_length(destination)
        + octet_length(schema_name)
        + coalesce(octet_length(ordering_key), 0)
        + octet_length(payload)
        + octet_length(metadata) <= 294912
    ),
    CONSTRAINT outbox_events_attempts_check CHECK (
        cycle_attempt_count >= 0 AND total_attempt_count >= cycle_attempt_count
    ),
    CONSTRAINT outbox_events_lease_check CHECK (
        (lease_token IS NULL) = (lease_expires_at IS NULL)
        AND (
            lease_token IS NULL
            OR (
                octet_length(lease_token) BETWEEN 1 AND 256
                AND lease_token !~ '[[:cntrl:]]'
            )
        )
    ),
    CONSTRAINT outbox_events_terminal_check CHECK (
        NOT (published_at IS NOT NULL AND poisoned_at IS NOT NULL)
        AND (published_at IS NULL OR lease_token IS NULL)
        AND (poisoned_at IS NULL OR lease_token IS NULL)
    ),
    CONSTRAINT outbox_events_error_class_check CHECK (
        last_error_class IS NULL
        OR (
            octet_length(last_error_class) BETWEEN 1 AND 64
            AND last_error_class !~ '[[:cntrl:]]'
        )
    ),
    CONSTRAINT outbox_events_redrive_check CHECK (
        redrive_count >= 0
        AND (
            (redrive_count = 0 AND last_redrive_id IS NULL AND last_redriven_at IS NULL)
            OR (
                redrive_count > 0
                AND last_redrive_id IS NOT NULL
                AND octet_length(last_redrive_id) BETWEEN 1 AND 256
                AND last_redrive_id !~ '[[:cntrl:]]'
                AND last_redriven_at IS NOT NULL
            )
        )
    )
) WITH (
    fillfactor = 45,
    autovacuum_vacuum_scale_factor = 0.01,
    autovacuum_vacuum_threshold = 10000,
    autovacuum_vacuum_insert_scale_factor = 0.01,
    autovacuum_vacuum_insert_threshold = 10000,
    autovacuum_analyze_scale_factor = 0.01,
    autovacuum_analyze_threshold = 5000
);

-- Claim order and eligibility in one index. Excluding rows that are not their
-- key's claimable head keeps a hot key's backlog out of the scan entirely.
CREATE INDEX outbox_events_claim_idx
    ON outbox_events (available_at, created_at, id)
    WHERE published_at IS NULL
      AND poisoned_at IS NULL
      AND (ordering_key IS NULL OR ordering_ready);

-- At most one claimable event per ordering key. This is what lets the relay
-- publish a batch concurrently without reordering a key, and what lets one
-- statement finalize a whole lease: a lease can never hold two events of the
-- same key.
CREATE UNIQUE INDEX outbox_events_ordering_ready_key
    ON outbox_events (ordering_key)
    WHERE published_at IS NULL
      AND ordering_key IS NOT NULL
      AND ordering_ready;

-- Successor lookup during ordered finalization, and the uniqueness that still
-- has work to do: at most one unpublished event per (key, sequence). Spanning
-- published rows as well would reject nothing the retained high-water mark
-- already rejects, while every ordered append paid to maintain it.
CREATE UNIQUE INDEX outbox_events_ordering_pending_key
    ON outbox_events (ordering_key, ordering_sequence)
    WHERE published_at IS NULL AND ordering_key IS NOT NULL;

CREATE INDEX outbox_events_cleanup_idx
    ON outbox_events (published_at, id)
    WHERE published_at IS NOT NULL;

CREATE INDEX outbox_events_poison_idx
    ON outbox_events (poisoned_at, id)
    WHERE poisoned_at IS NOT NULL;

-- State observation reads the pending set instead of every retained published
-- row, so its cost tracks backlog rather than retention volume.
CREATE INDEX outbox_events_pending_idx
    ON outbox_events (created_at)
    WHERE published_at IS NULL;

CREATE TABLE outbox_redrives (
    audit_id text PRIMARY KEY,
    event_id text NOT NULL REFERENCES outbox_events (id) ON DELETE CASCADE,
    redriven_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    cycle_number integer NOT NULL,
    CONSTRAINT outbox_redrives_audit_id_check CHECK (
        octet_length(audit_id) BETWEEN 1 AND 256 AND audit_id !~ '[[:cntrl:]]'
    ),
    CONSTRAINT outbox_redrives_cycle_check CHECK (cycle_number > 0)
);

CREATE INDEX outbox_redrives_event_idx ON outbox_redrives (event_id, cycle_number);

-- Wake relays on commit instead of only on the poll timer. A statement-level
-- trigger costs the appending transaction no extra client round trip, and
-- PostgreSQL collapses duplicate (channel, payload) notifications inside one
-- transaction. The signal is an optimization only: a relay that misses it still
-- claims the row on its next poll.
-- +goose StatementBegin
CREATE FUNCTION outbox_notify_appended() RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $$
BEGIN
    PERFORM pg_notify('outbox_appended', '');
    RETURN NULL;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER outbox_events_notify_appended
    AFTER INSERT ON outbox_events
    FOR EACH STATEMENT
    EXECUTE FUNCTION outbox_notify_appended();

-- +goose Down

DROP TRIGGER outbox_events_notify_appended ON outbox_events;
DROP FUNCTION outbox_notify_appended();
DROP TABLE outbox_redrives;
DROP TABLE outbox_events;
DROP TABLE outbox_ordering_heads;
