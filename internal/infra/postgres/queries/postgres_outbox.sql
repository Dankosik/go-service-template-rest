-- Every event of one append that owns no ordering head, in one statement.
--
-- Each column arrives as one array rather than each event as one statement.
-- That is what makes a multi-event append cheap: PostgreSQL sets up the
-- executor once instead of once per event, and setting it up opens this
-- table's indexes again every time, which costs more than the insert itself.
-- A feature that emits several events per business transaction therefore pays
-- one round trip and one executor setup, and holds its own row locks for that
-- much less time.
--
-- A call carrying no ordering key uses this statement rather than the one
-- below, which is worth a second statement to maintain: the ordering-head
-- insert and its arbiter index are initialized on every execution even when no
-- event touches a head, and that setup costs more than a whole single-event
-- insert. Ordering columns stay at their table defaults here.
-- name: InsertOutboxEvents :exec
WITH receipt AS (
    INSERT INTO outbox_commit_receipts (
        event_id,
        fingerprint_version,
        envelope_fingerprint
    )
    SELECT
        unnest(sqlc.arg(ids)::text[]),
        1,
        unnest(sqlc.arg(envelope_fingerprints)::bytea[])
)
INSERT INTO outbox_events (
    id,
    event_type,
    source,
    destination,
    schema_name,
    occurred_at,
    payload,
    metadata,
    trace_context
)
-- Set-returning functions in one select list advance in lockstep, which is how
-- the column arrays are zipped back into rows.
SELECT
    unnest(sqlc.arg(ids)::text[]),
    unnest(sqlc.arg(event_types)::text[]),
    unnest(sqlc.arg(sources)::text[]),
    unnest(sqlc.arg(destinations)::text[]),
    unnest(sqlc.arg(schema_names)::text[]),
    unnest(sqlc.arg(occurred_ats)::timestamptz[]),
    unnest(sqlc.arg(payloads)::bytea[]),
    unnest(sqlc.arg(metadatas)::bytea[]),
    unnest(sqlc.arg(trace_contexts)::bytea[]);

