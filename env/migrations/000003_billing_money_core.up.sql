CREATE TABLE billing_accounts (
    account_id UUID PRIMARY KEY,
    account_scope_key TEXT NOT NULL,
    account_type TEXT NOT NULL,
    subject_authority TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    state TEXT NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ,
    CONSTRAINT billing_accounts_scope_key_unique UNIQUE (account_scope_key),
    CONSTRAINT billing_accounts_subject_unique UNIQUE (subject_authority, account_type, subject_id),
    CONSTRAINT billing_accounts_id_scope_unique UNIQUE (account_id, account_scope_key),
    CONSTRAINT billing_accounts_type_check CHECK (account_type IN ('user', 'organization')),
    CONSTRAINT billing_accounts_state_check CHECK (state IN ('active', 'suspended', 'closed', 'manual_review')),
    CONSTRAINT billing_accounts_closed_at_check CHECK ((state = 'closed') = (closed_at IS NOT NULL)),
    CONSTRAINT billing_accounts_version_check CHECK (version > 0),
    CONSTRAINT billing_accounts_scope_format_check CHECK (
        (account_type = 'user' AND account_scope_key = 'user:' || subject_id)
        OR (account_type = 'organization' AND account_scope_key = 'org:' || subject_id)
    )
);

CREATE INDEX idx_billing_accounts_state_updated
ON billing_accounts (state, updated_at, account_id);

CREATE TABLE account_balances (
    account_id UUID PRIMARY KEY,
    account_scope_key TEXT NOT NULL,
    currency TEXT NOT NULL,
    settled_usd_atoms BIGINT NOT NULL,
    reserved_usd_atoms BIGINT NOT NULL,
    available_usd_atoms BIGINT NOT NULL,
    pending_usd_atoms BIGINT NOT NULL,
    version BIGINT NOT NULL,
    last_ledger_entry_id UUID,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT account_balances_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT account_balances_currency_check CHECK (currency = 'USD'),
    CONSTRAINT account_balances_settled_nonnegative CHECK (settled_usd_atoms >= 0),
    CONSTRAINT account_balances_reserved_nonnegative CHECK (reserved_usd_atoms >= 0),
    CONSTRAINT account_balances_available_nonnegative CHECK (available_usd_atoms >= 0),
    CONSTRAINT account_balances_pending_nonnegative CHECK (pending_usd_atoms >= 0),
    CONSTRAINT account_balances_available_check CHECK (available_usd_atoms = settled_usd_atoms - reserved_usd_atoms),
    CONSTRAINT account_balances_version_check CHECK (version > 0)
);

CREATE INDEX idx_account_balances_scope
ON account_balances (account_scope_key);

CREATE INDEX idx_account_balances_updated
ON account_balances (updated_at DESC, account_id);

CREATE TABLE idempotency_records (
    idempotency_record_id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES billing_accounts(account_id),
    operation_kind TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_fingerprint TEXT NOT NULL,
    state TEXT NOT NULL,
    stored_outcome_id UUID,
    conflict_reason TEXT,
    retention_class TEXT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    committed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    CONSTRAINT idempotency_records_key_unique UNIQUE (account_id, operation_kind, idempotency_key),
    CONSTRAINT idempotency_records_state_check CHECK (state IN ('started', 'committed', 'failed_stored', 'conflict', 'reconcile_required')),
    CONSTRAINT idempotency_records_operation_kind_check CHECK (operation_kind IN ('reserve', 'finalize', 'write_off', 'reversal', 'compensation', 'topup_create', 'topup_presentation_sync', 'topup_evidence', 'payment_reversal', 'operator_adjustment', 'migration_import', 'reconciliation_correction')),
    CONSTRAINT idempotency_records_retention_class_check CHECK (retention_class IN ('hot_replay', 'audit', 'legal_hold', 'expired_ready')),
    CONSTRAINT idempotency_records_stored_outcome_check CHECK ((state IN ('committed', 'failed_stored')) = (stored_outcome_id IS NOT NULL)),
    CONSTRAINT idempotency_records_conflict_reason_check CHECK ((state = 'conflict') = (conflict_reason IS NOT NULL)),
    CONSTRAINT idempotency_records_expires_at_check CHECK (expires_at IS NULL OR (committed_at IS NOT NULL AND expires_at > committed_at))
);

CREATE INDEX idx_idempotency_account_recent
ON idempotency_records (account_id, last_seen_at DESC, idempotency_record_id DESC);

CREATE INDEX idx_idempotency_expiry
ON idempotency_records (retention_class, expires_at, idempotency_record_id)
WHERE expires_at IS NOT NULL;

