-- name: ListPostgresWebhookRelations :many
SELECT c.relname::text AS relation_name
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema()
  AND c.relkind = 'r'
  AND c.relname = ANY(ARRAY[
    'webhook_clock', 'webhook_destinations', 'webhook_events', 'webhook_fanouts',
    'webhook_deliveries', 'webhook_cycles', 'webhook_attempts', 'webhook_capacity_slots',
    'webhook_destination_tombstones', 'webhook_operator_actions', 'webhook_tombstones'
  ]::text[])
ORDER BY c.relname;

-- name: CheckPostgresWebhookWriter :one
SELECT COALESCE(bool_and(has_table_privilege(current_user, relation_name, 'SELECT,INSERT,UPDATE,DELETE')), false)
       AND has_sequence_privilege(current_user, 'webhook_fairness_sequence', 'USAGE') AS writable
FROM unnest(ARRAY[
    'webhook_clock', 'webhook_destinations', 'webhook_events', 'webhook_fanouts',
    'webhook_deliveries', 'webhook_cycles', 'webhook_attempts', 'webhook_capacity_slots',
    'webhook_destination_tombstones', 'webhook_operator_actions', 'webhook_tombstones'
]::text[]) AS relation_name;

-- name: AdvanceWebhookClock :one
WITH locked AS (
    SELECT high_water, regression FROM webhook_clock WHERE singleton FOR UPDATE
), sampled AS (
    SELECT clock_timestamp() AS sampled_at, high_water, regression FROM locked
), advanced AS (
    UPDATE webhook_clock c
    SET high_water = s.sampled_at, observed_at = s.sampled_at
    FROM sampled s
    WHERE c.singleton AND NOT s.regression AND s.sampled_at >= s.high_water
    RETURNING c.high_water
)
SELECT high_water AS sampled_at FROM advanced;

-- name: ObserveWebhookClock :one
WITH locked AS (
    SELECT high_water FROM webhook_clock WHERE singleton FOR UPDATE
), sampled AS (
    SELECT clock_timestamp() AS sampled_at, high_water FROM locked
)
UPDATE webhook_clock c
SET high_water = GREATEST(c.high_water, s.sampled_at),
    regression = s.sampled_at < c.high_water,
    observed_at = s.sampled_at
FROM sampled s
WHERE c.singleton
RETURNING c.high_water, c.regression, c.observed_at;

-- name: ReadWebhookClock :one
SELECT high_water, regression, observed_at FROM webhook_clock WHERE singleton;

-- name: LockWebhookAdvisoryKey :exec
SELECT pg_advisory_xact_lock(sqlc.arg(lock_key)::bigint);

-- name: LockWebhookCapacityTable :exec
LOCK TABLE webhook_capacity_slots IN ACCESS EXCLUSIVE MODE;

-- name: ReadWebhookCapacity :one
SELECT COALESCE(max(capacity_revision), 0)::bigint AS capacity_revision,
       count(*)::integer AS slot_count,
       count(*) FILTER (WHERE attempt_id IS NOT NULL)::integer AS leased_count,
       count(DISTINCT capacity_revision)::integer AS revision_count
FROM webhook_capacity_slots;

-- name: ClearWebhookCapacity :exec
DELETE FROM webhook_capacity_slots;

-- name: InsertWebhookCapacity :exec
INSERT INTO webhook_capacity_slots (slot_number, capacity_revision)
SELECT generate_series(1, sqlc.arg(slot_count)::integer), sqlc.arg(capacity_revision)::bigint;

-- name: ReadWebhookMaxRequiredSecretRevision :one
SELECT COALESCE(max(required_secret_revision), 0)::bigint
FROM webhook_destinations
WHERE disposition IN ('active', 'automatically_paused');

-- name: ListWebhookSecretBindings :many
SELECT owner_scope, destination_id, required_secret_revision, active_key_reference, policy,
       predecessor_key_reference, predecessor_valid_until
FROM webhook_destinations
WHERE disposition IN ('active', 'automatically_paused')
ORDER BY owner_scope, destination_id, generation;

-- name: ReadWebhookMinimumDeclaredConcurrency :one
SELECT COALESCE(min(global_concurrency), 0)::integer
FROM webhook_destinations
WHERE disposition IN ('active', 'automatically_paused');

-- name: ReadWebhookTombstone :one
SELECT target_kind, target_id, acceptance_id, fanout_snapshot_id, last_semantic_class,
       action_id, action_encoding_version, request_fingerprint, first_disposition,
       deletion_authority, created_at
FROM webhook_tombstones
WHERE owner_scope = sqlc.arg(owner_scope)
  AND ((target_kind = 'namespace' AND target_id = sqlc.arg(owner_scope))
    OR (target_kind = 'event' AND (
        target_id = sqlc.arg(business_event_id)
        OR acceptance_id = sqlc.arg(acceptance_id)
        OR fanout_snapshot_id = sqlc.arg(fanout_snapshot_id)
        OR EXISTS (
            SELECT 1
            FROM jsonb_array_elements(delivery_identities) identity
            WHERE identity->>0 = ANY(sqlc.arg(delivery_ids)::text[])
        )
    )))
ORDER BY target_kind DESC
LIMIT 1;

-- name: ReadWebhookTombstoneAction :one
SELECT target_kind, target_id, acceptance_id, fanout_snapshot_id, last_semantic_class,
       action_id, action_encoding_version, request_fingerprint, first_disposition,
       deletion_authority, created_at
FROM webhook_tombstones
WHERE owner_scope = sqlc.arg(owner_scope) AND action_id = sqlc.arg(action_id);

-- name: InsertWebhookDestination :execrows
INSERT INTO webhook_destinations (
    owner_scope, destination_id, generation, ownership_verification_receipt, url,
    selection_revision, payload_version_preference, signature_profile,
    signing_authority_binding, policy, policy_fingerprint, destination_concurrency,
    global_concurrency, required_secret_revision, active_key_reference, created_at, updated_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(destination_id), sqlc.arg(generation),
    sqlc.arg(ownership_verification_receipt), sqlc.arg(url), sqlc.arg(selection_revision),
    sqlc.arg(payload_version_preference), sqlc.arg(signature_profile),
    sqlc.arg(signing_authority_binding), sqlc.arg(policy), sqlc.arg(policy_fingerprint),
    sqlc.arg(destination_concurrency), sqlc.arg(global_concurrency),
    sqlc.arg(required_secret_revision), sqlc.arg(active_key_reference),
    sqlc.arg(created_at), sqlc.arg(created_at)
)
ON CONFLICT (owner_scope, destination_id, generation) DO NOTHING;

-- name: ReadWebhookDestination :one
SELECT * FROM webhook_destinations
WHERE owner_scope = sqlc.arg(owner_scope)
  AND destination_id = sqlc.arg(destination_id)
  AND generation = sqlc.arg(generation);

-- name: ReadWebhookDestinationTombstone :one
SELECT retired_at FROM webhook_destination_tombstones
WHERE owner_scope = sqlc.arg(owner_scope)
  AND destination_id = sqlc.arg(destination_id)
  AND generation = sqlc.arg(generation);

-- name: InsertWebhookEvent :execrows
INSERT INTO webhook_events (
    owner_scope, business_event_id, acceptance_id, fanout_snapshot_id, event_type,
    business_schema_version, content_type, body, delivery_envelope_version,
    subscriber_policy_revision, intent_fingerprint, retention_policy_identity, accepted_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(business_event_id), sqlc.arg(acceptance_id),
    sqlc.arg(fanout_snapshot_id), sqlc.arg(event_type), sqlc.arg(business_schema_version),
    sqlc.arg(content_type), sqlc.arg(body), sqlc.arg(delivery_envelope_version),
    sqlc.arg(subscriber_policy_revision), sqlc.arg(intent_fingerprint),
    sqlc.arg(retention_policy_identity), sqlc.arg(accepted_at)
)
ON CONFLICT DO NOTHING;

