package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dankosik/billing-service/internal/app/billingauthority"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestBillingRepositoriesPersistAuthorityCutoverLifecycle(t *testing.T) {
	t.Parallel()

	pool := setupRepositoryIntegrationPool(t)
	authorityRepo, err := NewBillingAuthorityRepositoryFromPGXPool(pool)
	if err != nil {
		t.Fatalf("create authority repository: %v", err)
	}
	microleaseRepo, err := NewMicroleaseRepositoryFromPGXPool(pool)
	if err != nil {
		t.Fatalf("create microlease repository: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now().UTC()
	accountID, scope := createRepositoryAccount(ctx, t, pool, 9100, 10_000)
	insertRepositoryLegacyImport(ctx, t, pool, accountID, scope, now)

	account, err := authorityRepo.ResolveAccount(ctx, billingauthority.AccountResolveRequest{RepresentedSubjectID: "user-9100"})
	if err != nil {
		t.Fatalf("ResolveAccount() error = %v", err)
	}
	if account.AccountScopeKey != scope || !account.BalanceReadEligible || account.ImportState != "accepted" {
		t.Fatalf("ResolveAccount() = %+v, want accepted migrated account", account)
	}
	balance, err := authorityRepo.ReadBalance(ctx, billingauthority.BalanceReadRequest{AccountScopeKey: scope})
	if err != nil {
		t.Fatalf("ReadBalance() error = %v", err)
	}
	if balance.AvailableUSDAtoms != 10_000 || balance.RuntimeGateState != "ready" {
		t.Fatalf("ReadBalance() = %+v, want ready 10000 available", balance)
	}

	issue := repositoryIssueCommand(accountID, scope, now)
	if _, err := microleaseRepo.Issue(ctx, issue); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	expectRepositoryBalance(ctx, t, pool, accountID, 10_000, 1_000, 9_000)

	reserve := repositoryReserveCommand(scope, issue, 9110, 250)
	reserved, err := authorityRepo.ReserveUsage(ctx, reserve)
	if err != nil {
		t.Fatalf("ReserveUsage() error = %v", err)
	}
	if reserved.ResultCode != "accepted" || reserved.State != "reserved" || reserved.StoredOutcomeID == "" {
		t.Fatalf("ReserveUsage() = %+v, want accepted reserve", reserved)
	}
	child, err := authorityRepo.ReadTerminalChildDebitByAuthorization(ctx, reserve.MicroleaseID, reserve.DebitAuthorizationID)
	if err != nil {
		t.Fatalf("ReadTerminalChildDebitByAuthorization() error = %v", err)
	}
	if child.UsageOperationID != reserve.UsageOperationID || child.AccountScopeKey != scope {
		t.Fatalf("terminal child debit = %+v, want reserved operation linkage", child)
	}
	replay, err := authorityRepo.ReserveUsage(ctx, reserve)
	if err != nil {
		t.Fatalf("ReserveUsage(replay) error = %v", err)
	}
	if replay.ResultCode != "duplicate_stored_outcome" || replay.StoredOutcomeID != reserved.StoredOutcomeID {
		t.Fatalf("ReserveUsage(replay) = %+v, want stored outcome %s", replay, reserved.StoredOutcomeID)
	}

	finalized, err := authorityRepo.CompleteUsage(ctx, billingauthority.UsageTerminalCommand{
		AccountScopeKey:        scope,
		UsageOperationID:       reserve.UsageOperationID,
		TerminalKind:           "finalize",
		IdempotencyKey:         "terminal-idem-9110",
		RequestFingerprint:     "terminal-fingerprint-9110",
		MicroleaseID:           reserve.MicroleaseID,
		MicroleaseChildDebitID: reserve.MicroleaseChildDebitID,
		DebitAuthorizationID:   reserve.DebitAuthorizationID,
		TerminalOutcomeID:      repositoryUUID(9114),
		TerminalFingerprint:    "terminal-basis-9110",
		ChargedUSDAtoms:        100,
		ReleasedUSDAtoms:       150,
		Pricing:                reserve.Pricing,
		Metadata:               map[string]string{"terminal_class": "ok"},
	})
	if err != nil {
		t.Fatalf("CompleteUsage() error = %v", err)
	}
	if finalized.State != "finalized" || finalized.ResultCode != "accepted" {
		t.Fatalf("CompleteUsage() = %+v, want finalized accepted", finalized)
	}
	expectRepositoryBalance(ctx, t, pool, accountID, 9_900, 750, 9_150)

	if err := microleaseRepo.RecordCheckpoint(ctx, repositoryCheckpointCommand(accountID, scope, now)); err != nil {
		t.Fatalf("RecordCheckpoint() error = %v", err)
	}
	closed, err := microleaseRepo.Close(ctx, repositoryCloseCommand(accountID, now))
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if closed.State != "closed" || closed.AvailableChildUSDAtoms != 0 {
		t.Fatalf("Close() = %+v, want closed exhausted microlease", closed)
	}
	expectRepositoryBalance(ctx, t, pool, accountID, 9_900, 0, 9_900)

	if err := microleaseRepo.UpsertAdmissionControl(ctx, AdmissionControlCommand{
		AdmissionControlID:          repositoryUUID(9160),
		ScopeKind:                   "global",
		ScopeKey:                    "all",
		UseClass:                    "chat",
		State:                       "open",
		ReasonCode:                  "worker_ready",
		TerminalLagBucket:           "none",
		StaleAgeBucket:              "none",
		ReconciliationBacklogBucket: "none",
		AuditedActorKind:            "worker",
		AuditedActorID:              "billing-service",
		ExpiresAt:                   now.Add(time.Minute),
		RenewedAt:                   now,
		CreatedAt:                   now,
		SafeMetadata:                map[string]string{"worker_role": "admission_control_renew"},
	}); err != nil {
		t.Fatalf("UpsertAdmissionControl() error = %v", err)
	}
	if err := microleaseRepo.RecordQuarantine(ctx, QuarantineCommand{
		InboxID:          repositoryUUID(9165),
		Topic:            "billing.microlease.terminal.v1",
		PartitionID:      3,
		OffsetValue:      33,
		EventID:          "quarantined-9165",
		ProducerIdentity: "gonka-proxy",
		BusinessIdentity: "terminal:quarantined-9165",
		EventFingerprint: "quarantine-fingerprint-9165",
		ReasonClass:      "schema_contract_mismatch",
		QuarantinedAt:    now.Add(4 * time.Second),
		SafeMetadata:     map[string]string{"reason_class": "schema_contract_mismatch"},
	}); err != nil {
		t.Fatalf("RecordQuarantine() error = %v", err)
	}
	if retried, err := microleaseRepo.RetryEligibleInbox(ctx, now.Add(time.Minute), 5); err != nil || retried == 0 {
		t.Fatalf("RetryEligibleInbox() = %d, %v; want retried quarantine", retried, err)
	}
	if _, err := microleaseRepo.RetryEligibleInbox(ctx, now.Add(time.Minute), 5); err != nil {
		t.Fatalf("RetryEligibleInbox() error = %v", err)
	}
	if stale, err := microleaseRepo.ScanStaleMicroleases(ctx, now.Add(time.Hour), 5); err != nil || len(stale) != 0 {
		t.Fatalf("ScanStaleMicroleases() = %d, %v; want none after close", len(stale), err)
	}

	outbox, err := microleaseRepo.ClaimOutbox(ctx, now.Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("ClaimOutbox() error = %v", err)
	}
	if len(outbox) < 2 {
		t.Fatalf("ClaimOutbox() returned %d records, want issue and terminal facts", len(outbox))
	}
	if err := microleaseRepo.MarkOutboxPublished(ctx, uuidString(outbox[0].OutboxID), now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkOutboxPublished() error = %v", err)
	}
	if err := microleaseRepo.MarkOutboxRetry(ctx, uuidString(outbox[1].OutboxID), now.Add(2*time.Minute), "publish_failed"); err != nil {
		t.Fatalf("MarkOutboxRetry() error = %v", err)
	}

	readback, err := authorityRepo.ReadUsageOperation(ctx, billingauthority.UsageReadbackRequest{AccountScopeKey: scope, UsageOperationID: reserve.UsageOperationID})
	if err != nil {
		t.Fatalf("ReadUsageOperation() error = %v", err)
	}
	if readback.State != "finalized" || readback.BillingOperationID != reserve.MicroleaseChildDebitID {
		t.Fatalf("ReadUsageOperation() = %+v, want finalized child debit readback", readback)
	}
	insertRepositoryReconciliationCase(ctx, t, pool, accountID, reserve.UsageOperationID)
	cases, err := authorityRepo.ListReconciliationCases(ctx, billingauthority.ReconciliationCasesRequest{
		AccountScopeKey: scope,
		State:           "open",
		Severity:        "medium",
		Limit:           5,
	})
	if err != nil {
		t.Fatalf("ListReconciliationCases() error = %v", err)
	}
	if len(cases) != 1 || cases[0].SafeLineageID != reserve.UsageOperationID {
		t.Fatalf("ListReconciliationCases() = %+v, want one usage-linked case", cases)
	}
	ledger, err := authorityRepo.ListLedgerEntries(ctx, billingauthority.AdminLedgerRequest{AccountScopeKey: scope, Limit: 10})
	if err != nil {
		t.Fatalf("ListLedgerEntries() error = %v", err)
	}
	if len(ledger) == 0 {
		t.Fatal("ListLedgerEntries() returned no entries")
	}
	chargeLedgerID := latestRepositoryLedgerByEffect(ctx, t, pool, accountID, "microlease_child_charge")
	reversed, err := authorityRepo.ReverseUsage(ctx, billingauthority.UsageReversalCommand{
		AccountScopeKey:       scope,
		UsageOperationID:      reserve.UsageOperationID,
		IdempotencyKey:        "reversal-idem-9110",
		RequestFingerprint:    "reversal-fingerprint-9110",
		OriginalLedgerEntryID: chargeLedgerID,
		ReversalUSDAtoms:      100,
		ReasonCode:            "operator_reversal",
		Metadata:              map[string]string{"reversal_class": "operator"},
	})
	if err != nil {
		t.Fatalf("ReverseUsage() error = %v", err)
	}
	if reversed.State != "reversed" || reversed.ResultCode != "accepted" {
		t.Fatalf("ReverseUsage() = %+v, want reversed accepted", reversed)
	}
	expectRepositoryBalance(ctx, t, pool, accountID, 10_000, 0, 10_000)

	workerIssue := repositoryIssueCommand(accountID, scope, now)
	workerIssue.MicroleaseID = repositoryUUID(9180)
	workerIssue.MicroleaseGeneration = 2
	workerIssue.LeaseFence = "fence-b"
	workerIssue.IssuedCapUSDAtoms = 500
	workerIssue.IdempotencyRecordID = repositoryUUID(9181)
	workerIssue.IdempotencyKey = "issue-idem-9180"
	workerIssue.RequestFingerprint = "issue-fingerprint-9180"
	workerIssue.StoredOutcomeID = repositoryUUID(9182)
	workerIssue.LedgerEntryID = repositoryUUID(9183)
	workerIssue.OutboxID = repositoryUUID(9184)
	workerIssue.EventFingerprint = "issue-event-fingerprint-9180"
	if _, err := microleaseRepo.Issue(ctx, workerIssue); err != nil {
		t.Fatalf("Issue(worker microlease) error = %v", err)
	}
	readLease, err := microleaseRepo.ReadMicrolease(ctx, workerIssue.MicroleaseID)
	if err != nil {
		t.Fatalf("ReadMicrolease() error = %v", err)
	}
	if readLease.MicroleaseID != workerIssue.MicroleaseID || readLease.AvailableChildUSDAtoms != 500 {
		t.Fatalf("ReadMicrolease() = %+v, want worker lease", readLease)
	}
	if err := microleaseRepo.ApplyTerminalSettlement(ctx, repositoryDirectTerminalCommand(accountID, scope, now)); err != nil {
		t.Fatalf("ApplyTerminalSettlement() error = %v", err)
	}
	expectRepositoryBalance(ctx, t, pool, accountID, 9_950, 300, 9_650)
	exposure, err := authorityRepo.ReadExposure(ctx, billingauthority.AdminExposureRequest{AccountScopeKey: scope})
	if err != nil {
		t.Fatalf("ReadExposure() error = %v", err)
	}
	if exposure.ActiveMicroleaseUSDAtoms != 300 || exposure.UnresolvedChildDebitUSDAtoms != 0 {
		t.Fatalf("ReadExposure() = %+v, want worker residual exposure", exposure)
	}
}

func setupRepositoryIntegrationPool(tb testing.TB) *pgxpool.Pool {
	tb.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := runRepositoryPostgresContainer(ctx)
	if err != nil {
		if repositoryDockerUnavailable(err) {
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
	root := repositoryRoot(tb)
	if _, err := MigrateUp(ctx, MigrationOptions{DSN: dsn, SourceFS: os.DirFS(root), SourcePath: "env/migrations"}); err != nil {
		tb.Fatalf("apply migrations: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		tb.Fatalf("create postgres pool: %v", err)
	}
	tb.Cleanup(pool.Close)
	return pool
}

func runRepositoryPostgresContainer(ctx context.Context) (container *tcpostgres.PostgresContainer, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("testcontainers panic: %v", recovered)
		}
	}()
	container, err = tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("run repository postgres container: %w", err)
	}
	return container, nil
}

func repositoryDockerUnavailable(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cannot connect to the docker daemon") ||
		strings.Contains(msg, "is the docker daemon running") ||
		strings.Contains(msg, "docker socket") ||
		strings.Contains(msg, "checked path: $xdg_runtime_dir")
}

func repositoryRoot(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("get cwd: %v", err)
	}
	for {
		if info, statErr := os.Stat(filepath.Join(dir, "env", "migrations")); statErr == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("repository root with env/migrations not found")
		}
		dir = parent
	}
}