-- Every event of one append when at least one of them owns an ordering head.
-- It stores the ordered and unordered events together and advances the head of
-- every key the call touches, so a mixed append is still one statement and one
-- round trip.
--
-- The ordering columns carry the same zero value the Go event does: an empty
-- key and a zero sequence mean the event owns no ordering head.
--
-- `head` advances each key's retained high-water mark once per key instead of
-- once per event, so several events for one key take that head's row lock a
-- single time. `last_sequence` is the retained authority that rejects a
-- replayed sequence, and cleanup never deletes a head, so the rejection
-- outlives the events it was derived from. `current_sequence` names the key's
-- earliest unpublished sequence, and `ordering_ready` marks the one event that
-- matches it as the key's claimable head.
--
-- The guard compares the retained mark against the call's *first* new sequence,
-- so a call is accepted only when all of its sequences clear that mark. Within
-- one call the events of a key may arrive in any order; they are stored and
-- published in sequence order either way.
--
-- The statement's own result is the rejection report: one row per key whose
-- first new sequence did not clear its retained mark, and no rows at all on the
-- normal path. One rejected key stores nothing, which is the same all-or-
-- nothing outcome the caller's transaction would produce anyway. A
-- data-modifying CTE runs to completion whether or not the primary query reads
-- its output, so returning only rejections skips no work.
-- name: InsertOutboxEventsWithOrdering :many
WITH input AS (
    SELECT
        unnest(sqlc.arg(ids)::text[]) AS id,
        unnest(sqlc.arg(event_types)::text[]) AS event_type,
        unnest(sqlc.arg(sources)::text[]) AS source,
        unnest(sqlc.arg(destinations)::text[]) AS destination,
        unnest(sqlc.arg(schema_names)::text[]) AS schema_name,
        unnest(sqlc.arg(occurred_ats)::timestamptz[]) AS occurred_at,
        unnest(sqlc.arg(payloads)::bytea[]) AS payload,
        unnest(sqlc.arg(metadatas)::bytea[]) AS metadata,
        unnest(sqlc.arg(trace_contexts)::bytea[]) AS trace_context,
        unnest(sqlc.arg(envelope_fingerprints)::bytea[]) AS envelope_fingerprint,
        unnest(sqlc.arg(ordering_keys)::text[]) AS ordering_key,
        unnest(sqlc.arg(ordering_sequences)::bigint[]) AS ordering_sequence
), event AS (
    SELECT
        input.id,
        input.event_type,
        input.source,
        input.destination,
        input.schema_name,
        input.occurred_at,
        input.payload,
        input.metadata,
        input.trace_context,
        input.envelope_fingerprint,
        nullif(input.ordering_key, '') AS ordering_key,
        nullif(input.ordering_sequence, 0) AS ordering_sequence
    FROM input
), ordered_key AS (
    SELECT
        event.ordering_key,
        min(event.ordering_sequence) AS first_sequence,
        max(event.ordering_sequence) AS last_sequence
    FROM event
    WHERE event.ordering_key IS NOT NULL
    GROUP BY event.ordering_key
), head AS (
    INSERT INTO outbox_ordering_heads AS existing (ordering_key, last_sequence, current_sequence)
    SELECT ordered_key.ordering_key, ordered_key.last_sequence, ordered_key.first_sequence
    FROM ordered_key
    -- Concurrent appends that touch the same keys take their locks in the same
    -- order, so they queue behind each other instead of deadlocking.
    ORDER BY ordered_key.ordering_key
    ON CONFLICT (ordering_key) DO UPDATE
    SET last_sequence = EXCLUDED.last_sequence,
        current_sequence = coalesce(existing.current_sequence, EXCLUDED.current_sequence),
        updated_at = clock_timestamp()
    WHERE existing.last_sequence < EXCLUDED.current_sequence
    RETURNING existing.ordering_key, existing.current_sequence
), rejected AS (
    SELECT ordered_key.ordering_key, ordered_key.first_sequence
    FROM ordered_key
    WHERE NOT EXISTS (
        SELECT 1 FROM head WHERE head.ordering_key = ordered_key.ordering_key
    )
), receipt AS (
    INSERT INTO outbox_commit_receipts (
        event_id,
        fingerprint_version,
        envelope_fingerprint
    )
    SELECT event.id, 1, event.envelope_fingerprint
    FROM event
    WHERE NOT EXISTS (SELECT 1 FROM rejected)
), stored AS (
    INSERT INTO outbox_events (
        id,
        event_type,
        source,
        destination,
        schema_name,
        occurred_at,
        payload,
        metadata,
        trace_context,
        ordering_key,
        ordering_sequence,
        ordering_ready
    )
    SELECT
        event.id,
        event.event_type,
        event.source,
        event.destination,
        event.schema_name,
        event.occurred_at,
        event.payload,
        event.metadata,
        event.trace_context,
        event.ordering_key,
        event.ordering_sequence,
        event.ordering_key IS NOT NULL AND head.current_sequence = event.ordering_sequence
    FROM event
    LEFT JOIN head ON head.ordering_key = event.ordering_key
    WHERE NOT EXISTS (SELECT 1 FROM rejected)
)
SELECT
    rejected.ordering_key::text AS ordering_key,
    rejected.first_sequence::bigint AS first_sequence
FROM rejected
ORDER BY rejected.ordering_key;

-- One writer-pool read distinguishes durable evidence from an authoritative
-- absence. A missing receipt authorizes retry only when this same statement is
-- running on a writable PostgreSQL primary.
-- name: ReconcileOutboxCommit :one
SELECT
    (receipt.event_id IS NOT NULL)::boolean AS receipt_exists,
    coalesce(receipt.fingerprint_version, 0)::smallint AS fingerprint_version,
    coalesce(receipt.envelope_fingerprint, '\x'::bytea) AS envelope_fingerprint,
    (NOT pg_is_in_recovery()
        AND current_setting('transaction_read_only') = 'off')::boolean AS writer_primary
FROM (SELECT sqlc.arg(event_id)::text AS event_id) AS input
LEFT JOIN outbox_commit_receipts AS receipt USING (event_id);

