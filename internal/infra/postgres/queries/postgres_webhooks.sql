-- name: ListPostgresWebhookRelations :many
SELECT c.relname::text AS relation_name
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema()
  AND c.relkind = 'r'
  AND c.relname = ANY(ARRAY[
    'webhook_clock', 'webhook_destinations', 'webhook_events', 'webhook_fanouts',
    'webhook_deliveries', 'webhook_cycles', 'webhook_attempts', 'webhook_capacity_slots',
    'webhook_operator_actions', 'webhook_tombstones'
  ]::text[])
ORDER BY c.relname;

-- name: CheckPostgresWebhookWriter :one
SELECT COALESCE(bool_and(has_table_privilege(current_user, relation_name, 'SELECT,INSERT,UPDATE,DELETE')), false)
       AND has_sequence_privilege(current_user, 'webhook_fairness_sequence', 'USAGE') AS writable
FROM unnest(ARRAY[
    'webhook_clock', 'webhook_destinations', 'webhook_events', 'webhook_fanouts',
    'webhook_deliveries', 'webhook_cycles', 'webhook_attempts', 'webhook_capacity_slots',
    'webhook_operator_actions', 'webhook_tombstones'
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
SELECT owner_scope, destination_id, required_secret_revision, active_key_reference,
       predecessor_key_reference, predecessor_valid_until
FROM webhook_destinations
WHERE disposition IN ('active', 'automatically_paused') AND active_key_reference IS NOT NULL
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
    OR (target_kind = 'event' AND target_id = sqlc.arg(target_id)))
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
    global_concurrency, required_secret_revision, active_key_reference,
    destination_retained_until, key_references_retained_until, created_at, updated_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(destination_id), sqlc.arg(generation),
    sqlc.arg(ownership_verification_receipt), sqlc.arg(url), sqlc.arg(selection_revision),
    sqlc.arg(payload_version_preference), sqlc.arg(signature_profile),
    sqlc.arg(signing_authority_binding), sqlc.arg(policy), sqlc.arg(policy_fingerprint),
    sqlc.arg(destination_concurrency), sqlc.arg(global_concurrency),
    sqlc.arg(required_secret_revision), sqlc.arg(active_key_reference),
    sqlc.arg(destination_retained_until), sqlc.arg(key_references_retained_until),
    sqlc.arg(created_at), sqlc.arg(created_at)
)
ON CONFLICT (owner_scope, destination_id, generation) DO NOTHING;

-- name: ReadWebhookDestination :one
SELECT * FROM webhook_destinations
WHERE owner_scope = sqlc.arg(owner_scope)
  AND destination_id = sqlc.arg(destination_id)
  AND generation = sqlc.arg(generation);

-- name: ExtendWebhookDestinationRetention :execrows
UPDATE webhook_destinations
SET destination_retained_until = GREATEST(destination_retained_until, sqlc.arg(destination_retained_until)),
    key_references_retained_until = GREATEST(key_references_retained_until, sqlc.arg(key_references_retained_until)),
    updated_at = GREATEST(updated_at, sqlc.arg(updated_at))
WHERE owner_scope = sqlc.arg(owner_scope)
  AND destination_id = sqlc.arg(destination_id)
  AND generation = sqlc.arg(generation)
  AND disposition = 'active'
  AND active_key_reference IS NOT NULL;

