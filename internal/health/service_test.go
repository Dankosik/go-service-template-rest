package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

const testProbeBudget = time.Second

func TestServiceRefreshSuccess(t *testing.T) {
	t.Parallel()

	db := fakeProbe{name: "db"}
	cache := fakeProbe{name: "cache"}

	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: 1}, db, cache)

	if err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
}

func TestServiceRefreshFail(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	db := fakeProbe{name: "db", err: downErr}

	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: 1}, db)

	err := svc.Refresh(context.Background())
	if err == nil {
		t.Fatal("Refresh() expected error")
	}

	if !errors.Is(err, downErr) {
		t.Fatalf("Refresh() error = %v, want wrapped %v", err, downErr)
	}
}

// TestServiceRefreshNamesTheFailingProbe keeps a multi-probe service from
// reporting "not ready" without saying which dependency answered that way.
func TestServiceRefreshNamesTheFailingProbe(t *testing.T) {
	t.Parallel()

	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: 1},
		fakeProbe{name: "db"},
		fakeProbe{name: "cache", err: errors.New("down")},
	)

	err := svc.Refresh(context.Background())
	if err == nil {
		t.Fatal("Refresh() error = nil, want a failure")
	}
	if got := err.Error(); got[:len("cache probe failed")] != "cache probe failed" {
		t.Fatalf("Refresh() error = %q, want it to name the cache probe first", got)
	}
}

// TestServiceRefreshReportsObservedFailureRegardlessOfThreshold keeps admission
// honest: the threshold exists to protect an instance that was already healthy,
// not to grant a grace period to one that has never passed a probe.
func TestServiceRefreshReportsObservedFailureRegardlessOfThreshold(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: 100}, fakeProbe{name: "db", err: downErr})

	if err := svc.Refresh(context.Background()); !errors.Is(err, downErr) {
		t.Fatalf("Refresh() error = %v, want wrapped %v", err, downErr)
	}
}

func mustNew(t *testing.T, policy Policy, probes ...Probe) *Service {
	t.Helper()
	svc, err := New(policy, probes...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return svc
}

type fakeProbe struct {
	name string
	err  error
}

func (p fakeProbe) Name() string {
	return p.name
}

func (p fakeProbe) Check(context.Context) error {
	return p.err
}