-- name: InsertWebhookFanout :execrows
INSERT INTO webhook_fanouts (
    owner_scope, fanout_snapshot_id, business_event_id, member_count, member_fingerprint, accepted_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(fanout_snapshot_id), sqlc.arg(business_event_id),
    sqlc.arg(member_count), sqlc.arg(member_fingerprint), sqlc.arg(accepted_at)
)
ON CONFLICT DO NOTHING;

-- name: InsertWebhookDelivery :execrows
INSERT INTO webhook_deliveries (
    owner_scope, delivery_id, business_event_id, fanout_snapshot_id, destination_id,
    destination_generation, url_snapshot, policy_snapshot, next_due_at,
    redrive_eligible_until, payload_retained_until, active_retained_until,
    terminal_summary_retained_until, attempt_retained_until, action_retained_until,
    destination_generation_retained_until, receiver_dedup_retained_until,
    created_at, updated_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(delivery_id), sqlc.arg(business_event_id),
    sqlc.arg(fanout_snapshot_id), sqlc.arg(destination_id), sqlc.arg(destination_generation),
    sqlc.arg(url_snapshot), sqlc.arg(policy_snapshot), sqlc.arg(next_due_at),
    sqlc.arg(redrive_eligible_until), sqlc.arg(payload_retained_until),
    sqlc.arg(active_retained_until), sqlc.arg(terminal_summary_retained_until),
    sqlc.arg(attempt_retained_until), sqlc.arg(action_retained_until),
    sqlc.arg(destination_generation_retained_until), sqlc.arg(receiver_dedup_retained_until),
    sqlc.arg(created_at), sqlc.arg(created_at)
)
ON CONFLICT DO NOTHING;

-- name: InsertWebhookCycle :execrows
INSERT INTO webhook_cycles (
    owner_scope, delivery_id, cycle_number, cycle_kind, accepted_at, deadline_at, maximum_attempts
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(delivery_id), sqlc.arg(cycle_number),
    sqlc.arg(cycle_kind), sqlc.arg(accepted_at), sqlc.arg(deadline_at), sqlc.arg(maximum_attempts)
)
ON CONFLICT DO NOTHING;

-- name: ReadWebhookAcceptance :one
SELECT e.business_event_id, e.acceptance_id, e.fanout_snapshot_id, e.intent_fingerprint,
       f.member_count, f.member_fingerprint, e.accepted_at
FROM webhook_events e
JOIN webhook_fanouts f USING (owner_scope, fanout_snapshot_id)
WHERE e.owner_scope = sqlc.arg(owner_scope)
  AND e.acceptance_id = sqlc.arg(acceptance_id);

-- name: ListWebhookAcceptanceDeliveries :many
SELECT d.delivery_id, d.destination_id, d.destination_generation, d.current_cycle,
       c.accepted_at, c.deadline_at, c.maximum_attempts
FROM webhook_deliveries d
JOIN webhook_cycles c
  ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id AND c.cycle_number = 0
WHERE d.owner_scope = sqlc.arg(owner_scope)
  AND d.fanout_snapshot_id = sqlc.arg(fanout_snapshot_id)
ORDER BY d.destination_id, d.destination_generation;

-- name: ListWebhookClaimDestinations :many
SELECT d.*
FROM webhook_destinations d
WHERE d.disposition = 'active'
  AND d.required_secret_revision <= sqlc.arg(manifest_revision)
  AND EXISTS (
      SELECT 1 FROM webhook_deliveries w
      JOIN webhook_cycles c
        ON c.owner_scope = w.owner_scope AND c.delivery_id = w.delivery_id AND c.cycle_number = w.current_cycle
      WHERE w.owner_scope = d.owner_scope
        AND w.destination_id = d.destination_id
        AND w.destination_generation = d.generation
        AND w.state IN ('ready', 'scheduled')
        AND w.sendable
        AND w.next_due_at <= sqlc.arg(sampled_at)
        AND c.disposition = 'active' AND c.deadline_at > sqlc.arg(sampled_at)
        AND c.attempts_used < c.maximum_attempts
  )
ORDER BY d.last_considered_sequence, d.owner_scope, d.destination_id, d.generation
LIMIT sqlc.arg(page_size)
FOR UPDATE SKIP LOCKED;

-- name: AdvanceWebhookDestinationCursor :execrows
UPDATE webhook_destinations
SET last_considered_sequence = nextval('webhook_fairness_sequence'), updated_at = sqlc.arg(sampled_at)
WHERE owner_scope = sqlc.arg(owner_scope)
  AND destination_id = sqlc.arg(destination_id)
  AND generation = sqlc.arg(generation);

-- name: CountWebhookDestinationAttempts :one
SELECT count(*)::integer
FROM webhook_attempts a
JOIN webhook_deliveries d
  ON d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
WHERE d.owner_scope = sqlc.arg(owner_scope)
  AND d.destination_id = sqlc.arg(destination_id)
  AND d.destination_generation = sqlc.arg(generation)
  AND a.finalized_at IS NULL
  AND a.lease_expires_at > sqlc.arg(sampled_at);

-- name: LockWebhookDueDelivery :one
SELECT d.*, e.body, e.content_type, c.deadline_at,
       COALESCE((
           SELECT a.retry_delay_ns
           FROM webhook_attempts a
           WHERE a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id
             AND a.cycle_number = d.current_cycle AND a.finalized_at IS NOT NULL
             AND a.retry_delay_ns IS NOT NULL
           ORDER BY a.attempted_at DESC, a.attempt_id DESC
           LIMIT 1
       ), 0)::bigint AS previous_retry_delay_ns
FROM webhook_deliveries d
JOIN webhook_events e
  ON e.owner_scope = d.owner_scope AND e.business_event_id = d.business_event_id
JOIN webhook_cycles c
  ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id AND c.cycle_number = d.current_cycle
WHERE d.owner_scope = sqlc.arg(owner_scope)
  AND d.destination_id = sqlc.arg(destination_id)
  AND d.destination_generation = sqlc.arg(generation)
  AND d.state IN ('ready', 'scheduled') AND d.sendable
  AND d.next_due_at <= sqlc.arg(sampled_at)
  AND e.body IS NOT NULL AND d.payload_retained_until > sqlc.arg(sampled_at)
  AND c.disposition = 'active' AND c.deadline_at > sqlc.arg(sampled_at)
  AND c.attempts_used < c.maximum_attempts
ORDER BY d.next_due_at, d.delivery_id
LIMIT 1
FOR UPDATE OF d SKIP LOCKED;

-- name: LockWebhookCapacitySlot :one
SELECT slot_number
FROM webhook_capacity_slots
WHERE capacity_revision = sqlc.arg(capacity_revision)
  AND attempt_id IS NULL
ORDER BY slot_number
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: ClaimWebhookDelivery :execrows
UPDATE webhook_deliveries
SET state = 'in_flight', lease_owner = sqlc.arg(worker_id), lease_expires_at = sqlc.arg(lease_expires_at),
    fence = fence + 1, updated_at = sqlc.arg(sampled_at)
WHERE owner_scope = sqlc.arg(owner_scope) AND delivery_id = sqlc.arg(delivery_id)
  AND state IN ('ready', 'scheduled') AND fence = sqlc.arg(previous_fence);

-- name: InsertWebhookAttempt :exec
INSERT INTO webhook_attempts (
    owner_scope, delivery_id, cycle_number, attempt_id, fence, capacity_slot,
    attempted_at, lease_expires_at, payload_digest, payload_bytes
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(delivery_id), sqlc.arg(cycle_number), sqlc.arg(attempt_id),
    sqlc.arg(fence), sqlc.arg(capacity_slot), sqlc.arg(attempted_at), sqlc.arg(lease_expires_at),
    sqlc.arg(payload_digest), sqlc.arg(payload_bytes)
);

