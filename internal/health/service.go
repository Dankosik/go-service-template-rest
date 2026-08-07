// Package health owns readiness aggregation and the drain signal.
//
// Readiness is refreshed by a background loop and served from the last observed
// result. Evaluating probes per request looks simpler, but it makes the probe
// route consume the dependency capacity it is reporting on: a pooled database
// ping needs a pool connection, so a saturated pool fails readiness, the
// orchestrator evicts the instance, its traffic moves to instances that are
// already saturated, and a slow dependency becomes a fleet-wide eviction.
//
// Serving from cache means something has to keep the cache honest, which is what
// the staleness bound on Cached is for.
package health

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"time"
)

type Service struct {
	probes   []Probe
	draining atomic.Bool
	state    atomic.Pointer[readinessState]
	// staleAfter is how old a cached verdict may be before Cached refuses it,
	// in nanoseconds. Watch publishes it because Watch owns the cadence; zero,
	// the value before any refresher is running, disables the guard.
	staleAfter atomic.Int64
}

type Probe interface {
	Name() string
	Check(ctx context.Context) error
}

var (
	ErrDraining = errors.New("service is draining")

	// ErrNotEvaluated reports that no refresh has completed yet. Readiness must
	// fail closed until a probe has actually answered.
	ErrNotEvaluated = errors.New("readiness has not been evaluated yet")

	// ErrStale reports that the cached verdict is older than the refresher's own
	// cadence allows, which means nothing is refreshing it any more.
	ErrStale = errors.New("readiness verdict is stale")
)

// staleRefreshMultiplier is how many refresh periods a verdict may age through
// before it is refused. It is greater than one so an ordinary missed tick or a
// slow evaluation does not flip readiness, and finite so a dead refresher cannot
// leave a verdict standing for the life of the process.
const staleRefreshMultiplier = 3

// staleBudget is how old a verdict may get before Cached refuses it.
//
// The period is the larger of the two inputs, not the interval. A probe budget
// above the refresh interval makes the loop run at the budget's pace instead —
// Refresh is called serially, so a slow evaluation delays the next tick rather
// than stacking up — and sizing the bound off the interval alone would then
// expire a verdict that is being refreshed as fast as it can be. That reads as a
// dead refresher when the dependency is merely slow, which is the false eviction
// this whole package is built to avoid.
func staleBudget(interval, probeBudget time.Duration) time.Duration {
	return probeBudget + staleRefreshMultiplier*max(interval, probeBudget)
}

// readinessState is the immutable result of one refresh. consecutiveFailures
// counts failed evaluations since the last healthy one so a single slow
// round-trip cannot evict an instance that is still serving. evaluatedAt is when
// the probes last actually ran, which is what Cached measures staleness against.
type readinessState struct {
	err                 error
	consecutiveFailures int
	evaluatedAt         time.Time
}

func New(probes ...Probe) *Service {
	return &Service{probes: slices.Clone(probes)}
}

// Cached reports the most recently observed readiness without touching a
// dependency. The drain flag is read first so StartDrain takes effect on the
// next request rather than after the next refresh interval.
//
// A verdict older than the refresher's cadence is refused rather than served.
// Without that bound, a refresher that stopped — cancelled, panicked, or wedged on
// a dependency that never returns — leaves its last verdict standing forever while
// nothing checks anything. The failure is silent by construction, because the last
// thing a healthy service writes is "healthy", so the frozen answer is the
// reassuring one. This check turns it into an eviction.
func (s *Service) Cached() error {
	if s.draining.Load() {
		return ErrDraining
	}
	state := s.state.Load()
	if state == nil {
		return ErrNotEvaluated
	}
	if err := s.staleness(state); err != nil {
		return err
	}
	return state.err
}

func (s *Service) staleness(state *readinessState) error {
	staleAfter := time.Duration(s.staleAfter.Load())
	if staleAfter <= 0 || state.evaluatedAt.IsZero() {
		return nil
	}
	age := time.Since(state.evaluatedAt)
	if age <= staleAfter {
		return nil
	}
	return fmt.Errorf("%w: last evaluated %s ago, budget %s", ErrStale, age.Round(time.Millisecond), staleAfter)
}

