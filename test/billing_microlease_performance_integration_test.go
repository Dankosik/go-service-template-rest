//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Dankosik/billing-service/internal/infra/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBillingMicroleasePerformanceBudgets(t *testing.T) {
	pool := setupBillingMoneyCoreRawPool(t)
	repo, err := postgres.NewMicroleaseRepositoryFromPGXPool(pool)
	if err != nil {
		t.Fatalf("create microlease repository: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	accountID, scope := createBillingMoneyAccount(t, ctx, pool, 120000, 1_000_000_000)

	issueSamples, microleases := measureMicroleaseIssues(t, ctx, repo, accountID, scope, now, 30, 121000)
	assertDurationBudget(t, "billing_issue_replenish", issueSamples, 100*time.Millisecond, 250*time.Millisecond)

	terminalSamples := measureTerminals(t, ctx, repo, accountID, scope, now, microleases)
	assertDurationBudget(t, "terminal_ingestion", terminalSamples, 100*time.Millisecond, 250*time.Millisecond)

	checkpointSamples := measureCheckpoints(t, ctx, repo, accountID, scope, now, microleases)
	assertDurationBudget(t, "checkpoint_cadence", checkpointSamples, 100*time.Millisecond, 250*time.Millisecond)

	closeSamples := measureCloses(t, ctx, repo, accountID, now, microleases)
	assertDurationBudget(t, "close_cadence", closeSamples, 100*time.Millisecond, 250*time.Millisecond)

	coldSamples := measureColdReplenishment(t, ctx, pool, repo, accountID, scope, now, 16, 130000)
	assertDurationBudget(t, "cold_replenishment", coldSamples, 250*time.Millisecond, 500*time.Millisecond)

	staleScanSamples := measureStaleScans(t, ctx, pool, 24)
	assertDurationBudget(t, "stale_reconciliation_scan", staleScanSamples, 100*time.Millisecond, 250*time.Millisecond)

	contentionSamples := measureAccountContention(t, ctx, repo, accountID, scope, now, 8, 140000)
	assertDurationBudget(t, "account_contention", contentionSamples, 250*time.Millisecond, 500*time.Millisecond)
}

func measureMicroleaseIssues(tb testing.TB, ctx context.Context, repo *postgres.MicroleaseRepository, accountID, scope string, now time.Time, count int, seedStart int) ([]time.Duration, []performanceMicrolease) {
	tb.Helper()

	samples := make([]time.Duration, 0, count)
	microleases := make([]performanceMicrolease, 0, count)
	for i := 0; i < count; i++ {
		seed := seedStart + i*10
		cmd := performanceIssueCommand(seed, accountID, scope, now.Add(time.Duration(i)*time.Millisecond), 1_000)
		start := time.Now()
		issued, err := repo.Issue(ctx, cmd)
		if err != nil {
			tb.Fatalf("Issue(%d) error = %v", i, err)
		}
		samples = append(samples, time.Since(start))
		microleases = append(microleases, performanceMicrolease{ID: issued.MicroleaseID, Seed: seed})
	}
	return samples, microleases
}

func measureTerminals(tb testing.TB, ctx context.Context, repo *postgres.MicroleaseRepository, accountID, scope string, now time.Time, microleases []performanceMicrolease) []time.Duration {
	tb.Helper()

	samples := make([]time.Duration, 0, len(microleases))
	for i, microlease := range microleases {
		cmd := performanceTerminalCommand(microlease.Seed+100_000, microlease.ID, accountID, scope, now.Add(time.Duration(i)*time.Millisecond))
		start := time.Now()
		if err := repo.ApplyTerminalSettlement(ctx, cmd); err != nil {
			tb.Fatalf("ApplyTerminalSettlement(%d) error = %v", i, err)
		}
		samples = append(samples, time.Since(start))
	}
	return samples
}

func measureCheckpoints(tb testing.TB, ctx context.Context, repo *postgres.MicroleaseRepository, accountID, scope string, now time.Time, microleases []performanceMicrolease) []time.Duration {
	tb.Helper()

	samples := make([]time.Duration, 0, len(microleases))
	for i, microlease := range microleases {
		cmd := performanceCheckpointCommand(microlease.Seed+200_000, microlease.ID, accountID, scope, now.Add(time.Duration(i)*time.Millisecond))
		start := time.Now()
		if err := repo.RecordCheckpoint(ctx, cmd); err != nil {
			tb.Fatalf("RecordCheckpoint(%d) error = %v", i, err)
		}
		samples = append(samples, time.Since(start))
	}
	return samples
}

func measureCloses(tb testing.TB, ctx context.Context, repo *postgres.MicroleaseRepository, accountID string, now time.Time, microleases []performanceMicrolease) []time.Duration {
	tb.Helper()

	samples := make([]time.Duration, 0, len(microleases))
	for i, microlease := range microleases {
		cmd := performanceCloseCommand(microlease.Seed+300_000, microlease.ID, accountID, now.Add(time.Duration(i)*time.Millisecond))
		start := time.Now()
		if _, err := repo.Close(ctx, cmd); err != nil {
			tb.Fatalf("Close(%d) error = %v", i, err)
		}
		samples = append(samples, time.Since(start))
	}
	return samples
}

func measureColdReplenishment(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, repo *postgres.MicroleaseRepository, accountID, scope string, now time.Time, count int, seedStart int) []time.Duration {
	tb.Helper()

	insertExpiredMicroleases(tb, ctx, pool, accountID, scope, now, seedStart+50_000, 32)
	samples := make([]time.Duration, 0, count)
	for i := 0; i < count; i++ {
		seed := seedStart + i*10
		cmd := performanceIssueCommand(seed, accountID, scope, now.Add(time.Duration(i)*time.Millisecond), 1_000)
		start := time.Now()
		if err := scanStaleMicroleases(ctx, pool, now, 16); err != nil {
			tb.Fatalf("cold stale scan(%d) error = %v", i, err)
		}
		if _, err := repo.Issue(ctx, cmd); err != nil {
			tb.Fatalf("cold Issue(%d) error = %v", i, err)
		}
		samples = append(samples, time.Since(start))
	}
	return samples
}

func measureStaleScans(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, count int) []time.Duration {
	tb.Helper()

	samples := make([]time.Duration, 0, count)
	now := time.Date(2026, 6, 1, 12, 5, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		start := time.Now()
		if err := scanStaleMicroleases(ctx, pool, now, 50); err != nil {
			tb.Fatalf("stale scan(%d) error = %v", i, err)
		}
		samples = append(samples, time.Since(start))
	}
	return samples
}

func measureAccountContention(tb testing.TB, ctx context.Context, repo *postgres.MicroleaseRepository, accountID, scope string, now time.Time, count int, seedStart int) []time.Duration {
	tb.Helper()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	samples := make([]time.Duration, 0, count)
	for i := 0; i < count; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := performanceIssueCommand(seedStart+i*10, accountID, scope, now.Add(time.Duration(i)*time.Millisecond), 500)
			start := time.Now()
			_, err := repo.Issue(ctx, cmd)
			elapsed := time.Since(start)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = fmt.Errorf("contention Issue(%d): %w", i, err)
			}
			samples = append(samples, elapsed)
		}()
	}
	wg.Wait()
	if firstErr != nil {
		tb.Fatal(firstErr)
	}
	return samples
}