-- Retire the retained high-water mark of every terminal ordering key a call
-- names, in one statement, inside the caller's own transaction.
--
-- Heads are locked in key order, the same rows in the same order the ordered
-- append takes, which is the whole concurrency argument: an append for one of
-- these keys serializes either before this statement -- and then its unpublished
-- event puts the key in `active`, so the mark stays -- or after it, and then it
-- establishes a fresh mark from its own sequence. Neither can interleave with
-- the check. The locked head's current_sequence is the pending-work authority:
-- after waiting for a concurrent append, PostgreSQL refreshes that locked row
-- to the committed version. A second-table lookup would remain on the
-- statement's older READ COMMITTED snapshot and could miss the appended event.
--
-- A key with no head is neither locked, refused, nor deleted, which is what
-- makes a repeated or unknown retirement a no-op rather than an error.
--
-- The result is the rejection report, the shape the ordered append already uses:
-- one row per key that still owns unpublished events, and no rows on the normal
-- path. `retired` is unreferenced on purpose -- a data-modifying CTE runs to
-- completion whether or not the primary query reads its output.
--
-- name: RetireOutboxOrderingKeys :many
WITH locked AS MATERIALIZED (
    SELECT head.ordering_key, head.current_sequence
    FROM outbox_ordering_heads AS head
    WHERE head.ordering_key = ANY(sqlc.arg(ordering_keys)::text[])
    ORDER BY head.ordering_key
    FOR UPDATE
), active AS MATERIALIZED (
    SELECT locked.ordering_key
    FROM locked
    WHERE locked.current_sequence IS NOT NULL
), retired AS (
    DELETE FROM outbox_ordering_heads AS head
    USING locked
    WHERE head.ordering_key = locked.ordering_key
      AND NOT EXISTS (
          SELECT 1 FROM active WHERE active.ordering_key = head.ordering_key
      )
    RETURNING head.ordering_key
)
SELECT active.ordering_key::text AS ordering_key
FROM active
ORDER BY active.ordering_key;

-- One statement leases a whole batch under a single token. The partial unique
-- index on ready ordered rows keeps at most one claimable event per ordering
-- key, so a batch never holds two events that must stay ordered relative to
-- each other.
--
-- The update reaches each candidate by the physical address the select already
-- pinned rather than by its id, which is what keeps a claim proportional to the
-- batch instead of to the table. Matching on the id leaves the planner to
-- choose between one index descent per row and a sequential scan of every
-- retained row, and it takes the scan while the batch is a large enough share
-- of the estimated rows -- so a claim would cost what the retention window
-- costs. Measured on 100 claimed events: 4.6 ms against 10,000 retained rows
-- and 2.3 ms against 210,000, versus 1.6 ms either way here, because a TID scan
-- reads no index at all and cannot be planned another way.
--
-- The address holds only because `FOR UPDATE` holds every candidate row for the
-- rest of this statement: a locked row cannot be updated or deleted, so it
-- cannot move. It is valid inside this statement alone, and only while events
-- live in one table -- partitioning would have to return to the id. The `::tid`
-- cast is a no-op that lets sqlc, whose catalog carries no system columns,
-- type the projection.
-- name: ClaimOutboxEvents :many
WITH candidate AS (
    SELECT event.id,
           event.ctid::tid AS row_address,
           event.lease_expires_at IS NOT NULL AS recovery_due,
           event.publication_uncertain IS TRUE
               OR event.lease_token IS NOT NULL
               OR (
                   event.publication_uncertain IS NULL
                   AND event.total_attempt_count > 0
               ) AS publication_uncertain,
           (
               event.publication_uncertain IS TRUE
               OR event.lease_token IS NOT NULL
               OR (
                   event.publication_uncertain IS NULL
                   AND event.total_attempt_count > 0
               )
           )
               AND event.cycle_attempt_count >= sqlc.arg(max_attempts)::integer
               AS terminalize
    FROM outbox_events AS event
    WHERE event.published_at IS NULL
      AND event.poisoned_at IS NULL
      AND (event.ordering_key IS NULL OR event.ordering_ready)
      AND event.available_at <= statement_timestamp()
      AND (event.lease_expires_at IS NULL OR event.lease_expires_at <= statement_timestamp())
    ORDER BY event.available_at, event.created_at, event.id
    FOR UPDATE OF event SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
), transitioned AS (
    UPDATE outbox_events AS event
    SET publication_uncertain = candidate.publication_uncertain,
        lease_token = CASE
            WHEN candidate.terminalize THEN NULL
            ELSE sqlc.arg(lease_token)
        END,
        lease_expires_at = CASE
            WHEN candidate.terminalize THEN NULL
            ELSE statement_timestamp()
                + sqlc.arg(lease_milliseconds)::double precision * interval '1 millisecond'
        END,
        cycle_attempt_count = CASE
            WHEN candidate.terminalize THEN event.cycle_attempt_count
            ELSE event.cycle_attempt_count + 1
        END,
        total_attempt_count = CASE
            WHEN candidate.terminalize THEN event.total_attempt_count
            ELSE event.total_attempt_count + 1
        END,
        last_attempt_at = CASE
            WHEN candidate.terminalize THEN event.last_attempt_at
            ELSE statement_timestamp()
        END,
        poisoned_at = CASE
            WHEN candidate.terminalize THEN statement_timestamp()
            ELSE event.poisoned_at
        END,
        last_error_class = CASE
            WHEN candidate.terminalize THEN 'outcome_unknown'
            ELSE event.last_error_class
        END
    FROM candidate
    WHERE event.ctid = candidate.row_address
-- Projecting the envelope plus the attempt counters keeps decoding proportional
-- to what the relay publishes and fences on. trace_context belongs to that set:
-- the adapter is handed it to put on broker headers, and the relay links its
-- publish span to it. The lease, retry, terminal, and
-- redrive columns it would otherwise decode per event are already known to the
-- caller or belong to operator inspection through GetOutboxEvent.
    RETURNING event.id,
          event.event_type,
          event.source,
          event.destination,
          event.schema_name,
          event.occurred_at,
          event.payload,
          event.metadata,
          event.trace_context,
          event.ordering_key,
          event.ordering_sequence,
          event.cycle_attempt_count,
          event.total_attempt_count,
          event.publication_uncertain
)
SELECT
    transitioned.id,
    transitioned.event_type,
    transitioned.source,
    transitioned.destination,
    transitioned.schema_name,
    transitioned.occurred_at,
    transitioned.payload,
    transitioned.metadata,
    transitioned.trace_context,
    transitioned.ordering_key,
    transitioned.ordering_sequence,
    transitioned.cycle_attempt_count,
    transitioned.total_attempt_count,
    candidate.recovery_due::boolean AS recovery_due,
    transitioned.publication_uncertain