func createRepositoryAccount(ctx context.Context, tb testing.TB, pool *pgxpool.Pool, seed int, settled int64) (string, string) {
	tb.Helper()
	accountID := repositoryUUID(seed)
	subjectID := fmt.Sprintf("user-%d", seed)
	scope := "user:" + subjectID
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts (
			account_id, account_scope_key, account_type, subject_authority,
			subject_id, state, version, created_at, updated_at
		)
		VALUES ($1, $2, 'user', 'identity-service', $3, 'active', 1, now(), now())
	`, accountID, scope, subjectID); err != nil {
		tb.Fatalf("insert billing account: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_balances (
			account_id, account_scope_key, currency, settled_usd_atoms,
			reserved_usd_atoms, available_usd_atoms, pending_usd_atoms, version, updated_at
		)
		VALUES ($1, $2, 'USD', $3, 0, $3, 0, 1, now())
	`, accountID, scope, settled); err != nil {
		tb.Fatalf("insert account balance: %v", err)
	}
	return accountID, scope
}

func insertRepositoryLegacyImport(ctx context.Context, tb testing.TB, pool *pgxpool.Pool, accountID, scope string, createdAt time.Time) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO legacy_import_batches (
			legacy_import_batch_id, source_system, source_snapshot_fingerprint,
			state, account_count, derived_total_usd_atoms, created_at, updated_at, applied_at
		)
		VALUES ($1, 'gonka-proxy', 'repository-import-fingerprint', 'applied', 1, 100, $2, $2, $2)
	`, repositoryUUID(9101), createdAt); err != nil {
		tb.Fatalf("insert legacy import batch: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legacy_balance_imports (
			legacy_balance_import_id, legacy_import_batch_id, account_id,
			account_scope_key, legacy_source_system, legacy_subject_id,
			legacy_balance_ngonka_text, legacy_locked_rate_usd_text,
			derived_usd_atoms, import_fingerprint, parity_status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'gonka-proxy', 'legacy-user-9100', '10.0', '0.01', 100, 'repository-import-row', 'matched', $5, $5)
	`, repositoryUUID(9102), repositoryUUID(9101), accountID, scope, createdAt); err != nil {
		tb.Fatalf("insert legacy import row: %v", err)
	}
}

