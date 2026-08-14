package postgresjobs

import (
	"context"
	"fmt"
	"time"
)

func (e *Engine) observe(ctx context.Context) error {
	e.mu.Lock()
	due := e.lastObservation.IsZero() || time.Since(e.lastObservation) >= e.config.ObservationInterval
	e.mu.Unlock()
	if !due {
		return nil
	}
	observation, err := e.store.Observe(ctx, e.registry.Keys())
	if err != nil {
		e.mu.Lock()
		e.lastObservation = time.Time{}
		e.mu.Unlock()
		e.telemetry.MarkStale()
		return fmt.Errorf("observe jobs: %w", err)
	}
	e.mu.Lock()
	e.lastObservation = observation.ObservedAt
	if !observation.Compatible {
		e.compatible = false
		e.closeAdmissionLocked()
	}
	facts := EngineFacts{ClaimAdmissionOpen: e.admission, Compatible: e.compatible, InFlight: len(e.inflight), Capacity: e.config.MaxConcurrency, ObservationFresh: true}
	e.mu.Unlock()
	e.telemetry.Update(observation, facts)
	return nil
}
