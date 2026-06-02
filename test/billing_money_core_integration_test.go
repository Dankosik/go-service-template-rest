//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
)

func TestBillingMoneyCoreSchemaConstraintsAndLedgerDeltas(t *testing.T) {
	pool := setupBillingMoneyCoreRawPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, scope := createBillingMoneyAccount(t, ctx, pool, 100, 1_000)

	_, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts (account_id, account_scope_key, account_type, subject_authority, subject_id, state, version, created_at, updated_at)
		VALUES ($1, $2, 'user', 'identity-service', $3, 'active', 1, now(), now())
	`, testUUID(101), scope, "user-100")
	expectPgCode(t, err, "23505")

	_, err = pool.Exec(ctx, `
		INSERT INTO account_balances (account_id, account_scope_key, currency, settled_usd_atoms, reserved_usd_atoms, available_usd_atoms, pending_usd_atoms, version, updated_at)
		VALUES ($1, $2, 'USD', 100, 60, 50, 0, 1, now())
	`, accountID, scope)
	expectPgCode(t, err, "23514")

	ledgerID := testUUID(110)
	insertLedgerEntry(t, ctx, pool, ledgerEntry{
		ID:           ledgerID,
		AccountID:    accountID,
		Scope:        scope,
		Effect:       "topup_credit",
		Amount:       1_000,
		SettledDelta: 1_000,
		SettledAfter: 1_000,
		Available:    1_000,
		Reason:       "topup-settlement",
		CreatedBy:    "service",
	})
	_, err = pool.Exec(ctx, `
		UPDATE ledger_entries
		SET amount_usd_atoms = amount_usd_atoms + 1
		WHERE ledger_entry_id = $1
	`, ledgerID)
	expectPgCode(t, err, "23514")

	insertUsageOperation(t, ctx, pool, 120, accountID, scope, "reserved")
	insertUsageHold(t, ctx, pool, 121, 120, accountID, scope, "active", 200)
	insertLedgerEntry(t, ctx, pool, ledgerEntry{
		ID:            testUUID(122),
		AccountID:     accountID,
		Scope:         scope,
		Effect:        "usage_hold",
		Amount:        200,
		ReservedDelta: 200,
		SettledAfter:  1_000,
		ReservedAfter: 200,
		Available:     800,
		Reason:        "usage-reserve",
		CreatedBy:     "service",
	})
	_, err = pool.Exec(ctx, `
		UPDATE account_balances
		SET settled_usd_atoms = 1000,
		    reserved_usd_atoms = 200,
		    available_usd_atoms = 800,
		    pending_usd_atoms = 0,
		    version = version + 1,
		    last_ledger_entry_id = $2,
		    updated_at = now()
		WHERE account_id = $1
	`, accountID, testUUID(122))
	if err != nil {
		t.Fatalf("update balance after hold: %v", err)
	}

	_, err = pool.Exec(ctx, ledgerInsertSQL(), testUUID(123), accountID, scope, "USD", "usage_charge", -50, -50, -50, 1, 950, 150, 800, 1, 3, nil, nil, testUUID(120), nil, nil, nil, nil, nil, nil, time.Now(), time.Now(), "service", "bad-pending-delta", nil)
	expectPgCode(t, err, "23514")

	_, err = pool.Exec(ctx, `
		INSERT INTO idempotency_records (
			idempotency_record_id, account_id, operation_kind, idempotency_key,
			request_fingerprint, state, retention_class, first_seen_at, last_seen_at
		)
		VALUES ($1, $2, 'reserve', 'idem-constraints', 'fp', 'started', 'hot_replay', now(), now())
	`, testUUID(130), accountID)
	if err != nil {
		t.Fatalf("insert idempotency fixture: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO idempotency_records (
			idempotency_record_id, account_id, operation_kind, idempotency_key,
			request_fingerprint, state, retention_class, first_seen_at, last_seen_at
		)
		VALUES ($1, $2, 'reserve', 'idem-bad-retention', 'fp', 'started', 'raw_payload', now(), now())
	`, testUUID(131), accountID)
	expectPgCode(t, err, "23514")

	_, err = pool.Exec(ctx, `
		INSERT INTO operation_outcomes (
			stored_outcome_id, idempotency_record_id, account_id, operation_kind,
			outcome_status, primary_resource_type, primary_resource_id, created_at
		)
		VALUES ($1, $2, $3, 'reserve', 'success', 'raw_response', 'x', now())
	`, testUUID(132), testUUID(130), accountID)
	expectPgCode(t, err, "23514")

	topupID, attemptID := createTopupWithAttempt(t, ctx, pool, 140, accountID, scope)
	_, err = pool.Exec(ctx, `
		INSERT INTO payment_attempts (
			payment_attempt_row_id, topup_operation_id, payment_attempt_id,
			attempt_generation, state, presentation_version, created_at, updated_at
		)
		VALUES ($1, $2, $3, 2, 'raw_provider_state', 1, now(), now())
	`, testUUID(141), topupID, "attempt-bad-state")
	expectPgCode(t, err, "23514")

	insertPaymentEvidence(t, ctx, pool, paymentEvidenceFixture{
		ID:          "evidence-constraints",
		TopupID:     topupID,
		AttemptID:   attemptID,
		AccountID:   accountID,
		Scope:       scope,
		Fingerprint: "evidence-fingerprint-constraints",
		Amount:      1_000,
	})
	_, err = pool.Exec(ctx, `
		INSERT INTO payment_evidence (
			payment_evidence_id, topup_operation_id, payment_attempt_id, account_id,
			account_scope_key, state, evidence_payload_fingerprint, evidence_kind,
			schema_version, finality_class, rail_family, settlement_amount_usd_atoms,
			received_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, 'accepted', $6, 'settlement', 'v1', 'unknown', 'card', 1000, now(), now())
	`, "evidence-bad-finality", topupID, attemptID, accountID, scope, "evidence-fingerprint-bad-finality")
	expectPgCode(t, err, "23514")

	insertLedgerEntry(t, ctx, pool, ledgerEntry{
		ID:               testUUID(150),
		AccountID:        accountID,
		Scope:            scope,
		Effect:           "reconciliation_correction",
		Amount:           10,
		SettledDelta:     10,
		SettledAfter:     1_010,
		ReservedAfter:    200,
		Available:        810,
		SettlementEffect: ptrString(testUUID(151)),
		Reason:           "correction",
		CreatedBy:        "operator",
	})
	_, err = pool.Exec(ctx, ledgerInsertSQL(), testUUID(152), accountID, scope, "USD", "reconciliation_correction", 10, 10, 0, 0, 1020, 200, 820, 0, 4, testUUID(151), nil, nil, nil, nil, nil, nil, nil, nil, time.Now(), time.Now(), "operator", "duplicate-settlement-effect", nil)
	expectPgCode(t, err, "23505")

	var settled, reserved int64
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(settled_delta_usd_atoms), 0), COALESCE(SUM(reserved_delta_usd_atoms), 0)
		FROM ledger_entries
		WHERE account_id = $1
	`, accountID).Scan(&settled, &reserved); err != nil {
		t.Fatalf("recompute ledger deltas: %v", err)
	}
	if settled != 1_010 || reserved != 200 {
		t.Fatalf("ledger delta recompute = settled %d reserved %d, want settled 1010 reserved 200", settled, reserved)
	}
	var activeReserved int64
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(reserved_usd_atoms - released_usd_atoms - charged_usd_atoms), 0)
		FROM usage_holds
		WHERE account_id = $1 AND state = 'active'
	`, accountID).Scan(&activeReserved); err != nil {
		t.Fatalf("recompute active holds: %v", err)
	}
	if activeReserved != 200 {
		t.Fatalf("active hold recompute = %d, want 200", activeReserved)
	}
}