func repositoryIssueCommand(accountID, scope string, now time.Time) IssueMicroleaseCommand {
	return IssueMicroleaseCommand{
		MicroleaseID:               repositoryUUID(9103),
		AccountID:                  accountID,
		AccountScopeKey:            scope,
		ProxyAllocatorOwnerID:      "proxy-a",
		MicroleaseGeneration:       2,
		LeaseFence:                 "fence-a",
		IssuedCapUSDAtoms:          1_000,
		PricingSnapshotID:          "pricing-snapshot-9103",
		PricingSnapshotFingerprint: "pricing-fingerprint-9103",
		PricingPolicyVersion:       "pricing-v1",
		PricingDecisionAt:          now.Add(-time.Second),
		PricingSelectorKey:         "model:gpt-4.1:chat",
		PricingContractVersion:     "pricing-contract-v1",
		FeePolicyVersion:           "fee-v1",
		MicroleasePolicyVersion:    "microlease-v1",
		IssuedAt:                   now,
		DebitCutoffAt:              now.Add(25 * time.Second),
		ExpiresAt:                  now.Add(30 * time.Second),
		IdempotencyRecordID:        repositoryUUID(9104),
		IdempotencyKey:             "issue-idem-9103",
		RequestFingerprint:         "issue-fingerprint-9103",
		StoredOutcomeID:            repositoryUUID(9105),
		LedgerEntryID:              repositoryUUID(9106),
		OutboxID:                   repositoryUUID(9107),
		EventFingerprint:           "issue-event-fingerprint-9103",
		SafeMetadata:               map[string]string{"lag_bucket": "ok"},
	}
}

