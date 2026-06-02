package reconciliation

import (
	"testing"
	"time"
)

func TestDecideMicroleaseRepair(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 12, 5, 0, 0, time.UTC)
	policy := Policy{
		StaleChildCriticalAge: 180 * time.Second,
		ReconciliationSLA:     5 * time.Minute,
	}

	expired := DecideMicroleaseRepair(policy, MicroleaseSnapshot{
		MicroleaseID:              "microlease-1",
		State:                     "expired",
		ExpiresAt:                 now.Add(-time.Minute),
		UnresolvedChildCapUSDAtom: 500,
	}, now)
	if !expired.OpenCase || expired.Reason != "stale_microlease" || expired.Severity != "medium" {
		t.Fatalf("expired decision = %+v", expired)
	}

	staleChild := DecideMicroleaseRepair(policy, MicroleaseSnapshot{
		MicroleaseID:              "microlease-2",
		State:                     "active",
		ExpiresAt:                 now.Add(time.Minute),
		UnresolvedChildCapUSDAtom: 100,
		OldestUnresolvedChildAt:   now.Add(-4 * time.Minute),
	}, now)
	if !staleChild.OpenCase || staleChild.Reason != "stale_child_debit" || staleChild.Severity != "high" {
		t.Fatalf("stale child decision = %+v", staleChild)
	}

	healthy := DecideMicroleaseRepair(policy, MicroleaseSnapshot{
		MicroleaseID:              "microlease-3",
		State:                     "active",
		ExpiresAt:                 now.Add(time.Minute),
		UnresolvedChildCapUSDAtom: 0,
		OldestUnresolvedChildAt:   now.Add(-time.Minute),
	}, now)
	if healthy.OpenCase {
		t.Fatalf("healthy decision = %+v, want no case", healthy)
	}
}
