ALTER TABLE idempotency_records
DROP CONSTRAINT idempotency_records_operation_kind_check;

ALTER TABLE idempotency_records
ADD CONSTRAINT idempotency_records_operation_kind_check CHECK (operation_kind IN (
    'reserve',
    'finalize',
    'write_off',
    'reversal',
    'compensation',
    'topup_create',
    'topup_presentation_sync',
    'topup_evidence',
    'payment_reversal',
    'operator_adjustment',
    'migration_import',
    'reconciliation_correction',
    'microlease_issue',
    'microlease_close',
    'microlease_terminal',
    'microlease_checkpoint',
    'microlease_reconciliation'
));

ALTER TABLE operation_outcomes
DROP CONSTRAINT operation_outcomes_operation_kind_check;

ALTER TABLE operation_outcomes
ADD CONSTRAINT operation_outcomes_operation_kind_check CHECK (operation_kind IN (
    'reserve',
    'finalize',
    'write_off',
    'reversal',
    'compensation',
    'topup_create',
    'topup_presentation_sync',
    'topup_evidence',
    'payment_reversal',
    'operator_adjustment',
    'migration_import',
    'reconciliation_correction',
    'microlease_issue',
    'microlease_close',
    'microlease_terminal',
    'microlease_checkpoint',
    'microlease_reconciliation'
));

ALTER TABLE operation_outcomes
DROP CONSTRAINT operation_outcomes_resource_type_check;

ALTER TABLE operation_outcomes
ADD CONSTRAINT operation_outcomes_resource_type_check CHECK (primary_resource_type IN (
    'usage_operation',
    'topup_operation',
    'payment_evidence',
    'ledger_entry',
    'reconciliation_case',
    'spending_microlease',
    'microlease_child_debit',
    'microlease_checkpoint',
    'billing_event_inbox',
    'billing_outbox',
    'billing_admission_control'
));

ALTER TABLE ledger_entries
DROP CONSTRAINT ledger_entries_effect_type_check;

ALTER TABLE ledger_entries
ADD CONSTRAINT ledger_entries_effect_type_check CHECK (effect_type IN (
    'topup_credit',
    'usage_charge',
    'usage_hold',
    'usage_hold_release',
    'usage_write_off',
    'usage_reversal',
    'payment_reversal',
    'operator_adjustment',
    'migration_import',
    'reconciliation_correction',
    'topup_pending',
    'topup_pending_release',
    'microlease_reserve',
    'microlease_child_charge',
    'microlease_child_release',
    'microlease_write_off',
    'microlease_close_release',
    'microlease_reversal',
    'microlease_compensation'
));

ALTER TABLE ledger_entries
DROP CONSTRAINT ledger_entries_delta_pattern_check;

ALTER TABLE ledger_entries
ADD CONSTRAINT ledger_entries_delta_pattern_check CHECK (
    (
        effect_type IN ('topup_credit', 'migration_import')
        AND amount_usd_atoms = settled_delta_usd_atoms
        AND settled_delta_usd_atoms > 0
        AND reserved_delta_usd_atoms = 0
        AND pending_delta_usd_atoms = 0
    )
    OR (
        effect_type IN ('usage_hold', 'microlease_reserve')
        AND amount_usd_atoms = reserved_delta_usd_atoms
        AND reserved_delta_usd_atoms > 0
        AND settled_delta_usd_atoms = 0
        AND pending_delta_usd_atoms = 0
    )
    OR (
        effect_type IN ('usage_hold_release', 'usage_write_off', 'microlease_child_release', 'microlease_write_off', 'microlease_close_release')
        AND amount_usd_atoms = reserved_delta_usd_atoms
        AND reserved_delta_usd_atoms < 0
        AND settled_delta_usd_atoms = 0
        AND pending_delta_usd_atoms = 0
    )
    OR (
        effect_type IN ('usage_charge', 'microlease_child_charge')
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
        effect_type IN ('usage_reversal', 'payment_reversal', 'operator_adjustment', 'reconciliation_correction', 'microlease_reversal', 'microlease_compensation')
        AND amount_usd_atoms = settled_delta_usd_atoms
        AND reserved_delta_usd_atoms = 0
        AND pending_delta_usd_atoms = 0
    )
);

ALTER TABLE audit_events
DROP CONSTRAINT audit_events_operation_kind_check;