func repositoryReserveCommand(scope string, issue IssueMicroleaseCommand, seed int, childCap int64) billingauthority.UsageReserveCommand {
	return billingauthority.UsageReserveCommand{
		AccountScopeKey:        scope,
		UsageOperationID:       repositoryUUID(seed),
		AuthorityMode:          billingauthority.AuthorityModeMicroleaseChildDebit,
		IdempotencyKey:         fmt.Sprintf("reserve-idem-%d", seed),
		RequestFingerprint:     fmt.Sprintf("reserve-fingerprint-%d", seed),
		RequestID:              fmt.Sprintf("request-%d", seed),
		MicroleaseID:           issue.MicroleaseID,
		MicroleaseChildDebitID: repositoryUUID(seed + 1),
		DebitAuthorizationID:   fmt.Sprintf("debit-%d", seed),
		ProxyAllocatorOwnerID:  issue.ProxyAllocatorOwnerID,
		MicroleaseGeneration:   issue.MicroleaseGeneration,
		LeaseFence:             issue.LeaseFence,
		ChildSequence:          1,
		ChildCapUSDAtoms:       childCap,
		RepresentedSubjectID:   "user-9100",
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

func repositoryCheckpointCommand(accountID, scope string, now time.Time) CheckpointCommand {
	return CheckpointCommand{
		InboxID:                    repositoryUUID(9120),
		Topic:                      "billing.microlease.checkpoint.v1",
		PartitionID:                0,
		OffsetValue:                2,
		EventID:                    "checkpoint-9120",
		ProducerIdentity:           "proxy-a",
		EventFingerprint:           "checkpoint-event-fingerprint-9120",
		CheckpointID:               repositoryUUID(9121),
		MicroleaseID:               repositoryUUID(9103),
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
		CheckpointFingerprint:      "checkpoint-fingerprint-9120",
		CreatedAt:                  now.Add(2 * time.Second),
		AppliedAt:                  now.Add(2 * time.Second),
		SafeMetadata:               map[string]string{"checkpoint_class": "ok"},
	}
}

func repositoryCloseCommand(accountID string, now time.Time) CloseMicroleaseCommand {
	return CloseMicroleaseCommand{
		MicroleaseID:        repositoryUUID(9103),
		AccountID:           accountID,
		IdempotencyRecordID: repositoryUUID(9130),
		IdempotencyKey:      "close-idem-9130",
		RequestFingerprint:  "close-fingerprint-9130",
		StoredOutcomeID:     repositoryUUID(9131),
		LedgerEntryID:       repositoryUUID(9132),
		OutboxID:            repositoryUUID(9133),
		EventFingerprint:    "close-outbox-fingerprint-9130",
		ReleasedUSDAtoms:    750,
		CloseState:          "closed",
		ClosedAt:            now.Add(3 * time.Second),
		Now:                 now.Add(3 * time.Second),
		SafeMetadata:        map[string]string{"close_kind": "proof"},
	}
}

func repositoryDirectTerminalCommand(accountID, scope string, now time.Time) TerminalSettlementCommand {
	return TerminalSettlementCommand{
		InboxID:                    repositoryUUID(9190),
		Topic:                      "billing.microlease.terminal.v1",
		PartitionID:                1,
		OffsetValue:                10,
		EventID:                    "terminal-9190",
		ProducerIdentity:           "gonka-proxy",
		EventFingerprint:           "terminal-fingerprint-9190",
		MicroleaseChildDebitID:     repositoryUUID(9191),
		MicroleaseID:               repositoryUUID(9180),
		DebitAuthorizationID:       "debit-9190",
		AccountID:                  accountID,
		AccountScopeKey:            scope,
		ProxyAllocatorOwnerID:      "proxy-a",
		MicroleaseGeneration:       1,
		ChildSequence:              1,
		ChildCapUSDAtoms:           200,
		ChargedUSDAtoms:            50,
		ReleasedUSDAtoms:           150,
		RequestBasisFingerprint:    "request-fingerprint-9190",
		TerminalBasisFingerprint:   "terminal-basis-fingerprint-9190",
		PricingSnapshotID:          "pricing-snapshot-9103",
		PricingSnapshotFingerprint: "pricing-fingerprint-9103",
		TerminalKind:               "finalize",
		TerminalState:              "finalized",
		LedgerEntryID:              repositoryUUID(9192),
		SettlementEffectID:         repositoryUUID(9193),
		OutboxID:                   repositoryUUID(9194),
		OutboxEventFingerprint:     "terminal-outbox-fingerprint-9190",
		TerminalAt:                 now.Add(5 * time.Second),
		SettledAt:                  now.Add(6 * time.Second),
		SafeMetadata:               map[string]string{"terminal_class": "worker"},
	}
}

func expectRepositoryBalance(ctx context.Context, tb testing.TB, pool *pgxpool.Pool, accountID string, settled, reserved, available int64) {
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
		tb.Fatalf("balance = %d/%d/%d, want %d/%d/%d", gotSettled, gotReserved, gotAvailable, settled, reserved, available)
	}
}

func insertRepositoryReconciliationCase(ctx context.Context, tb testing.TB, pool *pgxpool.Pool, accountID, usageOperationID string) {
	tb.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO reconciliation_cases (
			reconciliation_case_id, account_id, reason, state, severity,
			usage_operation_id, attempt_count, next_attempt_at, created_at, updated_at
		)
		VALUES ($1, $2, 'stale_reservation', 'open', 'medium', $3, 0, $4, $4, $4)
	`, repositoryUUID(9170), accountID, usageOperationID, time.Now().UTC()); err != nil {
		tb.Fatalf("insert reconciliation case: %v", err)
	}
}

func latestRepositoryLedgerByEffect(ctx context.Context, tb testing.TB, pool *pgxpool.Pool, accountID, effect string) string {
	tb.Helper()
	var ledgerID string
	if err := pool.QueryRow(ctx, `
		SELECT ledger_entry_id::text
		FROM ledger_entries
		WHERE account_id = $1 AND effect_type = $2
		ORDER BY created_at DESC, ledger_entry_id DESC
		LIMIT 1
	`, accountID, effect).Scan(&ledgerID); err != nil {
		tb.Fatalf("read latest ledger by effect %s: %v", effect, err)
	}
	return ledgerID
}

func repositoryUUID(seed int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", seed)
}
