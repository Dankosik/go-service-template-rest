//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/Dankosik/billing-service/internal/infra/postgres"
	"github.com/jackc/pgx/v5"
)

func TestBillingMicroleaseRepositoryTransactions(t *testing.T) {
	pool := setupBillingMoneyCoreRawPool(t)
	repo, err := postgres.NewMicroleaseRepositoryFromPGXPool(pool)
	if err != nil {
		t.Fatalf("create microlease repository: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	accountID, scope := createBillingMoneyAccount(t, ctx, pool, 700, 10_000)

	issued, err := repo.Issue(ctx, issueCommand(accountID, scope, now))
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if issued.State != "active" || issued.IssuedCapUSDAtoms != 1_000 || issued.AvailableChildUSDAtoms != 1_000 {
		t.Fatalf("issued record = %+v", issued)
	}
	expectBalance(t, ctx, pool, accountID, 10_000, 1_000, 9_000)

	beforeLedger := countRows(t, ctx, pool, "ledger_entries")
	duplicate := issueCommand(accountID, scope, now)
	duplicate.MicroleaseID = testUUID(711)
	duplicate.IdempotencyRecordID = testUUID(712)
	duplicate.StoredOutcomeID = testUUID(713)
	duplicate.LedgerEntryID = testUUID(714)
	duplicate.OutboxID = testUUID(715)
	duplicate.RequestFingerprint = "changed-fingerprint"
	if _, err := repo.Issue(ctx, duplicate); err == nil {
		t.Fatal("Issue(changed idempotency fingerprint) error = nil, want conflict")
	}
	if afterLedger := countRows(t, ctx, pool, "ledger_entries"); afterLedger != beforeLedger {
		t.Fatalf("failed duplicate issue changed ledger rows: before=%d after=%d", beforeLedger, afterLedger)
	}

	badTerminal := terminalCommand(accountID, scope, now)
	badTerminal.MicroleaseChildDebitID = testUUID(721)
	badTerminal.InboxID = testUUID(722)
	badTerminal.OutboxID = testUUID(723)
	badTerminal.LedgerEntryID = testUUID(724)
	badTerminal.EventID = "terminal-over-child"
	badTerminal.EventFingerprint = "terminal-over-child-fingerprint"
	badTerminal.DebitAuthorizationID = "debit-over-child"
	badTerminal.ChargedUSDAtoms = 260
	if err := repo.ApplyTerminalSettlement(ctx, badTerminal); err == nil {
		t.Fatal("ApplyTerminalSettlement(over child cap) error = nil, want rollback")
	}
	if afterLedger := countRows(t, ctx, pool, "ledger_entries"); afterLedger != beforeLedger {
		t.Fatalf("failed terminal changed ledger rows: before=%d after=%d", beforeLedger, afterLedger)
	}

	if err := repo.ApplyTerminalSettlement(ctx, terminalCommand(accountID, scope, now)); err != nil {
		t.Fatalf("ApplyTerminalSettlement() error = %v", err)
	}
	expectBalance(t, ctx, pool, accountID, 9_900, 750, 9_150)
	expectInboxApplied(t, ctx, pool, testUUID(720))

	if err := repo.RecordCheckpoint(ctx, checkpointCommand(accountID, scope, now)); err != nil {
		t.Fatalf("RecordCheckpoint() error = %v", err)
	}
	expectInboxApplied(t, ctx, pool, testUUID(730))

	closed, err := repo.Close(ctx, closeCommand(accountID, now))
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if closed.State != "closed" || closed.TerminalReleaseUSDAtoms != 900 {
		t.Fatalf("closed record = %+v", closed)
	}
	expectBalance(t, ctx, pool, accountID, 9_900, 0, 9_900)

	if err := repo.UpsertAdmissionControl(ctx, postgres.AdmissionControlCommand{
		AdmissionControlID:          testUUID(760),
		ScopeKind:                   "account",
		ScopeKey:                    scope,
		UseClass:                    "chat",
		State:                       "fail_closed",
		ReasonCode:                  "rollout_default_closed",
		TerminalLagBucket:           "unknown",
		StaleAgeBucket:              "unknown",
		ReconciliationBacklogBucket: "unknown",
		AuditedActorKind:            "service",
		AuditedActorID:              "billing-service",
		ExpiresAt:                   now.Add(time.Minute),
		RenewedAt:                   now,
		CreatedAt:                   now,
	}); err != nil {
		t.Fatalf("UpsertAdmissionControl() error = %v", err)
	}

	claimed, err := repo.ClaimOutbox(ctx, now.Add(5*time.Second), 10)
	if err != nil {
		t.Fatalf("ClaimOutbox() error = %v", err)
	}
	if len(claimed) != 3 {
		t.Fatalf("claimed outbox len = %d, want issue terminal close", len(claimed))
	}
	if err := repo.MarkOutboxPublished(ctx, testUUID(719), now.Add(2*time.Second)); err != nil {
		t.Fatalf("MarkOutboxPublished() error = %v", err)
	}
}

func issueCommand(accountID, scope string, now time.Time) postgres.IssueMicroleaseCommand {
	return postgres.IssueMicroleaseCommand{
		MicroleaseID:               testUUID(710),
		AccountID:                  accountID,
		AccountScopeKey:            scope,
		ProxyAllocatorOwnerID:      "proxy-a",
		MicroleaseGeneration:       1,
		LeaseFence:                 "fence-a",
		IssuedCapUSDAtoms:          1_000,
		PricingSnapshotID:          "pricing-snapshot-710",
		PricingSnapshotFingerprint: "pricing-fingerprint-710",
		PricingPolicyVersion:       "pricing-v1",
		PricingDecisionAt:          now.Add(-time.Second),
		PricingSelectorKey:         "model:gpt-4.1:chat",
		PricingContractVersion:     "pricing-contract-v1",
		FeePolicyVersion:           "fee-v1",
		MicroleasePolicyVersion:    "microlease-v1",
		IssuedAt:                   now,
		DebitCutoffAt:              now.Add(25 * time.Second),
		ExpiresAt:                  now.Add(30 * time.Second),
		IdempotencyRecordID:        testUUID(716),
		IdempotencyKey:             "idem-issue-710",
		RequestFingerprint:         "issue-fingerprint-710",
		StoredOutcomeID:            testUUID(717),
		LedgerEntryID:              testUUID(718),
		OutboxID:                   testUUID(719),
		EventFingerprint:           "issue-event-fingerprint-710",
		SafeMetadata:               map[string]string{"lag_bucket": "ok"},
	}
}

func terminalCommand(accountID, scope string, now time.Time) postgres.TerminalSettlementCommand {
	return postgres.TerminalSettlementCommand{
		InboxID:                    testUUID(720),
		Topic:                      "billing.microlease.terminal.v1",
		PartitionID:                0,
		OffsetValue:                1,
		EventID:                    "terminal-720",
		ProducerIdentity:           "proxy-a",
		EventFingerprint:           "terminal-fingerprint-720",
		MicroleaseChildDebitID:     testUUID(721),
		MicroleaseID:               testUUID(710),
		DebitAuthorizationID:       "debit-720",
		AccountID:                  accountID,
		AccountScopeKey:            scope,
		ProxyAllocatorOwnerID:      "proxy-a",
		MicroleaseGeneration:       1,
		ChildSequence:              1,
		ChildCapUSDAtoms:           250,
		ChargedUSDAtoms:            100,
		ReleasedUSDAtoms:           150,
		RequestBasisFingerprint:    "request-fingerprint-720",
		TerminalBasisFingerprint:   "terminal-basis-fingerprint-720",
		PricingSnapshotID:          "pricing-snapshot-710",
		PricingSnapshotFingerprint: "pricing-fingerprint-710",
		TerminalKind:               "finalize",
		TerminalState:              "finalized",
		LedgerEntryID:              testUUID(724),
		SettlementEffectID:         testUUID(725),
		OutboxID:                   testUUID(726),
		OutboxEventFingerprint:     "terminal-outbox-fingerprint-720",
		TerminalAt:                 now.Add(100 * time.Millisecond),
		SettledAt:                  now.Add(time.Second),
		SafeMetadata:               map[string]string{"terminal_class": "ok"},
	}
}

func checkpointCommand(accountID, scope string, now time.Time) postgres.CheckpointCommand {
	return postgres.CheckpointCommand{
		InboxID:                    testUUID(730),
		Topic:                      "billing.microlease.checkpoint.v1",
		PartitionID:                0,
		OffsetValue:                2,
		EventID:                    "checkpoint-730",
		ProducerIdentity:           "proxy-a",
		EventFingerprint:           "checkpoint-event-fingerprint-730",
		CheckpointID:               testUUID(731),
		MicroleaseID:               testUUID(710),
		AccountID:                  accountID,
		AccountScopeKey:            scope,
		ProxyAllocatorOwnerID:      "proxy-a",
		MicroleaseGeneration:       1,
		CheckpointSequence:         1,
		CheckpointKind:             "progress",
		AllocatedChildHighWater:    1,
		AllocatedChildCount:        1,
		AllocatedChildCapUSDAtoms:  250,
		TerminalSubmittedCount:     1,
		TerminalPublishedCount:     1,
		TerminalAcceptedCount:      1,
		UnresolvedChildCount:       0,
		UnresolvedChildCapUSDAtoms: 0,
		LocalRemainingUSDAtoms:     750,
		CheckpointFingerprint:      "checkpoint-fingerprint-730",
		CreatedAt:                  now.Add(2 * time.Second),
		AppliedAt:                  now.Add(2 * time.Second),
	}
}

func closeCommand(accountID string, now time.Time) postgres.CloseMicroleaseCommand {
	return postgres.CloseMicroleaseCommand{
		MicroleaseID:        testUUID(710),
		AccountID:           accountID,
		IdempotencyRecordID: testUUID(740),
		IdempotencyKey:      "idem-close-740",
		RequestFingerprint:  "close-fingerprint-740",
		StoredOutcomeID:     testUUID(741),
		LedgerEntryID:       testUUID(742),
		OutboxID:            testUUID(743),
		EventFingerprint:    "close-outbox-fingerprint-740",
		ReleasedUSDAtoms:    750,
		CloseState:          "closed",
		ClosedAt:            now.Add(3 * time.Second),
		Now:                 now.Add(3 * time.Second),
		SafeMetadata:        map[string]string{"close_kind": "proof"},
	}
}

func expectBalance(tb testing.TB, ctx context.Context, pool pgQueryer, accountID string, settled, reserved, available int64) {
	tb.Helper()
	var gotSettled, gotReserved, gotAvailable int64
	if err := pool.QueryRow(ctx, `
		SELECT settled_usd_atoms, reserved_usd_atoms, available_usd_atoms
		FROM account_balances
		WHERE account_id = $1
	`, accountID).Scan(&gotSettled, &gotReserved, &gotAvailable); err != nil {
		tb.Fatalf("read balance: %v", err)
	}
	if gotSettled != settled || gotReserved != reserved || gotAvailable != available {
		tb.Fatalf("balance = settled %d reserved %d available %d, want %d/%d/%d", gotSettled, gotReserved, gotAvailable, settled, reserved, available)
	}
}

func expectInboxApplied(tb testing.TB, ctx context.Context, pool pgQueryer, inboxID string) {
	tb.Helper()
	var state string
	if err := pool.QueryRow(ctx, `SELECT state FROM billing_event_inbox WHERE inbox_id = $1`, inboxID).Scan(&state); err != nil {
		tb.Fatalf("read inbox state: %v", err)
	}
	if state != "applied" {
		tb.Fatalf("inbox state = %q, want applied", state)
	}
}

type pgQueryer interface {
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}