ALTER TABLE audit_events
ADD CONSTRAINT audit_events_operation_kind_check CHECK (operation_kind IS NULL OR operation_kind IN (
    'reserve',
    'finalize',
    'write_off',
    'reversal',
    'compensation',
    'topup_create',
    'topup_presentation_sync',
    'topup_evidence',
    'payment_reversal',
    'operator_adjustment',
    'migration_import',
    'reconciliation_correction',
    'microlease_issue',
    'microlease_close',
    'microlease_terminal',
    'microlease_checkpoint',
    'microlease_reconciliation'
));

CREATE TABLE billing_event_inbox (
    inbox_id UUID PRIMARY KEY,
    topic TEXT NOT NULL,
    partition_id INTEGER NOT NULL,
    offset_value BIGINT NOT NULL,
    event_id TEXT NOT NULL,
    producer_identity TEXT NOT NULL,
    business_identity_type TEXT NOT NULL,
    business_identity_value TEXT NOT NULL,
    event_fingerprint TEXT NOT NULL,
    state TEXT NOT NULL,
    failure_class TEXT,
    safe_metadata JSONB,
    received_at TIMESTAMPTZ NOT NULL,
    applied_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT billing_event_inbox_event_unique UNIQUE (topic, event_id),
    CONSTRAINT billing_event_inbox_offset_unique UNIQUE (topic, partition_id, offset_value),
    CONSTRAINT billing_event_inbox_partition_check CHECK (partition_id >= 0),
    CONSTRAINT billing_event_inbox_offset_check CHECK (offset_value >= 0),
    CONSTRAINT billing_event_inbox_state_check CHECK (state IN ('received', 'applied', 'duplicate', 'conflict', 'quarantined', 'reconcile_required')),
    CONSTRAINT billing_event_inbox_applied_check CHECK ((state = 'applied') = (applied_at IS NOT NULL)),
    CONSTRAINT billing_event_inbox_failure_check CHECK ((state IN ('conflict', 'quarantined', 'reconcile_required')) = (failure_class IS NOT NULL)),
    CONSTRAINT billing_event_inbox_safe_metadata_check CHECK (safe_metadata IS NULL OR jsonb_typeof(safe_metadata) = 'object')
);

CREATE INDEX idx_billing_event_inbox_business
ON billing_event_inbox (business_identity_type, business_identity_value, received_at DESC, inbox_id DESC);

CREATE INDEX idx_billing_event_inbox_state
ON billing_event_inbox (state, updated_at, inbox_id)
WHERE state IN ('received', 'conflict', 'quarantined', 'reconcile_required');

CREATE TABLE billing_outbox (
    outbox_id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_fingerprint TEXT NOT NULL,
    safe_payload JSONB NOT NULL,
    state TEXT NOT NULL,
    attempt_count INTEGER NOT NULL,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT billing_outbox_fingerprint_unique UNIQUE (event_fingerprint),
    CONSTRAINT billing_outbox_state_check CHECK (state IN ('pending', 'publishing', 'published', 'failed', 'quarantined')),
    CONSTRAINT billing_outbox_attempt_count_check CHECK (attempt_count >= 0),
    CONSTRAINT billing_outbox_published_check CHECK ((state = 'published') = (published_at IS NOT NULL)),
    CONSTRAINT billing_outbox_payload_check CHECK (jsonb_typeof(safe_payload) = 'object')
);

CREATE INDEX idx_billing_outbox_claim
ON billing_outbox (state, next_attempt_at, outbox_id)
WHERE state IN ('pending', 'failed');

CREATE INDEX idx_billing_outbox_aggregate
ON billing_outbox (aggregate_type, aggregate_id, created_at DESC, outbox_id DESC);