CREATE TABLE operation_outcomes (
    stored_outcome_id UUID PRIMARY KEY,
    idempotency_record_id UUID NOT NULL,
    account_id UUID NOT NULL REFERENCES billing_accounts(account_id),
    operation_kind TEXT NOT NULL,
    outcome_status TEXT NOT NULL,
    primary_resource_type TEXT NOT NULL,
    primary_resource_id TEXT NOT NULL,
    ledger_entry_id UUID,
    settlement_effect_id UUID,
    failure_class TEXT,
    safe_problem_code TEXT,
    safe_outcome JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT operation_outcomes_idempotency_unique UNIQUE (idempotency_record_id),
    CONSTRAINT operation_outcomes_idempotency_fk FOREIGN KEY (idempotency_record_id)
        REFERENCES idempotency_records(idempotency_record_id),
    CONSTRAINT operation_outcomes_operation_kind_check CHECK (operation_kind IN ('reserve', 'finalize', 'write_off', 'reversal', 'compensation', 'topup_create', 'topup_presentation_sync', 'topup_evidence', 'payment_reversal', 'operator_adjustment', 'migration_import', 'reconciliation_correction')),
    CONSTRAINT operation_outcomes_status_check CHECK (outcome_status IN ('success', 'stored_failure', 'conflict', 'reconcile_required')),
    CONSTRAINT operation_outcomes_resource_type_check CHECK (primary_resource_type IN ('usage_operation', 'topup_operation', 'payment_evidence', 'ledger_entry', 'reconciliation_case')),
    CONSTRAINT operation_outcomes_safe_outcome_check CHECK (safe_outcome IS NULL OR jsonb_typeof(safe_outcome) = 'object')
);

CREATE INDEX idx_operation_outcomes_account_recent
ON operation_outcomes (account_id, created_at DESC, stored_outcome_id DESC);

CREATE INDEX idx_operation_outcomes_resource
ON operation_outcomes (primary_resource_type, primary_resource_id, stored_outcome_id);

CREATE TABLE usage_operations (
    usage_operation_id UUID PRIMARY KEY,
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    state TEXT NOT NULL,
    operation_kind TEXT NOT NULL,
    client_usage_request_id TEXT NOT NULL,
    request_id TEXT,
    request_basis_fingerprint TEXT NOT NULL,
    terminal_basis_fingerprint TEXT,
    pricing_snapshot_id TEXT NOT NULL,
    pricing_snapshot_fingerprint TEXT NOT NULL,
    quote_expires_at TIMESTAMPTZ NOT NULL,
    fee_policy_version TEXT NOT NULL,
    reserve_policy_version TEXT NOT NULL,
    qualified_inference_evidence_id UUID,
    terminal_outcome_id UUID,
    settlement_effect_id UUID,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    reserved_at TIMESTAMPTZ,
    terminal_at TIMESTAMPTZ,
    CONSTRAINT usage_operations_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT usage_operations_state_check CHECK (state IN ('reserve_pending', 'reserved', 'finalize_pending', 'finalized', 'write_off_pending', 'written_off', 'reversed', 'compensated', 'reconcile_required', 'manual_review', 'expired')),
    CONSTRAINT usage_operations_kind_check CHECK (operation_kind IN ('reserve', 'finalize', 'write_off', 'reversal', 'compensation')),
    CONSTRAINT usage_operations_terminal_at_check CHECK ((state IN ('finalized', 'written_off', 'reversed', 'compensated', 'expired')) = (terminal_at IS NOT NULL)),
    CONSTRAINT usage_operations_quote_expires_check CHECK (quote_expires_at > created_at)
);

CREATE INDEX idx_usage_operations_account_recent
ON usage_operations (account_id, created_at DESC, usage_operation_id DESC);

CREATE INDEX idx_usage_operations_client_request
ON usage_operations (account_id, client_usage_request_id, usage_operation_id);

CREATE INDEX idx_usage_operations_state_updated
ON usage_operations (state, updated_at, usage_operation_id);

CREATE INDEX idx_usage_operations_request_id
ON usage_operations (request_id, usage_operation_id)
WHERE request_id IS NOT NULL;

CREATE UNIQUE INDEX idx_usage_operations_settlement_effect_unique
ON usage_operations (settlement_effect_id)
WHERE settlement_effect_id IS NOT NULL;

CREATE TABLE usage_holds (
    hold_id UUID PRIMARY KEY,
    usage_operation_id UUID NOT NULL,
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    state TEXT NOT NULL,
    reserved_usd_atoms BIGINT NOT NULL,
    released_usd_atoms BIGINT NOT NULL,
    charged_usd_atoms BIGINT NOT NULL,
    write_off_usd_atoms BIGINT NOT NULL,
    pricing_snapshot_id TEXT NOT NULL,
    pricing_snapshot_fingerprint TEXT NOT NULL,
    quote_expires_at TIMESTAMPTZ NOT NULL,
    fee_policy_version TEXT NOT NULL,
    reserve_policy_version TEXT NOT NULL,
    client_usage_request_id TEXT NOT NULL,
    request_basis_fingerprint TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    terminal_at TIMESTAMPTZ,
    CONSTRAINT usage_holds_usage_operation_unique UNIQUE (usage_operation_id),
    CONSTRAINT usage_holds_usage_operation_fk FOREIGN KEY (usage_operation_id)
        REFERENCES usage_operations(usage_operation_id),
    CONSTRAINT usage_holds_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT usage_holds_state_check CHECK (state IN ('active', 'finalized', 'released', 'written_off', 'expired', 'reversed', 'reconcile_required', 'manual_review')),
    CONSTRAINT usage_holds_reserved_positive CHECK (reserved_usd_atoms > 0),
    CONSTRAINT usage_holds_released_nonnegative CHECK (released_usd_atoms >= 0),
    CONSTRAINT usage_holds_charged_nonnegative CHECK (charged_usd_atoms >= 0),
    CONSTRAINT usage_holds_write_off_nonnegative CHECK (write_off_usd_atoms >= 0),
    CONSTRAINT usage_holds_release_charge_check CHECK (released_usd_atoms + charged_usd_atoms <= reserved_usd_atoms),
    CONSTRAINT usage_holds_terminal_at_check CHECK ((state IN ('finalized', 'released', 'written_off', 'expired', 'reversed')) = (terminal_at IS NOT NULL)),
    CONSTRAINT usage_holds_expires_quote_check CHECK (expires_at = quote_expires_at)
);