func scanStaleMicroleases(ctx context.Context, pool *pgxpool.Pool, now time.Time, limit int) error {
	rows, err := pool.Query(ctx, `
		SELECT microlease_id
		FROM spending_microleases
		WHERE state IN ('active', 'cutoff', 'closing', 'expired')
		  AND expires_at <= $1
		ORDER BY expires_at, microlease_id
		LIMIT $2
	`, now, limit)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var microleaseID string
		if err := rows.Scan(&microleaseID); err != nil {
			return err
		}
	}
	return rows.Err()
}

func insertExpiredMicroleases(tb testing.TB, ctx context.Context, pool *pgxpool.Pool, accountID, scope string, now time.Time, seedStart, count int) {
	tb.Helper()

	for i := 0; i < count; i++ {
		seed := seedStart + i*10
		createIdempotency(tb, ctx, pool, seed+1, accountID, "microlease_issue", fmt.Sprintf("expired-idem-%d", seed), fmt.Sprintf("expired-fingerprint-%d", seed), "started", nil)
		createOutcome(tb, ctx, pool, seed+2, seed+1, accountID, "microlease_issue", "success", "spending_microlease", testUUID(seed), nil)
		if _, err := pool.Exec(ctx, `
			INSERT INTO spending_microleases (
				microlease_id, account_id, account_scope_key, proxy_allocator_owner_id,
				microlease_generation, lease_fence, state, issued_cap_usd_atoms,
				available_child_cap_usd_atoms, allocated_child_cap_reported_usd_atoms,
				terminal_charged_usd_atoms, terminal_released_usd_atoms, write_off_usd_atoms,
				pricing_snapshot_id, pricing_snapshot_fingerprint, pricing_policy_version,
				pricing_decision_at, pricing_selector_key, pricing_contract_version,
				fee_policy_version, microlease_policy_version, issued_at, debit_cutoff_at,
				expires_at, last_checkpoint_sequence, idempotency_record_id,
				stored_outcome_id, created_at, updated_at
			)
			VALUES ($1, $2, $3, 'proxy-a', 1, $4, 'active', 1000, 1000, 0, 0, 0, 0,
				$5, $6, 'pricing-policy-v1', $7, 'model:gpt-4.1:chat',
				'pricing-contract-v1', 'fee-v1', 'microlease-v1', $8, $9, $10, 0,
				$11, $12, $8, $8)
		`, testUUID(seed), accountID, scope, fmt.Sprintf("expired-fence-%d", seed),
			fmt.Sprintf("expired-pricing-%d", seed), fmt.Sprintf("expired-pricing-fingerprint-%d", seed),
			now.Add(-time.Minute), now.Add(-time.Minute), now.Add(-30*time.Second), now.Add(-time.Second),
			testUUID(seed+1), testUUID(seed+2)); err != nil {
			tb.Fatalf("insert expired microlease %d: %v", seed, err)
		}
	}
}