// Watch refreshes cached readiness every interval until ctx is done, and returns
// nil on cancellation so it can be supervised as an ordinary background task.
// onTransition receives the cached failure on healthy-to-unhealthy changes and
// nil on recovery; the initial evaluation is not a transition.
//
// The first evaluation runs immediately: a service that has just been admitted
// must not report ErrNotEvaluated for a whole interval. failureThreshold applies
// only to the healthy-to-unhealthy transition; a service that has never been
// healthy reports the failure at once.
//
// probeBudget bounds one evaluation and is passed separately from interval. The
// two used to be one value, which clamped every configured probe timeout to the
// refresh period: a 3s postgres.healthcheck_timeout became a 2s budget in steady
// state while startup admission still granted the full 3s, so the same dependency
// could pass admission and then flap out of rotation. Configuration
// cross-validation keeps interval above probeBudget so evaluations cannot pile up
// behind a slow dependency.
func (s *Service) Watch(
	ctx context.Context,
	interval, probeBudget time.Duration,
	failureThreshold int,
	onTransition func(error),
) error {
	if interval <= 0 {
		return fmt.Errorf("health watch: interval must be > 0")
	}
	if probeBudget <= 0 {
		return fmt.Errorf("health watch: probe budget must be > 0")
	}
	if failureThreshold < 1 {
		return fmt.Errorf("health watch: failure threshold must be >= 1")
	}
	if ctx.Err() != nil {
		return nil
	}

	// Published before the first evaluation, so a refresher that dies on its very
	// first pass still leaves Cached able to refuse the verdict it never wrote.
	// It is deliberately never cleared: this function returning is one of the
	// ways the refresher stops, so clearing it on the way out would disarm the
	// guard exactly when it is needed. A drain is already reported by StartDrain,
	// which Cached reads first.
	s.staleAfter.Store(int64(staleBudget(interval, probeBudget)))

	_ = s.Refresh(ctx, probeBudget, failureThreshold)
	previousErr := s.state.Load().err

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			_ = s.Refresh(ctx, probeBudget, failureThreshold)
			currentErr := s.state.Load().err
			if onTransition != nil && (previousErr == nil) != (currentErr == nil) {
				onTransition(currentErr)
			}
			previousErr = currentErr
		}
	}
}

// Refresh performs one evaluation, folds it into the cached state, and returns
// what it observed.
//
// Startup admission calls it directly: it needs both the verdict, to decide
// whether to admit traffic at all, and the seeded cache, so the first probe after
// admission is answered from a real evaluation rather than reporting
// ErrNotEvaluated for a whole refresh interval.
//
// probeBudget bounds the evaluation. Watch passes the configured probe budget;
// admission passes the readiness budget it was given.
func (s *Service) Refresh(ctx context.Context, probeBudget time.Duration, failureThreshold int) error {
	probeCtx, cancel := context.WithTimeout(ctx, probeBudget)
	defer cancel()

	err := s.evaluate(probeCtx)
	evaluatedAt := time.Now()
	if err == nil {
		s.state.Store(&readinessState{evaluatedAt: evaluatedAt})
		return nil
	}

	failures := 1
	reported := err
	if previous := s.state.Load(); previous != nil {
		failures = previous.consecutiveFailures + 1
		// Hold the previous verdict until the streak reaches the threshold. A
		// previously healthy instance stays in rotation through a blip; one that
		// was already failing keeps reporting the newest cause. A service that
		// has never been healthy has no previous verdict and fails immediately.
		if failures < failureThreshold {
			reported = previous.err
		}
	}
	s.state.Store(&readinessState{err: reported, consecutiveFailures: failures, evaluatedAt: evaluatedAt})
	return err
}

func (s *Service) evaluate(ctx context.Context) error {
	for _, probe := range s.probes {
		if err := probe.Check(ctx); err != nil {
			return fmt.Errorf("%s probe failed: %w", probe.Name(), err)
		}
	}
	return nil
}

func (s *Service) StartDrain() {
	s.draining.Store(true)
}
