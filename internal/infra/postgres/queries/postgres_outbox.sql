-- name: AdvanceOutboxOrderingHead :one
INSERT INTO outbox_ordering_heads (ordering_key, last_sequence, current_sequence)
VALUES (sqlc.arg(ordering_key), sqlc.arg(ordering_sequence), sqlc.arg(ordering_sequence))
ON CONFLICT (ordering_key) DO UPDATE
SET last_sequence = EXCLUDED.last_sequence,
    current_sequence = coalesce(
        outbox_ordering_heads.current_sequence,
        EXCLUDED.current_sequence
    ),
    updated_at = clock_timestamp()
WHERE outbox_ordering_heads.last_sequence < EXCLUDED.last_sequence
RETURNING current_sequence;

-- name: InsertOutboxEvent :exec
INSERT INTO outbox_events (
    id,
    event_type,
    source,
    destination,
    schema_name,
    occurred_at,
    payload,
    metadata,
    ordering_key,
    ordering_sequence,
    ordering_ready
) VALUES (
    sqlc.arg(id),
    sqlc.arg(event_type),
    sqlc.arg(source),
    sqlc.arg(destination),
    sqlc.arg(schema_name),
    sqlc.arg(occurred_at),
    sqlc.arg(payload),
    sqlc.arg(metadata),
    sqlc.narg(ordering_key),
    sqlc.narg(ordering_sequence),
    sqlc.arg(ordering_ready)
);

