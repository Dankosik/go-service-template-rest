-- +goose Up

CREATE TABLE outbox_ordering_heads (
    ordering_key text PRIMARY KEY,
    last_sequence bigint NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT outbox_ordering_heads_key_check CHECK (
        octet_length(ordering_key) BETWEEN 1 AND 256
        AND ordering_key !~ '[[:cntrl:]]'
    ),
    CONSTRAINT outbox_ordering_heads_sequence_check CHECK (last_sequence > 0)
);

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
    CONSTRAINT outbox_events_payload_check CHECK (
        octet_length(payload) BETWEEN 1 AND 262144
        AND convert_from(payload, 'UTF8')::jsonb IS NOT NULL
    ),
    CONSTRAINT outbox_events_metadata_check CHECK (
        octet_length(metadata) BETWEEN 2 AND 32768
        AND jsonb_typeof(convert_from(metadata, 'UTF8')::jsonb) = 'object'
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
);

CREATE UNIQUE INDEX outbox_events_ordering_sequence_key
    ON outbox_events (ordering_key, ordering_sequence)
    WHERE ordering_key IS NOT NULL;
CREATE INDEX outbox_events_claim_idx
    ON outbox_events (available_at, created_at, id)
    WHERE published_at IS NULL AND poisoned_at IS NULL;
CREATE INDEX outbox_events_ordering_head_idx
    ON outbox_events (ordering_key, ordering_sequence)
    WHERE published_at IS NULL AND ordering_key IS NOT NULL;
CREATE INDEX outbox_events_cleanup_idx
    ON outbox_events (published_at, id)
    WHERE published_at IS NOT NULL;
CREATE INDEX outbox_events_poison_idx
    ON outbox_events (poisoned_at, id)
    WHERE poisoned_at IS NOT NULL;

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

-- +goose Down

DROP TABLE outbox_redrives;
DROP TABLE outbox_events;
DROP TABLE outbox_ordering_heads;