CREATE INDEX idx_usage_holds_account_state
ON usage_holds (account_id, state, updated_at DESC, hold_id DESC);

CREATE INDEX idx_usage_holds_stale
ON usage_holds (state, expires_at, hold_id)
WHERE state = 'active';

CREATE INDEX idx_usage_holds_operation
ON usage_holds (usage_operation_id, hold_id);

CREATE TABLE usage_terminal_outcomes (
    usage_terminal_outcome_id UUID PRIMARY KEY,
    usage_operation_id UUID NOT NULL REFERENCES usage_operations(usage_operation_id),
    terminal_kind TEXT NOT NULL,
    idempotency_record_id UUID NOT NULL REFERENCES idempotency_records(idempotency_record_id),
    stored_outcome_id UUID NOT NULL REFERENCES operation_outcomes(stored_outcome_id),
    ledger_entry_id UUID,
    settlement_effect_id UUID,
    charged_usd_atoms BIGINT NOT NULL,
    released_usd_atoms BIGINT NOT NULL,
    write_off_usd_atoms BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT usage_terminal_idempotency_unique UNIQUE (idempotency_record_id),
    CONSTRAINT usage_terminal_stored_outcome_unique UNIQUE (stored_outcome_id),
    CONSTRAINT usage_terminal_kind_check CHECK (terminal_kind IN ('finalize', 'write_off', 'reversal', 'compensation')),
    CONSTRAINT usage_terminal_charged_nonnegative CHECK (charged_usd_atoms >= 0),
    CONSTRAINT usage_terminal_released_nonnegative CHECK (released_usd_atoms >= 0),
    CONSTRAINT usage_terminal_write_off_nonnegative CHECK (write_off_usd_atoms >= 0)
);

CREATE UNIQUE INDEX idx_usage_terminal_settlement_effect_unique
ON usage_terminal_outcomes (settlement_effect_id)
WHERE settlement_effect_id IS NOT NULL;

CREATE UNIQUE INDEX idx_usage_terminal_one_terminal_path
ON usage_terminal_outcomes (usage_operation_id)
WHERE terminal_kind IN ('finalize', 'write_off');

CREATE UNIQUE INDEX idx_usage_terminal_kind_unique
ON usage_terminal_outcomes (usage_operation_id, terminal_kind)
WHERE terminal_kind IN ('finalize', 'write_off');

CREATE INDEX idx_usage_terminal_operation_recent
ON usage_terminal_outcomes (usage_operation_id, created_at DESC, usage_terminal_outcome_id DESC);

CREATE TABLE qualified_inference_evidence (
    qualified_inference_evidence_id UUID PRIMARY KEY,
    usage_operation_id UUID NOT NULL REFERENCES usage_operations(usage_operation_id),
    provider_family TEXT NOT NULL,
    verification_surface TEXT NOT NULL,
    proof_scope TEXT NOT NULL,
    inference_id TEXT NOT NULL,
    evidence_fingerprint TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT qualified_inference_scope_unique UNIQUE (provider_family, verification_surface, proof_scope, inference_id),
    CONSTRAINT qualified_inference_fingerprint_unique UNIQUE (evidence_fingerprint)
);

CREATE INDEX idx_inference_evidence_usage
ON qualified_inference_evidence (usage_operation_id, created_at DESC);

CREATE TABLE topup_operations (
    topup_operation_id UUID PRIMARY KEY,
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    state TEXT NOT NULL,
    accepted_quote_id TEXT NOT NULL,
    credited_usd_atoms BIGINT NOT NULL,
    deposit_fee_usd_atoms BIGINT NOT NULL,
    payin_amount_value TEXT,
    payin_currency TEXT,
    pricing_snapshot_id TEXT NOT NULL,
    pricing_snapshot_fingerprint TEXT NOT NULL,
    settlement_policy_version TEXT NOT NULL,
    billing_fee_policy_version TEXT NOT NULL,
    current_payment_attempt_id TEXT,
    attempt_generation INTEGER NOT NULL,
    presentation_version INTEGER NOT NULL,
    presentation_fingerprint TEXT,
    settlement_effect_id UUID,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    settlement_applied_at TIMESTAMPTZ,
    CONSTRAINT topup_operations_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT topup_operations_state_check CHECK (state IN ('created', 'payment_pending', 'presentation_synced', 'evidence_pending', 'settlement_applied', 'duplicate_evidence', 'evidence_conflict', 'late_evidence_review', 'reversed', 'expired', 'manual_review', 'reconcile_required')),
    CONSTRAINT topup_operations_credited_positive CHECK (credited_usd_atoms > 0),
    CONSTRAINT topup_operations_deposit_fee_nonnegative CHECK (deposit_fee_usd_atoms >= 0),
    CONSTRAINT topup_operations_attempt_generation_check CHECK (attempt_generation >= 0),
    CONSTRAINT topup_operations_presentation_version_check CHECK (presentation_version >= 0),
    CONSTRAINT topup_operations_applied_at_required CHECK (state <> 'settlement_applied' OR settlement_applied_at IS NOT NULL),
    CONSTRAINT topup_operations_applied_state_check CHECK (settlement_applied_at IS NULL OR state IN ('settlement_applied', 'duplicate_evidence', 'reversed', 'manual_review', 'reconcile_required'))
);