func TestBillingMoneyCoreIdempotencyAndStoredOutcomes(t *testing.T) {
	pool := setupBillingMoneyCoreRawPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, scope := createBillingMoneyAccount(t, ctx, pool, 200, 10_000)
	beforeLedger := countRows(t, ctx, pool, "ledger_entries")

	createIdempotency(t, ctx, pool, 201, accountID, "reserve", "idem-reserve", "reserve-fingerprint", "started", nil)
	createOutcome(t, ctx, pool, 202, 201, accountID, "reserve", "success", "usage_operation", testUUID(203), nil)
	_, err := pool.Exec(ctx, `
		UPDATE idempotency_records
		SET state = 'committed', stored_outcome_id = $2, committed_at = now(), last_seen_at = now()
		WHERE idempotency_record_id = $1
	`, testUUID(201), testUUID(202))
	if err != nil {
		t.Fatalf("commit idempotency fixture: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO idempotency_records (
			idempotency_record_id, account_id, operation_kind, idempotency_key,
			request_fingerprint, state, retention_class, first_seen_at, last_seen_at
		)
		VALUES ($1, $2, 'reserve', 'idem-reserve', 'changed-fingerprint', 'started', 'hot_replay', now(), now())
	`, testUUID(204), accountID)
	expectPgCode(t, err, "23505")
	if after := countRows(t, ctx, pool, "ledger_entries"); after != beforeLedger {
		t.Fatalf("changed-fingerprint idempotency conflict mutated ledger rows: before=%d after=%d", beforeLedger, after)
	}

	var replayFingerprint string
	var replayOutcome string
	if err := pool.QueryRow(ctx, `
		SELECT request_fingerprint, stored_outcome_id::text
		FROM idempotency_records
		WHERE account_id = $1 AND operation_kind = 'reserve' AND idempotency_key = 'idem-reserve'
	`, accountID).Scan(&replayFingerprint, &replayOutcome); err != nil {
		t.Fatalf("read replay idempotency: %v", err)
	}
	if replayFingerprint != "reserve-fingerprint" || replayOutcome != testUUID(202) {
		t.Fatalf("replay record = (%q,%q), want original fingerprint and outcome", replayFingerprint, replayOutcome)
	}

	createIdempotency(t, ctx, pool, 210, accountID, "finalize", "idem-finalize-failure", "finalize-fingerprint", "started", nil)
	createOutcome(t, ctx, pool, 211, 210, accountID, "finalize", "stored_failure", "usage_operation", testUUID(212), nil)
	_, err = pool.Exec(ctx, `
		UPDATE idempotency_records
		SET state = 'failed_stored', stored_outcome_id = $2, committed_at = now(), last_seen_at = now()
		WHERE idempotency_record_id = $1
	`, testUUID(210), testUUID(211))
	if err != nil {
		t.Fatalf("store failure outcome: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO operation_outcomes (
			stored_outcome_id, idempotency_record_id, account_id, operation_kind,
			outcome_status, primary_resource_type, primary_resource_id, created_at
		)
		VALUES ($1, $2, $3, 'finalize', 'stored_failure', 'usage_operation', $4, now())
	`, testUUID(213), testUUID(210), accountID, testUUID(212))
	expectPgCode(t, err, "23505")

	for i, kind := range []string{"write_off", "reversal", "compensation", "topup_evidence", "migration_import", "reconciliation_correction"} {
		seed := 220 + i
		createIdempotency(t, ctx, pool, seed, accountID, kind, "idem-"+kind, kind+"-fingerprint", "conflict", ptrString("changed_fingerprint"))
	}

	createLegacyImportWithLedger(t, ctx, pool, 240, accountID, scope)
}