CREATE TABLE billing_admission_controls (
    admission_control_id UUID PRIMARY KEY,
    scope_kind TEXT NOT NULL,
    scope_key TEXT NOT NULL,
    use_class TEXT NOT NULL,
    state TEXT NOT NULL,
    reason_code TEXT NOT NULL,
    terminal_lag_bucket TEXT NOT NULL,
    stale_age_bucket TEXT NOT NULL,
    reconciliation_backlog_bucket TEXT NOT NULL,
    audited_actor_kind TEXT NOT NULL,
    audited_actor_id TEXT NOT NULL,
    safe_metadata JSONB,
    expires_at TIMESTAMPTZ NOT NULL,
    renewed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT billing_admission_controls_scope_unique UNIQUE (scope_kind, scope_key, use_class),
    CONSTRAINT billing_admission_controls_scope_kind_check CHECK (scope_kind IN ('global', 'account', 'allocator_owner', 'use_class')),
    CONSTRAINT billing_admission_controls_use_class_check CHECK (use_class IN ('chat', 'web_search', 'tool_call', 'batch', 'unknown')),
    CONSTRAINT billing_admission_controls_state_check CHECK (state IN ('open', 'throttle', 'strict', 'fail_closed')),
    CONSTRAINT billing_admission_controls_actor_kind_check CHECK (audited_actor_kind IN ('service', 'worker', 'operator', 'migration')),
    CONSTRAINT billing_admission_controls_expires_check CHECK (expires_at > renewed_at),
    CONSTRAINT billing_admission_controls_safe_metadata_check CHECK (safe_metadata IS NULL OR jsonb_typeof(safe_metadata) = 'object')
);

CREATE INDEX idx_billing_admission_controls_state
ON billing_admission_controls (state, expires_at, admission_control_id);

CREATE TABLE spending_microleases (
    microlease_id UUID PRIMARY KEY,
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    proxy_allocator_owner_id TEXT NOT NULL,
    microlease_generation BIGINT NOT NULL,
    lease_fence TEXT NOT NULL,
    state TEXT NOT NULL,
    issued_cap_usd_atoms BIGINT NOT NULL,
    available_child_cap_usd_atoms BIGINT NOT NULL,
    allocated_child_cap_reported_usd_atoms BIGINT NOT NULL,
    terminal_charged_usd_atoms BIGINT NOT NULL,
    terminal_released_usd_atoms BIGINT NOT NULL,
    write_off_usd_atoms BIGINT NOT NULL,
    pricing_snapshot_id TEXT NOT NULL,
    pricing_snapshot_fingerprint TEXT NOT NULL,
    pricing_policy_version TEXT NOT NULL,
    pricing_decision_at TIMESTAMPTZ NOT NULL,
    pricing_selector_key TEXT NOT NULL,
    pricing_contract_version TEXT NOT NULL,
    fee_policy_version TEXT NOT NULL,
    microlease_policy_version TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    debit_cutoff_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ,
    last_checkpoint_sequence BIGINT NOT NULL,
    last_checkpoint_fingerprint TEXT,
    idempotency_record_id UUID NOT NULL REFERENCES idempotency_records(idempotency_record_id),
    stored_outcome_id UUID NOT NULL REFERENCES operation_outcomes(stored_outcome_id),
    safe_metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT spending_microleases_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT spending_microleases_idempotency_unique UNIQUE (idempotency_record_id),
    CONSTRAINT spending_microleases_stored_outcome_unique UNIQUE (stored_outcome_id),
    CONSTRAINT spending_microleases_owner_generation_unique UNIQUE (proxy_allocator_owner_id, microlease_generation, lease_fence),
    CONSTRAINT spending_microleases_state_check CHECK (state IN ('active', 'cutoff', 'closing', 'closed', 'expired', 'reconcile_required', 'manual_review', 'canceled')),
    CONSTRAINT spending_microleases_generation_check CHECK (microlease_generation > 0),
    CONSTRAINT spending_microleases_issued_cap_positive CHECK (issued_cap_usd_atoms > 0),
    CONSTRAINT spending_microleases_available_nonnegative CHECK (available_child_cap_usd_atoms >= 0),
    CONSTRAINT spending_microleases_allocated_nonnegative CHECK (allocated_child_cap_reported_usd_atoms >= 0),
    CONSTRAINT spending_microleases_terminal_nonnegative CHECK (terminal_charged_usd_atoms >= 0 AND terminal_released_usd_atoms >= 0 AND write_off_usd_atoms >= 0),
    CONSTRAINT spending_microleases_available_cap_check CHECK (available_child_cap_usd_atoms <= issued_cap_usd_atoms),
    CONSTRAINT spending_microleases_allocated_cap_check CHECK (allocated_child_cap_reported_usd_atoms <= issued_cap_usd_atoms),
    CONSTRAINT spending_microleases_terminal_cap_check CHECK (terminal_charged_usd_atoms + terminal_released_usd_atoms + write_off_usd_atoms <= issued_cap_usd_atoms),
    CONSTRAINT spending_microleases_checkpoint_sequence_check CHECK (last_checkpoint_sequence >= 0),
    CONSTRAINT spending_microleases_time_check CHECK (issued_at < debit_cutoff_at AND debit_cutoff_at < expires_at),
    CONSTRAINT spending_microleases_closed_at_check CHECK (closed_at IS NULL OR state IN ('closed', 'canceled', 'reconcile_required', 'manual_review')),
    CONSTRAINT spending_microleases_safe_metadata_check CHECK (safe_metadata IS NULL OR jsonb_typeof(safe_metadata) = 'object')
);