CREATE UNIQUE INDEX idx_topup_settlement_effect_unique
ON topup_operations (settlement_effect_id)
WHERE settlement_effect_id IS NOT NULL;

CREATE INDEX idx_topup_account_recent
ON topup_operations (account_id, created_at DESC, topup_operation_id DESC);

CREATE INDEX idx_topup_state_updated
ON topup_operations (state, updated_at, topup_operation_id);

CREATE INDEX idx_topup_current_attempt
ON topup_operations (topup_operation_id, current_payment_attempt_id)
WHERE current_payment_attempt_id IS NOT NULL;

CREATE TABLE payment_attempts (
    payment_attempt_row_id UUID PRIMARY KEY,
    topup_operation_id UUID NOT NULL REFERENCES topup_operations(topup_operation_id),
    payment_attempt_id TEXT NOT NULL,
    attempt_generation INTEGER NOT NULL,
    state TEXT NOT NULL,
    presentation_version INTEGER NOT NULL,
    presentation_fingerprint TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    CONSTRAINT payment_attempts_topup_attempt_unique UNIQUE (topup_operation_id, payment_attempt_id),
    CONSTRAINT payment_attempts_topup_generation_unique UNIQUE (topup_operation_id, attempt_generation),
    CONSTRAINT payment_attempts_state_check CHECK (state IN ('created', 'presentation_synced', 'evidence_pending', 'evidence_accepted', 'expired', 'conflict', 'manual_review')),
    CONSTRAINT payment_attempts_generation_check CHECK (attempt_generation >= 0),
    CONSTRAINT payment_attempts_presentation_version_check CHECK (presentation_version >= 0)
);

CREATE INDEX idx_payment_attempts_topup_recent
ON payment_attempts (topup_operation_id, attempt_generation DESC);

CREATE INDEX idx_payment_attempts_id
ON payment_attempts (payment_attempt_id, payment_attempt_row_id);

ALTER TABLE topup_operations
ADD CONSTRAINT topup_operations_current_attempt_fk
FOREIGN KEY (topup_operation_id, current_payment_attempt_id)
REFERENCES payment_attempts(topup_operation_id, payment_attempt_id);

CREATE TABLE payment_evidence (
    payment_evidence_id TEXT PRIMARY KEY,
    topup_operation_id UUID NOT NULL,
    payment_attempt_id TEXT NOT NULL,
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    state TEXT NOT NULL,
    evidence_payload_fingerprint TEXT NOT NULL,
    evidence_kind TEXT NOT NULL,
    schema_version TEXT NOT NULL,
    finality_class TEXT NOT NULL,
    rail_family TEXT NOT NULL,
    settlement_amount_usd_atoms BIGINT NOT NULL,
    settlement_effect_id UUID,
    ledger_entry_id UUID,
    prior_payment_evidence_id TEXT,
    received_at TIMESTAMPTZ NOT NULL,
    provider_event_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT payment_evidence_topup_fk FOREIGN KEY (topup_operation_id)
        REFERENCES topup_operations(topup_operation_id),
    CONSTRAINT payment_evidence_attempt_fk FOREIGN KEY (topup_operation_id, payment_attempt_id)
        REFERENCES payment_attempts(topup_operation_id, payment_attempt_id),
    CONSTRAINT payment_evidence_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT payment_evidence_fingerprint_unique UNIQUE (evidence_payload_fingerprint),
    CONSTRAINT payment_evidence_prior_fk FOREIGN KEY (prior_payment_evidence_id)
        REFERENCES payment_evidence(payment_evidence_id),
    CONSTRAINT payment_evidence_amount_positive CHECK (settlement_amount_usd_atoms > 0),
    CONSTRAINT payment_evidence_state_check CHECK (state IN ('accepted', 'duplicate', 'conflict', 'late_review', 'reversed', 'reconcile_required', 'manual_review')),
    CONSTRAINT payment_evidence_kind_check CHECK (evidence_kind IN ('settlement', 'duplicate_notice', 'reversal', 'refund', 'adjustment')),
    CONSTRAINT payment_evidence_finality_check CHECK (finality_class IN ('final', 'provisional', 'reversed', 'disputed')),
    CONSTRAINT payment_evidence_rail_check CHECK (rail_family IN ('card', 'bank_transfer', 'crypto', 'internal', 'manual'))
);

CREATE UNIQUE INDEX idx_payment_evidence_settlement_effect_unique
ON payment_evidence (settlement_effect_id)
WHERE settlement_effect_id IS NOT NULL;

CREATE INDEX idx_payment_evidence_topup_recent
ON payment_evidence (topup_operation_id, created_at DESC, payment_evidence_id DESC);

CREATE INDEX idx_payment_evidence_account_recent
ON payment_evidence (account_id, created_at DESC, payment_evidence_id DESC);

CREATE INDEX idx_payment_evidence_state
ON payment_evidence (state, created_at, payment_evidence_id);