func assertDurationBudget(tb testing.TB, name string, samples []time.Duration, p95Budget, p99Budget time.Duration) {
	tb.Helper()

	p95 := percentileDuration(samples, 0.95)
	p99 := percentileDuration(samples, 0.99)
	tb.Logf("microlease billing performance %s: samples=%d p95=%s p99=%s p95_budget=%s p99_budget=%s", name, len(samples), p95, p99, p95Budget, p99Budget)
	if p95 > p95Budget {
		tb.Fatalf("%s p95 = %s, budget %s", name, p95, p95Budget)
	}
	if p99 > p99Budget {
		tb.Fatalf("%s p99 = %s, budget %s", name, p99, p99Budget)
	}
}

func percentileDuration(samples []time.Duration, percentile float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func performanceIssueCommand(seed int, accountID, scope string, now time.Time, capAtoms int64) postgres.IssueMicroleaseCommand {
	return postgres.IssueMicroleaseCommand{
		MicroleaseID:               testUUID(seed),
		AccountID:                  accountID,
		AccountScopeKey:            scope,
		ProxyAllocatorOwnerID:      "proxy-a",
		MicroleaseGeneration:       1,
		LeaseFence:                 fmt.Sprintf("fence-%d", seed),
		IssuedCapUSDAtoms:          capAtoms,
		PricingSnapshotID:          fmt.Sprintf("pricing-snapshot-%d", seed),
		PricingSnapshotFingerprint: fmt.Sprintf("pricing-fingerprint-%d", seed),
		PricingPolicyVersion:       "pricing-v1",
		PricingDecisionAt:          now.Add(-time.Second),
		PricingSelectorKey:         "model:gpt-4.1:chat",
		PricingContractVersion:     "pricing-contract-v1",
		FeePolicyVersion:           "fee-v1",
		MicroleasePolicyVersion:    "microlease-v1",
		IssuedAt:                   now,
		DebitCutoffAt:              now.Add(25 * time.Second),
		ExpiresAt:                  now.Add(30 * time.Second),
		IdempotencyRecordID:        testUUID(seed + 1),
		IdempotencyKey:             fmt.Sprintf("idem-issue-%d", seed),
		RequestFingerprint:         fmt.Sprintf("issue-fingerprint-%d", seed),
		StoredOutcomeID:            testUUID(seed + 2),
		LedgerEntryID:              testUUID(seed + 3),
		OutboxID:                   testUUID(seed + 4),
		EventFingerprint:           fmt.Sprintf("issue-event-fingerprint-%d", seed),
		SafeMetadata:               map[string]string{"lag_bucket": "ok"},
	}
}

func performanceTerminalCommand(seed int, microleaseID, accountID, scope string, now time.Time) postgres.TerminalSettlementCommand {
	return postgres.TerminalSettlementCommand{
		InboxID:                    testUUID(seed + 1),
		Topic:                      "billing.microlease.terminal.v1",
		PartitionID:                0,
		OffsetValue:                int64(seed),
		EventID:                    fmt.Sprintf("terminal-%d", seed),
		ProducerIdentity:           "proxy-a",
		EventFingerprint:           fmt.Sprintf("terminal-fingerprint-%d", seed),
		MicroleaseChildDebitID:     testUUID(seed + 2),
		MicroleaseID:               microleaseID,
		DebitAuthorizationID:       fmt.Sprintf("debit-%d", seed),
		AccountID:                  accountID,
		AccountScopeKey:            scope,
		ProxyAllocatorOwnerID:      "proxy-a",
		MicroleaseGeneration:       1,
		ChildSequence:              1,
		ChildCapUSDAtoms:           250,
		ChargedUSDAtoms:            100,
		ReleasedUSDAtoms:           150,
		RequestBasisFingerprint:    fmt.Sprintf("request-fingerprint-%d", seed),
		TerminalBasisFingerprint:   fmt.Sprintf("terminal-basis-fingerprint-%d", seed),
		PricingSnapshotID:          "pricing-snapshot-terminal",
		PricingSnapshotFingerprint: "pricing-fingerprint-terminal",
		TerminalKind:               "finalize",
		TerminalState:              "finalized",
		LedgerEntryID:              testUUID(seed + 3),
		SettlementEffectID:         testUUID(seed + 4),
		OutboxID:                   testUUID(seed + 5),
		OutboxEventFingerprint:     fmt.Sprintf("terminal-outbox-fingerprint-%d", seed),
		TerminalAt:                 now.Add(100 * time.Millisecond),
		SettledAt:                  now.Add(time.Second),
		SafeMetadata:               map[string]string{"terminal_class": "ok"},
	}
}

func performanceCheckpointCommand(seed int, microleaseID, accountID, scope string, now time.Time) postgres.CheckpointCommand {
	return postgres.CheckpointCommand{
		InboxID:                    testUUID(seed + 1),
		Topic:                      "billing.microlease.checkpoint.v1",
		PartitionID:                0,
		OffsetValue:                int64(seed),
		EventID:                    fmt.Sprintf("checkpoint-%d", seed),
		ProducerIdentity:           "proxy-a",
		EventFingerprint:           fmt.Sprintf("checkpoint-event-fingerprint-%d", seed),
		CheckpointID:               testUUID(seed + 2),
		MicroleaseID:               microleaseID,
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
		CheckpointFingerprint:      fmt.Sprintf("checkpoint-fingerprint-%d", seed),
		CreatedAt:                  now.Add(2 * time.Second),
		AppliedAt:                  now.Add(2 * time.Second),
		SafeMetadata:               map[string]string{"checkpoint_kind": "progress"},
	}
}

func performanceCloseCommand(seed int, microleaseID, accountID string, now time.Time) postgres.CloseMicroleaseCommand {
	return postgres.CloseMicroleaseCommand{
		MicroleaseID:        microleaseID,
		AccountID:           accountID,
		IdempotencyRecordID: testUUID(seed + 1),
		IdempotencyKey:      fmt.Sprintf("idem-close-%d", seed),
		RequestFingerprint:  fmt.Sprintf("close-fingerprint-%d", seed),
		StoredOutcomeID:     testUUID(seed + 2),
		LedgerEntryID:       testUUID(seed + 3),
		OutboxID:            testUUID(seed + 4),
		EventFingerprint:    fmt.Sprintf("close-outbox-fingerprint-%d", seed),
		ReleasedUSDAtoms:    750,
		CloseState:          "closed",
		ClosedAt:            now.Add(3 * time.Second),
		Now:                 now.Add(3 * time.Second),
		SafeMetadata:        map[string]string{"close_kind": "proof"},
	}
}

type performanceMicrolease struct {
	ID   string
	Seed int
}