CREATE INDEX idx_spending_microleases_account_state
ON spending_microleases (account_id, state, updated_at DESC, microlease_id DESC);

CREATE INDEX idx_spending_microleases_owner_state
ON spending_microleases (proxy_allocator_owner_id, state, expires_at, microlease_id);

CREATE INDEX idx_spending_microleases_stale
ON spending_microleases (state, expires_at, microlease_id)
WHERE state IN ('active', 'cutoff', 'closing', 'expired');

CREATE TABLE microlease_child_debits (
    microlease_child_debit_id UUID PRIMARY KEY,
    microlease_id UUID NOT NULL REFERENCES spending_microleases(microlease_id),
    debit_authorization_id TEXT NOT NULL,
    usage_operation_id UUID,
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    proxy_allocator_owner_id TEXT NOT NULL,
    microlease_generation BIGINT NOT NULL,
    child_sequence BIGINT NOT NULL,
    child_cap_usd_atoms BIGINT NOT NULL,
    charged_usd_atoms BIGINT NOT NULL,
    released_usd_atoms BIGINT NOT NULL,
    write_off_usd_atoms BIGINT NOT NULL,
    request_basis_fingerprint TEXT NOT NULL,
    terminal_basis_fingerprint TEXT,
    pricing_snapshot_id TEXT NOT NULL,
    pricing_snapshot_fingerprint TEXT NOT NULL,
    terminal_kind TEXT NOT NULL,
    state TEXT NOT NULL,
    qualified_inference_evidence_id UUID REFERENCES qualified_inference_evidence(qualified_inference_evidence_id),
    terminal_event_id TEXT,
    terminal_inbox_id UUID REFERENCES billing_event_inbox(inbox_id),
    ledger_entry_id UUID REFERENCES ledger_entries(ledger_entry_id),
    settlement_effect_id UUID,
    safe_metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    terminal_at TIMESTAMPTZ,
    settled_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT microlease_child_debits_authorization_unique UNIQUE (microlease_id, debit_authorization_id),
    CONSTRAINT microlease_child_debits_sequence_unique UNIQUE (microlease_id, child_sequence),
    CONSTRAINT microlease_child_debits_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT microlease_child_debits_usage_operation_fk FOREIGN KEY (usage_operation_id)
        REFERENCES usage_operations(usage_operation_id),
    CONSTRAINT microlease_child_debits_generation_check CHECK (microlease_generation > 0),
    CONSTRAINT microlease_child_debits_sequence_check CHECK (child_sequence > 0),
    CONSTRAINT microlease_child_debits_child_cap_positive CHECK (child_cap_usd_atoms > 0),
    CONSTRAINT microlease_child_debits_amounts_nonnegative CHECK (charged_usd_atoms >= 0 AND released_usd_atoms >= 0 AND write_off_usd_atoms >= 0),
    CONSTRAINT microlease_child_debits_terminal_totals_check CHECK (charged_usd_atoms + released_usd_atoms + write_off_usd_atoms <= child_cap_usd_atoms),
    CONSTRAINT microlease_child_debits_terminal_kind_check CHECK (terminal_kind IN ('pending', 'finalize', 'write_off', 'abort_release', 'reversal', 'compensation')),
    CONSTRAINT microlease_child_debits_state_check CHECK (state IN ('terminal_pending', 'finalized', 'written_off', 'released', 'reversed', 'reconcile_required', 'conflict')),
    CONSTRAINT microlease_child_debits_terminal_at_check CHECK ((state = 'terminal_pending') = (terminal_at IS NULL)),
    CONSTRAINT microlease_child_debits_safe_metadata_check CHECK (safe_metadata IS NULL OR jsonb_typeof(safe_metadata) = 'object')
);

CREATE INDEX idx_microlease_child_debits_microlease_state
ON microlease_child_debits (microlease_id, state, child_sequence);