CREATE TABLE ledger_entries (
    ledger_entry_id UUID PRIMARY KEY,
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    currency TEXT NOT NULL,
    effect_type TEXT NOT NULL,
    amount_usd_atoms BIGINT NOT NULL,
    settled_delta_usd_atoms BIGINT NOT NULL,
    reserved_delta_usd_atoms BIGINT NOT NULL,
    pending_delta_usd_atoms BIGINT NOT NULL,
    settled_after_usd_atoms BIGINT NOT NULL,
    reserved_after_usd_atoms BIGINT NOT NULL,
    available_after_usd_atoms BIGINT NOT NULL,
    pending_after_usd_atoms BIGINT NOT NULL,
    balance_version_after BIGINT NOT NULL,
    settlement_effect_id UUID,
    idempotency_record_id UUID,
    usage_operation_id UUID,
    topup_operation_id UUID,
    payment_attempt_id TEXT,
    payment_evidence_id TEXT,
    qualified_inference_evidence_id UUID,
    reversal_of_ledger_entry_id UUID,
    correction_of_ledger_entry_id UUID,
    effective_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    created_by_kind TEXT NOT NULL,
    reason_code TEXT NOT NULL,
    safe_metadata JSONB,
    CONSTRAINT ledger_entries_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT ledger_entries_idempotency_fk FOREIGN KEY (idempotency_record_id)
        REFERENCES idempotency_records(idempotency_record_id),
    CONSTRAINT ledger_entries_usage_fk FOREIGN KEY (usage_operation_id)
        REFERENCES usage_operations(usage_operation_id),
    CONSTRAINT ledger_entries_topup_fk FOREIGN KEY (topup_operation_id)
        REFERENCES topup_operations(topup_operation_id),
    CONSTRAINT ledger_entries_payment_evidence_fk FOREIGN KEY (payment_evidence_id)
        REFERENCES payment_evidence(payment_evidence_id),
    CONSTRAINT ledger_entries_inference_fk FOREIGN KEY (qualified_inference_evidence_id)
        REFERENCES qualified_inference_evidence(qualified_inference_evidence_id),
    CONSTRAINT ledger_entries_reversal_fk FOREIGN KEY (reversal_of_ledger_entry_id)
        REFERENCES ledger_entries(ledger_entry_id),
    CONSTRAINT ledger_entries_correction_fk FOREIGN KEY (correction_of_ledger_entry_id)
        REFERENCES ledger_entries(ledger_entry_id),
    CONSTRAINT ledger_entries_currency_check CHECK (currency = 'USD'),
    CONSTRAINT ledger_entries_amount_nonzero CHECK (amount_usd_atoms <> 0),
    CONSTRAINT ledger_entries_settled_after_nonnegative CHECK (settled_after_usd_atoms >= 0),
    CONSTRAINT ledger_entries_reserved_after_nonnegative CHECK (reserved_after_usd_atoms >= 0),
    CONSTRAINT ledger_entries_available_after_nonnegative CHECK (available_after_usd_atoms >= 0),
    CONSTRAINT ledger_entries_pending_after_nonnegative CHECK (pending_after_usd_atoms >= 0),
    CONSTRAINT ledger_entries_available_after_check CHECK (available_after_usd_atoms = settled_after_usd_atoms - reserved_after_usd_atoms),
    CONSTRAINT ledger_entries_balance_version_check CHECK (balance_version_after > 0),
    CONSTRAINT ledger_entries_effect_type_check CHECK (effect_type IN ('topup_credit', 'usage_charge', 'usage_hold', 'usage_hold_release', 'usage_write_off', 'usage_reversal', 'payment_reversal', 'operator_adjustment', 'migration_import', 'reconciliation_correction', 'topup_pending', 'topup_pending_release')),
    CONSTRAINT ledger_entries_created_by_kind_check CHECK (created_by_kind IN ('service', 'worker', 'operator', 'migration')),
    CONSTRAINT ledger_entries_safe_metadata_check CHECK (safe_metadata IS NULL OR jsonb_typeof(safe_metadata) = 'object'),
    CONSTRAINT ledger_entries_delta_pattern_check CHECK (
        (
            effect_type IN ('topup_credit', 'migration_import')
            AND amount_usd_atoms = settled_delta_usd_atoms
            AND settled_delta_usd_atoms > 0
            AND reserved_delta_usd_atoms = 0
            AND pending_delta_usd_atoms = 0
        )
        OR (
            effect_type = 'usage_hold'
            AND amount_usd_atoms = reserved_delta_usd_atoms
            AND reserved_delta_usd_atoms > 0
            AND settled_delta_usd_atoms = 0
            AND pending_delta_usd_atoms = 0
        )
        OR (
            effect_type IN ('usage_hold_release', 'usage_write_off')
            AND amount_usd_atoms = reserved_delta_usd_atoms
            AND reserved_delta_usd_atoms < 0
            AND settled_delta_usd_atoms = 0
            AND pending_delta_usd_atoms = 0
        )
        OR (
            effect_type = 'usage_charge'
            AND amount_usd_atoms = settled_delta_usd_atoms
            AND settled_delta_usd_atoms < 0
            AND reserved_delta_usd_atoms <= 0
            AND pending_delta_usd_atoms = 0
        )
        OR (
            effect_type = 'topup_pending'
            AND amount_usd_atoms = pending_delta_usd_atoms
            AND pending_delta_usd_atoms > 0
            AND settled_delta_usd_atoms = 0
            AND reserved_delta_usd_atoms = 0
        )
        OR (
            effect_type = 'topup_pending_release'
            AND amount_usd_atoms = pending_delta_usd_atoms
            AND pending_delta_usd_atoms < 0
            AND settled_delta_usd_atoms = 0
            AND reserved_delta_usd_atoms = 0
        )
        OR (
            effect_type IN ('usage_reversal', 'payment_reversal', 'operator_adjustment', 'reconciliation_correction')
            AND amount_usd_atoms = settled_delta_usd_atoms
            AND reserved_delta_usd_atoms = 0
            AND pending_delta_usd_atoms = 0
        )
    )
);

