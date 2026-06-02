-- name: CreateBillingAccount :one
INSERT INTO billing_accounts (
    account_id,
    account_scope_key,
    account_type,
    subject_authority,
    subject_id,
    state,
    version,
    created_at,
    updated_at,
    closed_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetBillingAccountByScope :one
SELECT *
FROM billing_accounts
WHERE account_scope_key = $1;

-- name: CreateAccountBalance :one
INSERT INTO account_balances (
    account_id,
    account_scope_key,
    currency,
    settled_usd_atoms,
    reserved_usd_atoms,
    available_usd_atoms,
    pending_usd_atoms,
    version,
    last_ledger_entry_id,
    updated_at
)
VALUES ($1, $2, 'USD', $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetAccountBalanceByScope :one
SELECT *
FROM account_balances
WHERE account_scope_key = $1;

-- name: GetLatestAcceptedLegacyImportByAccountScope :one
SELECT lbi.*
FROM legacy_balance_imports AS lbi
JOIN legacy_import_batches AS lib
  ON lib.legacy_import_batch_id = lbi.legacy_import_batch_id
WHERE lbi.account_scope_key = $1
  AND lib.state IN ('parity_checked', 'applied')
ORDER BY lbi.created_at DESC, lbi.legacy_balance_import_id DESC
LIMIT 1;

-- name: GetActiveExposureByAccountScope :one
SELECT
    COALESCE((
        SELECT SUM(sm.issued_cap_usd_atoms - sm.terminal_charged_usd_atoms - sm.terminal_released_usd_atoms - sm.write_off_usd_atoms)
        FROM spending_microleases AS sm
        WHERE sm.account_scope_key = $1
          AND sm.state IN ('active', 'cutoff', 'closing', 'expired')
    ), 0)::bigint AS active_microlease_usd_atoms,
    COALESCE((
        SELECT SUM(uh.reserved_usd_atoms - uh.released_usd_atoms - uh.charged_usd_atoms - uh.write_off_usd_atoms)
        FROM usage_holds AS uh
        WHERE uh.account_scope_key = $1
          AND uh.state = 'active'
    ), 0)::bigint AS active_usage_hold_usd_atoms,
    COALESCE((
        SELECT SUM(mcd.child_cap_usd_atoms - mcd.charged_usd_atoms - mcd.released_usd_atoms - mcd.write_off_usd_atoms)
        FROM microlease_child_debits AS mcd
        WHERE mcd.account_scope_key = $1
          AND mcd.state IN ('authorized', 'terminal_pending', 'reconcile_required', 'manual_review')
    ), 0)::bigint AS unresolved_child_debit_usd_atoms;

-- name: LockAccountBalanceByAccountID :one
SELECT *
FROM account_balances
WHERE account_id = $1
FOR UPDATE;

-- name: UpdateAccountBalanceAfterLedger :one
UPDATE account_balances
SET
    settled_usd_atoms = $2,
    reserved_usd_atoms = $3,
    available_usd_atoms = $4,
    pending_usd_atoms = $5,
    version = version + 1,
    last_ledger_entry_id = $6,
    updated_at = $7
WHERE account_id = $1
RETURNING *;

-- name: GetIdempotencyRecord :one
SELECT *
FROM idempotency_records
WHERE account_id = $1
  AND operation_kind = $2
  AND idempotency_key = $3;

-- name: LockIdempotencyRecord :one
SELECT *
FROM idempotency_records
WHERE account_id = $1
  AND operation_kind = $2
  AND idempotency_key = $3
FOR UPDATE;

-- name: CreateIdempotencyRecord :one
INSERT INTO idempotency_records (
    idempotency_record_id,
    account_id,
    operation_kind,
    idempotency_key,
    request_fingerprint,
    state,
    stored_outcome_id,
    conflict_reason,
    retention_class,
    first_seen_at,
    last_seen_at,
    committed_at,
    expires_at
)
VALUES ($1, $2, $3, $4, $5, 'started', NULL, NULL, $6, $7, $7, NULL, NULL)
RETURNING *;

-- name: MarkIdempotencyCommitted :one
UPDATE idempotency_records
SET
    state = $2,
    stored_outcome_id = $3,
    conflict_reason = NULL,
    last_seen_at = $4,
    committed_at = $4
WHERE idempotency_record_id = $1
  AND state IN ('started', 'committed', 'failed_stored')
RETURNING *;

-- name: MarkIdempotencyConflict :one
UPDATE idempotency_records
SET
    state = 'conflict',
    conflict_reason = $2,
    last_seen_at = $3
WHERE idempotency_record_id = $1
RETURNING *;

-- name: CreateOperationOutcome :one
INSERT INTO operation_outcomes (
    stored_outcome_id,
    idempotency_record_id,
    account_id,
    operation_kind,
    outcome_status,
    primary_resource_type,
    primary_resource_id,
    ledger_entry_id,
    settlement_effect_id,
    failure_class,
    safe_problem_code,
    safe_outcome,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetOperationOutcomeByID :one
SELECT *
FROM operation_outcomes
WHERE stored_outcome_id = $1;

-- name: CreateUsageOperation :one
INSERT INTO usage_operations (
    usage_operation_id,
    account_id,
    account_scope_key,
    state,
    operation_kind,
    client_usage_request_id,
    request_id,
    request_basis_fingerprint,
    terminal_basis_fingerprint,
    pricing_snapshot_id,
    pricing_snapshot_fingerprint,
    quote_expires_at,
    fee_policy_version,
    reserve_policy_version,
    qualified_inference_evidence_id,
    terminal_outcome_id,
    settlement_effect_id,
    created_at,
    updated_at,
    reserved_at,
    terminal_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $18, $19, $20)
RETURNING *;

-- name: GetUsageOperation :one
SELECT *
FROM usage_operations
WHERE usage_operation_id = $1;

-- name: LockUsageOperation :one
SELECT *
FROM usage_operations
WHERE usage_operation_id = $1
FOR UPDATE;

-- name: GetMicroleaseChildDebitByUsageOperation :one
SELECT *
FROM microlease_child_debits
WHERE usage_operation_id = $1
ORDER BY created_at DESC, microlease_child_debit_id DESC
LIMIT 1;

-- name: UpdateUsageOperationTerminal :one
UPDATE usage_operations
SET
    state = $2,
    operation_kind = $3,
    terminal_basis_fingerprint = $4,
    terminal_outcome_id = $5,
    settlement_effect_id = $6,
    updated_at = $7,
    terminal_at = $7
WHERE usage_operation_id = $1
RETURNING *;

-- name: CreateUsageHold :one
INSERT INTO usage_holds (
    hold_id,
    usage_operation_id,
    account_id,
    account_scope_key,
    state,
    reserved_usd_atoms,
    released_usd_atoms,
    charged_usd_atoms,
    write_off_usd_atoms,
    pricing_snapshot_id,
    pricing_snapshot_fingerprint,
    quote_expires_at,
    fee_policy_version,
    reserve_policy_version,
    client_usage_request_id,
    request_basis_fingerprint,
    created_at,
    updated_at,
    expires_at,
    terminal_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $17, $18, $19)
RETURNING *;

-- name: LockUsageHoldByOperation :one
SELECT *
FROM usage_holds
WHERE usage_operation_id = $1
FOR UPDATE;

-- name: CreateLedgerEntry :one
INSERT INTO ledger_entries (
    ledger_entry_id,
    account_id,
    account_scope_key,
    currency,
    effect_type,
    amount_usd_atoms,
    settled_delta_usd_atoms,
    reserved_delta_usd_atoms,
    pending_delta_usd_atoms,
    settled_after_usd_atoms,
    reserved_after_usd_atoms,
    available_after_usd_atoms,
    pending_after_usd_atoms,
    balance_version_after,
    settlement_effect_id,
    idempotency_record_id,
    usage_operation_id,
    topup_operation_id,
    payment_attempt_id,
    payment_evidence_id,
    qualified_inference_evidence_id,
    reversal_of_ledger_entry_id,
    correction_of_ledger_entry_id,
    effective_at,
    created_at,
    created_by_kind,
    reason_code,
    safe_metadata
)
VALUES ($1, $2, $3, 'USD', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
RETURNING *;

-- name: ListLedgerEntriesByAccount :many
SELECT *
FROM ledger_entries
WHERE account_id = $1
  AND (
      sqlc.narg('before_created_at')::timestamptz IS NULL
      OR (created_at, ledger_entry_id) < (sqlc.narg('before_created_at')::timestamptz, sqlc.narg('before_ledger_entry_id')::uuid)
  )
ORDER BY created_at DESC, ledger_entry_id DESC
LIMIT $2;

-- name: CreateUsageTerminalOutcome :one
INSERT INTO usage_terminal_outcomes (
    usage_terminal_outcome_id,
    usage_operation_id,
    terminal_kind,
    idempotency_record_id,
    stored_outcome_id,
    ledger_entry_id,
    settlement_effect_id,
    charged_usd_atoms,
    released_usd_atoms,
    write_off_usd_atoms,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: CreateQualifiedInferenceEvidence :one
INSERT INTO qualified_inference_evidence (
    qualified_inference_evidence_id,
    usage_operation_id,
    provider_family,
    verification_surface,
    proof_scope,
    inference_id,
    evidence_fingerprint,
    observed_at,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: CreateTopupOperation :one
INSERT INTO topup_operations (
    topup_operation_id,
    account_id,
    account_scope_key,
    state,
    accepted_quote_id,
    credited_usd_atoms,
    deposit_fee_usd_atoms,
    payin_amount_value,
    payin_currency,
    pricing_snapshot_id,
    pricing_snapshot_fingerprint,
    settlement_policy_version,
    billing_fee_policy_version,
    current_payment_attempt_id,
    attempt_generation,
    presentation_version,
    presentation_fingerprint,
    settlement_effect_id,
    created_at,
    updated_at,
    expires_at,
    settlement_applied_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULL, $14, $15, $16, $17, $18, $18, $19, $20)
RETURNING *;

-- name: LockTopupOperation :one
SELECT *
FROM topup_operations
WHERE topup_operation_id = $1
FOR UPDATE;

-- name: CreatePaymentAttempt :one
INSERT INTO payment_attempts (
    payment_attempt_row_id,
    topup_operation_id,
    payment_attempt_id,
    attempt_generation,
    state,
    presentation_version,
    presentation_fingerprint,
    created_at,
    updated_at,
    expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $9)
RETURNING *;

-- name: LockPaymentAttempt :one
SELECT *
FROM payment_attempts
WHERE topup_operation_id = $1
  AND payment_attempt_id = $2
FOR UPDATE;

-- name: CreatePaymentEvidence :one
INSERT INTO payment_evidence (
    payment_evidence_id,
    topup_operation_id,
    payment_attempt_id,
    account_id,
    account_scope_key,
    state,
    evidence_payload_fingerprint,
    evidence_kind,
    schema_version,
    finality_class,
    rail_family,
    settlement_amount_usd_atoms,
    settlement_effect_id,
    ledger_entry_id,
    prior_payment_evidence_id,
    received_at,
    provider_event_at,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
RETURNING *;

-- name: GetPaymentEvidence :one
SELECT *
FROM payment_evidence
WHERE payment_evidence_id = $1;

-- name: GetPaymentEvidenceByFingerprint :one
SELECT *
FROM payment_evidence
WHERE evidence_payload_fingerprint = $1;

-- name: CreateReconciliationCase :one
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
    resolved_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $21, $22)
RETURNING *;

-- name: ClaimReconciliationCases :many
UPDATE reconciliation_cases
SET
    state = 'leased',
    lease_owner = $2,
    lease_deadline_at = $3,
    attempt_count = attempt_count + 1,
    updated_at = $4
WHERE reconciliation_case_id IN (
    SELECT candidate.reconciliation_case_id
    FROM reconciliation_cases AS candidate
    WHERE candidate.state IN ('open', 'waiting_evidence')
      AND candidate.next_attempt_at <= $1
    ORDER BY candidate.next_attempt_at, candidate.reconciliation_case_id
    FOR UPDATE SKIP LOCKED
    LIMIT $5
)
RETURNING *;

-- name: ListReconciliationCasesByAccount :many
SELECT *
FROM reconciliation_cases
WHERE account_id = $1
ORDER BY created_at DESC, reconciliation_case_id DESC
LIMIT $2;

-- name: CreateLegacyImportBatch :one
INSERT INTO legacy_import_batches (
    legacy_import_batch_id,
    source_system,
    source_snapshot_fingerprint,
    state,
    account_count,
    derived_total_usd_atoms,
    created_at,
    updated_at,
    applied_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8)
RETURNING *;

-- name: CreateLegacyBalanceImport :one
INSERT INTO legacy_balance_imports (
    legacy_balance_import_id,
    legacy_import_batch_id,
    account_id,
    account_scope_key,
    legacy_source_system,
    legacy_subject_id,
    legacy_balance_ngonka_text,
    legacy_locked_rate_usd_text,
    derived_usd_atoms,
    import_fingerprint,
    parity_status,
    migration_ledger_entry_id,
    correction_ledger_entry_id,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
RETURNING *;

-- name: GetLegacyBalanceImportByAccount :one
SELECT *
FROM legacy_balance_imports
WHERE legacy_import_batch_id = $1
  AND account_id = $2;
