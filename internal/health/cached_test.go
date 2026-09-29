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

	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: 1}, fakeProbe{name: "db"})

	if err := svc.Cached(); !errors.Is(err, ErrNotEvaluated) {
		t.Fatalf("Cached() error = %v, want ErrNotEvaluated", err)
	}
}

// TestCachedServesWithoutTouchingProbes is the property the whole change exists
// for: the probe route must not consume dependency capacity per request.
func TestCachedServesWithoutTouchingProbes(t *testing.T) {
	t.Parallel()

	probe := &countingProbe{name: "db"}
	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, probe)
	_ = svc.Refresh(context.Background())

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

func TestWatchReusesReadinessSeededByStartupAdmission(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		probe := &countingProbe{name: "db"}
		svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: 1}, probe)
		if err := svc.Refresh(t.Context()); err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() {
			done <- svc.Watch(ctx, time.Hour, nil)
		}()
		synctest.Wait()
		if got := probe.calls.Load(); got != 1 {
			t.Fatalf("probe calls before first tick = %d, want seeded evaluation only", got)
		}

		cancel()
		if err := <-done; err != nil {
			t.Fatalf("Watch() error = %v", err)
		}
	})
}

func TestCachedHoldsHealthyUntilFailureThreshold(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	probe := &countingProbe{name: "db"}
	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, probe)

	_ = svc.Refresh(context.Background())
	if err := svc.Cached(); err != nil {
		t.Fatalf("Cached() after healthy refresh error = %v", err)
	}

	probe.err.Store(&downErr)
	for attempt := 1; attempt < testFailureThreshold; attempt++ {
		_ = svc.Refresh(context.Background())
		if err := svc.Cached(); err != nil {
			t.Fatalf("Cached() after %d/%d failures error = %v, want nil", attempt, testFailureThreshold, err)
		}
	}

	_ = svc.Refresh(context.Background())
	if err := svc.Cached(); !errors.Is(err, downErr) {
		t.Fatalf("Cached() at threshold error = %v, want wrapped %v", err, downErr)
	}
}

// TestCachedReportsFirstFailureImmediately keeps the threshold from granting a
// grace period to an instance that was never healthy in the first place.
func TestCachedReportsFirstFailureImmediately(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, fakeProbe{name: "db", err: downErr})

	_ = svc.Refresh(context.Background())

	if err := svc.Cached(); !errors.Is(err, downErr) {
		t.Fatalf("Cached() error = %v, want wrapped %v", err, downErr)
	}
}

func TestCachedReportsNewestFailureWhileAlreadyUnhealthy(t *testing.T) {
	t.Parallel()

	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")
	probe := &countingProbe{name: "db"}
	probe.err.Store(&firstErr)
	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, probe)

	_ = svc.Refresh(context.Background())
	probe.err.Store(&secondErr)
	_ = svc.Refresh(context.Background())

	if err := svc.Cached(); !errors.Is(err, secondErr) {
		t.Fatalf("Cached() error = %v, want newest wrapped %v", err, secondErr)
	}
}

func TestCachedRecoversAfterHealthyRefresh(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	probe := &countingProbe{name: "db"}
	probe.err.Store(&downErr)
	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, probe)

	_ = svc.Refresh(context.Background())
	if err := svc.Cached(); err == nil {
		t.Fatal("Cached() error = nil, want failure")
	}

	probe.err.Store(nil)
	_ = svc.Refresh(context.Background())
	if err := svc.Cached(); err != nil {
		t.Fatalf("Cached() after recovery error = %v", err)
	}
}

// TestCachedDrainWinsOverFreshState keeps StartDrain effective on the next
// request rather than after the next refresh interval.
func TestCachedDrainWinsOverFreshState(t *testing.T) {
	t.Parallel()

	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, fakeProbe{name: "db"})
	_ = svc.Refresh(context.Background())
	svc.StartDrain()

	if err := svc.Cached(); !errors.Is(err, ErrDraining) {
		t.Fatalf("Cached() error = %v, want ErrDraining", err)
	}
}

func TestWatchRefreshesOnIntervalAndStopsOnCancel(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		probe := &countingProbe{name: "db"}
		svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, probe)

		ctx, cancel := context.WithCancel(context.Background())
		watchErr := make(chan error, 1)
		go func() { watchErr <- svc.Watch(ctx, testRefreshInterval, nil) }()

		synctest.Wait()
		if got := probe.calls.Load(); got != 1 {
			t.Fatalf("probe calls before first tick = %d, want 1 immediate evaluation", got)
		}

		synctest.Sleep(3 * testRefreshInterval)
		if got := probe.calls.Load(); got != 4 {
			t.Fatalf("probe calls after 3 intervals = %d, want 4", got)
		}

		cancel()
		synctest.Wait()
		if err := <-watchErr; err != nil {
			t.Fatalf("Watch() error = %v, want nil on cancellation", err)
		}

		observed := probe.calls.Load()
		synctest.Sleep(3 * testRefreshInterval)
		if got := probe.calls.Load(); got != observed {
			t.Fatalf("probe calls after cancellation = %d, want %d", got, observed)
		}
	})
}

// TestWatchBoundsEachEvaluationByProbeBudget stops refreshes from piling up
// behind a probe that outlives its own budget.
func TestWatchBoundsEachEvaluationByProbeBudget(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: 1}, blockingProbe{name: "db"})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() { _ = svc.Watch(ctx, testRefreshInterval, nil) }()

		synctest.Sleep(testProbeBudget + time.Millisecond)

		err := svc.Cached()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Cached() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

