// Package health owns readiness aggregation and the drain signal.
//
// Readiness is refreshed by a background loop and served from the last observed
// result. Evaluating probes per request looks simpler, but it makes the probe
// route consume the dependency capacity it is reporting on: a pooled database
// ping needs a pool connection, so a saturated pool fails readiness, the
// orchestrator evicts the instance, its traffic moves to instances that are
// already saturated, and a slow dependency becomes a fleet-wide eviction.
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
)

// readinessState is the immutable result of one refresh. consecutiveFailures
// counts failed evaluations since the last healthy one so a single slow
// round-trip cannot evict an instance that is still serving.
type readinessState struct {
	err                 error
	consecutiveFailures int
}

func New(probes ...Probe) *Service {
	return &Service{probes: slices.Clone(probes)}
}

// Cached reports the most recently observed readiness without touching a
// dependency. The drain flag is read first so StartDrain takes effect on the
// next request rather than after the next refresh interval.
func (s *Service) Cached() error {
	if s.draining.Load() {
		return ErrDraining
	}
	state := s.state.Load()
	if state == nil {
		return ErrNotEvaluated
	}
	return state.err
}

// Watch refreshes cached readiness every interval until ctx is done, and returns
// nil on cancellation so it can be supervised as an ordinary background task.
//
// The first evaluation runs immediately: a service that has just been admitted
// must not report ErrNotEvaluated for a whole interval. failureThreshold applies
// only to the healthy-to-unhealthy transition; a service that has never been
// healthy reports the failure at once.
func (s *Service) Watch(ctx context.Context, interval time.Duration, failureThreshold int) error {
	if interval <= 0 {
		return fmt.Errorf("health watch: interval must be > 0")
	}
	if failureThreshold < 1 {
		return fmt.Errorf("health watch: failure threshold must be >= 1")
	}
	if ctx.Err() != nil {
		return nil
	}

	_ = s.Refresh(ctx, interval, failureThreshold)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			_ = s.Refresh(ctx, interval, failureThreshold)
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
// probeBudget bounds the evaluation. Watch passes its own interval, so a probe
// that outlives its refresh period cannot let refreshes pile up behind a hung
// dependency; admission passes the readiness budget it was given.
func (s *Service) Refresh(ctx context.Context, probeBudget time.Duration, failureThreshold int) error {
	probeCtx, cancel := context.WithTimeout(ctx, probeBudget)
	defer cancel()

	err := s.evaluate(probeCtx)
	if err == nil {
		s.state.Store(&readinessState{})
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
	s.state.Store(&readinessState{err: reported, consecutiveFailures: failures})
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