-- name: IncrementWebhookCycleAttempts :execrows
UPDATE webhook_cycles
SET attempts_used = attempts_used + 1
WHERE owner_scope = sqlc.arg(owner_scope) AND delivery_id = sqlc.arg(delivery_id)
  AND cycle_number = sqlc.arg(cycle_number) AND disposition = 'active'
  AND attempts_used < maximum_attempts;

-- name: LeaseWebhookCapacitySlot :execrows
UPDATE webhook_capacity_slots
SET owner_scope = sqlc.arg(owner_scope), delivery_id = sqlc.arg(delivery_id),
    cycle_number = sqlc.arg(cycle_number), attempt_id = sqlc.arg(attempt_id),
    lease_expires_at = sqlc.arg(lease_expires_at), fence = sqlc.arg(fence)
WHERE slot_number = sqlc.arg(slot_number) AND capacity_revision = sqlc.arg(capacity_revision)
  AND attempt_id IS NULL;

-- name: AuthorizeWebhookAttempt :execrows
UPDATE webhook_attempts a
SET key_reference = sqlc.arg(key_reference), key_references = sqlc.arg(key_references),
    signature_header_digest = sqlc.arg(signature_header_digest),
    dns_set_digest = sqlc.arg(dns_set_digest), selected_address = sqlc.arg(selected_address),
    send_authorized = true, may_have_sent = true
FROM webhook_deliveries d, webhook_cycles c, webhook_destinations g, webhook_capacity_slots s
WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.delivery_id = sqlc.arg(delivery_id)
  AND a.cycle_number = sqlc.arg(cycle_number) AND a.attempt_id = sqlc.arg(attempt_id)
  AND a.fence = sqlc.arg(fence) AND a.finalized_at IS NULL
  AND a.lease_expires_at > sqlc.arg(sampled_at)
  AND d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
  AND d.state = 'in_flight' AND d.fence = a.fence AND d.sendable
  AND d.lease_expires_at > sqlc.arg(sampled_at)
  AND c.owner_scope = a.owner_scope AND c.delivery_id = a.delivery_id
  AND c.cycle_number = a.cycle_number AND c.disposition = 'active'
  AND c.deadline_at > sqlc.arg(sampled_at)
  AND g.owner_scope = d.owner_scope AND g.destination_id = d.destination_id
  AND g.generation = d.destination_generation AND g.disposition = 'active'
  AND g.control_revision = sqlc.arg(control_revision)
  AND g.key_state_revision = sqlc.arg(key_state_revision)
  AND g.required_secret_revision <= sqlc.arg(manifest_revision)
  AND s.slot_number = a.capacity_slot AND s.capacity_revision = sqlc.arg(capacity_revision)
  AND s.owner_scope = a.owner_scope AND s.delivery_id = a.delivery_id
  AND s.cycle_number = a.cycle_number AND s.attempt_id = a.attempt_id
  AND s.fence = a.fence AND s.lease_expires_at > sqlc.arg(sampled_at);

-- name: LockWebhookSendBarrier :one
SELECT g.control_revision, g.key_state_revision, g.required_secret_revision,
       g.active_key_reference, g.predecessor_key_reference, g.predecessor_valid_until,
       d.destination_id, d.destination_generation, a.capacity_slot
FROM webhook_attempts a
JOIN webhook_deliveries d
  ON d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
JOIN webhook_cycles c
  ON c.owner_scope = a.owner_scope AND c.delivery_id = a.delivery_id AND c.cycle_number = a.cycle_number
JOIN webhook_destinations g
  ON g.owner_scope = d.owner_scope AND g.destination_id = d.destination_id
 AND g.generation = d.destination_generation
JOIN webhook_capacity_slots s
  ON s.slot_number = a.capacity_slot AND s.capacity_revision = sqlc.arg(capacity_revision)
 AND s.owner_scope = a.owner_scope AND s.delivery_id = a.delivery_id
 AND s.cycle_number = a.cycle_number AND s.attempt_id = a.attempt_id
 AND s.fence = a.fence
WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.delivery_id = sqlc.arg(delivery_id)
  AND a.cycle_number = sqlc.arg(cycle_number) AND a.attempt_id = sqlc.arg(attempt_id)
  AND a.fence = sqlc.arg(fence) AND a.finalized_at IS NULL
  AND a.lease_expires_at > sqlc.arg(sampled_at)
  AND d.lease_expires_at > sqlc.arg(sampled_at)
  AND c.disposition = 'active' AND c.deadline_at > sqlc.arg(sampled_at)
  AND s.lease_expires_at > sqlc.arg(sampled_at)
FOR UPDATE OF g, d, s, a, c;

-- name: LockWebhookFinalization :one
SELECT a.send_authorized, a.may_have_sent, a.capacity_slot,
       d.cumulative_summary, d.state, d.fence,
       c.attempts_used, c.maximum_attempts, c.deadline_at
FROM webhook_attempts a
JOIN webhook_deliveries d
  ON d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
JOIN webhook_cycles c
  ON c.owner_scope = a.owner_scope AND c.delivery_id = a.delivery_id AND c.cycle_number = a.cycle_number
JOIN webhook_capacity_slots s
  ON s.slot_number = a.capacity_slot AND s.capacity_revision = sqlc.arg(capacity_revision)
 AND s.owner_scope = a.owner_scope
 AND s.delivery_id = a.delivery_id AND s.cycle_number = a.cycle_number
 AND s.attempt_id = a.attempt_id AND s.fence = a.fence
WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.delivery_id = sqlc.arg(delivery_id)
  AND a.cycle_number = sqlc.arg(cycle_number) AND a.attempt_id = sqlc.arg(attempt_id)
  AND a.fence = sqlc.arg(fence) AND a.finalized_at IS NULL
FOR UPDATE OF d, s, c, a;

-- name: FinalizeWebhookAttempt :one
WITH finalized AS (
    UPDATE webhook_attempts a
    SET response_header_bytes = sqlc.narg(response_header_bytes),
        response_body_bytes = sqlc.narg(response_body_bytes),
        response_status = sqlc.narg(response_status), retry_after = NULL,
        retry_after_delay_ns = sqlc.narg(retry_after_delay_ns),
        retry_delay_ns = sqlc.narg(retry_delay_ns),
        outcome_class = sqlc.arg(outcome_class), finalized_at = sqlc.arg(finalized_at)
    WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.delivery_id = sqlc.arg(delivery_id)
      AND a.cycle_number = sqlc.arg(cycle_number) AND a.attempt_id = sqlc.arg(attempt_id)
      AND a.fence = sqlc.arg(fence) AND a.finalized_at IS NULL
    RETURNING a.capacity_slot
), released AS (
    UPDATE webhook_capacity_slots s
    SET owner_scope = NULL, delivery_id = NULL, cycle_number = NULL, attempt_id = NULL,
        lease_expires_at = NULL, fence = NULL
    FROM finalized f
    WHERE s.slot_number = f.capacity_slot
      AND s.capacity_revision = sqlc.arg(capacity_revision)
      AND s.owner_scope = sqlc.arg(owner_scope) AND s.delivery_id = sqlc.arg(delivery_id)
      AND s.cycle_number = sqlc.arg(cycle_number) AND s.attempt_id = sqlc.arg(attempt_id)
      AND s.fence = sqlc.arg(fence)
), delivery AS (
    UPDATE webhook_deliveries d
    SET state = sqlc.arg(delivery_state), next_due_at = sqlc.arg(next_due_at),
        lease_owner = NULL, lease_expires_at = NULL, cumulative_summary = sqlc.arg(cumulative_summary),
        terminal_at = sqlc.narg(terminal_at), updated_at = sqlc.arg(finalized_at)
    FROM finalized
    WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.delivery_id = sqlc.arg(delivery_id)
      AND d.fence = sqlc.arg(fence)
    RETURNING d.delivery_id
), cycle AS (
    UPDATE webhook_cycles c
    SET disposition = sqlc.arg(cycle_disposition), finalized_at = sqlc.narg(terminal_at)
    FROM finalized
    WHERE c.owner_scope = sqlc.arg(owner_scope) AND c.delivery_id = sqlc.arg(delivery_id)
      AND c.cycle_number = sqlc.arg(cycle_number) AND c.disposition = 'active'
      AND sqlc.arg(cycle_disposition) <> 'active'
)
SELECT count(*) FROM delivery;

