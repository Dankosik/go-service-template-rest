//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBillingMicroleaseSchemaConstraintsAndReplayState(t *testing.T) {
	pool := setupBillingMoneyCoreRawPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, scope := createBillingMoneyAccount(t, ctx, pool, 600, 10_000)
	microleaseID := createMicroleaseFixture(t, ctx, pool, 610, accountID, scope, 1_000)

	insertLedgerEntry(t, ctx, pool, ledgerEntry{
		ID:            testUUID(620),
		AccountID:     accountID,
		Scope:         scope,
		Effect:        "microlease_reserve",
		Amount:        1_000,
		ReservedDelta: 1_000,
		SettledAfter:  10_000,
		ReservedAfter: 1_000,
		Available:     9_000,
		Reason:        "microlease-issue",
		CreatedBy:     "service",
	})
	_, err := pool.Exec(ctx, `
		UPDATE account_balances
		SET reserved_usd_atoms = 1000,
		    available_usd_atoms = 9000,
		    version = version + 1,
		    last_ledger_entry_id = $2,
		    updated_at = now()
		WHERE account_id = $1
	`, accountID, testUUID(620))
	if err != nil {
		t.Fatalf("update balance after microlease reserve: %v", err)
	}

	_, err = pool.Exec(ctx, ledgerInsertSQL(), testUUID(621), accountID, scope, "USD", "microlease_child_charge", 25, 25, 0, 0, 9_975, 1_000, 8_975, 0, 3, nil, nil, nil, nil, nil, nil, nil, nil, nil, time.Now(), time.Now(), "service", "bad-child-charge-delta", nil)
	expectPgCode(t, err, "23514")

	inboxID := testUUID(630)
	insertBillingEventInbox(t, ctx, pool, inboxID, "billing.microlease.terminal.v1", 0, 42, "event-630", "microlease_child_debit", "debit-630", "event-fingerprint-630")
	_, err = pool.Exec(ctx, `
		INSERT INTO billing_event_inbox (
			inbox_id, topic, partition_id, offset_value, event_id, producer_identity,
			business_identity_type, business_identity_value, event_fingerprint,
			state, received_at, updated_at
		)
		VALUES ($1, 'billing.microlease.terminal.v1', 0, 43, 'event-630', 'proxy-a',
			'microlease_child_debit', 'debit-630', 'changed-fingerprint',
			'received', now(), now())
	`, testUUID(631))
	expectPgCode(t, err, "23505")

	childID := testUUID(640)
	insertMicroleaseChildDebit(t, ctx, pool, childID, microleaseID, "debit-640", accountID, scope, 1, 250)
	_, err = pool.Exec(ctx, `
		INSERT INTO microlease_child_debits (
			microlease_child_debit_id, microlease_id, debit_authorization_id,
			account_id, account_scope_key, proxy_allocator_owner_id,
			microlease_generation, child_sequence, child_cap_usd_atoms,
			charged_usd_atoms, released_usd_atoms, write_off_usd_atoms,
			request_basis_fingerprint, pricing_snapshot_id, pricing_snapshot_fingerprint,
			terminal_kind, state, created_at, updated_at
		)
		VALUES ($1, $2, 'debit-640', $3, $4, 'proxy-a', 1, 2, 250, 0, 0, 0,
			'changed-request-fingerprint', 'pricing-snapshot-610', 'pricing-fingerprint-610',
			'pending', 'terminal_pending', now(), now())
	`, testUUID(641), microleaseID, accountID, scope)
	expectPgCode(t, err, "23505")

	_, err = pool.Exec(ctx, `
		INSERT INTO microlease_child_debits (
			microlease_child_debit_id, microlease_id, debit_authorization_id,
			account_id, account_scope_key, proxy_allocator_owner_id,
			microlease_generation, child_sequence, child_cap_usd_atoms,
			charged_usd_atoms, released_usd_atoms, write_off_usd_atoms,
			request_basis_fingerprint, terminal_basis_fingerprint,
			pricing_snapshot_id, pricing_snapshot_fingerprint,
			terminal_kind, state, terminal_inbox_id, created_at, terminal_at, updated_at
		)
		VALUES ($1, $2, 'debit-over-cap', $3, $4, 'proxy-a', 1, 3, 100,
			90, 20, 0, 'request-fingerprint-over-cap', 'terminal-fingerprint-over-cap',
			'pricing-snapshot-610', 'pricing-fingerprint-610',
			'finalize', 'finalized', $5, now(), now(), now())
	`, testUUID(642), microleaseID, accountID, scope, inboxID)
	expectPgCode(t, err, "23514")

	checkpointID := testUUID(650)
	insertMicroleaseCheckpoint(t, ctx, pool, checkpointID, microleaseID, accountID, scope, inboxID, 1, "progress", 1, 250, 1, 1, 1, 0, 0, 750)
	_, err = pool.Exec(ctx, `
		INSERT INTO microlease_checkpoints (
			checkpoint_id, microlease_id, account_id, account_scope_key,
			proxy_allocator_owner_id, microlease_generation, checkpoint_sequence,
			checkpoint_kind, allocated_child_high_water, allocated_child_count,
			allocated_child_cap_sum_usd_atoms, terminal_submitted_count,
			terminal_published_count, terminal_accepted_count, unresolved_child_count,
			unresolved_child_cap_sum_usd_atoms, local_remaining_usd_atoms,
			checkpoint_fingerprint, created_at
		)
		VALUES ($1, $2, $3, $4, 'proxy-a', 1, 2, 'progress', 1, 1, 250, 1, 0, 1, 0, 0, 750, 'bad-checkpoint-fingerprint', now())
	`, testUUID(651), microleaseID, accountID, scope)
	expectPgCode(t, err, "23514")

	insertMicroleaseReconciliationCase(t, ctx, pool, testUUID(660), accountID, "microlease_close_gap", checkpointID)
	_, err = pool.Exec(ctx, `
		INSERT INTO reconciliation_cases (
			reconciliation_case_id, account_id, reason, state, severity,
			attempt_count, next_attempt_at, created_at, updated_at
		)
		VALUES ($1, $2, 'stale_microlease', 'open', 'medium', 0, now(), now(), now())
	`, testUUID(661), accountID)
	expectPgCode(t, err, "23514")

	_, err = pool.Exec(ctx, `
		INSERT INTO billing_admission_controls (
			admission_control_id, scope_kind, scope_key, use_class, state,
			reason_code, terminal_lag_bucket, stale_age_bucket,
			reconciliation_backlog_bucket, audited_actor_kind, audited_actor_id,
			expires_at, renewed_at, created_at, updated_at
		)
		VALUES ($1, 'account', $2, 'chat', 'fail_closed', 'rollout_default_closed',
			'unknown', 'unknown', 'unknown', 'service', 'billing-service',
			now() + interval '1 minute', now(), now(), now())
	`, testUUID(670), scope)
	if err != nil {
		t.Fatalf("insert default-closed admission control: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO billing_admission_controls (
			admission_control_id, scope_kind, scope_key, use_class, state,
			reason_code, terminal_lag_bucket, stale_age_bucket,
			reconciliation_backlog_bucket, audited_actor_kind, audited_actor_id,
			expires_at, renewed_at, created_at, updated_at
		)
		VALUES ($1, 'account', $2, 'chat', 'open', 'duplicate',
			'ok', 'ok', 'ok', 'service', 'billing-service',
			now() + interval '1 minute', now(), now(), now())
	`, testUUID(671), scope)
	expectPgCode(t, err, "23505")

	expectMicroleaseLockTimeout(t, ctx, pool, microleaseID)
}

func createMicroleaseFixture(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed int, accountID, scope string, capAtoms int64) string {
	tb.Helper()

	microleaseID := testUUID(seed)
	createIdempotency(tb, ctx, pool, seed+1, accountID, "microlease_issue", "microlease-idem", "issue-fingerprint", "started", nil)
	createOutcome(tb, ctx, pool, seed+2, seed+1, accountID, "microlease_issue", "success", "spending_microlease", microleaseID, nil)
	if _, err := pool.Exec(ctx, `
		INSERT INTO spending_microleases (
			microlease_id, account_id, account_scope_key, proxy_allocator_owner_id,
			microlease_generation, lease_fence, state, issued_cap_usd_atoms,
			available_child_cap_usd_atoms, allocated_child_cap_reported_usd_atoms,
			terminal_charged_usd_atoms, terminal_released_usd_atoms,
			write_off_usd_atoms, pricing_snapshot_id, pricing_snapshot_fingerprint,
			pricing_policy_version, pricing_decision_at, pricing_selector_key,
			pricing_contract_version, fee_policy_version, microlease_policy_version,
			issued_at, debit_cutoff_at, expires_at, last_checkpoint_sequence,
			idempotency_record_id, stored_outcome_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, 'proxy-a', 1, 'fence-a', 'active', $4, $4, 0, 0, 0,
			0, $5, $6, 'pricing-policy-v1', now(), 'model:gpt-4.1:chat',
			'pricing-contract-v1', 'fee-v1', 'microlease-v1',
			now(), now() + interval '25 seconds', now() + interval '30 seconds',
			0, $7, $8, now(), now())
	`, microleaseID, accountID, scope, capAtoms, "pricing-snapshot-610", "pricing-fingerprint-610", testUUID(seed+1), testUUID(seed+2)); err != nil {
		tb.Fatalf("insert microlease fixture: %v", err)
	}
	return microleaseID
}

func insertBillingEventInbox(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, inboxID, topic string, partition int, offset int64, eventID, identityType, identityValue, fingerprint string) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_event_inbox (
			inbox_id, topic, partition_id, offset_value, event_id, producer_identity,
			business_identity_type, business_identity_value, event_fingerprint,
			state, received_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'proxy-a', $6, $7, $8, 'received', now(), now())
	`, inboxID, topic, partition, offset, eventID, identityType, identityValue, fingerprint); err != nil {
		tb.Fatalf("insert inbox fixture: %v", err)
	}
}

