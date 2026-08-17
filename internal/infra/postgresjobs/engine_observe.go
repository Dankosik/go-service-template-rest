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
	observedLocallyAt := time.Now()
	e.mu.Lock()
	e.lastObservation = observedLocallyAt
	if !observation.Compatible {
		e.compatible = false
		e.closeAdmissionLocked()
	}
	facts := e.factsLocked(observedLocallyAt)
	e.mu.Unlock()
	e.telemetry.Update(observation, facts, observedLocallyAt.Add(e.config.ObservationMaxAge))
	return nil
}