-- name: ListExpiredWebhookAttempts :many
SELECT a.owner_scope, a.delivery_id, a.cycle_number, a.attempt_id, a.fence, a.capacity_slot,
       a.send_authorized, a.may_have_sent, a.attempted_at, a.lease_expires_at,
       c.attempts_used, c.maximum_attempts, c.deadline_at, d.cumulative_summary,
       d.policy_snapshot,
       COALESCE((
           SELECT prior.retry_delay_ns
           FROM webhook_attempts prior
           WHERE prior.owner_scope = a.owner_scope AND prior.delivery_id = a.delivery_id
             AND prior.cycle_number = a.cycle_number AND prior.finalized_at IS NOT NULL
             AND prior.retry_delay_ns IS NOT NULL
           ORDER BY prior.attempted_at DESC, prior.attempt_id DESC
           LIMIT 1
       ), 0)::bigint AS previous_retry_delay_ns
FROM webhook_attempts a
JOIN webhook_cycles c
  ON c.owner_scope = a.owner_scope AND c.delivery_id = a.delivery_id AND c.cycle_number = a.cycle_number
JOIN webhook_deliveries d
  ON d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
WHERE a.finalized_at IS NULL AND a.lease_expires_at <= sqlc.arg(sampled_at)
ORDER BY a.lease_expires_at, a.owner_scope, a.delivery_id, a.cycle_number, a.attempt_id
LIMIT sqlc.arg(batch_size)
FOR UPDATE OF a SKIP LOCKED;

-- name: FinalizeExpiredWebhookCycles :one
WITH candidates AS (
    SELECT c.owner_scope, c.delivery_id, c.cycle_number,
           CASE WHEN d.cumulative_summary = 'outcome_unknown' THEN 'outcome_unknown' ELSE 'attempts_exhausted' END AS disposition
    FROM webhook_cycles c
    JOIN webhook_deliveries d
      ON d.owner_scope = c.owner_scope AND d.delivery_id = c.delivery_id AND d.current_cycle = c.cycle_number
    WHERE c.disposition = 'active'
      AND d.state IN ('ready', 'scheduled')
      AND (c.deadline_at <= sqlc.arg(sampled_at) OR c.attempts_used >= c.maximum_attempts)
      AND NOT EXISTS (
          SELECT 1 FROM webhook_attempts a
          WHERE a.owner_scope = c.owner_scope AND a.delivery_id = c.delivery_id
            AND a.cycle_number = c.cycle_number AND a.finalized_at IS NULL
      )
    ORDER BY c.deadline_at, c.owner_scope, c.delivery_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF c, d SKIP LOCKED
), cycles AS (
    UPDATE webhook_cycles c
    SET disposition = candidates.disposition, finalized_at = sqlc.arg(sampled_at)
    FROM candidates
    WHERE c.owner_scope = candidates.owner_scope AND c.delivery_id = candidates.delivery_id
      AND c.cycle_number = candidates.cycle_number AND c.disposition = 'active'
    RETURNING c.owner_scope, c.delivery_id, c.cycle_number, c.disposition
), deliveries AS (
    UPDATE webhook_deliveries d
    SET state = CASE WHEN cycles.disposition = 'outcome_unknown' THEN 'suspended' ELSE 'terminal' END,
        cumulative_summary = cycles.disposition, sendable = false, next_due_at = sqlc.arg(sampled_at),
        lease_owner = NULL, lease_expires_at = NULL, terminal_at = sqlc.arg(sampled_at), updated_at = sqlc.arg(sampled_at)
    FROM cycles
    WHERE d.owner_scope = cycles.owner_scope AND d.delivery_id = cycles.delivery_id
      AND d.current_cycle = cycles.cycle_number
    RETURNING 1
)
SELECT count(*) FROM deliveries;

-- name: ReleaseExpiredWebhookOrphanCapacity :one
WITH candidates AS (
    SELECT s.slot_number
    FROM webhook_capacity_slots s
    WHERE s.capacity_revision = sqlc.arg(capacity_revision)
      AND s.attempt_id IS NOT NULL AND s.lease_expires_at <= sqlc.arg(sampled_at)
      AND NOT EXISTS (
          SELECT 1 FROM webhook_attempts a
          WHERE a.owner_scope = s.owner_scope AND a.delivery_id = s.delivery_id
            AND a.cycle_number = s.cycle_number AND a.attempt_id = s.attempt_id
            AND a.fence = s.fence AND a.finalized_at IS NULL
      )
    ORDER BY s.lease_expires_at, s.slot_number
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), released AS (
    UPDATE webhook_capacity_slots s
    SET owner_scope = NULL, delivery_id = NULL, cycle_number = NULL, attempt_id = NULL,
        lease_expires_at = NULL, fence = NULL
    FROM candidates c
    WHERE s.slot_number = c.slot_number
    RETURNING 1
)
SELECT count(*) FROM released;

-- name: QuarantineInconsistentWebhookDeliveries :one
WITH candidates AS (
    SELECT d.owner_scope, d.delivery_id
    FROM webhook_deliveries d
    LEFT JOIN webhook_cycles c
      ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id
     AND c.cycle_number = d.current_cycle
    WHERE d.state <> 'quarantined'
      AND NOT EXISTS (
          SELECT 1 FROM webhook_attempts pending
          WHERE pending.owner_scope = d.owner_scope AND pending.delivery_id = d.delivery_id
            AND pending.finalized_at IS NULL
      )
      AND (
          ((d.active_retained_until > sqlc.arg(sampled_at) OR d.legal_hold) AND (
              c.delivery_id IS NULL
              OR (d.state IN ('ready', 'scheduled', 'in_flight') AND c.disposition <> 'active')
              OR (d.state IN ('terminal', 'suspended') AND c.disposition = 'active')
          ))
          OR ((d.attempt_retained_until > sqlc.arg(sampled_at) OR d.legal_hold)
              AND d.cumulative_summary <> 'http_accepted' AND EXISTS (
              SELECT 1 FROM webhook_attempts accepted
              WHERE accepted.owner_scope = d.owner_scope AND accepted.delivery_id = d.delivery_id
                AND accepted.finalized_at IS NOT NULL AND accepted.outcome_class = 'http_accepted'
          ))
          OR ((d.attempt_retained_until > sqlc.arg(sampled_at) OR d.legal_hold)
              AND d.cumulative_summary IN ('none', 'http_rejected', 'locally_denied', 'attempts_exhausted') AND EXISTS (
              SELECT 1 FROM webhook_attempts ambiguous
              WHERE ambiguous.owner_scope = d.owner_scope AND ambiguous.delivery_id = d.delivery_id
                AND ambiguous.finalized_at IS NOT NULL
                AND (ambiguous.may_have_sent OR ambiguous.outcome_class IN ('transport_ambiguous', 'retryable_http_ambiguous', 'outcome_unknown'))
          ))
      )
    ORDER BY d.updated_at, d.owner_scope, d.delivery_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF d SKIP LOCKED
), quarantined AS (
    UPDATE webhook_deliveries d
    SET state = 'quarantined', sendable = false, lease_owner = NULL,
        lease_expires_at = NULL, terminal_at = COALESCE(d.terminal_at, sqlc.arg(sampled_at)),
        updated_at = sqlc.arg(sampled_at)
    FROM candidates c
    WHERE d.owner_scope = c.owner_scope AND d.delivery_id = c.delivery_id
    RETURNING 1
)
SELECT count(*) FROM quarantined;