func insertMicroleaseChildDebit(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, childID, microleaseID, authorizationID, accountID, scope string, sequence int64, capAtoms int64) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO microlease_child_debits (
			microlease_child_debit_id, microlease_id, debit_authorization_id,
			account_id, account_scope_key, proxy_allocator_owner_id,
			microlease_generation, child_sequence, child_cap_usd_atoms,
			charged_usd_atoms, released_usd_atoms, write_off_usd_atoms,
			request_basis_fingerprint, pricing_snapshot_id, pricing_snapshot_fingerprint,
			terminal_kind, state, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'proxy-a', 1, $6, $7, 0, 0, 0,
			'request-fingerprint-' || $3, 'pricing-snapshot-610',
			'pricing-fingerprint-610', 'pending', 'terminal_pending', now(), now())
	`, childID, microleaseID, authorizationID, accountID, scope, sequence, capAtoms); err != nil {
		tb.Fatalf("insert child debit fixture: %v", err)
	}
}

func insertMicroleaseCheckpoint(
	tb testing.TB,
	ctx context.Context,
	pool *pgxpool.Pool,
	checkpointID string,
	microleaseID string,
	accountID string,
	scope string,
	inboxID string,
	sequence int64,
	kind string,
	allocatedCount int64,
	allocatedCap int64,
	submitted int64,
	published int64,
	accepted int64,
	unresolvedCount int64,
	unresolvedCap int64,
	localRemaining int64,
) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO microlease_checkpoints (
			checkpoint_id, microlease_id, account_id, account_scope_key,
			proxy_allocator_owner_id, microlease_generation, checkpoint_sequence,
			checkpoint_kind, allocated_child_high_water, allocated_child_count,
			allocated_child_cap_sum_usd_atoms, terminal_submitted_count,
			terminal_published_count, terminal_accepted_count, unresolved_child_count,
			unresolved_child_cap_sum_usd_atoms, local_remaining_usd_atoms,
			checkpoint_fingerprint, inbox_id, created_at, applied_at
		)
		VALUES ($1, $2, $3, $4, 'proxy-a', 1, $5, $6, $7, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, now(), now())
	`, checkpointID, microleaseID, accountID, scope, sequence, kind, allocatedCount, allocatedCap, submitted, published, accepted, unresolvedCount, unresolvedCap, localRemaining, "checkpoint-fingerprint-"+checkpointID, inboxID); err != nil {
		tb.Fatalf("insert checkpoint fixture: %v", err)
	}
}