func TestBillingMoneyCoreReconciliationAndLegacyImport(t *testing.T) {
	pool := setupBillingMoneyCoreRawPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, scope := createBillingMoneyAccount(t, ctx, pool, 300, 5_000)
	insertUsageOperation(t, ctx, pool, 301, accountID, scope, "reserved")
	insertUsageHold(t, ctx, pool, 302, 301, accountID, scope, "active", 500)
	insertReconciliationCase(t, ctx, pool, reconciliationCase{
		ID:        testUUID(303),
		AccountID: accountID,
		Reason:    "stale_reservation",
		State:     "open",
		UsageID:   ptrString(testUUID(301)),
	})
	_, err := pool.Exec(ctx, reconciliationInsertSQL(), testUUID(304), accountID, "stale_reservation", "open", "medium", testUUID(301), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 0, time.Now(), nil, time.Now(), nil)
	expectPgCode(t, err, "23505")

	topupID, attemptID := createTopupWithAttempt(t, ctx, pool, 310, accountID, scope)
	insertPaymentEvidence(t, ctx, pool, paymentEvidenceFixture{
		ID:          "evidence-reconciliation",
		TopupID:     topupID,
		AttemptID:   attemptID,
		AccountID:   accountID,
		Scope:       scope,
		Fingerprint: "evidence-fingerprint-reconciliation",
		Amount:      1_000,
	})
	_, err = pool.Exec(ctx, `
		INSERT INTO payment_evidence (
			payment_evidence_id, topup_operation_id, payment_attempt_id, account_id,
			account_scope_key, state, evidence_payload_fingerprint, evidence_kind,
			schema_version, finality_class, rail_family, settlement_amount_usd_atoms,
			received_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, 'accepted', $6, 'settlement', 'v1', 'final', 'card', 1000, now(), now())
	`, "evidence-duplicate-fingerprint", topupID, attemptID, accountID, scope, "evidence-fingerprint-reconciliation")
	expectPgCode(t, err, "23505")
	_, err = pool.Exec(ctx, `
		INSERT INTO payment_evidence (
			payment_evidence_id, topup_operation_id, payment_attempt_id, account_id,
			account_scope_key, state, evidence_payload_fingerprint, evidence_kind,
			schema_version, finality_class, rail_family, settlement_amount_usd_atoms,
			received_at, created_at
		)
		VALUES ('evidence-reconciliation', $1, $2, $3, $4, 'accepted', 'changed-fingerprint', 'settlement', 'v1', 'final', 'card', 1000, now(), now())
	`, topupID, attemptID, accountID, scope)
	expectPgCode(t, err, "23505")

	_, err = pool.Exec(ctx, reconciliationInsertSQL(), testUUID(312), accountID, "missing_inference_evidence", "open", "medium", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 0, time.Now(), nil, time.Now(), nil)
	expectPgCode(t, err, "23514")

	insertReconciliationCase(t, ctx, pool, reconciliationCase{
		ID:        testUUID(313),
		AccountID: accountID,
		Reason:    "provider_reference_mismatch",
		State:     "open",
		TopupID:   ptrString(topupID),
		AttemptID: ptrString(attemptID),
	})
	_, err = pool.Exec(ctx, reconciliationInsertSQL(), testUUID(314), accountID, "provider_reference_mismatch", "open", "medium", nil, topupID, attemptID, nil, nil, nil, nil, nil, nil, nil, nil, nil, 0, time.Now(), nil, time.Now(), nil)
	expectPgCode(t, err, "23505")

	insertReconciliationCase(t, ctx, pool, reconciliationCase{
		ID:         testUUID(320),
		AccountID:  accountID,
		Reason:     "duplicate_payment_evidence",
		State:      "open",
		EvidenceID: ptrString("evidence-reconciliation"),
	})
	claimA := lockOneReconciliationCase(t, ctx, pool)
	claimB := lockOneReconciliationCase(t, ctx, pool)
	if claimA == claimB {
		t.Fatalf("FOR UPDATE SKIP LOCKED returned same case twice: %s", claimA)
	}

	legacyID := createLegacyImportWithLedger(t, ctx, pool, 330, accountID, scope)
	insertReconciliationCase(t, ctx, pool, reconciliationCase{
		ID:             testUUID(340),
		AccountID:      accountID,
		Reason:         "legacy_import_mismatch",
		State:          "open",
		LegacyImportID: ptrString(legacyID),
	})
	_, err = pool.Exec(ctx, reconciliationInsertSQL(), testUUID(341), accountID, "legacy_import_mismatch", "open", "medium", nil, nil, nil, nil, nil, nil, nil, legacyID, nil, nil, nil, nil, 0, time.Now(), nil, time.Now(), nil)
	expectPgCode(t, err, "23505")

	var settled int64
	if err := pool.QueryRow(ctx, `
		SELECT settled_usd_atoms
		FROM account_balances
		WHERE account_id = $1
	`, accountID).Scan(&settled); err != nil {
		t.Fatalf("read live balance after legacy import: %v", err)
	}
	if settled != 5_000 {
		t.Fatalf("live balance consumed legacy import evidence: settled=%d, want 5000", settled)
	}
}