-- name: ReadWebhookOperatorAction :one
SELECT actions.*,
       COALESCE(NULLIF(actions.result_cycle, 0), (
           SELECT cycles.cycle_number
           FROM webhook_cycles cycles
           WHERE cycles.owner_scope = actions.owner_scope
             AND cycles.authorizing_action_id = actions.action_id
           ORDER BY cycles.cycle_number DESC
           LIMIT 1
       ), 0)::bigint AS resolved_result_cycle
FROM webhook_operator_actions actions
WHERE actions.owner_scope = sqlc.arg(owner_scope) AND actions.action_id = sqlc.arg(action_id);

-- name: ApplyWebhookDestinationState :execrows
UPDATE webhook_destinations
SET disposition = sqlc.arg(disposition), control_revision = control_revision + 1,
    updated_at = sqlc.arg(updated_at)
WHERE owner_scope = sqlc.arg(owner_scope) AND destination_id = sqlc.arg(destination_id)
  AND generation = sqlc.arg(generation) AND control_revision = sqlc.arg(expected_revision)
  AND disposition <> 'retired';

-- name: ApplyWebhookKeyRotation :execrows
UPDATE webhook_destinations
SET required_secret_revision = sqlc.arg(required_secret_revision),
    key_state_revision = sqlc.arg(key_state_revision), active_key_reference = sqlc.arg(active_key_reference),
    predecessor_key_reference = sqlc.narg(predecessor_key_reference),
    predecessor_valid_until = sqlc.narg(predecessor_valid_until),
    control_revision = control_revision + 1, updated_at = sqlc.arg(updated_at)
WHERE owner_scope = sqlc.arg(owner_scope) AND destination_id = sqlc.arg(destination_id)
  AND generation = sqlc.arg(generation) AND control_revision = sqlc.arg(expected_revision)
  AND required_secret_revision < sqlc.arg(required_secret_revision)
  AND key_state_revision < sqlc.arg(key_state_revision) AND disposition <> 'retired'
  AND active_key_reference = sqlc.arg(predecessor_key_reference);

-- name: LockWebhookDeliveryForAction :one
SELECT d.current_cycle, d.state, d.cumulative_summary, d.redrive_eligible_until,
       d.destination_id, d.destination_generation, d.sendable, c.disposition, c.deadline_at,
       d.payload_retained_until, d.active_retained_until, d.terminal_summary_retained_until,
       d.attempt_retained_until, d.action_retained_until,
       d.destination_generation_retained_until, d.receiver_dedup_retained_until,
       d.legal_hold, d.policy_snapshot, g.disposition AS destination_disposition,
       g.required_secret_revision, g.active_key_reference,
       g.predecessor_key_reference, g.predecessor_valid_until,
       (e.body IS NOT NULL)::boolean AS payload_present
FROM webhook_deliveries d
JOIN webhook_cycles c
  ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id AND c.cycle_number = d.current_cycle
JOIN webhook_destinations g
  ON g.owner_scope = d.owner_scope AND g.destination_id = d.destination_id AND g.generation = d.destination_generation
JOIN webhook_events e
  ON e.owner_scope = d.owner_scope AND e.business_event_id = d.business_event_id
WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.delivery_id = sqlc.arg(delivery_id)
FOR UPDATE OF d, c, g, e;

-- name: InsertWebhookRedriveCycle :execrows
INSERT INTO webhook_cycles (
    owner_scope, delivery_id, cycle_number, cycle_kind, authorizing_action_id,
    accepted_at, deadline_at, maximum_attempts
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(delivery_id), sqlc.arg(cycle_number), 'redrive',
    sqlc.arg(action_id), sqlc.arg(accepted_at), sqlc.arg(deadline_at), sqlc.arg(maximum_attempts)
)
ON CONFLICT DO NOTHING;

-- name: ActivateWebhookRedriveCycle :execrows
UPDATE webhook_deliveries
SET current_cycle = sqlc.arg(cycle_number), state = 'ready', next_due_at = sqlc.arg(accepted_at),
    sendable = true, terminal_at = NULL, updated_at = sqlc.arg(accepted_at)
WHERE owner_scope = sqlc.arg(owner_scope) AND delivery_id = sqlc.arg(delivery_id)
  AND current_cycle = sqlc.arg(previous_cycle) AND state IN ('terminal', 'suspended');

-- name: CloseWebhookUnknownDelivery :execrows
UPDATE webhook_deliveries d
SET state = 'terminal', cumulative_summary = 'closed_unknown', sendable = false,
    terminal_at = sqlc.arg(finalized_at), updated_at = sqlc.arg(finalized_at)
WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.delivery_id = sqlc.arg(delivery_id)
  AND d.current_cycle = sqlc.arg(cycle_number) AND d.cumulative_summary = 'outcome_unknown';

-- name: CloseWebhookUnknownCycle :execrows
UPDATE webhook_cycles
SET disposition = 'closed_unknown', finalized_at = sqlc.arg(finalized_at)
WHERE owner_scope = sqlc.arg(owner_scope) AND delivery_id = sqlc.arg(delivery_id)
  AND cycle_number = sqlc.arg(cycle_number) AND disposition = 'outcome_unknown';

-- name: ApplyWebhookRetentionHold :execrows
UPDATE webhook_deliveries
SET legal_hold = sqlc.arg(legal_hold), updated_at = sqlc.arg(updated_at)
WHERE owner_scope = sqlc.arg(owner_scope) AND delivery_id = sqlc.arg(delivery_id)
  AND current_cycle = sqlc.arg(expected_cycle) AND legal_hold <> sqlc.arg(legal_hold);

-- name: InsertWebhookOperatorAction :execrows
INSERT INTO webhook_operator_actions (
    owner_scope, action_id, encoding_version, request_fingerprint, actor_reference,
    action_kind, target_kind, target_id, target_generation, expected_state, reason,
    duplicate_risk_acknowledged, result, created_at, completed_at, retain_until,
    request_payload, result_cycle
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(action_id), 'webhook-operator-action-v1',
    sqlc.arg(request_fingerprint), sqlc.arg(actor_reference), sqlc.arg(action_kind),
    sqlc.arg(target_kind), sqlc.arg(target_id), sqlc.arg(target_generation),
    sqlc.arg(expected_state), sqlc.arg(reason), sqlc.arg(duplicate_risk_acknowledged),
    sqlc.arg(result), sqlc.arg(created_at), sqlc.arg(created_at), sqlc.arg(retain_until),
    sqlc.arg(request_payload), sqlc.arg(result_cycle)
)
ON CONFLICT DO NOTHING;

-- name: InsertWebhookTombstone :execrows
INSERT INTO webhook_tombstones (
    owner_scope, target_kind, target_id, acceptance_id, fanout_snapshot_id,
    delivery_identities, destination_identities, last_semantic_class, action_id,
    action_encoding_version, request_fingerprint, first_disposition,
    deletion_authority, created_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(target_kind), sqlc.arg(target_id), sqlc.narg(acceptance_id),
    sqlc.narg(fanout_snapshot_id), sqlc.arg(delivery_identities), sqlc.arg(destination_identities),
    sqlc.arg(last_semantic_class), sqlc.arg(action_id), 'webhook-operator-action-v1',
    sqlc.arg(request_fingerprint), sqlc.arg(first_disposition), sqlc.arg(deletion_authority),
    sqlc.arg(created_at)
)
ON CONFLICT DO NOTHING;