func insertMicroleaseReconciliationCase(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, caseID, accountID, reason, checkpointID string) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO reconciliation_cases (
			reconciliation_case_id, account_id, reason, state, severity,
			microlease_checkpoint_id, attempt_count, next_attempt_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, 'open', 'medium', $4, 0, now(), now(), now())
	`, caseID, accountID, reason, checkpointID); err != nil {
		tb.Fatalf("insert microlease reconciliation case: %v", err)
	}
}

func expectMicroleaseLockTimeout(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, microleaseID string) {
	tb.Helper()

	locker, err := pool.Begin(ctx)
	if err != nil {
		tb.Fatalf("begin microlease lock holder: %v", err)
	}
	defer func() {
		_ = locker.Rollback(ctx)
	}()
	if _, err := locker.Exec(ctx, `SELECT microlease_id FROM spending_microleases WHERE microlease_id = $1 FOR UPDATE`, microleaseID); err != nil {
		tb.Fatalf("lock microlease for timeout test: %v", err)
	}

	waiter, err := pool.Begin(ctx)
	if err != nil {
		tb.Fatalf("begin microlease lock waiter: %v", err)
	}
	defer func() {
		_ = waiter.Rollback(ctx)
	}()
	if _, err := waiter.Exec(ctx, `SET LOCAL lock_timeout = '50ms'`); err != nil {
		tb.Fatalf("set microlease lock timeout: %v", err)
	}
	_, err = waiter.Exec(ctx, `SELECT microlease_id FROM spending_microleases WHERE microlease_id = $1 FOR UPDATE`, microleaseID)
	expectPgCode(tb, err, "55P03")
}
