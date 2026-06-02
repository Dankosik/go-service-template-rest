-- name: CreateBillingEventInboxReceipt :one
INSERT INTO billing_event_inbox (
    inbox_id,
    topic,
    partition_id,
    offset_value,
    event_id,
    producer_identity,
    business_identity_type,
    business_identity_value,
    event_fingerprint,
    state,
    failure_class,
    safe_metadata,
    received_at,
    applied_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'received', NULL, $10, $11, NULL, $11)
RETURNING *;

-- name: GetBillingEventInboxByEvent :one
SELECT *
FROM billing_event_inbox
WHERE topic = $1
  AND event_id = $2;

-- name: LockBillingEventInboxByBusinessIdentity :one
SELECT *
FROM billing_event_inbox
WHERE topic = $1
  AND business_identity_type = $2
  AND business_identity_value = $3
FOR UPDATE;

-- name: MarkBillingEventInboxApplied :one
UPDATE billing_event_inbox
SET
    state = 'applied',
    failure_class = NULL,
    applied_at = $2,
    updated_at = $2
WHERE inbox_id = $1
  AND state IN ('received', 'duplicate', 'applied')
RETURNING *;

-- name: MarkBillingEventInboxConflict :one
UPDATE billing_event_inbox
SET
    state = 'conflict',
    failure_class = $2,
    safe_metadata = $3,
    updated_at = $4
WHERE inbox_id = $1
RETURNING *;

-- name: CreateBillingOutbox :one
INSERT INTO billing_outbox (
    outbox_id,
    event_type,
    aggregate_type,
    aggregate_id,
    event_fingerprint,
    safe_payload,
    state,
    attempt_count,
    next_attempt_at,
    published_at,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, 'pending', 0, $7, NULL, $8, $8)
RETURNING *;

-- name: ClaimBillingOutbox :many
UPDATE billing_outbox
SET
    state = 'publishing',
    attempt_count = attempt_count + 1,
    updated_at = $2
WHERE outbox_id IN (
    SELECT candidate.outbox_id
    FROM billing_outbox AS candidate
    WHERE candidate.state IN ('pending', 'failed')
      AND candidate.next_attempt_at <= $1
    ORDER BY candidate.next_attempt_at, candidate.outbox_id
    FOR UPDATE SKIP LOCKED
    LIMIT $3
)
RETURNING *;

-- name: MarkBillingOutboxPublished :one
UPDATE billing_outbox
SET
    state = 'published',
    published_at = $2,
    updated_at = $2
WHERE outbox_id = $1
RETURNING *;

-- name: MarkBillingOutboxFailed :one
UPDATE billing_outbox
SET
    state = 'failed',
    next_attempt_at = $2,
    updated_at = $3
WHERE outbox_id = $1
RETURNING *;

-- name: CreateSpendingMicrolease :one
INSERT INTO spending_microleases (
    microlease_id,
    account_id,
    account_scope_key,
    proxy_allocator_owner_id,
    microlease_generation,
    lease_fence,
    state,
    issued_cap_usd_atoms,
    available_child_cap_usd_atoms,
    allocated_child_cap_reported_usd_atoms,
    terminal_charged_usd_atoms,
    terminal_released_usd_atoms,
    write_off_usd_atoms,
    pricing_snapshot_id,
    pricing_snapshot_fingerprint,
    pricing_policy_version,
    pricing_decision_at,
    pricing_selector_key,
    pricing_contract_version,
    fee_policy_version,
    microlease_policy_version,
    issued_at,
    debit_cutoff_at,
    expires_at,
    closed_at,
    last_checkpoint_sequence,
    last_checkpoint_fingerprint,
    idempotency_record_id,
    stored_outcome_id,
    safe_metadata,
    created_at,
    updated_at
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, 0, 0, 0, 0, $10, $11, $12, $13, $14, $15,
    $16, $17, $18, $19, $20, NULL, 0, NULL, $21, $22, $23, $24, $24
)
RETURNING *;

-- name: GetSpendingMicrolease :one
SELECT *
FROM spending_microleases
WHERE microlease_id = $1;

-- name: LockSpendingMicrolease :one
SELECT *
FROM spending_microleases
WHERE microlease_id = $1
FOR UPDATE;

-- name: GetSpendingMicroleaseByIdempotency :one
SELECT *
FROM spending_microleases
WHERE idempotency_record_id = $1;

-- name: ListStaleSpendingMicroleases :many
SELECT *
FROM spending_microleases
WHERE state IN ('active', 'cutoff', 'closing', 'expired')
  AND expires_at <= $1
ORDER BY expires_at, microlease_id
LIMIT $2;

-- name: UpdateSpendingMicroleaseSettlementTotals :one
UPDATE spending_microleases
SET
    state = $2,
    available_child_cap_usd_atoms = $3,
    allocated_child_cap_reported_usd_atoms = $4,
    terminal_charged_usd_atoms = $5,
    terminal_released_usd_atoms = $6,
    write_off_usd_atoms = $7,
    last_checkpoint_sequence = $8,
    last_checkpoint_fingerprint = $9,
    closed_at = $10,
    updated_at = $11
WHERE microlease_id = $1
RETURNING *;