CREATE FUNCTION prevent_ledger_money_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF (
        OLD.account_id,
        OLD.account_scope_key,
        OLD.currency,
        OLD.effect_type,
        OLD.amount_usd_atoms,
        OLD.settled_delta_usd_atoms,
        OLD.reserved_delta_usd_atoms,
        OLD.pending_delta_usd_atoms,
        OLD.settled_after_usd_atoms,
        OLD.reserved_after_usd_atoms,
        OLD.available_after_usd_atoms,
        OLD.pending_after_usd_atoms,
        OLD.balance_version_after,
        OLD.settlement_effect_id,
        OLD.idempotency_record_id,
        OLD.usage_operation_id,
        OLD.topup_operation_id,
        OLD.payment_attempt_id,
        OLD.payment_evidence_id,
        OLD.qualified_inference_evidence_id,
        OLD.reversal_of_ledger_entry_id,
        OLD.correction_of_ledger_entry_id,
        OLD.effective_at,
        OLD.created_at,
        OLD.created_by_kind
    ) IS DISTINCT FROM (
        NEW.account_id,
        NEW.account_scope_key,
        NEW.currency,
        NEW.effect_type,
        NEW.amount_usd_atoms,
        NEW.settled_delta_usd_atoms,
        NEW.reserved_delta_usd_atoms,
        NEW.pending_delta_usd_atoms,
        NEW.settled_after_usd_atoms,
        NEW.reserved_after_usd_atoms,
        NEW.available_after_usd_atoms,
        NEW.pending_after_usd_atoms,
        NEW.balance_version_after,
        NEW.settlement_effect_id,
        NEW.idempotency_record_id,
        NEW.usage_operation_id,
        NEW.topup_operation_id,
        NEW.payment_attempt_id,
        NEW.payment_evidence_id,
        NEW.qualified_inference_evidence_id,
        NEW.reversal_of_ledger_entry_id,
        NEW.correction_of_ledger_entry_id,
        NEW.effective_at,
        NEW.created_at,
        NEW.created_by_kind
    ) THEN
        RAISE EXCEPTION 'posted ledger money fields are immutable'
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_prevent_ledger_money_mutation
BEFORE UPDATE ON ledger_entries
FOR EACH ROW
EXECUTE FUNCTION prevent_ledger_money_mutation();

CREATE INDEX idx_ledger_account_recent
ON ledger_entries (account_id, created_at DESC, ledger_entry_id DESC);

CREATE INDEX idx_ledger_scope_recent
ON ledger_entries (account_scope_key, created_at DESC, ledger_entry_id DESC);

CREATE INDEX idx_ledger_usage
ON ledger_entries (usage_operation_id, created_at, ledger_entry_id)
WHERE usage_operation_id IS NOT NULL;

CREATE INDEX idx_ledger_topup
ON ledger_entries (topup_operation_id, created_at, ledger_entry_id)
WHERE topup_operation_id IS NOT NULL;

CREATE INDEX idx_ledger_payment_evidence
ON ledger_entries (payment_evidence_id, ledger_entry_id)
WHERE payment_evidence_id IS NOT NULL;

CREATE UNIQUE INDEX idx_ledger_settlement_effect_unique
ON ledger_entries (settlement_effect_id)
WHERE settlement_effect_id IS NOT NULL;

CREATE INDEX idx_ledger_reversal_links
ON ledger_entries (reversal_of_ledger_entry_id, correction_of_ledger_entry_id);

ALTER TABLE account_balances
ADD CONSTRAINT account_balances_last_ledger_fk
FOREIGN KEY (last_ledger_entry_id)
REFERENCES ledger_entries(ledger_entry_id);

ALTER TABLE idempotency_records
ADD CONSTRAINT idempotency_records_stored_outcome_fk
FOREIGN KEY (stored_outcome_id)
REFERENCES operation_outcomes(stored_outcome_id);

ALTER TABLE operation_outcomes
ADD CONSTRAINT operation_outcomes_ledger_fk
FOREIGN KEY (ledger_entry_id)
REFERENCES ledger_entries(ledger_entry_id);

ALTER TABLE usage_operations
ADD CONSTRAINT usage_operations_inference_fk
FOREIGN KEY (qualified_inference_evidence_id)
REFERENCES qualified_inference_evidence(qualified_inference_evidence_id);

ALTER TABLE usage_operations
ADD CONSTRAINT usage_operations_terminal_outcome_fk
FOREIGN KEY (terminal_outcome_id)
REFERENCES operation_outcomes(stored_outcome_id);

ALTER TABLE usage_terminal_outcomes
ADD CONSTRAINT usage_terminal_ledger_fk
FOREIGN KEY (ledger_entry_id)
REFERENCES ledger_entries(ledger_entry_id);

ALTER TABLE payment_evidence
ADD CONSTRAINT payment_evidence_ledger_fk
FOREIGN KEY (ledger_entry_id)
REFERENCES ledger_entries(ledger_entry_id);