FROM transitioned
JOIN candidate USING (id)
WHERE NOT candidate.terminalize;

-- name: GetOutboxEvent :one
SELECT * FROM outbox_events WHERE id = sqlc.arg(id);

-- Unordered events finalize together because none of them owns an ordering
-- head. Returning the finalized ids lets the caller resolve only the events
-- this statement could not claim against durable state, instead of re-checking
-- the whole batch because one lease was lost.
-- name: MarkOutboxPublishedBatch :many
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
  AND event.poisoned_at IS NULL
RETURNING event.id;

-- Ordered acknowledgements of one lease finalize together. Each one also
-- advances its own key's head and unblocks that key's successor, and those
-- effects are independent across keys, so one statement covers the whole batch.
-- The partial unique index on ready ordered rows means a lease never holds two
-- events for the same key, so every key here has at most one input row and each
-- head is advanced at most once.
--
-- Heads are locked in key order to keep concurrent relays that recovered
-- overlapping leases from waiting on each other in opposite orders. Every input
-- row reports its own outcome: `marked` finalized it, `snapshot_conflict` means
-- an appended successor is not yet visible in this snapshot and the caller
-- should retry, and neither means the lease no longer owns the event.
-- name: MarkOrderedOutboxPublishedBatch :many
WITH claim AS (
    SELECT
        unnest(sqlc.arg(ids)::text[]) AS id,
        unnest(sqlc.arg(ordering_keys)::text[]) AS ordering_key,
        unnest(sqlc.arg(ordering_sequences)::bigint[]) AS ordering_sequence
), locked_head AS MATERIALIZED (
    SELECT head.ordering_key, head.current_sequence, head.last_sequence
    FROM outbox_ordering_heads AS head
    WHERE head.ordering_key = ANY(sqlc.arg(ordering_keys)::text[])
    ORDER BY head.ordering_key
    FOR UPDATE
), successor AS MATERIALIZED (
    SELECT head.ordering_key, next_event.id, next_event.ordering_sequence
    FROM locked_head AS head
    CROSS JOIN LATERAL (
        SELECT event.id, event.ordering_sequence
        FROM outbox_events AS event
        WHERE event.ordering_key = head.ordering_key
          AND event.ordering_sequence > head.current_sequence
          AND event.published_at IS NULL
        ORDER BY event.ordering_sequence
        LIMIT 1
    ) AS next_event
), eligible AS (
    SELECT claim.id, claim.ordering_key, claim.ordering_sequence
    FROM claim
    JOIN locked_head AS head
      ON head.ordering_key = claim.ordering_key
     AND head.current_sequence = claim.ordering_sequence
    WHERE head.current_sequence = head.last_sequence
       OR EXISTS (
           SELECT 1 FROM successor
           WHERE successor.ordering_key = head.ordering_key
       )
), marked AS (
    UPDATE outbox_events AS event
    SET published_at = statement_timestamp(),
        lease_token = NULL,
        lease_expires_at = NULL,
        last_error_class = NULL
    FROM eligible
    WHERE event.id = eligible.id
      AND event.ordering_key = eligible.ordering_key
      AND event.ordering_sequence = eligible.ordering_sequence
      AND event.lease_token = sqlc.arg(lease_token)
      AND event.lease_expires_at > statement_timestamp()
      AND event.published_at IS NULL
      AND event.poisoned_at IS NULL
    RETURNING event.id, eligible.ordering_key
), advanced AS (
    UPDATE outbox_ordering_heads AS head
    SET current_sequence = (
            SELECT successor.ordering_sequence
            FROM successor
            WHERE successor.ordering_key = head.ordering_key
        ),
        updated_at = statement_timestamp()
    FROM marked
    WHERE head.ordering_key = marked.ordering_key
    RETURNING head.ordering_key
), unblocked AS (
    UPDATE outbox_events AS event
    SET ordering_ready = true
    FROM successor
    JOIN advanced ON advanced.ordering_key = successor.ordering_key
    WHERE event.id = successor.id
    RETURNING event.id
)
SELECT
    claim.id::text AS id,
    EXISTS (SELECT 1 FROM marked WHERE marked.id = claim.id) AS marked,
    EXISTS (
        SELECT 1
        FROM locked_head AS head
        WHERE head.ordering_key = claim.ordering_key
          AND head.current_sequence = claim.ordering_sequence
          AND head.current_sequence < head.last_sequence
          AND NOT EXISTS (
              SELECT 1 FROM successor
              WHERE successor.ordering_key = head.ordering_key
          )
    ) AS snapshot_conflict