func TestBillingMoneyCoreConcurrencyAndLockClassification(t *testing.T) {
	pool := setupBillingMoneyCoreRawPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	accountID, scope := createBillingMoneyAccount(t, ctx, pool, 400, 1_000)
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := reserveIfAvailable(ctx, pool, accountID, 700)
			if err != nil {
				errs <- err
				return
			}
			results <- ok
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("reserve race error: %v", err)
	}
	successes := 0
	for ok := range results {
		if ok {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("reserve race successes = %d, want 1", successes)
	}
	var available int64
	if err := pool.QueryRow(ctx, `
		SELECT available_usd_atoms
		FROM account_balances
		WHERE account_id = $1
	`, accountID).Scan(&available); err != nil {
		t.Fatalf("read available after reserve race: %v", err)
	}
	if available < 0 {
		t.Fatalf("available balance went negative: %d", available)
	}

	insertUsageOperation(t, ctx, pool, 410, accountID, scope, "reserved")
	insertUsageHold(t, ctx, pool, 411, 410, accountID, scope, "active", 100)
	createIdempotency(t, ctx, pool, 412, accountID, "finalize", "idem-finalize-race", "fp-finalize", "started", nil)
	createOutcome(t, ctx, pool, 413, 412, accountID, "finalize", "success", "usage_operation", testUUID(410), nil)
	createIdempotency(t, ctx, pool, 414, accountID, "write_off", "idem-writeoff-race", "fp-writeoff", "started", nil)
	createOutcome(t, ctx, pool, 415, 414, accountID, "write_off", "success", "usage_operation", testUUID(410), nil)

	terminalErrs := make(chan error, 2)
	for _, terminal := range []struct {
		seed int
		kind string
		idem int
		out  int
	}{
		{seed: 416, kind: "finalize", idem: 412, out: 413},
		{seed: 417, kind: "write_off", idem: 414, out: 415},
	} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := pool.Exec(ctx, `
				INSERT INTO usage_terminal_outcomes (
					usage_terminal_outcome_id, usage_operation_id, terminal_kind,
					idempotency_record_id, stored_outcome_id, charged_usd_atoms,
					released_usd_atoms, write_off_usd_atoms, created_at
				)
				VALUES ($1, $2, $3, $4, $5, 0, 100, 0, now())
			`, testUUID(terminal.seed), testUUID(410), terminal.kind, testUUID(terminal.idem), testUUID(terminal.out))
			terminalErrs <- err
		}()
	}
	wg.Wait()
	close(terminalErrs)
	terminalSuccesses := 0
	terminalConflicts := 0
	for err := range terminalErrs {
		if err == nil {
			terminalSuccesses++
			continue
		}
		if isPgCode(err, "23505") {
			terminalConflicts++
			continue
		}
		t.Fatalf("terminal race unexpected error: %v", err)
	}
	if terminalSuccesses != 1 || terminalConflicts != 1 {
		t.Fatalf("terminal race successes=%d conflicts=%d, want 1/1", terminalSuccesses, terminalConflicts)
	}

	topupID, attemptID := createTopupWithAttempt(t, ctx, pool, 420, accountID, scope)
	evidenceErrs := make(chan error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := pool.Exec(ctx, `
				INSERT INTO payment_evidence (
					payment_evidence_id, topup_operation_id, payment_attempt_id, account_id,
					account_scope_key, state, evidence_payload_fingerprint, evidence_kind,
					schema_version, finality_class, rail_family, settlement_amount_usd_atoms,
					received_at, created_at
				)
				VALUES ($1, $2, $3, $4, $5, 'accepted', 'duplicate-topup-fingerprint', 'settlement', 'v1', 'final', 'card', 1000, now(), now())
			`, fmt.Sprintf("duplicate-topup-evidence-%d", i), topupID, attemptID, accountID, scope)
			evidenceErrs <- err
		}(i)
	}
	wg.Wait()
	close(evidenceErrs)
	evidenceSuccesses := 0
	evidenceConflicts := 0
	for err := range evidenceErrs {
		if err == nil {
			evidenceSuccesses++
			continue
		}
		if isPgCode(err, "23505") {
			evidenceConflicts++
			continue
		}
		t.Fatalf("duplicate evidence unexpected error: %v", err)
	}
	if evidenceSuccesses != 1 || evidenceConflicts != 1 {
		t.Fatalf("duplicate evidence successes=%d conflicts=%d, want 1/1", evidenceSuccesses, evidenceConflicts)
	}

	expectLockTimeout(t, ctx, pool, accountID)
	expectDeadlock(t, ctx, pool)
}

func BenchmarkBillingMoneyCoreReserve(b *testing.B) {
	pool, accountID := setupBillingMoneyCoreBenchmark(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := pool.Begin(ctx)
		if err != nil {
			b.Fatalf("begin reserve benchmark tx: %v", err)
		}
		if _, err := tx.Exec(ctx, `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`, accountID); err != nil {
			_ = tx.Rollback(ctx)
			b.Fatalf("lock account balance: %v", err)
		}
		if err := tx.Rollback(ctx); err != nil {
			b.Fatalf("rollback reserve benchmark tx: %v", err)
		}
	}
}

func BenchmarkBillingMoneyCoreFinalize(b *testing.B) {
	pool, _ := setupBillingMoneyCoreBenchmark(b)
	ctx := context.Background()
	usageID := testUUID(9003)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pool.Exec(ctx, `SELECT usage_operation_id FROM usage_operations WHERE usage_operation_id = $1`, usageID); err != nil {
			b.Fatalf("lookup usage operation: %v", err)
		}
		if _, err := pool.Exec(ctx, `SELECT hold_id FROM usage_holds WHERE usage_operation_id = $1`, usageID); err != nil {
			b.Fatalf("lookup usage hold: %v", err)
		}
	}
}