-- name: DeleteWebhookEvent :execrows
WITH target_deliveries AS MATERIALIZED (
    SELECT d.delivery_id
    FROM webhook_deliveries d
    WHERE d.owner_scope = sqlc.arg(owner_scope)
      AND d.business_event_id = sqlc.arg(business_event_id)
), deleted_actions AS (
    DELETE FROM webhook_operator_actions a
    WHERE a.owner_scope = sqlc.arg(owner_scope)
      AND (
        (a.target_kind = 'event' AND a.target_id = sqlc.arg(business_event_id))
        OR (a.target_kind = 'delivery' AND EXISTS (
            SELECT 1 FROM target_deliveries d WHERE d.delivery_id = a.target_id
        ))
      )
)
DELETE FROM webhook_events
WHERE webhook_events.owner_scope = sqlc.arg(owner_scope)
  AND webhook_events.business_event_id = sqlc.arg(business_event_id);

-- name: ReadWebhookEventForPrivacy :one
WITH e AS MATERIALIZED (
    SELECT events.acceptance_id, events.fanout_snapshot_id
    FROM webhook_events events
    WHERE events.owner_scope = sqlc.arg(owner_scope) AND events.business_event_id = sqlc.arg(business_event_id)
    FOR UPDATE
), barrier AS MATERIALIZED (
    SELECT a.attempt_id
    FROM e
    JOIN webhook_deliveries d
      ON d.owner_scope = sqlc.arg(owner_scope) AND d.business_event_id = sqlc.arg(business_event_id)
    JOIN webhook_attempts a
      ON a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id AND a.finalized_at IS NULL
    JOIN webhook_destinations g
      ON g.owner_scope = d.owner_scope AND g.destination_id = d.destination_id AND g.generation = d.destination_generation
    JOIN webhook_capacity_slots s
      ON s.owner_scope = a.owner_scope AND s.delivery_id = a.delivery_id
     AND s.cycle_number = a.cycle_number AND s.attempt_id = a.attempt_id AND s.fence = a.fence
    FOR UPDATE OF g, d, s, a
), locked_deliveries AS MATERIALIZED (
    SELECT d.*
    FROM e
    JOIN webhook_deliveries d
      ON d.owner_scope = sqlc.arg(owner_scope) AND d.business_event_id = sqlc.arg(business_event_id)
    WHERE (SELECT count(*) FROM barrier) >= 0
    FOR UPDATE OF d
), locked_attempts AS MATERIALIZED (
    SELECT a.*
    FROM locked_deliveries d
    JOIN webhook_attempts a ON a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id
    FOR UPDATE OF a
)
SELECT e.acceptance_id, e.fanout_snapshot_id,
       COALESCE(jsonb_agg(DISTINCT jsonb_build_array(d.delivery_id)) FILTER (WHERE d.delivery_id IS NOT NULL), '[]') AS delivery_identities,
       COALESCE(jsonb_agg(DISTINCT jsonb_build_array(d.destination_id, d.destination_generation)) FILTER (WHERE d.destination_id IS NOT NULL), '[]') AS destination_identities,
       CASE
         WHEN COALESCE(bool_or(a.send_authorized OR a.may_have_sent), false)
              AND COALESCE(max(d.cumulative_summary), 'none') = 'none' THEN 'outcome_unknown'
         ELSE COALESCE(max(d.cumulative_summary), 'none')::text
       END AS last_semantic_class
FROM e
LEFT JOIN locked_deliveries d ON true
LEFT JOIN locked_attempts a ON a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id
GROUP BY e.acceptance_id, e.fanout_snapshot_id;

-- name: CountWebhookNamespaceRows :one
SELECT (
  (SELECT count(*) FROM webhook_events events WHERE events.owner_scope = sqlc.arg(owner_scope)) +
  (SELECT count(*) FROM webhook_destinations destinations WHERE destinations.owner_scope = sqlc.arg(owner_scope)) +
  (SELECT count(*) FROM webhook_operator_actions actions WHERE actions.owner_scope = sqlc.arg(owner_scope)) +
  (SELECT count(*) FROM webhook_destination_tombstones tombstones WHERE tombstones.owner_scope = sqlc.arg(owner_scope)) +
  (SELECT count(*) FROM webhook_capacity_slots slots WHERE slots.owner_scope = sqlc.arg(owner_scope))
)::bigint;