FROM claim;

-- Each failed event carries its own jittered delay and error class, so one
-- statement releases the whole failing part of a batch.
--
-- `not_attempted` returns the attempt the claim charged. A claim counts an
-- attempt up front because a relay that dies mid-publication must still be seen
-- to have tried, but an event the relay never handed to the publisher — its
-- batch budget was already spent on earlier events — was not tried at all.
-- Leaving that attempt charged makes a slow broker walk such an event to
-- `max_attempts` and quarantine it as `outcome_unknown` without one publication
-- ever having been attempted, so the counter is given back and the attempt cap
-- keeps meaning attempts actually made. `greatest(...,0)` only guards the
-- counters' own CHECK; the claim that produced this lease always incremented
-- both.
--
-- `last_attempt_at` deliberately keeps the timestamp the claim wrote. Its
-- previous value is gone by the time this statement runs, and an operator
-- reading it wants to know when this row was last picked up, which is exactly
-- what it still reports.
-- name: ScheduleOutboxRetryBatch :execrows
UPDATE outbox_events AS event
SET available_at = statement_timestamp()
        + retry.delay_milliseconds * interval '1 millisecond',
    lease_token = NULL,
    lease_expires_at = NULL,
    publication_uncertain = event.publication_uncertain IS TRUE
        OR retry.publication_uncertain,
    cycle_attempt_count = CASE
        WHEN retry.not_attempted THEN greatest(event.cycle_attempt_count - 1, 0)
        ELSE event.cycle_attempt_count
    END,
    total_attempt_count = CASE
        WHEN retry.not_attempted THEN greatest(event.total_attempt_count - 1, 0)
        ELSE event.total_attempt_count
    END,
    last_error_class = retry.error_class