func BenchmarkBillingMoneyCoreWriteOff(b *testing.B) {
	pool, _ := setupBillingMoneyCoreBenchmark(b)
	ctx := context.Background()
	usageID := testUUID(9003)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := pool.Begin(ctx)
		if err != nil {
			b.Fatalf("begin writeoff benchmark tx: %v", err)
		}
		if _, err := tx.Exec(ctx, `SELECT hold_id FROM usage_holds WHERE usage_operation_id = $1 FOR UPDATE`, usageID); err != nil {
			_ = tx.Rollback(ctx)
			b.Fatalf("lock usage hold: %v", err)
		}
		if err := tx.Rollback(ctx); err != nil {
			b.Fatalf("rollback writeoff benchmark tx: %v", err)
		}
	}
}

func BenchmarkBillingMoneyCoreTopupEvidence(b *testing.B) {
	pool, _ := setupBillingMoneyCoreBenchmark(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pool.Exec(ctx, `SELECT payment_evidence_id FROM payment_evidence WHERE payment_evidence_id = $1`, "benchmark-evidence"); err != nil {
			b.Fatalf("lookup payment evidence by id: %v", err)
		}
		if _, err := pool.Exec(ctx, `SELECT payment_evidence_id FROM payment_evidence WHERE evidence_payload_fingerprint = $1`, "benchmark-evidence-fingerprint"); err != nil {
			b.Fatalf("lookup payment evidence by fingerprint: %v", err)
		}
	}
}

func BenchmarkBillingMoneyCoreReconciliationClaim(b *testing.B) {
	pool, _ := setupBillingMoneyCoreBenchmark(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := pool.Begin(ctx)
		if err != nil {
			b.Fatalf("begin reconciliation benchmark tx: %v", err)
		}
		if _, err := tx.Exec(ctx, `
			SELECT reconciliation_case_id
			FROM reconciliation_cases
			WHERE state IN ('open', 'waiting_evidence')
			  AND next_attempt_at <= now()
			ORDER BY next_attempt_at, reconciliation_case_id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		`); err != nil {
			_ = tx.Rollback(ctx)
			b.Fatalf("claim reconciliation case: %v", err)
		}
		if err := tx.Rollback(ctx); err != nil {
			b.Fatalf("rollback reconciliation benchmark tx: %v", err)
		}
	}
}

func BenchmarkBillingMoneyCoreSupportReadback(b *testing.B) {
	pool, accountID := setupBillingMoneyCoreBenchmark(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pool.Exec(ctx, `
			SELECT ledger_entry_id
			FROM ledger_entries
			WHERE account_id = $1
			ORDER BY created_at DESC, ledger_entry_id DESC
			LIMIT 20
		`, accountID); err != nil {
			b.Fatalf("support ledger readback: %v", err)
		}
	}
}

func setupBillingMoneyCoreRawPool(tb testing.TB) *pgxpool.Pool {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := runPostgresContainer(ctx)
	if err != nil {
		if isDockerUnavailable(err) {
			if requireDockerForIntegration() {
				tb.Fatalf("docker is required for integration tests: %v", err)
			}
			tb.Skipf("docker is unavailable: %v", err)
		}
		tb.Fatalf("start postgres container: %v", err)
	}
	tb.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(container); termErr != nil {
			tb.Errorf("terminate postgres container: %v", termErr)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		tb.Fatalf("build postgres dsn: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		tb.Fatalf("create postgres pool: %v", err)
	}
	tb.Cleanup(pool.Close)

	if err := applyMigrationFiles(ctx, pool, migrationGlobUp); err != nil {
		tb.Fatalf("apply migrations: %v", err)
	}
	return pool
}

func setupBillingMoneyCoreBenchmark(b *testing.B) (*pgxpool.Pool, string) {
	b.Helper()

	pool := setupBillingMoneyCoreRawPool(b)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, scope := createBillingMoneyAccount(b, ctx, pool, 9000, 1_000_000)
	insertUsageOperation(b, ctx, pool, 9003, accountID, scope, "reserved")
	insertUsageHold(b, ctx, pool, 9004, 9003, accountID, scope, "active", 100)
	topupID, attemptID := createTopupWithAttempt(b, ctx, pool, 9010, accountID, scope)
	insertPaymentEvidence(b, ctx, pool, paymentEvidenceFixture{
		ID:          "benchmark-evidence",
		TopupID:     topupID,
		AttemptID:   attemptID,
		AccountID:   accountID,
		Scope:       scope,
		Fingerprint: "benchmark-evidence-fingerprint",
		Amount:      1_000,
	})
	insertLedgerEntry(b, ctx, pool, ledgerEntry{
		ID:           testUUID(9020),
		AccountID:    accountID,
		Scope:        scope,
		Effect:       "topup_credit",
		Amount:       1_000,
		SettledDelta: 1_000,
		SettledAfter: 1_001_000,
		Available:    1_001_000,
		Reason:       "benchmark-credit",
		CreatedBy:    "service",
	})
	insertReconciliationCase(b, ctx, pool, reconciliationCase{
		ID:        testUUID(9030),
		AccountID: accountID,
		Reason:    "stale_reservation",
		State:     "open",
		UsageID:   ptrString(testUUID(9003)),
	})
	return pool, accountID
}

func createBillingMoneyAccount(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed int, settled int64) (string, string) {
	tb.Helper()

	accountID := testUUID(seed)
	subjectID := fmt.Sprintf("user-%d", seed)
	scope := "user:" + subjectID
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts (
			account_id, account_scope_key, account_type, subject_authority,
			subject_id, state, version, created_at, updated_at
		)
		VALUES ($1, $2, 'user', 'identity-service', $3, 'active', 1, now(), now())
	`, accountID, scope, subjectID); err != nil {
		tb.Fatalf("insert billing account %s: %v", accountID, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_balances (
			account_id, account_scope_key, currency, settled_usd_atoms,
			reserved_usd_atoms, available_usd_atoms, pending_usd_atoms, version, updated_at
		)
		VALUES ($1, $2, 'USD', $3, 0, $3, 0, 1, now())
	`, accountID, scope, settled); err != nil {
		tb.Fatalf("insert account balance %s: %v", accountID, err)
	}
	return accountID, scope
}