-- One statement leases a whole batch under a single token. The partial unique
-- index on ready ordered rows keeps at most one claimable event per ordering
-- key, so a batch never holds two events that must stay ordered relative to
-- each other.
-- name: ClaimOutboxEvents :many
WITH candidate AS (
    SELECT event.id,
           event.lease_expires_at IS NOT NULL AS recovery_due
    FROM outbox_events AS event
    WHERE event.published_at IS NULL
      AND event.poisoned_at IS NULL
      AND (event.ordering_key IS NULL OR event.ordering_ready)
      AND event.available_at <= statement_timestamp()
      AND (event.lease_expires_at IS NULL OR event.lease_expires_at <= statement_timestamp())
    ORDER BY event.available_at, event.created_at, event.id
    FOR UPDATE OF event SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
UPDATE outbox_events AS event
SET lease_token = sqlc.arg(lease_token),
    lease_expires_at = statement_timestamp()
        + sqlc.arg(lease_milliseconds)::double precision * interval '1 millisecond',
    cycle_attempt_count = event.cycle_attempt_count + 1,
    total_attempt_count = event.total_attempt_count + 1,
    last_attempt_at = statement_timestamp()
FROM candidate
WHERE event.id = candidate.id
RETURNING event.*, candidate.recovery_due::boolean AS recovery_due;

-- name: GetOutboxEvent :one
SELECT * FROM outbox_events WHERE id = sqlc.arg(id);

-- name: MarkOutboxPublished :execrows
UPDATE outbox_events
SET published_at = statement_timestamp(),
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error_class = NULL
WHERE id = sqlc.arg(id)
  AND lease_token = sqlc.arg(lease_token)
  AND ordering_key IS NOT DISTINCT FROM sqlc.narg(ordering_key)
  AND ordering_sequence IS NOT DISTINCT FROM sqlc.narg(ordering_sequence)
  AND lease_expires_at > statement_timestamp()
  AND published_at IS NULL
  AND poisoned_at IS NULL;

-- Unordered events finalize together because none of them owns an ordering
-- head. A short row count means at least one lease was lost, which the caller
-- resolves per event against durable state.
-- name: MarkOutboxPublishedBatch :execrows
UPDATE outbox_events AS event
SET published_at = statement_timestamp(),
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error_class = NULL
WHERE event.id = ANY(sqlc.arg(ids)::text[])
  AND event.lease_token = sqlc.arg(lease_token)
  AND event.ordering_key IS NULL
  AND event.lease_expires_at > statement_timestamp()
  AND event.published_at IS NULL
  AND event.poisoned_at IS NULL;

-- name: MarkOrderedOutboxPublished :one
WITH locked_head AS MATERIALIZED (
    SELECT head.ordering_key, head.current_sequence, head.last_sequence
    FROM outbox_ordering_heads AS head
    WHERE head.ordering_key = sqlc.arg(ordering_key)
      AND head.current_sequence = sqlc.arg(ordering_sequence)
    FOR UPDATE
), next_event AS MATERIALIZED (
    SELECT event.id, event.ordering_sequence
    FROM outbox_events AS event
    JOIN locked_head AS head ON head.ordering_key = event.ordering_key
    WHERE event.ordering_sequence > head.current_sequence
      AND event.published_at IS NULL
    ORDER BY event.ordering_sequence
    LIMIT 1
), marked AS (
    UPDATE outbox_events AS event
    SET published_at = statement_timestamp(),
        lease_token = NULL,
        lease_expires_at = NULL,
        last_error_class = NULL
    WHERE event.id = sqlc.arg(id)
      AND event.lease_token = sqlc.arg(lease_token)
      AND event.ordering_key = sqlc.arg(ordering_key)
      AND event.ordering_sequence = sqlc.arg(ordering_sequence)
      AND event.lease_expires_at > statement_timestamp()
      AND event.published_at IS NULL
      AND event.poisoned_at IS NULL
      AND EXISTS (SELECT 1 FROM locked_head)
      AND EXISTS (
          SELECT 1
          FROM locked_head AS head
          WHERE head.current_sequence = head.last_sequence
             OR EXISTS (SELECT 1 FROM next_event)
      )
    RETURNING event.id
), advanced AS (
    UPDATE outbox_ordering_heads AS head
    SET current_sequence = (SELECT ordering_sequence FROM next_event),
        updated_at = statement_timestamp()
    WHERE head.ordering_key = sqlc.arg(ordering_key)
      AND EXISTS (SELECT 1 FROM marked)
    RETURNING head.current_sequence
), unblocked AS (
    UPDATE outbox_events AS event
    SET ordering_ready = true
    WHERE event.id = (SELECT id FROM next_event)
      AND EXISTS (SELECT 1 FROM advanced)
    RETURNING event.id
)
SELECT
    (SELECT count(*) FROM marked)::bigint AS marked_count,
    (SELECT count(*) FROM advanced)::bigint AS advanced_count,
    (SELECT count(*) FROM unblocked)::bigint AS unblocked_count,
    (SELECT current_sequence FROM advanced) AS current_sequence,
    EXISTS (
        SELECT 1
        FROM locked_head AS head
        WHERE head.current_sequence < head.last_sequence
          AND NOT EXISTS (SELECT 1 FROM next_event)
    ) AS snapshot_conflict;

-- Each failed event carries its own jittered delay and error class, so one
-- statement releases the whole failing part of a batch.
-- name: ScheduleOutboxRetryBatch :execrows
UPDATE outbox_events AS event
SET available_at = statement_timestamp()
        + retry.delay_milliseconds * interval '1 millisecond',
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error_class = retry.error_class
FROM (
    SELECT
        unnest(sqlc.arg(ids)::text[]) AS id,
        unnest(sqlc.arg(delay_milliseconds)::double precision[]) AS delay_milliseconds,
        unnest(sqlc.arg(error_classes)::text[]) AS error_class
) AS retry
WHERE event.id = retry.id
  AND event.lease_token = sqlc.arg(lease_token)
  AND event.lease_expires_at > statement_timestamp()
  AND event.published_at IS NULL
  AND event.poisoned_at IS NULL;

-- name: MarkOutboxPoisonedBatch :execrows
UPDATE outbox_events AS event
SET poisoned_at = statement_timestamp(),
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error_class = poison.error_class
FROM (
    SELECT
        unnest(sqlc.arg(ids)::text[]) AS id,
        unnest(sqlc.arg(error_classes)::text[]) AS error_class
) AS poison
WHERE event.id = poison.id
  AND event.lease_token = sqlc.arg(lease_token)
  AND event.lease_expires_at > statement_timestamp()
  AND event.published_at IS NULL
  AND event.poisoned_at IS NULL;

-- name: FindOutboxRedrive :one
SELECT event_id FROM outbox_redrives WHERE audit_id = sqlc.arg(audit_id);

-- name: LockOutboxEventForRedrive :one
SELECT * FROM outbox_events WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: InsertOutboxRedrive :exec
INSERT INTO outbox_redrives (audit_id, event_id, cycle_number)
VALUES (sqlc.arg(audit_id), sqlc.arg(event_id), sqlc.arg(cycle_number));

-- name: RedriveOutboxEvent :execrows
UPDATE outbox_events
SET available_at = statement_timestamp(),
    cycle_attempt_count = 0,
    lease_token = NULL,
    lease_expires_at = NULL,
    poisoned_at = NULL,
    last_error_class = NULL,
    redrive_count = redrive_count + 1,
    last_redrive_id = sqlc.arg(audit_id),
    last_redriven_at = statement_timestamp()
WHERE id = sqlc.arg(id)
  AND poisoned_at IS NOT NULL
  AND published_at IS NULL;

-- name: CleanupPublishedOutboxEvents :execrows
WITH expired AS (
    SELECT id
    FROM outbox_events
    WHERE published_at < statement_timestamp()
        - sqlc.arg(retention_milliseconds)::double precision * interval '1 millisecond'
    ORDER BY published_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
DELETE FROM outbox_events AS event
USING expired
WHERE event.id = expired.id;

-- Backlog states are exact and read only unpublished rows through the pending
-- partial index, so observation cost tracks backlog rather than retention
-- volume. Retained published rows are reported as the planner's own row
-- estimate minus that exact pending count, because counting them would scan
-- the entire retention window on every observation.
-- name: ObserveOutbox :one
WITH pending AS (
    SELECT
        event.created_at,
        CASE
            WHEN event.poisoned_at IS NOT NULL THEN 'poison'
            WHEN event.lease_expires_at > statement_timestamp() THEN 'in_progress'
            WHEN event.lease_expires_at IS NOT NULL THEN 'recovery_due'
            WHEN event.ordering_key IS NOT NULL AND NOT event.ordering_ready THEN 'ordering_blocked'
            WHEN event.available_at > statement_timestamp() THEN 'retry_wait'
            ELSE 'eligible'
        END AS state
    FROM outbox_events AS event
    WHERE event.published_at IS NULL
), backlog AS (
    SELECT
        count(*)::bigint AS pending_count,
        count(*) FILTER (WHERE state = 'eligible')::bigint AS eligible_count,
        coalesce(extract(epoch FROM min(created_at) FILTER (
            WHERE state = 'eligible'
        )), 0)::double precision AS eligible_oldest_unix,
        count(*) FILTER (WHERE state = 'in_progress')::bigint AS in_progress_count,
        coalesce(extract(epoch FROM min(created_at) FILTER (
            WHERE state = 'in_progress'
        )), 0)::double precision AS in_progress_oldest_unix,
        count(*) FILTER (WHERE state = 'retry_wait')::bigint AS retry_wait_count,
        coalesce(extract(epoch FROM min(created_at) FILTER (
            WHERE state = 'retry_wait'
        )), 0)::double precision AS retry_wait_oldest_unix,
        count(*) FILTER (WHERE state = 'recovery_due')::bigint AS recovery_due_count,
        coalesce(extract(epoch FROM min(created_at) FILTER (
            WHERE state = 'recovery_due'
        )), 0)::double precision AS recovery_due_oldest_unix,
        count(*) FILTER (WHERE state = 'ordering_blocked')::bigint AS ordering_blocked_count,
        coalesce(extract(epoch FROM min(created_at) FILTER (
            WHERE state = 'ordering_blocked'
        )), 0)::double precision AS ordering_blocked_oldest_unix,
        count(*) FILTER (WHERE state = 'poison')::bigint AS poison_count,
        coalesce(extract(epoch FROM min(created_at) FILTER (
            WHERE state = 'poison'
        )), 0)::double precision AS poison_oldest_unix
    FROM pending
)
SELECT
    backlog.eligible_count,
    backlog.eligible_oldest_unix,
    backlog.in_progress_count,
    backlog.in_progress_oldest_unix,
    backlog.retry_wait_count,
    backlog.retry_wait_oldest_unix,
    backlog.recovery_due_count,
    backlog.recovery_due_oldest_unix,
    backlog.ordering_blocked_count,
    backlog.ordering_blocked_oldest_unix,
    backlog.poison_count,
    backlog.poison_oldest_unix,
    greatest(
        coalesce((
            SELECT nullif(class.reltuples, -1)
            FROM pg_catalog.pg_class AS class
            WHERE class.oid = 'outbox_events'::regclass
        ), 0) - backlog.pending_count,
        0
    )::bigint AS published_retained_estimate,
    coalesce((
        SELECT extract(epoch FROM min(event.published_at))
        FROM outbox_events AS event
        WHERE event.published_at IS NOT NULL
    ), 0)::double precision AS published_retained_oldest_unix,
    (SELECT count(*)::bigint FROM outbox_ordering_heads) AS ordering_head_count,
    pg_total_relation_size('outbox_events')::bigint AS events_bytes,
    pg_indexes_size('outbox_events')::bigint AS events_index_bytes,
    pg_total_relation_size('outbox_ordering_heads')::bigint AS ordering_heads_bytes,
    pg_indexes_size('outbox_ordering_heads')::bigint AS ordering_heads_index_bytes,
    pg_total_relation_size('outbox_redrives')::bigint AS redrives_bytes,
    pg_indexes_size('outbox_redrives')::bigint AS redrives_index_bytes
FROM backlog;