-- name: LockPendingWebhookNamespaceRetirement :one
SELECT owner_scope, action_id
FROM webhook_tombstones
WHERE target_kind = 'namespace' AND first_disposition = 'pending'
ORDER BY created_at, owner_scope, action_id
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: DeleteWebhookNamespaceActions :one
WITH candidates AS (
    SELECT a.action_id FROM webhook_operator_actions a
    WHERE a.owner_scope = sqlc.arg(owner_scope)
    ORDER BY a.action_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_operator_actions a USING candidates c
    WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.action_id = c.action_id
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: DeleteWebhookNamespaceDestinations :one
WITH candidates AS (
    SELECT d.destination_id, d.generation
    FROM webhook_destinations d
    WHERE d.owner_scope = sqlc.arg(owner_scope)
      AND NOT EXISTS (SELECT 1 FROM webhook_deliveries w WHERE w.owner_scope = d.owner_scope AND w.destination_id = d.destination_id AND w.destination_generation = d.generation)
    ORDER BY d.destination_id, d.generation
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_destinations d USING candidates c
    WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.destination_id = c.destination_id AND d.generation = c.generation
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: DeleteWebhookNamespaceDestinationTombstones :one
WITH candidates AS (
    SELECT t.destination_id, t.generation FROM webhook_destination_tombstones t
    WHERE t.owner_scope = sqlc.arg(owner_scope)
    ORDER BY t.destination_id, t.generation
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_destination_tombstones d USING candidates c
    WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.destination_id = c.destination_id AND d.generation = c.generation
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CompleteWebhookNamespaceTombstone :execrows
UPDATE webhook_tombstones
SET first_disposition = 'completed'
WHERE owner_scope = sqlc.arg(owner_scope) AND target_kind = 'namespace'
  AND target_id = sqlc.arg(owner_scope) AND action_id = sqlc.arg(action_id)
  AND first_disposition = 'pending';

-- name: MarkWebhookNamespacePossibleSend :exec
UPDATE webhook_tombstones
SET last_semantic_class = 'outcome_unknown'
WHERE owner_scope = sqlc.arg(owner_scope) AND target_kind = 'namespace'
  AND target_id = sqlc.arg(owner_scope) AND action_id = sqlc.arg(action_id);

-- name: BackfillWebhookDeliveryRetention :one
WITH candidates AS (
    SELECT d.owner_scope, d.delivery_id
    FROM webhook_deliveries d
    WHERE NOT isfinite(d.payload_retained_until)
       OR NOT isfinite(d.active_retained_until)
       OR NOT isfinite(d.terminal_summary_retained_until)
       OR NOT isfinite(d.attempt_retained_until)
       OR NOT isfinite(d.action_retained_until)
       OR NOT isfinite(d.destination_generation_retained_until)
       OR NOT isfinite(d.receiver_dedup_retained_until)
    ORDER BY d.owner_scope, d.delivery_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), updated AS (
    UPDATE webhook_deliveries d
    SET payload_retained_until = d.created_at + ((d.policy_snapshot->'horizons'->>0)::double precision / 1000000000.0) * interval '1 second',
        active_retained_until = d.created_at + ((d.policy_snapshot->'horizons'->>1)::double precision / 1000000000.0) * interval '1 second',
        terminal_summary_retained_until = d.created_at + ((d.policy_snapshot->'horizons'->>2)::double precision / 1000000000.0) * interval '1 second',
        attempt_retained_until = d.created_at + ((d.policy_snapshot->'horizons'->>3)::double precision / 1000000000.0) * interval '1 second',
        action_retained_until = d.created_at + ((d.policy_snapshot->'horizons'->>4)::double precision / 1000000000.0) * interval '1 second',
        destination_generation_retained_until = d.created_at + ((d.policy_snapshot->'horizons'->>5)::double precision / 1000000000.0) * interval '1 second',
        receiver_dedup_retained_until = d.created_at + ((d.policy_snapshot->'horizons'->>7)::double precision / 1000000000.0) * interval '1 second'
    FROM candidates c
    WHERE d.owner_scope = c.owner_scope AND d.delivery_id = c.delivery_id
    RETURNING 1
)
SELECT count(*) FROM updated;

-- name: BackfillWebhookActionRetention :one
WITH candidates AS (
    SELECT a.owner_scope, a.action_id
    FROM webhook_operator_actions a
    WHERE NOT isfinite(a.retain_until) OR (a.action_kind = 'redrive' AND a.result_cycle = 0)
    ORDER BY a.owner_scope, a.action_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), updated AS (
    UPDATE webhook_operator_actions a
    SET retain_until = CASE WHEN isfinite(a.retain_until) THEN a.retain_until ELSE COALESCE(
            (SELECT d.action_retained_until FROM webhook_deliveries d
             WHERE a.target_kind = 'delivery' AND d.owner_scope = a.owner_scope AND d.delivery_id = a.target_id),
            (SELECT a.created_at + ((g.policy->'horizons'->>4)::double precision / 1000000000.0) * interval '1 second'
             FROM webhook_destinations g
             WHERE a.target_kind = 'destination' AND g.owner_scope = a.owner_scope
               AND g.destination_id = a.target_id AND g.generation = a.target_generation),
            a.created_at + interval '3650 days'
        ) END,
        result_cycle = CASE WHEN a.result_cycle <> 0 THEN a.result_cycle ELSE COALESCE(
            (SELECT c.cycle_number FROM webhook_cycles c
             WHERE c.owner_scope = a.owner_scope AND c.authorizing_action_id = a.action_id
             ORDER BY c.cycle_number DESC LIMIT 1), 0
        ) END
    FROM candidates c
    WHERE a.owner_scope = c.owner_scope AND a.action_id = c.action_id
    RETURNING 1
)
SELECT count(*) FROM updated;

-- name: CleanupRetainedWebhookEvents :one
WITH candidates AS (
    SELECT e.owner_scope, e.business_event_id
    FROM webhook_events e
    WHERE NOT EXISTS (
        SELECT 1 FROM webhook_deliveries d
        WHERE d.owner_scope = e.owner_scope AND d.business_event_id = e.business_event_id
          AND (d.state <> 'terminal' OR d.legal_hold
            OR GREATEST(d.payload_retained_until, d.active_retained_until,
                        d.terminal_summary_retained_until, d.attempt_retained_until,
                        d.action_retained_until, d.destination_generation_retained_until,
                        d.redrive_eligible_until, d.receiver_dedup_retained_until) > sqlc.arg(sampled_at))
    )
    ORDER BY e.owner_scope, e.business_event_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_events e USING candidates c
    WHERE e.owner_scope = c.owner_scope AND e.business_event_id = c.business_event_id
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CleanupRetainedWebhookPayloads :one
WITH candidates AS (
    SELECT e.owner_scope, e.business_event_id
    FROM webhook_events e
    WHERE e.body IS NOT NULL
      AND NOT EXISTS (
          SELECT 1 FROM webhook_deliveries d
          WHERE d.owner_scope = e.owner_scope AND d.business_event_id = e.business_event_id
            AND (d.state <> 'terminal' OR d.legal_hold
              OR d.payload_retained_until > sqlc.arg(sampled_at)
              OR d.redrive_eligible_until > sqlc.arg(sampled_at))
      )
    ORDER BY e.owner_scope, e.business_event_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), erased AS (
    UPDATE webhook_events e
    SET body = NULL, origin_trace_link = NULL
    FROM candidates c
    WHERE e.owner_scope = c.owner_scope AND e.business_event_id = c.business_event_id
    RETURNING 1
)
SELECT count(*) FROM erased;

-- name: CleanupRetainedWebhookAttempts :one
WITH candidates AS (
    SELECT a.owner_scope, a.delivery_id, a.cycle_number, a.attempt_id
    FROM webhook_attempts a
    JOIN webhook_deliveries d
      ON d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
    WHERE a.finalized_at IS NOT NULL AND d.state = 'terminal' AND NOT d.legal_hold
      AND d.attempt_retained_until <= sqlc.arg(sampled_at)
    ORDER BY a.owner_scope, a.delivery_id, a.cycle_number, a.attempted_at
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF a SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_attempts a USING candidates c
    WHERE a.owner_scope = c.owner_scope AND a.delivery_id = c.delivery_id
      AND a.cycle_number = c.cycle_number AND a.attempt_id = c.attempt_id
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CleanupRetainedWebhookCycles :one
WITH candidates AS (
    SELECT c.owner_scope, c.delivery_id, c.cycle_number
    FROM webhook_cycles c
    JOIN webhook_deliveries d
      ON d.owner_scope = c.owner_scope AND d.delivery_id = c.delivery_id
    WHERE d.state = 'terminal' AND NOT d.legal_hold
      AND d.active_retained_until <= sqlc.arg(sampled_at)
      AND d.redrive_eligible_until <= sqlc.arg(sampled_at)
      AND NOT EXISTS (
          SELECT 1 FROM webhook_attempts a
          WHERE a.owner_scope = c.owner_scope AND a.delivery_id = c.delivery_id
            AND a.cycle_number = c.cycle_number
      )
    ORDER BY c.owner_scope, c.delivery_id, c.cycle_number
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF c SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_cycles c USING candidates x
    WHERE c.owner_scope = x.owner_scope AND c.delivery_id = x.delivery_id
      AND c.cycle_number = x.cycle_number
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CleanupRetainedWebhookSummaries :one
WITH candidates AS (
    SELECT d.owner_scope, d.delivery_id
    FROM webhook_deliveries d
    WHERE d.state = 'terminal' AND NOT d.legal_hold
      AND d.cumulative_summary <> 'retained'
      AND d.terminal_summary_retained_until <= sqlc.arg(sampled_at)
      AND d.redrive_eligible_until <= sqlc.arg(sampled_at)
    ORDER BY d.owner_scope, d.delivery_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), scrubbed AS (
    UPDATE webhook_deliveries d
    SET cumulative_summary = 'retained', terminal_at = NULL, updated_at = sqlc.arg(sampled_at)
    FROM candidates c
    WHERE d.owner_scope = c.owner_scope AND d.delivery_id = c.delivery_id
    RETURNING 1
)
SELECT count(*) FROM scrubbed;

-- name: CleanupRetiredWebhookDestinations :one
WITH candidates AS (
    SELECT d.owner_scope, d.destination_id, d.generation, d.updated_at
    FROM webhook_destinations d
    WHERE d.disposition = 'retired'
      AND NOT EXISTS (
          SELECT 1 FROM webhook_deliveries w
          WHERE w.owner_scope = d.owner_scope AND w.destination_id = d.destination_id
            AND w.destination_generation = d.generation
      )
      AND NOT EXISTS (
          SELECT 1 FROM webhook_operator_actions a
          WHERE a.owner_scope = d.owner_scope AND a.target_kind = 'destination'
            AND a.target_id = d.destination_id AND a.target_generation = d.generation
      )
    ORDER BY d.updated_at, d.owner_scope, d.destination_id, d.generation
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), tombstoned AS (
    INSERT INTO webhook_destination_tombstones (owner_scope, destination_id, generation, retired_at)
    SELECT owner_scope, destination_id, generation, updated_at FROM candidates
    ON CONFLICT DO NOTHING
    RETURNING owner_scope, destination_id, generation
), deleted AS (
    DELETE FROM webhook_destinations d USING tombstoned t
    WHERE d.owner_scope = t.owner_scope AND d.destination_id = t.destination_id AND d.generation = t.generation
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CleanupRetainedWebhookActions :one
WITH candidates AS (
    SELECT a.owner_scope, a.action_id
    FROM webhook_operator_actions a
    WHERE a.retain_until <= sqlc.arg(sampled_at)
      AND NOT EXISTS (
          SELECT 1 FROM webhook_deliveries d
          WHERE a.target_kind = 'delivery' AND d.owner_scope = a.owner_scope
            AND d.delivery_id = a.target_id AND d.legal_hold
      )
    ORDER BY a.retain_until, a.owner_scope, a.action_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_operator_actions a USING candidates c
    WHERE a.owner_scope = c.owner_scope AND a.action_id = c.action_id
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: ReadWebhookDeliveryInspection :one
SELECT e.acceptance_id, e.business_event_id, e.fanout_snapshot_id,
       e.event_type, e.business_schema_version, e.content_type,
       d.delivery_id, d.destination_id, d.destination_generation,
       d.state, d.current_cycle, d.cumulative_summary, d.sendable,
       d.next_due_at, d.redrive_eligible_until, d.terminal_at,
       d.created_at, d.updated_at, d.legal_hold,
       g.disposition AS destination_disposition,
       g.control_revision, g.key_state_revision, g.required_secret_revision
FROM webhook_deliveries d
JOIN webhook_events e
  ON e.owner_scope = d.owner_scope AND e.business_event_id = d.business_event_id
JOIN webhook_destinations g
  ON g.owner_scope = d.owner_scope AND g.destination_id = d.destination_id
 AND g.generation = d.destination_generation
WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.delivery_id = sqlc.arg(delivery_id);

-- name: ListWebhookInspectionCycles :many
SELECT cycle_number, cycle_kind, authorizing_action_id, accepted_at, deadline_at,
       maximum_attempts, attempts_used, disposition, finalized_at
FROM webhook_cycles
WHERE owner_scope = sqlc.arg(owner_scope) AND delivery_id = sqlc.arg(delivery_id)
  AND cycle_number >= sqlc.arg(cycle_from)
ORDER BY cycle_number
LIMIT sqlc.arg(page_size);

-- name: ListWebhookInspectionAttempts :many
SELECT cycle_number, attempt_id, fence, attempted_at, lease_expires_at,
       send_authorized, may_have_sent, response_header_bytes, response_body_bytes,
       response_status, retry_after_delay_ns, retry_delay_ns, outcome_class, finalized_at
FROM webhook_attempts
WHERE owner_scope = sqlc.arg(owner_scope) AND delivery_id = sqlc.arg(delivery_id)
  AND (cycle_number > sqlc.arg(after_cycle)
    OR (cycle_number = sqlc.arg(after_cycle)
      AND (attempted_at, attempt_id) > (sqlc.arg(after_attempted_at), sqlc.arg(after_attempt_id)::text)))
ORDER BY cycle_number, attempted_at, attempt_id
LIMIT sqlc.arg(page_size);

-- name: ListWebhookInspectionActions :many
SELECT action_id, actor_reference, action_kind, target_kind, target_id,
       target_generation, expected_state, reason, duplicate_risk_acknowledged,
       state, result, created_at, completed_at, result_cycle
FROM webhook_operator_actions
WHERE owner_scope = sqlc.arg(owner_scope) AND target_kind = 'delivery'
  AND target_id = sqlc.arg(delivery_id)
  AND (created_at, action_id) > (sqlc.arg(after_created_at), sqlc.arg(after_action_id)::text)
ORDER BY created_at, action_id
LIMIT sqlc.arg(page_size);

-- name: DeleteWebhookNamespaceBatch :one
WITH candidates AS MATERIALIZED (
    SELECT e.owner_scope, e.business_event_id
    FROM webhook_events e
    WHERE e.owner_scope = sqlc.arg(owner_scope)
    ORDER BY e.business_event_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE
), locked_deliveries AS MATERIALIZED (
    SELECT d.*
    FROM candidates c
    JOIN webhook_deliveries d
      ON d.owner_scope = c.owner_scope AND d.business_event_id = c.business_event_id
    FOR UPDATE OF d
), locked_attempts AS MATERIALIZED (
    SELECT a.*
    FROM locked_deliveries d
    JOIN webhook_attempts a ON a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id
    FOR UPDATE OF a
), deleted AS (
    DELETE FROM webhook_events e USING candidates c
    WHERE e.owner_scope = c.owner_scope AND e.business_event_id = c.business_event_id
      AND (SELECT count(*) FROM locked_attempts) >= 0
    RETURNING 1
)
SELECT (SELECT count(*) FROM deleted)::bigint AS deleted,
       COALESCE((SELECT bool_or(send_authorized OR may_have_sent) FROM locked_attempts), false)::boolean AS possible_send;

-- name: ObservePostgresWebhooks :one
SELECT
    (SELECT count(*) FROM webhook_deliveries WHERE state = 'ready')::bigint AS ready,
    (SELECT count(*) FROM webhook_deliveries WHERE state = 'scheduled')::bigint AS scheduled,
    (SELECT count(*) FROM webhook_deliveries WHERE state = 'in_flight')::bigint AS in_flight,
    (SELECT count(*) FROM webhook_deliveries WHERE state = 'terminal')::bigint AS terminal,
    (SELECT count(*) FROM webhook_deliveries WHERE state = 'suspended')::bigint AS suspended,
    (SELECT count(*) FROM webhook_deliveries WHERE state = 'quarantined')::bigint AS quarantined,
    (SELECT count(*) FROM webhook_destinations WHERE disposition = 'administratively_disabled')::bigint AS disabled,
    COALESCE((
        SELECT GREATEST(0, extract(epoch FROM (max(clock.high_water) - min(d.next_due_at)))::bigint)
        FROM webhook_deliveries d CROSS JOIN webhook_clock clock
        WHERE clock.singleton AND d.state IN ('ready', 'scheduled')
    ), 0)::bigint AS oldest_due_age_seconds,
    (SELECT count(*) FROM webhook_deliveries WHERE cumulative_summary = 'http_accepted')::bigint AS http_accepted,
    (SELECT count(*) FROM webhook_deliveries WHERE cumulative_summary = 'http_rejected')::bigint AS http_rejected,
    (SELECT count(*) FROM webhook_deliveries WHERE cumulative_summary = 'locally_denied')::bigint AS locally_denied,
    (SELECT count(*) FROM webhook_deliveries WHERE cumulative_summary = 'outcome_unknown')::bigint AS outcome_unknown,
    (SELECT count(*) FROM webhook_deliveries WHERE cumulative_summary = 'attempts_exhausted')::bigint AS attempts_exhausted,
    (SELECT count(*) FROM webhook_cycles WHERE cycle_kind = 'redrive' AND disposition = 'attempts_exhausted')::bigint AS redrive_exhausted,
    (SELECT count(*) FROM webhook_capacity_slots WHERE attempt_id IS NOT NULL)::bigint AS leased_slots,
    (SELECT count(*) FROM webhook_capacity_slots)::bigint AS total_slots,
    ((SELECT count(*) FROM webhook_deliveries
      WHERE NOT isfinite(payload_retained_until) OR NOT isfinite(active_retained_until)
         OR NOT isfinite(terminal_summary_retained_until) OR NOT isfinite(attempt_retained_until)
         OR NOT isfinite(action_retained_until) OR NOT isfinite(destination_generation_retained_until)
         OR NOT isfinite(receiver_dedup_retained_until))
     + (SELECT count(*) FROM webhook_operator_actions WHERE NOT isfinite(retain_until)))::bigint AS retention_backfill_pending,
    (SELECT count(*) FROM webhook_tombstones
     WHERE target_kind = 'namespace' AND first_disposition = 'pending')::bigint AS privacy_pending,
    (SELECT regression FROM webhook_clock WHERE singleton) AS clock_regression;