CREATE TABLE legacy_import_batches (
    legacy_import_batch_id UUID PRIMARY KEY,
    source_system TEXT NOT NULL,
    source_snapshot_fingerprint TEXT NOT NULL,
    state TEXT NOT NULL,
    account_count BIGINT NOT NULL,
    derived_total_usd_atoms BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    applied_at TIMESTAMPTZ,
    CONSTRAINT legacy_import_batches_source_unique UNIQUE (source_system, source_snapshot_fingerprint),
    CONSTRAINT legacy_import_batches_state_check CHECK (state IN ('loaded', 'parity_checked', 'applied', 'failed', 'superseded')),
    CONSTRAINT legacy_import_batches_account_count_check CHECK (account_count >= 0),
    CONSTRAINT legacy_import_batches_total_check CHECK (derived_total_usd_atoms >= 0)
);

CREATE INDEX idx_legacy_import_batches_state
ON legacy_import_batches (state, created_at DESC, legacy_import_batch_id DESC);

CREATE TABLE legacy_balance_imports (
    legacy_balance_import_id UUID PRIMARY KEY,
    legacy_import_batch_id UUID NOT NULL REFERENCES legacy_import_batches(legacy_import_batch_id),
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    legacy_source_system TEXT NOT NULL,
    legacy_subject_id TEXT NOT NULL,
    legacy_balance_ngonka_text TEXT NOT NULL,
    legacy_locked_rate_usd_text TEXT NOT NULL,
    derived_usd_atoms BIGINT NOT NULL,
    import_fingerprint TEXT NOT NULL,
    parity_status TEXT NOT NULL,
    migration_ledger_entry_id UUID,
    correction_ledger_entry_id UUID,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT legacy_balance_imports_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT legacy_balance_imports_source_subject_unique UNIQUE (legacy_import_batch_id, legacy_source_system, legacy_subject_id),
    CONSTRAINT legacy_balance_imports_batch_account_unique UNIQUE (legacy_import_batch_id, account_id),
    CONSTRAINT legacy_balance_imports_fingerprint_unique UNIQUE (import_fingerprint),
    CONSTRAINT legacy_balance_imports_derived_nonnegative CHECK (derived_usd_atoms >= 0),
    CONSTRAINT legacy_balance_imports_parity_check CHECK (parity_status IN ('pending', 'matched', 'mismatch', 'corrected', 'ignored')),
    CONSTRAINT legacy_balance_imports_migration_ledger_fk FOREIGN KEY (migration_ledger_entry_id)
        REFERENCES ledger_entries(ledger_entry_id),
    CONSTRAINT legacy_balance_imports_correction_ledger_fk FOREIGN KEY (correction_ledger_entry_id)
        REFERENCES ledger_entries(ledger_entry_id)
);

CREATE INDEX idx_legacy_imports_account
ON legacy_balance_imports (account_id, created_at DESC, legacy_balance_import_id DESC);

CREATE INDEX idx_legacy_imports_parity
ON legacy_balance_imports (legacy_import_batch_id, parity_status, legacy_balance_import_id);

CREATE TABLE reconciliation_cases (
    reconciliation_case_id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES billing_accounts(account_id),
    reason TEXT NOT NULL,
    state TEXT NOT NULL,
    severity TEXT NOT NULL,
    usage_operation_id UUID,
    topup_operation_id UUID,
    payment_attempt_id TEXT,
    payment_evidence_id TEXT,
    settlement_effect_id UUID,
    qualified_inference_evidence_id UUID,
    ledger_entry_id UUID,
    legacy_balance_import_id UUID,
    resolution_ledger_entry_id UUID,
    resolution_settlement_effect_id UUID,
    lease_owner TEXT,
    lease_deadline_at TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    support_safe_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    CONSTRAINT reconciliation_cases_usage_fk FOREIGN KEY (usage_operation_id)
        REFERENCES usage_operations(usage_operation_id),
    CONSTRAINT reconciliation_cases_attempt_fk FOREIGN KEY (topup_operation_id, payment_attempt_id)
        REFERENCES payment_attempts(topup_operation_id, payment_attempt_id),
    CONSTRAINT reconciliation_cases_evidence_fk FOREIGN KEY (payment_evidence_id)
        REFERENCES payment_evidence(payment_evidence_id),
    CONSTRAINT reconciliation_cases_inference_fk FOREIGN KEY (qualified_inference_evidence_id)
        REFERENCES qualified_inference_evidence(qualified_inference_evidence_id),
    CONSTRAINT reconciliation_cases_ledger_fk FOREIGN KEY (ledger_entry_id)
        REFERENCES ledger_entries(ledger_entry_id),
    CONSTRAINT reconciliation_cases_import_fk FOREIGN KEY (legacy_balance_import_id)
        REFERENCES legacy_balance_imports(legacy_balance_import_id),
    CONSTRAINT reconciliation_cases_resolution_ledger_fk FOREIGN KEY (resolution_ledger_entry_id)
        REFERENCES ledger_entries(ledger_entry_id),
    CONSTRAINT reconciliation_cases_reason_check CHECK (reason IN ('stale_reservation', 'ambiguous_terminal_state', 'duplicate_payment_evidence', 'evidence_conflict', 'missing_inference_evidence', 'late_payment_evidence', 'provider_reference_mismatch', 'legacy_import_mismatch', 'operator_adjustment_required')),
    CONSTRAINT reconciliation_cases_state_check CHECK (state IN ('open', 'leased', 'waiting_evidence', 'manual_review', 'resolved', 'canceled')),
    CONSTRAINT reconciliation_cases_severity_check CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT reconciliation_cases_attempt_count_check CHECK (attempt_count >= 0),
    CONSTRAINT reconciliation_cases_lease_check CHECK ((state = 'leased') = (lease_owner IS NOT NULL AND lease_deadline_at IS NOT NULL)),
    CONSTRAINT reconciliation_cases_resolved_check CHECK ((state = 'resolved') = (resolved_at IS NOT NULL)),
    CONSTRAINT reconciliation_cases_lineage_check CHECK (
        (reason IN ('stale_reservation', 'ambiguous_terminal_state', 'missing_inference_evidence') AND usage_operation_id IS NOT NULL)
        OR (reason IN ('duplicate_payment_evidence', 'evidence_conflict', 'late_payment_evidence') AND payment_evidence_id IS NOT NULL)
        OR (reason = 'provider_reference_mismatch' AND ((topup_operation_id IS NOT NULL AND payment_attempt_id IS NOT NULL) OR settlement_effect_id IS NOT NULL))
        OR (reason = 'legacy_import_mismatch' AND legacy_balance_import_id IS NOT NULL)
        OR (reason = 'operator_adjustment_required' AND (ledger_entry_id IS NOT NULL OR settlement_effect_id IS NOT NULL OR (ledger_entry_id IS NULL AND settlement_effect_id IS NULL)))
    )
);