CREATE INDEX idx_microlease_child_debits_account_recent
ON microlease_child_debits (account_id, created_at DESC, microlease_child_debit_id DESC);

CREATE INDEX idx_microlease_child_debits_stale
ON microlease_child_debits (state, created_at, microlease_child_debit_id)
WHERE state IN ('terminal_pending', 'reconcile_required', 'conflict');

CREATE UNIQUE INDEX idx_microlease_child_debits_settlement_effect_unique
ON microlease_child_debits (settlement_effect_id)
WHERE settlement_effect_id IS NOT NULL;

CREATE TABLE microlease_checkpoints (
    checkpoint_id UUID PRIMARY KEY,
    microlease_id UUID NOT NULL REFERENCES spending_microleases(microlease_id),
    account_id UUID NOT NULL,
    account_scope_key TEXT NOT NULL,
    proxy_allocator_owner_id TEXT NOT NULL,
    microlease_generation BIGINT NOT NULL,
    checkpoint_sequence BIGINT NOT NULL,
    checkpoint_kind TEXT NOT NULL,
    allocated_child_high_water BIGINT NOT NULL,
    allocated_child_count BIGINT NOT NULL,
    allocated_child_cap_sum_usd_atoms BIGINT NOT NULL,
    terminal_submitted_count BIGINT NOT NULL,
    terminal_published_count BIGINT NOT NULL,
    terminal_accepted_count BIGINT NOT NULL,
    unresolved_child_count BIGINT NOT NULL,
    unresolved_child_cap_sum_usd_atoms BIGINT NOT NULL,
    local_remaining_usd_atoms BIGINT NOT NULL,
    checkpoint_fingerprint TEXT NOT NULL,
    inbox_id UUID REFERENCES billing_event_inbox(inbox_id),
    created_at TIMESTAMPTZ NOT NULL,
    applied_at TIMESTAMPTZ,
    CONSTRAINT microlease_checkpoints_sequence_unique UNIQUE (microlease_id, checkpoint_sequence),
    CONSTRAINT microlease_checkpoints_account_scope_fk FOREIGN KEY (account_id, account_scope_key)
        REFERENCES billing_accounts(account_id, account_scope_key),
    CONSTRAINT microlease_checkpoints_generation_check CHECK (microlease_generation > 0),
    CONSTRAINT microlease_checkpoints_sequence_check CHECK (checkpoint_sequence >= 0),
    CONSTRAINT microlease_checkpoints_kind_check CHECK (checkpoint_kind IN ('progress', 'cutoff', 'close', 'shutdown', 'repair')),
    CONSTRAINT microlease_checkpoints_counts_nonnegative CHECK (
        allocated_child_high_water >= 0
        AND allocated_child_count >= 0
        AND allocated_child_cap_sum_usd_atoms >= 0
        AND terminal_submitted_count >= 0
        AND terminal_published_count >= 0
        AND terminal_accepted_count >= 0
        AND unresolved_child_count >= 0
        AND unresolved_child_cap_sum_usd_atoms >= 0
        AND local_remaining_usd_atoms >= 0
    ),
    CONSTRAINT microlease_checkpoints_progression_check CHECK (
        terminal_accepted_count <= terminal_published_count
        AND terminal_published_count <= terminal_submitted_count
        AND unresolved_child_count <= allocated_child_count
        AND unresolved_child_cap_sum_usd_atoms <= allocated_child_cap_sum_usd_atoms
    ),
    CONSTRAINT microlease_checkpoints_applied_check CHECK (applied_at IS NULL OR applied_at >= created_at)
);

CREATE INDEX idx_microlease_checkpoints_microlease_recent
ON microlease_checkpoints (microlease_id, checkpoint_sequence DESC, checkpoint_id DESC);

CREATE INDEX idx_microlease_checkpoints_kind
ON microlease_checkpoints (checkpoint_kind, created_at DESC, checkpoint_id DESC);

ALTER TABLE reconciliation_cases
ADD COLUMN microlease_id UUID,
ADD COLUMN microlease_child_debit_id UUID,
ADD COLUMN microlease_checkpoint_id UUID,
ADD COLUMN billing_event_inbox_id UUID;

ALTER TABLE reconciliation_cases
ADD CONSTRAINT reconciliation_cases_microlease_fk FOREIGN KEY (microlease_id)
    REFERENCES spending_microleases(microlease_id),
