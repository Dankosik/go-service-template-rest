package postgresjobs

import (
	"context"
	"time"
)

func (e *Engine) renew(ctx context.Context) error {
	e.mu.Lock()
	if e.lastLease.IsZero() || time.Since(e.lastLease) < e.config.LeaseDuration/3 || len(e.inflight) == 0 {
		e.mu.Unlock()
		return nil
	}
	attempts := make([]AttemptIdentity, 0, len(e.inflight))
	for attempt := range e.inflight {
		attempts = append(attempts, attempt)
	}
	e.mu.Unlock()

	renewals, err := e.store.Renew(ctx, attempts, e.config.LeaseDuration)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	seen := make(map[AttemptIdentity]Renewal, len(renewals))
	for _, renewal := range renewals {
		seen[renewal.Attempt] = renewal
		if renewal.ObservedAt.After(e.lastLease) {
			e.lastLease = renewal.ObservedAt
		}
		if renewal.CancelRequested {
			if cancel := e.inflight[renewal.Attempt]; cancel != nil {
				cancel()
			}
		}
	}
	for _, attempt := range attempts {
		if _, ok := seen[attempt]; !ok {
			if cancel := e.inflight[attempt]; cancel != nil {
				cancel()
			}
		}
	}
	e.telemetry.RecordRenewal(ctx)
	return nil
}
