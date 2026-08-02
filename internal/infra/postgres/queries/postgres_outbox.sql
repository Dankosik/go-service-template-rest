-- name: AdvanceOutboxOrderingHead :one
INSERT INTO outbox_ordering_heads (ordering_key, last_sequence)
VALUES (sqlc.arg(ordering_key), sqlc.arg(ordering_sequence))
ON CONFLICT (ordering_key) DO UPDATE
SET last_sequence = EXCLUDED.last_sequence,
    updated_at = clock_timestamp()
WHERE outbox_ordering_heads.last_sequence < EXCLUDED.last_sequence
RETURNING last_sequence;

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
    ordering_sequence
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
    sqlc.narg(ordering_sequence)
);

-- name: ClaimOutboxEvent :one
WITH candidate AS (
    SELECT event.id,
           event.lease_expires_at IS NOT NULL AS recovery_due
    FROM outbox_events AS event
    WHERE event.published_at IS NULL
      AND event.poisoned_at IS NULL
      AND event.available_at <= clock_timestamp()
      AND (event.lease_expires_at IS NULL OR event.lease_expires_at <= clock_timestamp())
      AND (
          event.ordering_key IS NULL
          OR NOT EXISTS (
              SELECT 1
              FROM outbox_events AS predecessor
              WHERE predecessor.ordering_key = event.ordering_key
                AND predecessor.ordering_sequence < event.ordering_sequence
                AND predecessor.published_at IS NULL
          )
      )
    ORDER BY event.available_at, event.created_at, event.id
    FOR UPDATE OF event SKIP LOCKED
    LIMIT 1
)
UPDATE outbox_events AS event
SET lease_token = sqlc.arg(lease_token),
    lease_expires_at = clock_timestamp()
        + sqlc.arg(lease_milliseconds)::double precision * interval '1 millisecond',
    cycle_attempt_count = event.cycle_attempt_count + 1,
    total_attempt_count = event.total_attempt_count + 1,
    last_attempt_at = clock_timestamp()
FROM candidate
WHERE event.id = candidate.id
RETURNING event.*, candidate.recovery_due::boolean AS recovery_due;

-- name: GetOutboxEvent :one
SELECT * FROM outbox_events WHERE id = sqlc.arg(id);

-- name: MarkOutboxPublished :execrows
UPDATE outbox_events
SET published_at = clock_timestamp(),
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error_class = NULL
WHERE id = sqlc.arg(id)
  AND lease_token = sqlc.arg(lease_token)
  AND lease_expires_at > clock_timestamp()
  AND published_at IS NULL
  AND poisoned_at IS NULL;

-- name: ScheduleOutboxRetry :execrows
UPDATE outbox_events
SET available_at = clock_timestamp()
        + sqlc.arg(delay_milliseconds)::double precision * interval '1 millisecond',
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error_class = sqlc.arg(error_class)
WHERE id = sqlc.arg(id)
  AND lease_token = sqlc.arg(lease_token)
  AND lease_expires_at > clock_timestamp()
  AND published_at IS NULL
  AND poisoned_at IS NULL;

-- name: MarkOutboxPoisoned :execrows
UPDATE outbox_events
SET poisoned_at = clock_timestamp(),
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error_class = sqlc.arg(error_class)
WHERE id = sqlc.arg(id)
  AND lease_token = sqlc.arg(lease_token)
  AND lease_expires_at > clock_timestamp()
  AND published_at IS NULL
  AND poisoned_at IS NULL;

-- name: FindOutboxRedrive :one
SELECT event_id FROM outbox_redrives WHERE audit_id = sqlc.arg(audit_id);

-- name: LockOutboxEventForRedrive :one
SELECT * FROM outbox_events WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: InsertOutboxRedrive :exec
INSERT INTO outbox_redrives (audit_id, event_id, cycle_number)
VALUES (sqlc.arg(audit_id), sqlc.arg(event_id), sqlc.arg(cycle_number));

-- name: RedriveOutboxEvent :execrows
UPDATE outbox_events
SET available_at = clock_timestamp(),
    cycle_attempt_count = 0,
    lease_token = NULL,
    lease_expires_at = NULL,
    poisoned_at = NULL,
    last_error_class = NULL,
    redrive_count = redrive_count + 1,
    last_redrive_id = sqlc.arg(audit_id),
    last_redriven_at = clock_timestamp()
WHERE id = sqlc.arg(id)
  AND poisoned_at IS NOT NULL
  AND published_at IS NULL;

-- name: CleanupPublishedOutboxEvents :many
WITH expired AS (
    SELECT id
    FROM outbox_events
    WHERE published_at < clock_timestamp()
        - sqlc.arg(retention_milliseconds)::double precision * interval '1 millisecond'
    ORDER BY published_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
DELETE FROM outbox_events AS event
USING expired
WHERE event.id = expired.id
RETURNING event.id;

-- name: ObserveOutbox :one
WITH observation_clock AS (
    SELECT clock_timestamp() AS observed_at
), classified AS (
    SELECT
        event.created_at,
        event.published_at,
        CASE
            WHEN event.published_at IS NOT NULL THEN 'published_retained'
            WHEN event.poisoned_at IS NOT NULL THEN 'poison'
            WHEN event.lease_token IS NOT NULL AND event.lease_expires_at > observation_clock.observed_at THEN 'in_progress'
            WHEN event.lease_token IS NOT NULL THEN 'recovery_due'
            WHEN event.ordering_key IS NOT NULL AND EXISTS (
                SELECT 1
                FROM outbox_events AS predecessor
                WHERE predecessor.ordering_key = event.ordering_key
                  AND predecessor.ordering_sequence < event.ordering_sequence
                  AND predecessor.published_at IS NULL
            ) THEN 'ordering_blocked'
            WHEN event.available_at > observation_clock.observed_at THEN 'retry_wait'
            ELSE 'eligible'
        END AS state
    FROM outbox_events AS event
    CROSS JOIN observation_clock
)
SELECT
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
    )), 0)::double precision AS poison_oldest_unix,
    count(*) FILTER (WHERE state = 'published_retained')::bigint AS published_retained_count,
    coalesce(extract(epoch FROM min(published_at) FILTER (
        WHERE state = 'published_retained'
    )), 0)::double precision AS published_retained_oldest_unix
FROM classified;

-- name: ObserveOutboxStorage :one
SELECT
    (SELECT count(*)::bigint FROM outbox_ordering_heads) AS ordering_head_count,
    pg_total_relation_size('outbox_events')::bigint AS events_bytes,
    pg_indexes_size('outbox_events')::bigint AS events_index_bytes,
    pg_total_relation_size('outbox_ordering_heads')::bigint AS ordering_heads_bytes,
    pg_indexes_size('outbox_ordering_heads')::bigint AS ordering_heads_index_bytes;