FROM (
    SELECT
        unnest(sqlc.arg(ids)::text[]) AS id,
        unnest(sqlc.arg(delay_milliseconds)::double precision[]) AS delay_milliseconds,
        unnest(sqlc.arg(error_classes)::text[]) AS error_class,
        unnest(sqlc.arg(publication_uncertain)::boolean[]) AS publication_uncertain,
        unnest(sqlc.arg(not_attempted)::boolean[]) AS not_attempted
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
    publication_uncertain = poison.publication_uncertain,
    last_error_class = poison.error_class
FROM (
    SELECT
        unnest(sqlc.arg(ids)::text[]) AS id,
        unnest(sqlc.arg(error_classes)::text[]) AS error_class,
        unnest(sqlc.arg(publication_uncertain)::boolean[]) AS publication_uncertain
) AS poison
WHERE event.id = poison.id
  AND event.lease_token = sqlc.arg(lease_token)
  AND event.lease_expires_at > statement_timestamp()
  AND event.published_at IS NULL
  AND event.poisoned_at IS NULL;

-- name: FindOutboxAction :one
SELECT event_id, action_kind
FROM outbox_redrives
WHERE audit_id = sqlc.arg(audit_id);

-- name: LockOutboxEventForAction :one
SELECT * FROM outbox_events WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: InsertOutboxAction :execrows
INSERT INTO outbox_redrives (audit_id, event_id, action_kind, cycle_number)
VALUES (
    sqlc.arg(audit_id),
    sqlc.arg(event_id),
    sqlc.arg(action_kind),
    sqlc.narg(cycle_number)
)
ON CONFLICT (audit_id) DO NOTHING;

-- name: RedrivePoisonedOutboxEvent :execrows
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
  AND published_at IS NULL
  AND publication_uncertain IS FALSE;

-- name: RedriveUnknownOutboxEvent :execrows
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
  AND published_at IS NULL
  AND poisoned_at IS NOT NULL
  AND publication_uncertain IS TRUE
  AND last_error_class = 'outcome_unknown';

-- name: ConfirmUnorderedOutboxAccepted :execrows
UPDATE outbox_events
SET published_at = statement_timestamp(),
    poisoned_at = NULL,
    lease_token = NULL,
    lease_expires_at = NULL,
    last_error_class = NULL
WHERE id = sqlc.arg(id)
  AND ordering_key IS NULL
  AND published_at IS NULL
  AND poisoned_at IS NOT NULL
  AND publication_uncertain IS TRUE
  AND last_error_class = 'outcome_unknown';

-- name: ConfirmOrderedOutboxAccepted :one
WITH locked_head AS MATERIALIZED (
    SELECT head.ordering_key, head.current_sequence, head.last_sequence
    FROM outbox_ordering_heads AS head
    JOIN outbox_events AS event
      ON event.id = sqlc.arg(id)
     AND event.ordering_key = head.ordering_key
     AND event.ordering_sequence = head.current_sequence
    WHERE event.published_at IS NULL
      AND event.poisoned_at IS NOT NULL
      AND event.publication_uncertain IS TRUE
      AND event.last_error_class = 'outcome_unknown'
    FOR UPDATE OF head
), successor AS MATERIALIZED (
    SELECT next_event.id, next_event.ordering_sequence
    FROM locked_head AS head
    CROSS JOIN LATERAL (
        SELECT event.id, event.ordering_sequence
        FROM outbox_events AS event
        WHERE event.ordering_key = head.ordering_key
          AND event.ordering_sequence > head.current_sequence
          AND event.published_at IS NULL
        ORDER BY event.ordering_sequence
        LIMIT 1
    ) AS next_event
), eligible AS (
    SELECT head.ordering_key
    FROM locked_head AS head
    WHERE head.current_sequence = head.last_sequence
       OR EXISTS (SELECT 1 FROM successor)
), marked AS (
    UPDATE outbox_events AS event
    SET published_at = statement_timestamp(),
        poisoned_at = NULL,
        lease_token = NULL,
        lease_expires_at = NULL,
        last_error_class = NULL
    FROM eligible
    WHERE event.id = sqlc.arg(id)
      AND event.ordering_key = eligible.ordering_key
      AND event.published_at IS NULL
      AND event.poisoned_at IS NOT NULL
      AND event.publication_uncertain IS TRUE
      AND event.last_error_class = 'outcome_unknown'
    RETURNING event.id, eligible.ordering_key
), advanced AS (
    UPDATE outbox_ordering_heads AS head
    SET current_sequence = (SELECT ordering_sequence FROM successor),
        updated_at = statement_timestamp()
    FROM marked
    WHERE head.ordering_key = marked.ordering_key
    RETURNING head.ordering_key
), unblocked AS (
    UPDATE outbox_events AS event
    SET ordering_ready = true
    FROM successor, advanced
    WHERE event.id = successor.id
    RETURNING event.id
)
SELECT
    EXISTS (SELECT 1 FROM marked) AS marked,
    EXISTS (
        SELECT 1
        FROM locked_head AS head
        WHERE head.current_sequence < head.last_sequence
          AND NOT EXISTS (SELECT 1 FROM successor)
    ) AS snapshot_conflict;

-- name: ClassifyLegacyOutboxUncertainty :one
WITH candidate AS MATERIALIZED (
    SELECT event.ctid::tid AS row_address,
           event.published_at IS NULL
               AND (
                   event.total_attempt_count > 0
                   OR event.lease_token IS NOT NULL
                   OR event.poisoned_at IS NOT NULL
               ) AS publication_uncertain,
           event.published_at IS NULL
               AND (
                   event.total_attempt_count > 0
                   OR event.lease_token IS NOT NULL
                   OR event.poisoned_at IS NOT NULL
               )
               AND (
                   event.poisoned_at IS NOT NULL
                   OR event.cycle_attempt_count >= sqlc.arg(max_attempts)::integer
               ) AS terminalize
    FROM outbox_events AS event
    WHERE event.publication_uncertain IS NULL
    ORDER BY event.created_at, event.id
    FOR UPDATE OF event
    LIMIT sqlc.arg(batch_size)
), classified AS (
    UPDATE outbox_events AS event
    SET publication_uncertain = candidate.publication_uncertain,
        poisoned_at = CASE
            WHEN candidate.terminalize
            THEN coalesce(event.poisoned_at, statement_timestamp())
            ELSE event.poisoned_at
        END,
        lease_token = CASE
            WHEN candidate.terminalize THEN NULL
            ELSE event.lease_token
        END,
        lease_expires_at = CASE
            WHEN candidate.terminalize THEN NULL
            ELSE event.lease_expires_at
        END,
        last_error_class = CASE
            WHEN candidate.terminalize THEN 'outcome_unknown'
            ELSE event.last_error_class
        END
    FROM candidate
    WHERE event.ctid = candidate.row_address
    RETURNING 1
)
SELECT count(*)::integer AS classified_count FROM classified;

-- Retention deletes reach their rows by the address the locking select pinned,
-- for the reason the claim does and with the same lock holding it valid. The
-- batch is a fixed size, while matching on the id costs an index descent per
-- row against the whole retention window this statement exists to trim: 16.8 ms
-- against 8.0 ms for 1,000 rows out of 210,000.
-- name: CleanupPublishedOutboxEvents :execrows
WITH expired AS (
    SELECT ctid::tid AS row_address
    FROM outbox_events
    WHERE published_at < statement_timestamp()
        - sqlc.arg(retention_milliseconds)::double precision * interval '1 millisecond'
    ORDER BY published_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
DELETE FROM outbox_events AS event
USING expired
WHERE event.ctid = expired.row_address;

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
            WHEN event.poisoned_at IS NOT NULL
                 AND event.publication_uncertain IS TRUE
                 AND event.last_error_class = 'outcome_unknown'
            THEN 'outcome_unknown'
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
        )), 0)::double precision AS poison_oldest_unix,
        count(*) FILTER (WHERE state = 'outcome_unknown')::bigint AS outcome_unknown_count,
        coalesce(extract(epoch FROM min(created_at) FILTER (
            WHERE state = 'outcome_unknown'
        )), 0)::double precision AS outcome_unknown_oldest_unix
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
    backlog.outcome_unknown_count,
    backlog.outcome_unknown_oldest_unix,
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
    pg_indexes_size('outbox_redrives')::bigint AS redrives_index_bytes,
    pg_total_relation_size('outbox_commit_receipts')::bigint AS receipts_bytes,
    pg_indexes_size('outbox_commit_receipts')::bigint AS receipts_index_bytes
FROM backlog;
