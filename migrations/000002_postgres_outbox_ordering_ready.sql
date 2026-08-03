-- +goose Up

ALTER TABLE outbox_ordering_heads
    ADD COLUMN current_sequence bigint;

INSERT INTO outbox_ordering_heads (ordering_key, last_sequence)
SELECT ordering_key, max(ordering_sequence)
FROM outbox_events
WHERE ordering_key IS NOT NULL
GROUP BY ordering_key
ON CONFLICT (ordering_key) DO UPDATE
SET last_sequence = greatest(
        outbox_ordering_heads.last_sequence,
        EXCLUDED.last_sequence
    ),
    updated_at = statement_timestamp()
WHERE outbox_ordering_heads.last_sequence < EXCLUDED.last_sequence;

UPDATE outbox_ordering_heads AS head
SET current_sequence = (
        SELECT event.ordering_sequence
        FROM outbox_events AS event
        WHERE event.ordering_key = head.ordering_key
          AND event.published_at IS NULL
        ORDER BY event.ordering_sequence
        LIMIT 1
    ),
    updated_at = statement_timestamp()
WHERE EXISTS (
    SELECT 1
    FROM outbox_events AS event
    WHERE event.ordering_key = head.ordering_key
      AND event.published_at IS NULL
);

-- +goose StatementBegin
DO $$
DECLARE
    bounded_event_count bigint;
    ready_count bigint;
BEGIN
    SELECT count(*) INTO ready_count
    FROM outbox_ordering_heads
    WHERE current_sequence IS NOT NULL;
    SELECT count(*) INTO bounded_event_count
    FROM (
        SELECT 1
        FROM outbox_events
        LIMIT ready_count * 2
    ) AS bounded_events;

    IF bounded_event_count = ready_count * 2 THEN
        ALTER TABLE outbox_events
            ADD COLUMN ordering_ready boolean NOT NULL DEFAULT false;

        UPDATE outbox_events AS event
        SET ordering_ready = true
        FROM outbox_ordering_heads AS head
        WHERE event.ordering_key = head.ordering_key
          AND event.published_at IS NULL
          AND event.ordering_sequence = head.current_sequence;
    ELSE
        ALTER TABLE outbox_events
            ADD COLUMN ordering_ready boolean NOT NULL DEFAULT true;
        ALTER TABLE outbox_events
            ALTER COLUMN ordering_ready SET DEFAULT false;

        UPDATE outbox_events AS event
        SET ordering_ready = false
        WHERE event.ordering_key IS NULL
           OR event.published_at IS NOT NULL
           OR NOT EXISTS (
                SELECT 1
                FROM outbox_ordering_heads AS head
                WHERE head.ordering_key = event.ordering_key
                  AND head.current_sequence = event.ordering_sequence
            );
    END IF;
END
$$;
-- +goose StatementEnd

-- Keep the adaptive postcondition visible to static migration parsers.
ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS ordering_ready boolean NOT NULL DEFAULT false;

ALTER TABLE outbox_events
    DROP CONSTRAINT outbox_events_payload_check,
    ADD CONSTRAINT outbox_events_payload_check CHECK (
        octet_length(payload) BETWEEN 1 AND 262144
        AND convert_from(payload, 'UTF8')::json IS NOT NULL
    ),
    DROP CONSTRAINT outbox_events_metadata_check,
    ADD CONSTRAINT outbox_events_metadata_check CHECK (
        octet_length(metadata) BETWEEN 2 AND 32768
        AND convert_from(metadata, 'UTF8')::json IS NOT NULL
        AND ltrim(convert_from(metadata, 'UTF8'), E' \t\r\n') LIKE '{%'
    ),
    DROP CONSTRAINT outbox_events_ordering_check,
    ADD CONSTRAINT outbox_events_ordering_check CHECK (
        (ordering_key IS NULL AND ordering_sequence IS NULL)
        OR (
            ordering_key IS NOT NULL
            AND octet_length(ordering_key) BETWEEN 1 AND 256
            AND ordering_key !~ '[[:cntrl:]]'
            AND ordering_sequence IS NOT NULL
            AND ordering_sequence > 0
        )
    );

ALTER TABLE outbox_ordering_heads
    DROP CONSTRAINT outbox_ordering_heads_sequence_check,
    ADD CONSTRAINT outbox_ordering_heads_sequence_check CHECK (
        last_sequence > 0
        AND (
            current_sequence IS NULL
            OR current_sequence BETWEEN 1 AND last_sequence
        )
    );

DROP INDEX outbox_events_claim_idx;

CREATE INDEX outbox_events_claim_idx
    ON outbox_events (available_at, created_at, id)
    WHERE published_at IS NULL
      AND poisoned_at IS NULL
      AND (ordering_key IS NULL OR ordering_ready);

CREATE UNIQUE INDEX outbox_events_ordering_ready_key
    ON outbox_events (ordering_key)
    WHERE published_at IS NULL
      AND ordering_key IS NOT NULL
      AND ordering_ready;

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    BEGIN
        PERFORM convert_from(payload, 'UTF8')::jsonb
        FROM outbox_events;
        PERFORM convert_from(metadata, 'UTF8')::jsonb
        FROM outbox_events;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'cannot revert outbox JSON constraints: jsonb-incompatible event bytes exist';
    END;
END
$$;
-- +goose StatementEnd

DROP INDEX outbox_events_ordering_ready_key;
DROP INDEX outbox_events_claim_idx;

CREATE INDEX outbox_events_claim_idx
    ON outbox_events (available_at, created_at, id)
    WHERE published_at IS NULL AND poisoned_at IS NULL;

ALTER TABLE outbox_events
    DROP CONSTRAINT outbox_events_payload_check,
    ADD CONSTRAINT outbox_events_payload_check CHECK (
        octet_length(payload) BETWEEN 1 AND 262144
        AND convert_from(payload, 'UTF8')::jsonb IS NOT NULL
    ),
    DROP CONSTRAINT outbox_events_metadata_check,
    ADD CONSTRAINT outbox_events_metadata_check CHECK (
        octet_length(metadata) BETWEEN 2 AND 32768
        AND jsonb_typeof(convert_from(metadata, 'UTF8')::jsonb) = 'object'
    ),
    DROP CONSTRAINT outbox_events_ordering_check,
    ADD CONSTRAINT outbox_events_ordering_check CHECK (
        (ordering_key IS NULL AND ordering_sequence IS NULL)
        OR (
            ordering_key IS NOT NULL
            AND octet_length(ordering_key) BETWEEN 1 AND 256
            AND ordering_key !~ '[[:cntrl:]]'
            AND ordering_sequence IS NOT NULL
            AND ordering_sequence > 0
        )
    ),
    DROP COLUMN ordering_ready;

ALTER TABLE outbox_ordering_heads
    DROP CONSTRAINT outbox_ordering_heads_sequence_check,
    ADD CONSTRAINT outbox_ordering_heads_sequence_check CHECK (last_sequence > 0),
    DROP COLUMN current_sequence;