func insertUsageOperation(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed int, accountID, scope, state string) {
	tb.Helper()
	terminalAt := any(nil)
	if state == "finalized" || state == "written_off" || state == "reversed" || state == "compensated" || state == "expired" {
		terminalAt = time.Now()
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO usage_operations (
			usage_operation_id, account_id, account_scope_key, state, operation_kind,
			client_usage_request_id, request_basis_fingerprint, pricing_snapshot_id,
			pricing_snapshot_fingerprint, quote_expires_at, fee_policy_version,
			reserve_policy_version, created_at, updated_at, reserved_at, terminal_at
		)
		VALUES ($1, $2, $3, $4, 'reserve', $5, $6, 'pricing-snapshot', 'pricing-fingerprint', now() + interval '1 hour', 'fee-v1', 'reserve-v1', now(), now(), now(), $7)
	`, testUUID(seed), accountID, scope, state, fmt.Sprintf("client-request-%d", seed), fmt.Sprintf("basis-fingerprint-%d", seed), terminalAt); err != nil {
		tb.Fatalf("insert usage operation %d: %v", seed, err)
	}
}

func insertUsageHold(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed, usageSeed int, accountID, scope, state string, reserved int64) {
	tb.Helper()
	terminalAt := any(nil)
	if state == "finalized" || state == "released" || state == "written_off" || state == "expired" || state == "reversed" {
		terminalAt = time.Now()
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO usage_holds (
			hold_id, usage_operation_id, account_id, account_scope_key, state,
			reserved_usd_atoms, released_usd_atoms, charged_usd_atoms,
			write_off_usd_atoms, pricing_snapshot_id, pricing_snapshot_fingerprint,
			quote_expires_at, fee_policy_version, reserve_policy_version,
			client_usage_request_id, request_basis_fingerprint, created_at,
			updated_at, expires_at, terminal_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 0, 0, 0, 'pricing-snapshot', 'pricing-fingerprint', $7, 'fee-v1', 'reserve-v1', $8, $9, now(), now(), $7, $10)
	`, testUUID(seed), testUUID(usageSeed), accountID, scope, state, reserved, time.Now().Add(time.Hour), fmt.Sprintf("client-request-%d", usageSeed), fmt.Sprintf("basis-fingerprint-%d", usageSeed), terminalAt); err != nil {
		tb.Fatalf("insert usage hold %d: %v", seed, err)
	}
}

type ledgerEntry struct {
	ID               string
	AccountID        string
	Scope            string
	Effect           string
	Amount           int64
	SettledDelta     int64
	ReservedDelta    int64
	PendingDelta     int64
	SettledAfter     int64
	ReservedAfter    int64
	PendingAfter     int64
	Available        int64
	BalanceVersion   int64
	SettlementEffect *string
	Reason           string
	CreatedBy        string
}

func insertLedgerEntry(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, entry ledgerEntry) {
	tb.Helper()
	version := entry.BalanceVersion
	if version == 0 {
		version = 1
	}
	if _, err := pool.Exec(ctx, ledgerInsertSQL(),
		entry.ID,
		entry.AccountID,
		entry.Scope,
		"USD",
		entry.Effect,
		entry.Amount,
		entry.SettledDelta,
		entry.ReservedDelta,
		entry.PendingDelta,
		entry.SettledAfter,
		entry.ReservedAfter,
		entry.Available,
		entry.PendingAfter,
		version,
		entry.SettlementEffect,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		time.Now(),
		time.Now(),
		entry.CreatedBy,
		entry.Reason,
		nil,
	); err != nil {
		tb.Fatalf("insert ledger entry %s: %v", entry.ID, err)
	}
}

func ledgerInsertSQL() string {
	return `
		INSERT INTO ledger_entries (
			ledger_entry_id, account_id, account_scope_key, currency, effect_type,
			amount_usd_atoms, settled_delta_usd_atoms, reserved_delta_usd_atoms,
			pending_delta_usd_atoms, settled_after_usd_atoms, reserved_after_usd_atoms,
			available_after_usd_atoms, pending_after_usd_atoms, balance_version_after,
			settlement_effect_id, idempotency_record_id, usage_operation_id,
			topup_operation_id, payment_attempt_id, payment_evidence_id,
			qualified_inference_evidence_id, reversal_of_ledger_entry_id,
			correction_of_ledger_entry_id, effective_at, created_at, created_by_kind,
			reason_code, safe_metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
	`
}

