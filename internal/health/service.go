// Package health owns readiness aggregation and the drain signal.
//
// Readiness is refreshed by a background loop and served from the last observed
// result. Evaluating probes per request looks simpler, but it makes the probe
// route consume the dependency capacity it reports on: a pooled database ping
// needs a pool connection, so a saturated pool fails readiness, the orchestrator
// evicts the instance, and its traffic moves to instances already saturated.
//
// Serving from cache means something has to keep the cache honest, which is what
// the staleness bound on Cached is for.
package health

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
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
// The period is the larger of the two inputs, not the interval: Refresh is called
// serially, so a probe budget above the interval makes the loop run at the
// budget's pace, and sizing off the interval alone would expire a verdict that is
// being refreshed as fast as it can be.
func staleBudget(interval, probeBudget time.Duration) time.Duration {
	return probeBudget + staleRefreshMultiplier*max(interval, probeBudget)
}

// readinessState is the immutable result of one refresh. consecutiveFailures
// counts failed evaluations since the last healthy one, so a single slow
// round-trip cannot evict an instance that is still serving.
type readinessState struct {
	err                 error
	consecutiveFailures int
	evaluatedAt         time.Time
}

type readinessTransitions struct {
	mu          sync.Mutex
	previousErr error
	notify      func(error)
}

func (t *readinessTransitions) publish(currentErr error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.notify != nil && (t.previousErr == nil) != (currentErr == nil) {
		t.notify(currentErr)
	}
	t.previousErr = currentErr
}

func (t *readinessTransitions) publishStale(service *Service, state *readinessState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if service.draining.Load() || service.state.Load() != state || t.previousErr != nil {
		return
	}
	err := service.staleness(state)
	if !errors.Is(err, ErrStale) {
		return
	}
	t.previousErr = err
	t.notify(err)
}

func (s *Service) armStaleness(state *readinessState, transitions *readinessTransitions) *time.Timer {
	if transitions.notify == nil || state.err != nil || s.draining.Load() {
		return nil
	}
	staleAfter := time.Duration(s.staleAfter.Load())
	delay := max(time.Until(state.evaluatedAt.Add(staleAfter))+time.Nanosecond, time.Nanosecond)
	return time.AfterFunc(delay, func() {
		transitions.publishStale(s, state)
	})
}

func New(probes ...Probe) *Service {
	return &Service{probes: slices.Clone(probes)}
}

// Cached reports the most recently observed readiness without touching a
// dependency. The drain flag is read first so StartDrain takes effect on the
// next request rather than after the next refresh interval.
//
// A verdict older than the refresher's cadence is refused rather than served: a
// refresher that stopped leaves its last verdict standing forever, and since the
// last thing a healthy service writes is "healthy", that frozen answer is the
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
// If Watch stops before drain, one final callback may report ErrStale when the
// last healthy verdict expires. StartDrain suppresses that deferred transition.
//
// The first evaluation runs immediately unless startup admission already seeded
// the cache. failureThreshold applies only to the healthy-to-unhealthy
// transition; a service that has never been healthy reports the failure at once.
//
// probeBudget bounds one evaluation and is separate from interval so a configured
// probe timeout is not clamped to the refresh period, which would let a
// dependency pass startup admission and then flap out of rotation. Evaluations
// are serial, and the staleness budget uses the larger of the interval and probe
// budget when a probe is slower than the requested cadence.
func (s *Service) Watch(
	ctx context.Context,
	interval, probeBudget time.Duration,
	failureThreshold int,
	onTransition func(error),
) error {
	if interval <= 0 {
		return errors.New("health watch: interval must be > 0")
	}
	if probeBudget <= 0 {
		return errors.New("health watch: probe budget must be > 0")
	}
	if failureThreshold < 1 {
		return errors.New("health watch: failure threshold must be >= 1")
	}
	if ctx.Err() != nil {
		return nil
	}

	// Published before the first evaluation, so a refresher that dies on its very
	// first pass still leaves Cached able to refuse the verdict it never wrote.
	// Never cleared: this function returning is one of the ways the refresher
	// stops, so clearing it on the way out would disarm the guard exactly when it
	// is needed.
	s.staleAfter.Store(int64(staleBudget(interval, probeBudget)))

	if s.state.Load() == nil {
		_ = s.Refresh(ctx, probeBudget, failureThreshold)
	}
	state := s.state.Load()
	transitions := readinessTransitions{previousErr: state.err, notify: onTransition}
	staleTimer := s.armStaleness(state, &transitions)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if s.draining.Load() && staleTimer != nil {
				staleTimer.Stop()
			}
			return nil
		case <-ticker.C:
			_ = s.Refresh(ctx, probeBudget, failureThreshold)
			if staleTimer != nil {
				staleTimer.Stop()
			}
			state = s.state.Load()
			currentErr := state.err
			if s.draining.Load() {
				currentErr = ErrDraining
			}
			transitions.publish(currentErr)
			staleTimer = s.armStaleness(state, &transitions)
		}
	}
}

// Refresh performs one evaluation, folds it into the cached state, and returns
// what it observed.
//
// Startup admission calls it directly: it needs both the verdict, to decide
// whether to admit traffic at all, and the seeded cache, so the first probe after
// admission is answered from a real evaluation rather than ErrNotEvaluated.
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
		if previous.err == nil && failures < failureThreshold {
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