CREATE INDEX idx_reconciliation_claim
ON reconciliation_cases (state, next_attempt_at, reconciliation_case_id)
WHERE state IN ('open', 'waiting_evidence');

CREATE INDEX idx_reconciliation_leases
ON reconciliation_cases (state, lease_deadline_at, reconciliation_case_id)
WHERE state = 'leased';

CREATE INDEX idx_reconciliation_account_recent
ON reconciliation_cases (account_id, created_at DESC, reconciliation_case_id DESC);

CREATE UNIQUE INDEX idx_reconciliation_usage_open_unique
ON reconciliation_cases (reason, usage_operation_id)
WHERE reason IN ('stale_reservation', 'ambiguous_terminal_state', 'missing_inference_evidence')
  AND usage_operation_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_attempt_open_unique
ON reconciliation_cases (reason, topup_operation_id, payment_attempt_id)
WHERE reason = 'provider_reference_mismatch'
  AND topup_operation_id IS NOT NULL
  AND payment_attempt_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_evidence_open_unique
ON reconciliation_cases (reason, payment_evidence_id)
WHERE reason IN ('duplicate_payment_evidence', 'evidence_conflict', 'late_payment_evidence')
  AND payment_evidence_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_settlement_open_unique
ON reconciliation_cases (reason, settlement_effect_id)
WHERE reason IN ('ambiguous_terminal_state', 'provider_reference_mismatch', 'operator_adjustment_required')
  AND settlement_effect_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_inference_open_unique
ON reconciliation_cases (reason, qualified_inference_evidence_id)
WHERE qualified_inference_evidence_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_ledger_open_unique
ON reconciliation_cases (reason, ledger_entry_id)
WHERE reason IN ('operator_adjustment_required', 'legacy_import_mismatch', 'ambiguous_terminal_state')
  AND ledger_entry_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_import_open_unique
ON reconciliation_cases (reason, legacy_balance_import_id)
WHERE reason = 'legacy_import_mismatch'
  AND legacy_balance_import_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE TABLE audit_events (
    audit_event_id UUID PRIMARY KEY,
    account_id UUID REFERENCES billing_accounts(account_id),
    actor_kind TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    reason_code TEXT NOT NULL,
    operation_kind TEXT,
    usage_operation_id UUID REFERENCES usage_operations(usage_operation_id),
    topup_operation_id UUID REFERENCES topup_operations(topup_operation_id),
    payment_evidence_id TEXT REFERENCES payment_evidence(payment_evidence_id),
    ledger_entry_id UUID REFERENCES ledger_entries(ledger_entry_id),
    reconciliation_case_id UUID REFERENCES reconciliation_cases(reconciliation_case_id),
    before_state TEXT,
    after_state TEXT,
    amount_usd_atoms BIGINT,
    request_id TEXT,
    trace_id TEXT,
    safe_metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT audit_events_actor_kind_check CHECK (actor_kind IN ('service', 'worker', 'operator', 'migration')),
    CONSTRAINT audit_events_operation_kind_check CHECK (operation_kind IS NULL OR operation_kind IN ('reserve', 'finalize', 'write_off', 'reversal', 'compensation', 'topup_create', 'topup_presentation_sync', 'topup_evidence', 'payment_reversal', 'operator_adjustment', 'migration_import', 'reconciliation_correction')),
    CONSTRAINT audit_events_safe_metadata_check CHECK (safe_metadata IS NULL OR jsonb_typeof(safe_metadata) = 'object')
);

CREATE INDEX idx_audit_account_recent
ON audit_events (account_id, created_at DESC, audit_event_id DESC)
WHERE account_id IS NOT NULL;

CREATE INDEX idx_audit_ledger
ON audit_events (ledger_entry_id, audit_event_id)
WHERE ledger_entry_id IS NOT NULL;

CREATE INDEX idx_audit_reconciliation
ON audit_events (reconciliation_case_id, created_at, audit_event_id)
WHERE reconciliation_case_id IS NOT NULL;