func createTopupWithAttempt(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed int, accountID, scope string) (string, string) {
	tb.Helper()

	topupID := testUUID(seed)
	attemptID := fmt.Sprintf("payment-attempt-%d", seed)
	if _, err := pool.Exec(ctx, `
		INSERT INTO topup_operations (
			topup_operation_id, account_id, account_scope_key, state,
			accepted_quote_id, credited_usd_atoms, deposit_fee_usd_atoms,
			pricing_snapshot_id, pricing_snapshot_fingerprint,
			settlement_policy_version, billing_fee_policy_version,
			attempt_generation, presentation_version, created_at, updated_at
		)
		VALUES ($1, $2, $3, 'payment_pending', $4, 1000, 0, 'pricing-snapshot', 'pricing-fingerprint', 'settlement-v1', 'billing-fee-v1', 1, 1, now(), now())
	`, topupID, accountID, scope, fmt.Sprintf("quote-%d", seed)); err != nil {
		tb.Fatalf("insert topup operation %d: %v", seed, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO payment_attempts (
			payment_attempt_row_id, topup_operation_id, payment_attempt_id,
			attempt_generation, state, presentation_version, created_at, updated_at
		)
		VALUES ($1, $2, $3, 1, 'created', 1, now(), now())
	`, testUUID(seed+1), topupID, attemptID); err != nil {
		tb.Fatalf("insert payment attempt %d: %v", seed, err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE topup_operations
		SET current_payment_attempt_id = $2
		WHERE topup_operation_id = $1
	`, topupID, attemptID); err != nil {
		tb.Fatalf("link current payment attempt %d: %v", seed, err)
	}
	return topupID, attemptID
}

type paymentEvidenceFixture struct {
	ID          string
	TopupID     string
	AttemptID   string
	AccountID   string
	Scope       string
	Fingerprint string
	Amount      int64
}

func insertPaymentEvidence(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, evidence paymentEvidenceFixture) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO payment_evidence (
			payment_evidence_id, topup_operation_id, payment_attempt_id, account_id,
			account_scope_key, state, evidence_payload_fingerprint, evidence_kind,
			schema_version, finality_class, rail_family, settlement_amount_usd_atoms,
			received_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, 'accepted', $6, 'settlement', 'v1', 'final', 'card', $7, now(), now())
	`, evidence.ID, evidence.TopupID, evidence.AttemptID, evidence.AccountID, evidence.Scope, evidence.Fingerprint, evidence.Amount); err != nil {
		tb.Fatalf("insert payment evidence %s: %v", evidence.ID, err)
	}
}

func createIdempotency(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed int, accountID, operationKind, key, fingerprint, state string, conflictReason *string) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO idempotency_records (
			idempotency_record_id, account_id, operation_kind, idempotency_key,
			request_fingerprint, state, conflict_reason, retention_class,
			first_seen_at, last_seen_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'hot_replay', now(), now())
	`, testUUID(seed), accountID, operationKind, key, fingerprint, state, conflictReason); err != nil {
		tb.Fatalf("insert idempotency %d: %v", seed, err)
	}
}

func createOutcome(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed, idemSeed int, accountID, operationKind, status, resourceType, resourceID string, ledgerID *string) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO operation_outcomes (
			stored_outcome_id, idempotency_record_id, account_id, operation_kind,
			outcome_status, primary_resource_type, primary_resource_id,
			ledger_entry_id, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
	`, testUUID(seed), testUUID(idemSeed), accountID, operationKind, status, resourceType, resourceID, ledgerID); err != nil {
		tb.Fatalf("insert outcome %d: %v", seed, err)
	}
}

type reconciliationCase struct {
	ID             string
	AccountID      string
	Reason         string
	State          string
	UsageID        *string
	TopupID        *string
	AttemptID      *string
	EvidenceID     *string
	LegacyImportID *string
}

func insertReconciliationCase(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, c reconciliationCase) {
	tb.Helper()
	if _, err := pool.Exec(ctx, reconciliationInsertSQL(),
		c.ID,
		c.AccountID,
		c.Reason,
		c.State,
		"medium",
		c.UsageID,
		c.TopupID,
		c.AttemptID,
		c.EvidenceID,
		nil,
		nil,
		nil,
		c.LegacyImportID,
		nil,
		nil,
		nil,
		nil,
		0,
		time.Now().Add(-time.Second),
		nil,
		time.Now(),
		nil,
	); err != nil {
		tb.Fatalf("insert reconciliation case %s: %v", c.ID, err)
	}
}

func reconciliationInsertSQL() string {
	return `
		INSERT INTO reconciliation_cases (
			reconciliation_case_id, account_id, reason, state, severity,
			usage_operation_id, topup_operation_id, payment_attempt_id,
			payment_evidence_id, settlement_effect_id,
			qualified_inference_evidence_id, ledger_entry_id,
			legacy_balance_import_id, resolution_ledger_entry_id,
			resolution_settlement_effect_id, lease_owner, lease_deadline_at,
			attempt_count, next_attempt_at, support_safe_notes,
			created_at, resolved_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $21)
	`
}

func createLegacyImportWithLedger(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed int, accountID, scope string) string {
	tb.Helper()

	ledgerID := testUUID(seed)
	insertLedgerEntry(tb, ctx, pool, ledgerEntry{
		ID:           ledgerID,
		AccountID:    accountID,
		Scope:        scope,
		Effect:       "migration_import",
		Amount:       100,
		SettledDelta: 100,
		SettledAfter: 100,
		Available:    100,
		Reason:       "legacy-import",
		CreatedBy:    "migration",
	})
	batchID := testUUID(seed + 1)
	if _, err := pool.Exec(ctx, `
		INSERT INTO legacy_import_batches (
			legacy_import_batch_id, source_system, source_snapshot_fingerprint,
			state, account_count, derived_total_usd_atoms, created_at, updated_at
		)
		VALUES ($1, 'gonka-proxy', $2, 'loaded', 1, 100, now(), now())
	`, batchID, fmt.Sprintf("snapshot-fingerprint-%d", seed)); err != nil {
		tb.Fatalf("insert legacy import batch %d: %v", seed, err)
	}
	importID := testUUID(seed + 2)
	if _, err := pool.Exec(ctx, `
		INSERT INTO legacy_balance_imports (
			legacy_balance_import_id, legacy_import_batch_id, account_id,
			account_scope_key, legacy_source_system, legacy_subject_id,
			legacy_balance_ngonka_text, legacy_locked_rate_usd_text,
			derived_usd_atoms, import_fingerprint, parity_status,
			migration_ledger_entry_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'gonka-proxy', $5, '10.0', '0.01', 100, $6, 'mismatch', $7, now(), now())
	`, importID, batchID, accountID, scope, fmt.Sprintf("legacy-user-%d", seed), fmt.Sprintf("import-fingerprint-%d", seed), ledgerID); err != nil {
		tb.Fatalf("insert legacy balance import %d: %v", seed, err)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO legacy_balance_imports (
			legacy_balance_import_id, legacy_import_batch_id, account_id,
			account_scope_key, legacy_source_system, legacy_subject_id,
			legacy_balance_ngonka_text, legacy_locked_rate_usd_text,
			derived_usd_atoms, import_fingerprint, parity_status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'gonka-proxy', $5, '10.0', '0.01', 100, $6, 'pending', now(), now())
	`, testUUID(seed+3), batchID, accountID, scope, fmt.Sprintf("legacy-user-duplicate-%d", seed), fmt.Sprintf("import-fingerprint-duplicate-%d", seed))
	expectPgCode(tb, err, "23505")
	_, err = pool.Exec(ctx, `
		INSERT INTO legacy_balance_imports (
			legacy_balance_import_id, legacy_import_batch_id, account_id,
			account_scope_key, legacy_source_system, legacy_subject_id,
			legacy_balance_ngonka_text, legacy_locked_rate_usd_text,
			derived_usd_atoms, import_fingerprint, parity_status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'gonka-proxy', $5, '10.0', '0.01', 100, $6, 'pending', now(), now())
	`, testUUID(seed+4), batchID, accountID, scope, fmt.Sprintf("legacy-user-fingerprint-%d", seed), fmt.Sprintf("import-fingerprint-%d", seed))
	expectPgCode(tb, err, "23505")
	return importID
}