-- name: InsertWebhookEvent :execrows
INSERT INTO webhook_events (
    owner_scope, business_event_id, acceptance_id, fanout_snapshot_id, event_type,
    business_schema_version, content_type, body, delivery_envelope_version,
    subscriber_policy_revision, intent_fingerprint, retention_policy_identity,
    payload_retained_until, accepted_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(business_event_id), sqlc.arg(acceptance_id),
    sqlc.arg(fanout_snapshot_id), sqlc.arg(event_type), sqlc.arg(business_schema_version),
    sqlc.arg(content_type), sqlc.arg(body), sqlc.arg(delivery_envelope_version),
    sqlc.arg(subscriber_policy_revision), sqlc.arg(intent_fingerprint),
    sqlc.arg(retention_policy_identity), sqlc.arg(payload_retained_until), sqlc.arg(accepted_at)
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
    redrive_eligible_until, active_retained_until, terminal_retained_until,
    attempts_retained_until, actions_retained_until, created_at, updated_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(delivery_id), sqlc.arg(business_event_id),
    sqlc.arg(fanout_snapshot_id), sqlc.arg(destination_id), sqlc.arg(destination_generation),
    sqlc.arg(url_snapshot), sqlc.arg(policy_snapshot), sqlc.arg(next_due_at),
    sqlc.arg(redrive_eligible_until), sqlc.arg(active_retained_until),
    sqlc.arg(terminal_retained_until), sqlc.arg(attempts_retained_until),
    sqlc.arg(actions_retained_until), sqlc.arg(created_at), sqlc.arg(created_at)
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
  AND d.active_key_reference IS NOT NULL
  AND d.required_secret_revision <= sqlc.arg(manifest_revision)
  AND EXISTS (
      SELECT 1 FROM webhook_deliveries w
      WHERE w.owner_scope = d.owner_scope
        AND w.destination_id = d.destination_id
        AND w.destination_generation = d.generation
        AND w.state IN ('ready', 'scheduled')
        AND w.sendable
        AND w.next_due_at <= sqlc.arg(sampled_at)
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
SELECT d.*, e.body, e.content_type, c.attempts_used
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
  AND c.disposition = 'active' AND c.deadline_at > sqlc.arg(sampled_at)
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
    attempted_at, lease_expires_at, payload_digest, payload_bytes, retained_until
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(delivery_id), sqlc.arg(cycle_number), sqlc.arg(attempt_id),
    sqlc.arg(fence), sqlc.arg(capacity_slot), sqlc.arg(attempted_at), sqlc.arg(lease_expires_at),
    sqlc.arg(payload_digest), sqlc.arg(payload_bytes), sqlc.arg(retained_until)
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
SET key_reference = sqlc.arg(key_reference), signature_header_digest = sqlc.arg(signature_header_digest),
    dns_set_digest = sqlc.arg(dns_set_digest), selected_address = sqlc.arg(selected_address),
    send_authorized = true, may_have_sent = true
FROM webhook_deliveries d, webhook_destinations g, webhook_capacity_slots s
WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.delivery_id = sqlc.arg(delivery_id)
  AND a.cycle_number = sqlc.arg(cycle_number) AND a.attempt_id = sqlc.arg(attempt_id)
  AND a.fence = sqlc.arg(fence) AND a.finalized_at IS NULL
  AND d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
  AND d.state = 'in_flight' AND d.fence = a.fence AND d.sendable
  AND g.owner_scope = d.owner_scope AND g.destination_id = d.destination_id
  AND g.generation = d.destination_generation AND g.disposition = 'active'
  AND g.control_revision = sqlc.arg(control_revision)
  AND g.key_state_revision = sqlc.arg(key_state_revision)
  AND g.required_secret_revision <= sqlc.arg(manifest_revision)
  AND s.slot_number = a.capacity_slot AND s.owner_scope = a.owner_scope
  AND s.delivery_id = a.delivery_id AND s.attempt_id = a.attempt_id AND s.fence = a.fence;

-- name: LockWebhookSendBarrier :one
SELECT g.control_revision, g.key_state_revision, g.required_secret_revision,
       g.active_key_reference, g.predecessor_key_reference, g.predecessor_valid_until,
       d.destination_id, d.destination_generation, a.capacity_slot
FROM webhook_attempts a
JOIN webhook_deliveries d
  ON d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
JOIN webhook_destinations g
  ON g.owner_scope = d.owner_scope AND g.destination_id = d.destination_id
 AND g.generation = d.destination_generation
JOIN webhook_capacity_slots s
  ON s.slot_number = a.capacity_slot
WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.delivery_id = sqlc.arg(delivery_id)
  AND a.cycle_number = sqlc.arg(cycle_number) AND a.attempt_id = sqlc.arg(attempt_id)
  AND a.fence = sqlc.arg(fence) AND a.finalized_at IS NULL
FOR UPDATE OF g, d, s, a;

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
  ON s.slot_number = a.capacity_slot AND s.owner_scope = a.owner_scope
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
        response_status = sqlc.narg(response_status),
        retry_after_delay_ms = sqlc.narg(retry_after_delay_ms), retry_after_source = sqlc.narg(retry_after_source),
        outcome_class = sqlc.arg(outcome_class), finalized_at = sqlc.arg(finalized_at)
    WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.delivery_id = sqlc.arg(delivery_id)
      AND a.cycle_number = sqlc.arg(cycle_number) AND a.attempt_id = sqlc.arg(attempt_id)
      AND a.fence = sqlc.arg(fence) AND a.finalized_at IS NULL
    RETURNING a.owner_scope, a.delivery_id, a.cycle_number, a.attempt_id, a.fence, a.capacity_slot
), released AS (
    UPDATE webhook_capacity_slots s
    SET owner_scope = NULL, delivery_id = NULL, cycle_number = NULL, attempt_id = NULL,
        lease_expires_at = NULL, fence = NULL
    FROM finalized f
    WHERE s.slot_number = f.capacity_slot AND s.owner_scope = f.owner_scope
      AND s.delivery_id = f.delivery_id AND s.cycle_number = f.cycle_number
      AND s.attempt_id = f.attempt_id AND s.fence = f.fence
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
SELECT owner_scope, delivery_id, cycle_number, attempt_id, fence, capacity_slot,
       send_authorized, may_have_sent, attempted_at, lease_expires_at
FROM webhook_attempts
WHERE finalized_at IS NULL AND lease_expires_at <= sqlc.arg(sampled_at)
ORDER BY lease_expires_at, owner_scope, delivery_id, cycle_number, attempt_id
LIMIT sqlc.arg(batch_size)
FOR UPDATE SKIP LOCKED;

-- name: ReadWebhookOperatorAction :one
SELECT * FROM webhook_operator_actions
WHERE owner_scope = sqlc.arg(owner_scope) AND action_id = sqlc.arg(action_id);

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
  AND key_state_revision < sqlc.arg(key_state_revision) AND disposition <> 'retired';

-- name: LockWebhookDeliveryForAction :one
SELECT d.current_cycle, d.state, d.cumulative_summary, d.redrive_eligible_until,
       d.destination_id, d.destination_generation, d.sendable, c.disposition, c.deadline_at,
       d.policy_snapshot, d.active_retained_until, d.terminal_retained_until,
       d.attempts_retained_until, d.actions_retained_until,
       g.disposition AS destination_disposition, g.required_secret_revision,
       g.active_key_reference, g.destination_retained_until, g.key_references_retained_until,
       CASE WHEN e.body IS NULL THEN false ELSE true END AS payload_retained, e.payload_retained_until
FROM webhook_deliveries d
JOIN webhook_cycles c
  ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id AND c.cycle_number = d.current_cycle
JOIN webhook_destinations g
  ON g.owner_scope = d.owner_scope AND g.destination_id = d.destination_id AND g.generation = d.destination_generation
JOIN webhook_events e
  ON e.owner_scope = d.owner_scope AND e.business_event_id = d.business_event_id
WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.delivery_id = sqlc.arg(delivery_id)
FOR UPDATE OF d, c;

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

-- name: InsertWebhookOperatorAction :execrows
INSERT INTO webhook_operator_actions (
    owner_scope, action_id, encoding_version, request_fingerprint, actor_reference,
    action_kind, target_kind, target_id, target_generation, expected_state, reason,
    duplicate_risk_acknowledged, result, retained_until, created_at, completed_at
) VALUES (
    sqlc.arg(owner_scope), sqlc.arg(action_id), 'webhook-operator-action-v1',
    sqlc.arg(request_fingerprint), sqlc.arg(actor_reference), sqlc.arg(action_kind),
    sqlc.arg(target_kind), sqlc.arg(target_id), sqlc.arg(target_generation),
    sqlc.arg(expected_state), sqlc.arg(reason), sqlc.arg(duplicate_risk_acknowledged),
    sqlc.arg(result), sqlc.arg(retained_until), sqlc.arg(created_at), sqlc.arg(created_at)
)
ON CONFLICT DO NOTHING;

-- name: ReadWebhookActionRetainedUntil :one
SELECT COALESCE(
    CASE sqlc.arg(target_kind)::text
      WHEN 'delivery' THEN (
        SELECT d.actions_retained_until FROM webhook_deliveries d
        WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.delivery_id = sqlc.arg(target_id)
      )
      WHEN 'destination' THEN (
        SELECT g.created_at + ((g.policy->'horizons'->>4)::double precision / 1000000000) * interval '1 second'
        FROM webhook_destinations g
        WHERE g.owner_scope = sqlc.arg(owner_scope) AND g.destination_id = sqlc.arg(target_id)
          AND g.generation = sqlc.arg(target_generation)
      )
    END,
    sqlc.arg(sampled_at)::timestamptz + interval '365 days'
)::timestamptz AS retained_until;

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
DELETE FROM webhook_events
WHERE owner_scope = sqlc.arg(owner_scope) AND business_event_id = sqlc.arg(business_event_id);

-- name: ReleaseWebhookEventCapacity :exec
UPDATE webhook_capacity_slots s
SET owner_scope = NULL, delivery_id = NULL, cycle_number = NULL, attempt_id = NULL,
    lease_expires_at = NULL, fence = NULL
WHERE s.owner_scope = sqlc.arg(owner_scope)
  AND EXISTS (
      SELECT 1 FROM webhook_deliveries d
      WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.business_event_id = sqlc.arg(business_event_id)
        AND d.delivery_id = s.delivery_id
  );

-- name: ReleaseWebhookNamespaceCapacity :exec
UPDATE webhook_capacity_slots
SET owner_scope = NULL, delivery_id = NULL, cycle_number = NULL, attempt_id = NULL,
    lease_expires_at = NULL, fence = NULL
WHERE owner_scope = sqlc.arg(owner_scope);

-- name: ReadWebhookEventForPrivacy :one
WITH e AS (
    SELECT acceptance_id, fanout_snapshot_id
    FROM webhook_events
    WHERE owner_scope = sqlc.arg(owner_scope) AND business_event_id = sqlc.arg(business_event_id)
    FOR UPDATE
)
SELECT e.acceptance_id, e.fanout_snapshot_id,
       COALESCE(jsonb_agg(DISTINCT jsonb_build_array(d.delivery_id)) FILTER (WHERE d.delivery_id IS NOT NULL), '[]') AS delivery_identities,
       COALESCE(jsonb_agg(DISTINCT jsonb_build_array(d.destination_id, d.destination_generation)) FILTER (WHERE d.destination_id IS NOT NULL), '[]') AS destination_identities,
       COALESCE(max(d.cumulative_summary), 'none')::text AS last_semantic_class
FROM e
LEFT JOIN webhook_deliveries d
  ON d.owner_scope = sqlc.arg(owner_scope) AND d.business_event_id = sqlc.arg(business_event_id)
GROUP BY e.acceptance_id, e.fanout_snapshot_id;

-- name: LockWebhookEventDeliveriesForPrivacy :many
SELECT d.delivery_id
FROM webhook_deliveries d
JOIN webhook_cycles c
  ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id AND c.cycle_number = d.current_cycle
WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.business_event_id = sqlc.arg(business_event_id)
ORDER BY d.delivery_id
FOR UPDATE OF d, c;

-- name: LockWebhookEventAttemptsForPrivacy :many
SELECT a.may_have_sent, a.send_authorized
FROM webhook_attempts a
JOIN webhook_deliveries d
  ON d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id
JOIN webhook_capacity_slots s
  ON s.slot_number = a.capacity_slot AND s.owner_scope = a.owner_scope
  AND s.delivery_id = a.delivery_id AND s.cycle_number = a.cycle_number
  AND s.attempt_id = a.attempt_id AND s.fence = a.fence
WHERE d.owner_scope = sqlc.arg(owner_scope) AND d.business_event_id = sqlc.arg(business_event_id)
  AND a.finalized_at IS NULL
ORDER BY a.delivery_id, a.cycle_number, a.attempt_id
FOR UPDATE OF a, s;

-- name: LockWebhookNamespaceDestinationsForPrivacy :many
SELECT destination_id
FROM webhook_destinations
WHERE owner_scope = sqlc.arg(owner_scope)
ORDER BY destination_id, generation
FOR UPDATE;

-- name: LockWebhookNamespaceDeliveriesForPrivacy :many
SELECT d.delivery_id
FROM webhook_deliveries d
JOIN webhook_cycles c
  ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id AND c.cycle_number = d.current_cycle
WHERE d.owner_scope = sqlc.arg(owner_scope)
ORDER BY d.delivery_id
FOR UPDATE OF d, c;

-- name: LockWebhookNamespaceAttemptsForPrivacy :many
SELECT a.may_have_sent, a.send_authorized
FROM webhook_attempts a
JOIN webhook_capacity_slots s
  ON s.slot_number = a.capacity_slot AND s.owner_scope = a.owner_scope
  AND s.delivery_id = a.delivery_id AND s.cycle_number = a.cycle_number
  AND s.attempt_id = a.attempt_id AND s.fence = a.fence
WHERE a.owner_scope = sqlc.arg(owner_scope) AND a.finalized_at IS NULL
ORDER BY a.delivery_id, a.cycle_number, a.attempt_id
FOR UPDATE OF a, s;

-- name: MarkWebhookNamespaceTombstoneUnknown :execrows
UPDATE webhook_tombstones
SET last_semantic_class = 'outcome_unknown'
WHERE owner_scope = sqlc.arg(owner_scope) AND target_kind = 'namespace'
  AND target_id = sqlc.arg(owner_scope) AND action_id = sqlc.arg(action_id)
  AND last_semantic_class = 'none';

-- name: CountWebhookNamespaceRows :one
SELECT (
  (SELECT count(*) FROM webhook_events events WHERE events.owner_scope = sqlc.arg(owner_scope)) +
  (SELECT count(*) FROM webhook_destinations destinations WHERE destinations.owner_scope = sqlc.arg(owner_scope)) +
  (SELECT count(*) FROM webhook_operator_actions actions WHERE actions.owner_scope = sqlc.arg(owner_scope))
)::bigint;

-- name: DeleteWebhookNamespaceActions :exec
DELETE FROM webhook_operator_actions WHERE owner_scope = sqlc.arg(owner_scope);

-- name: DeleteWebhookNamespaceDestinations :exec
DELETE FROM webhook_destinations d WHERE d.owner_scope = sqlc.arg(owner_scope)
  AND NOT EXISTS (SELECT 1 FROM webhook_deliveries w WHERE w.owner_scope = d.owner_scope AND w.destination_id = d.destination_id AND w.destination_generation = d.generation);

-- name: CompleteWebhookNamespaceTombstone :execrows
UPDATE webhook_tombstones
SET first_disposition = 'completed'
WHERE owner_scope = sqlc.arg(owner_scope) AND target_kind = 'namespace'
  AND target_id = sqlc.arg(owner_scope) AND action_id = sqlc.arg(action_id)
  AND first_disposition = 'pending';

-- name: FinalizeExpiredWebhookCycles :one
WITH candidates AS (
    SELECT d.owner_scope, d.delivery_id, d.current_cycle, d.cumulative_summary
    FROM webhook_deliveries d
    JOIN webhook_cycles c
      ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id AND c.cycle_number = d.current_cycle
    WHERE d.state IN ('ready', 'scheduled') AND c.disposition = 'active'
      AND (c.deadline_at <= sqlc.arg(sampled_at) OR d.active_retained_until <= sqlc.arg(sampled_at))
      AND NOT EXISTS (
        SELECT 1 FROM webhook_attempts a
        WHERE a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id
          AND a.cycle_number = d.current_cycle AND a.finalized_at IS NULL
      )
    ORDER BY c.deadline_at, d.owner_scope, d.delivery_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF d, c SKIP LOCKED
), cycles AS (
    UPDATE webhook_cycles c
    SET disposition = CASE WHEN x.cumulative_summary = 'outcome_unknown' THEN 'outcome_unknown' ELSE 'attempts_exhausted' END,
        finalized_at = sqlc.arg(sampled_at)
    FROM candidates x
    WHERE c.owner_scope = x.owner_scope AND c.delivery_id = x.delivery_id
      AND c.cycle_number = x.current_cycle AND c.disposition = 'active'
    RETURNING c.owner_scope, c.delivery_id
), deliveries AS (
    UPDATE webhook_deliveries d
    SET state = CASE WHEN d.cumulative_summary = 'outcome_unknown' THEN 'suspended' ELSE 'terminal' END,
        cumulative_summary = CASE WHEN d.cumulative_summary = 'outcome_unknown' THEN 'outcome_unknown' ELSE 'attempts_exhausted' END,
        sendable = false, lease_owner = NULL, lease_expires_at = NULL,
        terminal_at = sqlc.arg(sampled_at), updated_at = sqlc.arg(sampled_at)
    FROM cycles c
    WHERE d.owner_scope = c.owner_scope AND d.delivery_id = c.delivery_id
    RETURNING 1
)
SELECT count(*) FROM deliveries;

-- name: EraseRetainedWebhookPayloads :one
WITH candidates AS (
    SELECT e.owner_scope, e.business_event_id
    FROM webhook_events e
    WHERE e.body IS NOT NULL AND e.payload_retained_until <= sqlc.arg(sampled_at)
      AND NOT EXISTS (
        SELECT 1 FROM webhook_deliveries d
        WHERE d.owner_scope = e.owner_scope AND d.business_event_id = e.business_event_id
          AND (d.state <> 'terminal' OR d.redrive_eligible_until > sqlc.arg(sampled_at)
            OR d.cumulative_summary = 'outcome_unknown')
    )
    ORDER BY e.owner_scope, e.business_event_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
), erased AS (
    UPDATE webhook_events e
    SET body = NULL, payload_erased_at = sqlc.arg(sampled_at)
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
    WHERE a.finalized_at IS NOT NULL AND a.retained_until <= sqlc.arg(sampled_at)
      AND d.state = 'terminal' AND d.redrive_eligible_until <= sqlc.arg(sampled_at)
      AND d.cumulative_summary <> 'outcome_unknown'
    ORDER BY a.retained_until, a.owner_scope, a.delivery_id, a.cycle_number, a.attempt_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF a SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_attempts a USING candidates c
    WHERE a.owner_scope = c.owner_scope AND a.delivery_id = c.delivery_id
      AND a.cycle_number = c.cycle_number AND a.attempt_id = c.attempt_id RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CleanupRetainedWebhookActions :one
WITH candidates AS (
    SELECT action.owner_scope, action.action_id
    FROM webhook_operator_actions action
    WHERE action.retained_until <= sqlc.arg(sampled_at)
      AND NOT EXISTS (
        SELECT 1 FROM webhook_cycles cycle
        JOIN webhook_deliveries d
          ON d.owner_scope = cycle.owner_scope AND d.delivery_id = cycle.delivery_id
        WHERE cycle.owner_scope = action.owner_scope AND cycle.authorizing_action_id = action.action_id
          AND (cycle.disposition = 'active' OR d.state <> 'terminal'
            OR d.cumulative_summary = 'outcome_unknown'
            OR d.redrive_eligible_until > sqlc.arg(sampled_at)
            OR d.terminal_retained_until > sqlc.arg(sampled_at)
            OR EXISTS (SELECT 1 FROM webhook_attempts a WHERE a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id))
      )
    ORDER BY action.retained_until, action.owner_scope, action.action_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF action SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_operator_actions action USING candidates c
    WHERE action.owner_scope = c.owner_scope AND action.action_id = c.action_id RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CleanupRetainedWebhookDeliveries :one
WITH candidates AS (
    SELECT d.owner_scope, d.delivery_id
    FROM webhook_deliveries d
    WHERE d.state = 'terminal' AND d.cumulative_summary <> 'outcome_unknown'
      AND d.redrive_eligible_until <= sqlc.arg(sampled_at)
      AND d.terminal_retained_until <= sqlc.arg(sampled_at)
      AND NOT EXISTS (SELECT 1 FROM webhook_attempts a WHERE a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id)
      AND NOT EXISTS (SELECT 1 FROM webhook_operator_actions action WHERE action.owner_scope = d.owner_scope AND action.target_kind = 'delivery' AND action.target_id = d.delivery_id)
    ORDER BY d.terminal_retained_until, d.owner_scope, d.delivery_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF d SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_deliveries d USING candidates c
    WHERE d.owner_scope = c.owner_scope AND d.delivery_id = c.delivery_id RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CleanupRetainedWebhookEvents :one
WITH candidates AS (
    SELECT e.owner_scope, e.business_event_id
    FROM webhook_events e
    WHERE e.body IS NULL
      AND NOT EXISTS (SELECT 1 FROM webhook_deliveries d WHERE d.owner_scope = e.owner_scope AND d.business_event_id = e.business_event_id)
    ORDER BY e.owner_scope, e.business_event_id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF e SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_events e USING candidates c
    WHERE e.owner_scope = c.owner_scope AND e.business_event_id = c.business_event_id RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: CleanupRetainedWebhookDestinations :one
WITH candidates AS (
    SELECT g.owner_scope, g.destination_id, g.generation
    FROM webhook_destinations g
    WHERE g.disposition = 'retired'
      AND g.destination_retained_until <= sqlc.arg(sampled_at)
      AND g.key_references_erased_at IS NOT NULL
      AND NOT EXISTS (SELECT 1 FROM webhook_deliveries d WHERE d.owner_scope = g.owner_scope AND d.destination_id = g.destination_id AND d.destination_generation = g.generation)
      AND NOT EXISTS (SELECT 1 FROM webhook_operator_actions action WHERE action.owner_scope = g.owner_scope AND action.target_kind = 'destination' AND action.target_id = g.destination_id AND action.target_generation = g.generation)
    ORDER BY g.destination_retained_until, g.owner_scope, g.destination_id, g.generation
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF g SKIP LOCKED
), deleted AS (
    DELETE FROM webhook_destinations g USING candidates c
    WHERE g.owner_scope = c.owner_scope AND g.destination_id = c.destination_id
      AND g.generation = c.generation RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: EraseRetainedWebhookKeyReferences :one
WITH candidates AS (
    SELECT g.owner_scope, g.destination_id, g.generation
    FROM webhook_destinations g
    WHERE g.disposition = 'retired' AND g.active_key_reference IS NOT NULL
      AND g.key_references_retained_until <= sqlc.arg(sampled_at)
      AND NOT EXISTS (SELECT 1 FROM webhook_deliveries d WHERE d.owner_scope = g.owner_scope AND d.destination_id = g.destination_id AND d.destination_generation = g.generation)
    ORDER BY g.key_references_retained_until, g.owner_scope, g.destination_id, g.generation
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF g SKIP LOCKED
), erased AS (
    UPDATE webhook_destinations g
    SET active_key_reference = NULL, predecessor_key_reference = NULL,
        predecessor_valid_until = NULL, key_references_erased_at = sqlc.arg(sampled_at),
        updated_at = sqlc.arg(sampled_at)
    FROM candidates c
    WHERE g.owner_scope = c.owner_scope AND g.destination_id = c.destination_id
      AND g.generation = c.generation
    RETURNING 1
)
SELECT count(*) FROM erased;

-- name: DeleteWebhookNamespaceBatch :one
WITH deleted AS (
    DELETE FROM webhook_events e
    WHERE e.ctid IN (
        SELECT candidate.ctid FROM webhook_events candidate WHERE candidate.owner_scope = sqlc.arg(owner_scope)
        ORDER BY candidate.business_event_id LIMIT sqlc.arg(batch_size)
    )
    RETURNING 1
)
SELECT count(*) FROM deleted;

-- name: ObservePostgresWebhooks :one
SELECT
    (SELECT count(*) FROM webhook_deliveries WHERE state IN ('ready', 'scheduled'))::bigint AS scheduled,
    (SELECT count(*) FROM webhook_deliveries WHERE state = 'in_flight')::bigint AS in_flight,
    (SELECT count(*) FROM webhook_deliveries WHERE state = 'terminal')::bigint AS terminal,
    (SELECT count(*) FROM webhook_destinations WHERE disposition = 'administratively_disabled')::bigint AS disabled,
    (SELECT count(*) FROM webhook_deliveries WHERE cumulative_summary = 'outcome_unknown')::bigint AS outcome_unknown,
    COALESCE((SELECT extract(epoch FROM min(next_due_at))::bigint FROM webhook_deliveries WHERE state IN ('ready', 'scheduled')), 0)::bigint AS oldest_due_timestamp,
    extract(epoch FROM clock_timestamp())::bigint AS observation_timestamp,
    (SELECT count(*) FROM webhook_capacity_slots WHERE attempt_id IS NOT NULL)::bigint AS leased_slots,
    (SELECT count(*) FROM webhook_capacity_slots)::bigint AS total_slots,
    (SELECT regression FROM webhook_clock WHERE singleton) AS clock_regression;
