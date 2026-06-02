//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Dankosik/billing-service/internal/app/billingauthority"
	"github.com/Dankosik/billing-service/internal/infra/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBillingAuthorityRepositoryImportReadbackAndUsageLifecycle(t *testing.T) {
	pool := setupBillingMoneyCoreRawPool(t)
	repo, err := postgres.NewBillingAuthorityRepositoryFromPGXPool(pool)
	if err != nil {
		t.Fatalf("create authority repository: %v", err)
	}
	leaseRepo, err := postgres.NewMicroleaseRepositoryFromPGXPool(pool)
	if err != nil {
		t.Fatalf("create microlease repository: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now().UTC()
	accountID, scope := createBillingMoneyAccount(t, ctx, pool, 810, 10_000)
	insertLegacyImportReadback(t, ctx, pool, 8100, accountID, scope, "parity_checked", "mismatch", now.Add(-2*time.Minute))
	insertLegacyImportReadback(t, ctx, pool, 8110, accountID, scope, "applied", "corrected", now.Add(-time.Minute))

	account, err := repo.ResolveAccount(ctx, billingauthority.AccountResolveRequest{RepresentedSubjectID: "user-810"})
	if err != nil {
		t.Fatalf("ResolveAccount() error = %v", err)
	}
	if account.AccountScopeKey != scope || account.ImportState != "accepted" || !account.BalanceReadEligible {
		t.Fatalf("ResolveAccount() = %+v, want accepted eligible account %s", account, scope)
	}

	issue := issueCommand(accountID, scope, now)
	if _, err := leaseRepo.Issue(ctx, issue); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	reserve := authorityReserveCommand(scope, issue, 8120, 1, 250)
	reserved, err := repo.ReserveUsage(ctx, reserve)
	if err != nil {
		t.Fatalf("ReserveUsage() error = %v", err)
	}
	if reserved.ResultCode != "accepted" || reserved.State != "reserved" || reserved.StoredOutcomeID == "" {
		t.Fatalf("ReserveUsage() = %+v, want accepted reserved with stored outcome", reserved)
	}
	expectChildDebit(t, ctx, pool, reserve.MicroleaseChildDebitID, reserve.UsageOperationID, "terminal_pending")
	expectSpendingMicroleaseAvailable(t, ctx, pool, issue.MicroleaseID, 750)
	if holds := countRows(t, ctx, pool, "usage_holds"); holds != 0 {
		t.Fatalf("usage_holds rows = %d, want no direct account-balance hold", holds)
	}

	duplicate, err := repo.ReserveUsage(ctx, reserve)
	if err != nil {
		t.Fatalf("ReserveUsage(duplicate) error = %v", err)
	}
	if duplicate.ResultCode != "duplicate_stored_outcome" || duplicate.StoredOutcomeID != reserved.StoredOutcomeID {
		t.Fatalf("ReserveUsage(duplicate) = %+v, want replay of %s", duplicate, reserved.StoredOutcomeID)
	}
	changed := reserve
	changed.RequestFingerprint = "changed-reserve-fingerprint"
	if _, err := repo.ReserveUsage(ctx, changed); !errors.Is(err, billingauthority.ErrConflict) {
		t.Fatalf("ReserveUsage(changed fingerprint) error = %v, want ErrConflict", err)
	}

	finalized, err := repo.CompleteUsage(ctx, billingauthority.UsageTerminalCommand{
		AccountScopeKey:        scope,
		UsageOperationID:       reserve.UsageOperationID,
		TerminalKind:           "finalize",
		IdempotencyKey:         "terminal-idem-8120",
		RequestFingerprint:     "terminal-fingerprint-8120",
		MicroleaseID:           reserve.MicroleaseID,
		MicroleaseChildDebitID: reserve.MicroleaseChildDebitID,
		DebitAuthorizationID:   reserve.DebitAuthorizationID,
		TerminalOutcomeID:      testUUID(8124),
		TerminalFingerprint:    "terminal-basis-8120",
		ChargedUSDAtoms:        100,
		ReleasedUSDAtoms:       150,
		Pricing:                reserve.Pricing,
		Metadata:               map[string]string{"terminal_class": "ok"},
	})
	if err != nil {
		t.Fatalf("CompleteUsage(finalize) error = %v", err)
	}
	if finalized.State != "finalized" || finalized.ResultCode != "accepted" {
		t.Fatalf("CompleteUsage(finalize) = %+v, want finalized accepted", finalized)
	}
	expectChildDebit(t, ctx, pool, reserve.MicroleaseChildDebitID, reserve.UsageOperationID, "finalized")
	expectBalance(t, ctx, pool, accountID, 9_900, 750, 9_150)

	writeOffReserve := authorityReserveCommand(scope, issue, 8130, 2, 100)
	if _, err := repo.ReserveUsage(ctx, writeOffReserve); err != nil {
		t.Fatalf("ReserveUsage(write off child) error = %v", err)
	}
	writtenOff, err := repo.CompleteUsage(ctx, billingauthority.UsageTerminalCommand{
		AccountScopeKey:        scope,
		UsageOperationID:       writeOffReserve.UsageOperationID,
		TerminalKind:           "write_off",
		IdempotencyKey:         "terminal-idem-8130",
		RequestFingerprint:     "terminal-fingerprint-8130",
		MicroleaseID:           writeOffReserve.MicroleaseID,
		MicroleaseChildDebitID: writeOffReserve.MicroleaseChildDebitID,
		DebitAuthorizationID:   writeOffReserve.DebitAuthorizationID,
		TerminalOutcomeID:      testUUID(8134),
		TerminalFingerprint:    "terminal-basis-8130",
		WriteOffUSDAtoms:       100,
		Pricing:                writeOffReserve.Pricing,
		Metadata:               map[string]string{"terminal_class": "write_off"},
	})
	if err != nil {
		t.Fatalf("CompleteUsage(write off) error = %v", err)
	}
	if writtenOff.State != "written_off" {
		t.Fatalf("CompleteUsage(write off) = %+v, want written_off", writtenOff)
	}
	expectBalance(t, ctx, pool, accountID, 9_900, 650, 9_250)

	chargeLedgerID := latestLedgerByEffect(t, ctx, pool, accountID, "microlease_child_charge")
	reversed, err := repo.ReverseUsage(ctx, billingauthority.UsageReversalCommand{
		AccountScopeKey:       scope,
		UsageOperationID:      reserve.UsageOperationID,
		IdempotencyKey:        "reversal-idem-8120",
		RequestFingerprint:    "reversal-fingerprint-8120",
		OriginalLedgerEntryID: chargeLedgerID,
		ReversalUSDAtoms:      100,
		ReasonCode:            "operator_reversal",
		Metadata:              map[string]string{"reversal_class": "operator"},
	})
	if err != nil {
		t.Fatalf("ReverseUsage() error = %v", err)
	}
	if reversed.State != "reversed" {
		t.Fatalf("ReverseUsage() = %+v, want reversed", reversed)
	}
	expectBalance(t, ctx, pool, accountID, 10_000, 650, 9_350)

	readback, err := repo.ReadUsageOperation(ctx, billingauthority.UsageReadbackRequest{
		AccountScopeKey:  scope,
		UsageOperationID: reserve.UsageOperationID,
		TraceRequestID:   "trace-readback",
		DeadlineAt:       now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("ReadUsageOperation() error = %v", err)
	}
	if readback.State != "reversed" || readback.BillingOperationID != reserve.MicroleaseChildDebitID {
		t.Fatalf("ReadUsageOperation() = %+v, want reversed linked to child debit", readback)
	}

	exposure, err := repo.ReadExposure(ctx, billingauthority.AdminExposureRequest{AccountScopeKey: scope})
	if err != nil {
		t.Fatalf("ReadExposure() error = %v", err)
	}
	if exposure.ActiveMicroleaseUSDAtoms != 650 || exposure.UnresolvedChildDebitUSDAtoms != 0 {
		t.Fatalf("ReadExposure() = %+v, want active microlease 650 and no unresolved child debit", exposure)
	}
	ledger, err := repo.ListLedgerEntries(ctx, billingauthority.AdminLedgerRequest{AccountScopeKey: scope, Limit: 10})
	if err != nil {
		t.Fatalf("ListLedgerEntries() error = %v", err)
	}
	if !containsLedgerEffect(ledger, "microlease_reversal") || !containsLedgerEffect(ledger, "microlease_write_off") || !containsLedgerEffect(ledger, "microlease_child_charge") {
		t.Fatalf("ledger entries = %+v, want charge/write-off/reversal effects", ledger)
	}
}

func authorityReserveCommand(scope string, issue postgres.IssueMicroleaseCommand, seed int, childSequence int64, childCap int64) billingauthority.UsageReserveCommand {
	return billingauthority.UsageReserveCommand{
		AccountScopeKey:        scope,
		UsageOperationID:       testUUID(seed),
		AuthorityMode:          billingauthority.AuthorityModeMicroleaseChildDebit,
		IdempotencyKey:         fmt.Sprintf("reserve-idem-%d", seed),
		RequestFingerprint:     fmt.Sprintf("reserve-fingerprint-%d", seed),
		RequestID:              fmt.Sprintf("request-%d", seed),
		MicroleaseID:           issue.MicroleaseID,
		MicroleaseChildDebitID: testUUID(seed + 1),
		DebitAuthorizationID:   fmt.Sprintf("debit-%d", seed),
		ProxyAllocatorOwnerID:  issue.ProxyAllocatorOwnerID,
		MicroleaseGeneration:   issue.MicroleaseGeneration,
		LeaseFence:             issue.LeaseFence,
		ChildSequence:          childSequence,
		ChildCapUSDAtoms:       childCap,
		RepresentedSubjectID:   "user-810",
		Pricing: billingauthority.PricingSnapshot{
			ID:              issue.PricingSnapshotID,
			Fingerprint:     issue.PricingSnapshotFingerprint,
			PolicyVersion:   issue.PricingPolicyVersion,
			DecisionAt:      issue.PricingDecisionAt,
			SelectorKey:     issue.PricingSelectorKey,
			UseClass:        "usage_reserve",
			ContractVersion: issue.PricingContractVersion,
		},
		Metadata: map[string]string{"surface": "chat"},
	}
}

func insertLegacyImportReadback(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, seed int, accountID, scope, batchState, parity string, createdAt time.Time) {
	tb.Helper()
	batchID := testUUID(seed)
	if _, err := pool.Exec(ctx, `
		INSERT INTO legacy_import_batches (
			legacy_import_batch_id, source_system, source_snapshot_fingerprint,
			state, account_count, derived_total_usd_atoms, created_at, updated_at
		)
		VALUES ($1, 'gonka-proxy', $2, $3, 1, 100, $4, $4)
	`, batchID, fmt.Sprintf("authority-import-snapshot-%d", seed), batchState, createdAt); err != nil {
		tb.Fatalf("insert authority legacy import batch %d: %v", seed, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legacy_balance_imports (
			legacy_balance_import_id, legacy_import_batch_id, account_id,
			account_scope_key, legacy_source_system, legacy_subject_id,
			legacy_balance_ngonka_text, legacy_locked_rate_usd_text,
			derived_usd_atoms, import_fingerprint, parity_status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'gonka-proxy', $5, '10.0', '0.01', 100, $6, $7, $8, $8)
	`, testUUID(seed+1), batchID, accountID, scope, fmt.Sprintf("legacy-authority-user-%d", seed), fmt.Sprintf("authority-import-fingerprint-%d", seed), parity, createdAt); err != nil {
		tb.Fatalf("insert authority legacy import row %d: %v", seed, err)
	}
}

func expectChildDebit(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, childID, usageID, state string) {
	tb.Helper()
	var gotUsageID, gotState string
	if err := pool.QueryRow(ctx, `
		SELECT usage_operation_id::text, state
		FROM microlease_child_debits
		WHERE microlease_child_debit_id = $1
	`, childID).Scan(&gotUsageID, &gotState); err != nil {
		tb.Fatalf("read child debit %s: %v", childID, err)
	}
	if gotUsageID != usageID || gotState != state {
		tb.Fatalf("child debit = usage %s state %s, want %s/%s", gotUsageID, gotState, usageID, state)
	}
}

func expectSpendingMicroleaseAvailable(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, microleaseID string, want int64) {
	tb.Helper()
	var got int64
	if err := pool.QueryRow(ctx, `
		SELECT available_child_cap_usd_atoms
		FROM spending_microleases
		WHERE microlease_id = $1
	`, microleaseID).Scan(&got); err != nil {
		tb.Fatalf("read spending microlease %s: %v", microleaseID, err)
	}
	if got != want {
		tb.Fatalf("available child cap = %d, want %d", got, want)
	}
}

func latestLedgerByEffect(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, accountID string, effect string) string {
	tb.Helper()
	var ledgerID string
	if err := pool.QueryRow(ctx, `
		SELECT ledger_entry_id::text
		FROM ledger_entries
		WHERE account_id = $1 AND effect_type = $2
		ORDER BY created_at DESC, ledger_entry_id DESC
		LIMIT 1
	`, accountID, effect).Scan(&ledgerID); err != nil {
		tb.Fatalf("read latest ledger %s: %v", effect, err)
	}
	return ledgerID
}

func containsLedgerEffect(entries []billingauthority.LedgerEntry, effect string) bool {
	for _, entry := range entries {
		if entry.EffectType == effect {
			return true
		}
	}
	return false
}