func reserveIfAvailable(ctx context.Context, pool *pgxpool.Pool, accountID string, amount int64) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var available int64
	if err := tx.QueryRow(ctx, `
		SELECT available_usd_atoms
		FROM account_balances
		WHERE account_id = $1
		FOR UPDATE
	`, accountID).Scan(&available); err != nil {
		return false, err
	}
	if available < amount {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE account_balances
		SET reserved_usd_atoms = reserved_usd_atoms + $2,
		    available_usd_atoms = available_usd_atoms - $2,
		    version = version + 1,
		    updated_at = now()
		WHERE account_id = $1
	`, accountID, amount); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func expectLockTimeout(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, accountID string) {
	tb.Helper()

	locker, err := pool.Begin(ctx)
	if err != nil {
		tb.Fatalf("begin lock holder: %v", err)
	}
	defer func() {
		_ = locker.Rollback(ctx)
	}()
	if _, err := locker.Exec(ctx, `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`, accountID); err != nil {
		tb.Fatalf("lock account balance for timeout test: %v", err)
	}

	waiter, err := pool.Begin(ctx)
	if err != nil {
		tb.Fatalf("begin lock waiter: %v", err)
	}
	defer func() {
		_ = waiter.Rollback(ctx)
	}()
	if _, err := waiter.Exec(ctx, `SET LOCAL lock_timeout = '50ms'`); err != nil {
		tb.Fatalf("set lock timeout: %v", err)
	}
	_, err = waiter.Exec(ctx, `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`, accountID)
	expectPgCode(tb, err, "55P03")
}

func expectDeadlock(tb testing.TB, ctx context.Context, pool *pgxpool.Pool) {
	tb.Helper()

	accountA, _ := createBillingMoneyAccount(tb, ctx, pool, 430, 100)
	accountB, _ := createBillingMoneyAccount(tb, ctx, pool, 431, 100)

	txA, err := pool.Begin(ctx)
	if err != nil {
		tb.Fatalf("begin deadlock tx A: %v", err)
	}
	defer func() {
		_ = txA.Rollback(ctx)
	}()
	txB, err := pool.Begin(ctx)
	if err != nil {
		tb.Fatalf("begin deadlock tx B: %v", err)
	}
	defer func() {
		_ = txB.Rollback(ctx)
	}()
	if _, err := txA.Exec(ctx, `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`, accountA); err != nil {
		tb.Fatalf("tx A lock account A: %v", err)
	}
	if _, err := txB.Exec(ctx, `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`, accountB); err != nil {
		tb.Fatalf("tx B lock account B: %v", err)
	}

	errs := make(chan error, 2)
	go func() {
		_, err := txA.Exec(ctx, `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`, accountB)
		errs <- err
	}()
	go func() {
		_, err := txB.Exec(ctx, `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`, accountA)
		errs <- err
	}()

	first := <-errs
	second := <-errs
	if !isPgCode(first, "40P01") && !isPgCode(second, "40P01") {
		tb.Fatalf("deadlock errors = [%v, %v], want one SQLSTATE 40P01", first, second)
	}
}

func lockOneReconciliationCase(tb testing.TB, ctx context.Context, pool *pgxpool.Pool) string {
	tb.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		tb.Fatalf("begin reconciliation lock tx: %v", err)
	}
	tb.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})
	var id string
	if err := tx.QueryRow(ctx, `
		SELECT reconciliation_case_id::text
		FROM reconciliation_cases
		WHERE state IN ('open', 'waiting_evidence')
		  AND next_attempt_at <= now()
		ORDER BY next_attempt_at, reconciliation_case_id
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&id); err != nil {
		tb.Fatalf("lock reconciliation case: %v", err)
	}
	return id
}

func countRows(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, table string) int64 {
	tb.Helper()
	var count int64
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
		tb.Fatalf("count %s: %v", table, err)
	}
	return count
}

func expectPgCode(tb testing.TB, err error, code string) {
	tb.Helper()
	if !isPgCode(err, code) {
		tb.Fatalf("postgres error code = %v, want SQLSTATE %s", err, code)
	}
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

func testUUID(seed int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", seed)
}

func ptrString(v string) *string {
	return &v
}

var _ pgx.Tx