ADD CONSTRAINT reconciliation_cases_microlease_child_fk FOREIGN KEY (microlease_child_debit_id)
    REFERENCES microlease_child_debits(microlease_child_debit_id),
ADD CONSTRAINT reconciliation_cases_microlease_checkpoint_fk FOREIGN KEY (microlease_checkpoint_id)
    REFERENCES microlease_checkpoints(checkpoint_id),
ADD CONSTRAINT reconciliation_cases_billing_event_inbox_fk FOREIGN KEY (billing_event_inbox_id)
    REFERENCES billing_event_inbox(inbox_id);

ALTER TABLE reconciliation_cases
DROP CONSTRAINT reconciliation_cases_reason_check;

ALTER TABLE reconciliation_cases
ADD CONSTRAINT reconciliation_cases_reason_check CHECK (reason IN (
    'stale_reservation',
    'ambiguous_terminal_state',
    'duplicate_payment_evidence',
    'evidence_conflict',
    'missing_inference_evidence',
    'late_payment_evidence',
    'provider_reference_mismatch',
    'legacy_import_mismatch',
    'operator_adjustment_required',
    'stale_microlease',
    'stale_child_debit',
    'microlease_over_debit',
    'microlease_close_gap',
    'microlease_event_conflict'
));

ALTER TABLE reconciliation_cases
DROP CONSTRAINT reconciliation_cases_lineage_check;

ALTER TABLE reconciliation_cases
ADD CONSTRAINT reconciliation_cases_lineage_check CHECK (
    (reason IN ('stale_reservation', 'ambiguous_terminal_state', 'missing_inference_evidence') AND usage_operation_id IS NOT NULL)
    OR (reason IN ('duplicate_payment_evidence', 'evidence_conflict', 'late_payment_evidence') AND payment_evidence_id IS NOT NULL)
    OR (reason = 'provider_reference_mismatch' AND ((topup_operation_id IS NOT NULL AND payment_attempt_id IS NOT NULL) OR settlement_effect_id IS NOT NULL))
    OR (reason = 'legacy_import_mismatch' AND legacy_balance_import_id IS NOT NULL)
    OR (reason = 'operator_adjustment_required' AND (ledger_entry_id IS NOT NULL OR settlement_effect_id IS NOT NULL OR (ledger_entry_id IS NULL AND settlement_effect_id IS NULL)))
    OR (reason = 'stale_microlease' AND microlease_id IS NOT NULL)
    OR (reason IN ('stale_child_debit', 'microlease_over_debit') AND microlease_child_debit_id IS NOT NULL)
    OR (reason = 'microlease_close_gap' AND microlease_checkpoint_id IS NOT NULL)
    OR (reason = 'microlease_event_conflict' AND billing_event_inbox_id IS NOT NULL)
);

CREATE UNIQUE INDEX idx_reconciliation_microlease_open_unique
ON reconciliation_cases (reason, microlease_id)
WHERE reason = 'stale_microlease'
  AND microlease_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_microlease_child_open_unique
ON reconciliation_cases (reason, microlease_child_debit_id)
WHERE reason IN ('stale_child_debit', 'microlease_over_debit')
  AND microlease_child_debit_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_microlease_checkpoint_open_unique
ON reconciliation_cases (reason, microlease_checkpoint_id)
WHERE reason = 'microlease_close_gap'
  AND microlease_checkpoint_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

CREATE UNIQUE INDEX idx_reconciliation_event_inbox_open_unique
ON reconciliation_cases (reason, billing_event_inbox_id)
WHERE reason = 'microlease_event_conflict'
  AND billing_event_inbox_id IS NOT NULL
  AND state NOT IN ('resolved', 'canceled');

ALTER TABLE audit_events
ADD COLUMN microlease_id UUID REFERENCES spending_microleases(microlease_id),
ADD COLUMN microlease_child_debit_id UUID REFERENCES microlease_child_debits(microlease_child_debit_id),
ADD COLUMN billing_event_inbox_id UUID REFERENCES billing_event_inbox(inbox_id);

CREATE INDEX idx_audit_microlease
ON audit_events (microlease_id, created_at DESC, audit_event_id DESC)
WHERE microlease_id IS NOT NULL;

CREATE INDEX idx_audit_microlease_child
ON audit_events (microlease_child_debit_id, created_at DESC, audit_event_id DESC)
WHERE microlease_child_debit_id IS NOT NULL;