-- name: CreateMicroleaseChildDebit :one
INSERT INTO microlease_child_debits (
    microlease_child_debit_id,
    microlease_id,
    debit_authorization_id,
    usage_operation_id,
    account_id,
    account_scope_key,
    proxy_allocator_owner_id,
    microlease_generation,
    child_sequence,
    child_cap_usd_atoms,
    charged_usd_atoms,
    released_usd_atoms,
    write_off_usd_atoms,
    request_basis_fingerprint,
    terminal_basis_fingerprint,
    pricing_snapshot_id,
    pricing_snapshot_fingerprint,
    terminal_kind,
    state,
    qualified_inference_evidence_id,
    terminal_event_id,
    terminal_inbox_id,
    ledger_entry_id,
    settlement_effect_id,
    safe_metadata,
    created_at,
    terminal_at,
    settled_at,
    updated_at
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
    $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $26
)
RETURNING *;

-- name: GetMicroleaseChildDebit :one
SELECT *
FROM microlease_child_debits
WHERE microlease_child_debit_id = $1;

-- name: GetMicroleaseChildDebitByAuthorization :one
SELECT *
FROM microlease_child_debits
WHERE microlease_id = $1
  AND debit_authorization_id = $2;

-- name: LockMicroleaseChildDebit :one
SELECT *
FROM microlease_child_debits
WHERE microlease_child_debit_id = $1
FOR UPDATE;

-- name: UpdateMicroleaseChildDebitTerminal :one
UPDATE microlease_child_debits
SET
    terminal_basis_fingerprint = $2,
    terminal_kind = $3,
    state = $4,
    charged_usd_atoms = $5,
    released_usd_atoms = $6,
    write_off_usd_atoms = $7,
    qualified_inference_evidence_id = $8,
    terminal_event_id = $9,
    terminal_inbox_id = $10,
    ledger_entry_id = $11,
    settlement_effect_id = $12,
    terminal_at = $13,
    settled_at = $14,
    updated_at = $15
WHERE microlease_child_debit_id = $1
RETURNING *;

-- name: CreateMicroleaseCheckpoint :one
INSERT INTO microlease_checkpoints (
    checkpoint_id,
    microlease_id,
    account_id,
    account_scope_key,
    proxy_allocator_owner_id,
    microlease_generation,
    checkpoint_sequence,
    checkpoint_kind,
    allocated_child_high_water,
    allocated_child_count,
    allocated_child_cap_sum_usd_atoms,
    terminal_submitted_count,
    terminal_published_count,
    terminal_accepted_count,
    unresolved_child_count,
    unresolved_child_cap_sum_usd_atoms,
    local_remaining_usd_atoms,
    checkpoint_fingerprint,
    inbox_id,
    created_at,
    applied_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
RETURNING *;

-- name: GetLatestMicroleaseCheckpoint :one
SELECT *
FROM microlease_checkpoints
WHERE microlease_id = $1
ORDER BY checkpoint_sequence DESC, checkpoint_id DESC
LIMIT 1;

-- name: UpsertBillingAdmissionControl :one
INSERT INTO billing_admission_controls (
    admission_control_id,
    scope_kind,
    scope_key,
    use_class,
    state,
    reason_code,
    terminal_lag_bucket,
    stale_age_bucket,
    reconciliation_backlog_bucket,
    audited_actor_kind,
    audited_actor_id,
    safe_metadata,
    expires_at,
    renewed_at,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)
ON CONFLICT (scope_kind, scope_key, use_class)
DO UPDATE SET
    state = EXCLUDED.state,
    reason_code = EXCLUDED.reason_code,
    terminal_lag_bucket = EXCLUDED.terminal_lag_bucket,
    stale_age_bucket = EXCLUDED.stale_age_bucket,
    reconciliation_backlog_bucket = EXCLUDED.reconciliation_backlog_bucket,
    audited_actor_kind = EXCLUDED.audited_actor_kind,
    audited_actor_id = EXCLUDED.audited_actor_id,
    safe_metadata = EXCLUDED.safe_metadata,
    expires_at = EXCLUDED.expires_at,
    renewed_at = EXCLUDED.renewed_at,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: GetBillingAdmissionControl :one
SELECT *
FROM billing_admission_controls
WHERE scope_kind = $1
  AND scope_key = $2
  AND use_class = $3;

-- name: CreateMicroleaseReconciliationCase :one
INSERT INTO reconciliation_cases (
    reconciliation_case_id,
    account_id,
    reason,
    state,
    severity,
    usage_operation_id,
    topup_operation_id,
    payment_attempt_id,
    payment_evidence_id,
    settlement_effect_id,
    qualified_inference_evidence_id,
    ledger_entry_id,
    legacy_balance_import_id,
    resolution_ledger_entry_id,
    resolution_settlement_effect_id,
    lease_owner,
    lease_deadline_at,
    attempt_count,
    next_attempt_at,
    support_safe_notes,
    created_at,
    updated_at,
    resolved_at,
    microlease_id,
    microlease_child_debit_id,
    microlease_checkpoint_id,
    billing_event_inbox_id
)
VALUES (
    $1, $2, $3, $4, $5, NULL, NULL, NULL, NULL, $6, NULL, $7, NULL, NULL, NULL, $8, $9,
    $10, $11, $12, $13, $13, NULL, $14, $15, $16, $17
)
RETURNING *;
