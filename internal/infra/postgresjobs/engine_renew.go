package postgresjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func (e *Engine) renew(ctx context.Context) error {
	e.mu.Lock()
	now := time.Now()
	attempts := make([]AttemptIdentity, 0, len(e.inflight))
	for identity, attempt := range e.inflight {
		if !now.Before(attempt.renewAt) {
			attempts = append(attempts, identity)
		}
	}
	if len(attempts) == 0 {
		e.mu.Unlock()
		return nil
	}
	e.mu.Unlock()

	renewStartedAt := time.Now()
	renewals, err := e.store.Renew(ctx, attempts, e.config.LeaseDuration)
	if err != nil {
		return fmt.Errorf("renew job leases: %w", err)
	}
	e.mu.Lock()
	cancellations := 0
	seen := make(map[AttemptIdentity]Renewal, len(renewals))
	for _, renewal := range renewals {
		seen[renewal.Attempt] = renewal
		if attempt, ok := e.inflight[renewal.Attempt]; ok {
			attempt.renewAt = renewStartedAt.Add(e.config.LeaseDuration / 3)
			if renewal.CancelRequested && !attempt.cancelObserved {
				attempt.cancelObserved = true
				cancellations++
			}
			e.inflight[renewal.Attempt] = attempt
			if renewal.CancelRequested {
				attempt.cancel()
			}
		}
	}
	for _, attempt := range attempts {
		if _, ok := seen[attempt]; !ok {
			if inflight, ok := e.inflight[attempt]; ok {
				inflight.cancel()
			}
		}
	}
	e.mu.Unlock()
	for range cancellations {
		e.telemetry.RecordCancellation(ctx, jobs.OutcomeCancelled)
	}
	e.telemetry.RecordRenewal(ctx)
	return nil
}
