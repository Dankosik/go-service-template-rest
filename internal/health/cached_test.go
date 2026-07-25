package health

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

const (
	testRefreshInterval  = 2 * time.Second
	testFailureThreshold = 3
)

func TestCachedFailsClosedBeforeFirstEvaluation(t *testing.T) {
	t.Parallel()

	svc := New(fakeProbe{name: "db"})

	if err := svc.Cached(); !errors.Is(err, ErrNotEvaluated) {
		t.Fatalf("Cached() error = %v, want ErrNotEvaluated", err)
	}
}

// TestCachedServesWithoutTouchingProbes is the property the whole change exists
// for: the probe route must not consume dependency capacity per request.
func TestCachedServesWithoutTouchingProbes(t *testing.T) {
	t.Parallel()

	probe := &countingProbe{name: "db"}
	svc := New(probe)
	_ = svc.Refresh(context.Background(), testRefreshInterval, testFailureThreshold)

	if got := probe.calls.Load(); got != 1 {
		t.Fatalf("probe calls after one refresh = %d, want 1", got)
	}
	for range 100 {
		if err := svc.Cached(); err != nil {
			t.Fatalf("Cached() error = %v", err)
		}
	}
	if got := probe.calls.Load(); got != 1 {
		t.Fatalf("probe calls after 100 Cached() reads = %d, want 1", got)
	}
}

func TestCachedHoldsHealthyUntilFailureThreshold(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	probe := &countingProbe{name: "db"}
	svc := New(probe)

	_ = svc.Refresh(context.Background(), testRefreshInterval, testFailureThreshold)
	if err := svc.Cached(); err != nil {
		t.Fatalf("Cached() after healthy refresh error = %v", err)
	}

	probe.err.Store(&downErr)
	for attempt := 1; attempt < testFailureThreshold; attempt++ {
		_ = svc.Refresh(context.Background(), testRefreshInterval, testFailureThreshold)
		if err := svc.Cached(); err != nil {
			t.Fatalf("Cached() after %d/%d failures error = %v, want nil", attempt, testFailureThreshold, err)
		}
	}

	_ = svc.Refresh(context.Background(), testRefreshInterval, testFailureThreshold)
	if err := svc.Cached(); !errors.Is(err, downErr) {
		t.Fatalf("Cached() at threshold error = %v, want wrapped %v", err, downErr)
	}
}

// TestCachedReportsFirstFailureImmediately keeps the threshold from granting a
// grace period to an instance that was never healthy in the first place.
func TestCachedReportsFirstFailureImmediately(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	svc := New(fakeProbe{name: "db", err: downErr})

	_ = svc.Refresh(context.Background(), testRefreshInterval, testFailureThreshold)

	if err := svc.Cached(); !errors.Is(err, downErr) {
		t.Fatalf("Cached() error = %v, want wrapped %v", err, downErr)
	}
}

func TestCachedRecoversAfterHealthyRefresh(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	probe := &countingProbe{name: "db"}
	probe.err.Store(&downErr)
	svc := New(probe)

	_ = svc.Refresh(context.Background(), testRefreshInterval, testFailureThreshold)
	if err := svc.Cached(); err == nil {
		t.Fatal("Cached() error = nil, want failure")
	}

	probe.err.Store(nil)
	_ = svc.Refresh(context.Background(), testRefreshInterval, testFailureThreshold)
	if err := svc.Cached(); err != nil {
		t.Fatalf("Cached() after recovery error = %v", err)
	}
}

// TestCachedDrainWinsOverFreshState keeps StartDrain effective on the next
// request rather than after the next refresh interval.
func TestCachedDrainWinsOverFreshState(t *testing.T) {
	t.Parallel()

	svc := New(fakeProbe{name: "db"})
	_ = svc.Refresh(context.Background(), testRefreshInterval, testFailureThreshold)
	svc.StartDrain()

	if err := svc.Cached(); !errors.Is(err, ErrDraining) {
		t.Fatalf("Cached() error = %v, want ErrDraining", err)
	}
}

func TestWatchRefreshesOnIntervalAndStopsOnCancel(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		probe := &countingProbe{name: "db"}
		svc := New(probe)

		ctx, cancel := context.WithCancel(context.Background())
		watchErr := make(chan error, 1)
		go func() { watchErr <- svc.Watch(ctx, testRefreshInterval, testFailureThreshold) }()

		synctest.Wait()
		if got := probe.calls.Load(); got != 1 {
			t.Fatalf("probe calls before first tick = %d, want 1 immediate evaluation", got)
		}

		time.Sleep(3 * testRefreshInterval)
		synctest.Wait()
		if got := probe.calls.Load(); got != 4 {
			t.Fatalf("probe calls after 3 intervals = %d, want 4", got)
		}

		cancel()
		synctest.Wait()
		if err := <-watchErr; err != nil {
			t.Fatalf("Watch() error = %v, want nil on cancellation", err)
		}

		observed := probe.calls.Load()
		time.Sleep(3 * testRefreshInterval)
		synctest.Wait()
		if got := probe.calls.Load(); got != observed {
			t.Fatalf("probe calls after cancellation = %d, want %d", got, observed)
		}
	})
}

// TestWatchBoundsEachEvaluationByInterval stops refreshes from piling up behind
// a probe that outlives its own refresh period.
func TestWatchBoundsEachEvaluationByInterval(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		svc := New(blockingProbe{name: "db"})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() { _ = svc.Watch(ctx, testRefreshInterval, 1) }()

		time.Sleep(testRefreshInterval + time.Millisecond)
		synctest.Wait()

		err := svc.Cached()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Cached() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

func TestWatchRejectsUnusableSettings(t *testing.T) {
	t.Parallel()

	svc := New(fakeProbe{name: "db"})

	if err := svc.Watch(context.Background(), 0, 1); err == nil {
		t.Fatal("Watch(interval=0) error = nil, want non-nil")
	}
	if err := svc.Watch(context.Background(), testRefreshInterval, 0); err == nil {
		t.Fatal("Watch(threshold=0) error = nil, want non-nil")
	}
}

func TestWatchReturnsImmediatelyOnCanceledContext(t *testing.T) {
	t.Parallel()

	probe := &countingProbe{name: "db"}
	svc := New(probe)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := svc.Watch(ctx, testRefreshInterval, testFailureThreshold); err != nil {
		t.Fatalf("Watch() error = %v, want nil", err)
	}
	if got := probe.calls.Load(); got != 0 {
		t.Fatalf("probe calls = %d, want 0 for an already-canceled context", got)
	}
}

type countingProbe struct {
	name  string
	calls atomic.Int64
	err   atomic.Pointer[error]
}

func (p *countingProbe) Name() string {
	return p.name
}

func (p *countingProbe) Check(context.Context) error {
	p.calls.Add(1)
	if err := p.err.Load(); err != nil {
		return *err
	}
	return nil
}

type blockingProbe struct {
	name string
}

func (p blockingProbe) Name() string {
	return p.name
}

func (p blockingProbe) Check(ctx context.Context) error {
	<-ctx.Done()
	return fmt.Errorf("blocking probe: %w", ctx.Err())
}
