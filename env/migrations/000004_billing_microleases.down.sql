DROP INDEX IF EXISTS idx_audit_microlease_child;
DROP INDEX IF EXISTS idx_audit_microlease;

ALTER TABLE audit_events
DROP COLUMN IF EXISTS billing_event_inbox_id,
DROP COLUMN IF EXISTS microlease_child_debit_id,
DROP COLUMN IF EXISTS microlease_id;

DROP INDEX IF EXISTS idx_reconciliation_event_inbox_open_unique;
DROP INDEX IF EXISTS idx_reconciliation_microlease_checkpoint_open_unique;
DROP INDEX IF EXISTS idx_reconciliation_microlease_child_open_unique;
DROP INDEX IF EXISTS idx_reconciliation_microlease_open_unique;

ALTER TABLE reconciliation_cases
DROP CONSTRAINT IF EXISTS reconciliation_cases_lineage_check;

ALTER TABLE reconciliation_cases
ADD CONSTRAINT reconciliation_cases_lineage_check CHECK (
    (reason IN ('stale_reservation', 'ambiguous_terminal_state', 'missing_inference_evidence') AND usage_operation_id IS NOT NULL)
    OR (reason IN ('duplicate_payment_evidence', 'evidence_conflict', 'late_payment_evidence') AND payment_evidence_id IS NOT NULL)
    OR (reason = 'provider_reference_mismatch' AND ((topup_operation_id IS NOT NULL AND payment_attempt_id IS NOT NULL) OR settlement_effect_id IS NOT NULL))
    OR (reason = 'legacy_import_mismatch' AND legacy_balance_import_id IS NOT NULL)
    OR (reason = 'operator_adjustment_required' AND (ledger_entry_id IS NOT NULL OR settlement_effect_id IS NOT NULL OR (ledger_entry_id IS NULL AND settlement_effect_id IS NULL)))
);

ALTER TABLE reconciliation_cases
DROP CONSTRAINT IF EXISTS reconciliation_cases_reason_check;

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
    'operator_adjustment_required'
));

ALTER TABLE reconciliation_cases
DROP COLUMN IF EXISTS billing_event_inbox_id,
DROP COLUMN IF EXISTS microlease_checkpoint_id,
DROP COLUMN IF EXISTS microlease_child_debit_id,
DROP COLUMN IF EXISTS microlease_id;

DROP TABLE IF EXISTS microlease_checkpoints CASCADE;
DROP TABLE IF EXISTS microlease_child_debits CASCADE;
DROP TABLE IF EXISTS spending_microleases CASCADE;
DROP TABLE IF EXISTS billing_admission_controls CASCADE;
DROP TABLE IF EXISTS billing_outbox CASCADE;
DROP TABLE IF EXISTS billing_event_inbox CASCADE;

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
    'reconciliation_correction'
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
);

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
    'topup_pending_release'
));

ALTER TABLE operation_outcomes
DROP CONSTRAINT operation_outcomes_resource_type_check;

ALTER TABLE operation_outcomes
ADD CONSTRAINT operation_outcomes_resource_type_check CHECK (primary_resource_type IN (
    'usage_operation',
    'topup_operation',
    'payment_evidence',
    'ledger_entry',
    'reconciliation_case'
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
    'reconciliation_correction'
));

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
    'reconciliation_correction'
));