// TestWatchSpendsTheProbeBudgetNotTheInterval keeps a probe budget larger than
// the refresh period from being clamped to the period, which would let a
// dependency that passed startup admission flap in steady state.
func TestWatchSpendsTheProbeBudgetNotTheInterval(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		const (
			interval    = 500 * time.Millisecond
			probeBudget = 4 * time.Second
		)
		probe := &deadlineProbe{}
		svc := mustNew(t, Policy{ProbeBudget: probeBudget, FailureThreshold: 1}, probe)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() { _ = svc.Watch(ctx, interval, nil) }()
		synctest.Wait()

		if got := probe.budget.Load(); time.Duration(got) != probeBudget {
			t.Fatalf("probe budget = %s, want the configured %s", time.Duration(got), probeBudget)
		}
	})
}

// TestCachedRefusesAStaleVerdict is the second half of the readiness fix: even
// when the refresher stops for a reason this package cannot see, the verdict it
// left behind must expire rather than be served for the life of the process.
func TestCachedRefusesAStaleVerdict(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		probe := &countingProbe{name: "db"}
		svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, probe)
		transitions := make(chan error, 1)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			done <- svc.Watch(ctx, testRefreshInterval, func(err error) {
				transitions <- err
			})
		}()
		synctest.Wait()
		if err := svc.Cached(); err != nil {
			t.Fatalf("Cached() while refreshing error = %v, want nil", err)
		}

		// The refresher stops without the drain flag ever being set, which is what
		// an unrelated failure or a panic in the supervisor looks like from here.
		cancel()
		if err := <-done; err != nil {
			t.Fatalf("Watch() error = %v", err)
		}

		synctest.Sleep(staleBudget(testRefreshInterval, testProbeBudget) + time.Second)
		synctest.Wait()

		if err := <-transitions; !errors.Is(err, ErrStale) {
			t.Fatalf("stale transition error = %v, want ErrStale", err)
		}

		if err := svc.Cached(); !errors.Is(err, ErrStale) {
			t.Fatalf("Cached() after the refresher stopped error = %v, want ErrStale", err)
		}
	})
}

func TestNewRejectsUnusablePolicy(t *testing.T) {
	t.Parallel()

	if _, err := New(Policy{FailureThreshold: 1}, fakeProbe{name: "db"}); err == nil {
		t.Fatal("New(probeBudget=0) error = nil, want non-nil")
	}
	if _, err := New(Policy{ProbeBudget: testProbeBudget}, fakeProbe{name: "db"}); err == nil {
		t.Fatal("New(threshold=0) error = nil, want non-nil")
	}
}

func TestWatchRejectsUnusableInterval(t *testing.T) {
	t.Parallel()

	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: 1}, fakeProbe{name: "db"})

	if err := svc.Watch(context.Background(), 0, nil); err == nil {
		t.Fatal("Watch(interval=0) error = nil, want non-nil")
	}
}

func TestWatchReturnsImmediatelyOnCanceledContext(t *testing.T) {
	t.Parallel()

	probe := &countingProbe{name: "db"}
	svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, probe)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := svc.Watch(ctx, testRefreshInterval, nil); err != nil {
		t.Fatalf("Watch() error = %v, want nil", err)
	}
	if got := probe.calls.Load(); got != 0 {
		t.Fatalf("probe calls = %d, want 0 for an already-canceled context", got)
	}
}

func TestWatchReportsOnlyEffectiveReadinessTransitions(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		downErr := errors.New("down")
		probe := &countingProbe{name: "db"}
		svc := mustNew(t, Policy{ProbeBudget: testProbeBudget, FailureThreshold: testFailureThreshold}, probe)
		transitions := make(chan error, 2)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			_ = svc.Watch(ctx, testRefreshInterval, func(err error) {
				transitions <- err
			})
		}()
		synctest.Wait()
		if got := len(transitions); got != 0 {
			t.Fatalf("initial transition count = %d, want 0", got)
		}

		probe.err.Store(&downErr)
		synctest.Sleep(2 * testRefreshInterval)
		if got := len(transitions); got != 0 {
			t.Fatalf("transition count below failure threshold = %d, want 0", got)
		}

		synctest.Sleep(testRefreshInterval)
		if err := <-transitions; !errors.Is(err, downErr) {
			t.Fatalf("unhealthy transition error = %v, want wrapped %v", err, downErr)
		}

		synctest.Sleep(testRefreshInterval)
		if got := len(transitions); got != 0 {
			t.Fatalf("duplicate unhealthy transition count = %d, want 0", got)
		}

		probe.err.Store(nil)
		synctest.Sleep(testRefreshInterval)
		if err := <-transitions; err != nil {
			t.Fatalf("healthy transition error = %v, want nil", err)
		}
	})
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

// deadlineProbe records the budget the evaluation handed it, which is what
// distinguishes a configured probe budget from the refresh interval.
type deadlineProbe struct {
	budget atomic.Int64
}

func (p *deadlineProbe) Name() string { return "deadline" }

func (p *deadlineProbe) Check(ctx context.Context) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return errors.New("evaluation ran without a budget")
	}
	p.budget.Store(int64(time.Until(deadline)))
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
