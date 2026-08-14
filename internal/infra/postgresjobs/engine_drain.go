package postgresjobs

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/jobs"
)

// DrainResult reports whether dependency cleanup may safely run after drain.
// A false CleanupSafe means an attempt outlived cancellation and process exit
// retains ownership of the Session and pool.
type DrainResult struct {
	CleanupSafe bool
	Err         error
}

// StartDrain closes claim admission, waits for the active coordinator cycle to
// publish every committed claim to the join, then gives registered attempts the
// supplied soft-drain budget. A timed-out join leaves dependency cleanup to the
// process because handlers may still be using the reserved Session.
func (e *Engine) StartDrain(ctx context.Context) DrainResult {
	if e == nil {
		return DrainResult{Err: fmt.Errorf("%w: engine is required", ErrConfig)}
	}
	e.mu.Lock()
	e.closeAdmissionLocked()
	facts := EngineFacts{ClaimAdmissionOpen: e.admission, Compatible: e.compatible, InFlight: len(e.inflight), Capacity: e.config.MaxConcurrency, ObservationFresh: !e.lastObservation.IsZero()}
	e.mu.Unlock()
	e.telemetry.UpdateFacts(facts)

	// A cycle that began before admission closed may have durably claimed work.
	// Waiting for it here makes the map insertion in registerClaimLocked the
	// quiescence boundary rather than leaving an untracked handoff behind.
	e.cycleMu.Lock()
	//nolint:staticcheck,gocritic // Immediate unlock is the cycle-quiescence barrier.
	e.cycleMu.Unlock()

	done := make(chan struct{})
	go func() {
		e.attempts.Wait()
		close(done)
	}()
	softCtx, cancelSoftDrain := context.WithTimeout(ctx, e.config.DrainTimeout)
	defer cancelSoftDrain()
	select {
	case <-done:
		e.telemetry.RecordDrain(ctx, jobs.OutcomeSuccess)
		return DrainResult{CleanupSafe: true}
	case <-softCtx.Done():
	}

	e.mu.Lock()
	for _, cancel := range e.inflight {
		cancel()
	}
	e.mu.Unlock()
	select {
	case <-done:
		e.telemetry.RecordDrain(context.WithoutCancel(ctx), jobs.OutcomeCancelled)
		return DrainResult{CleanupSafe: true}
	case <-ctx.Done():
		e.telemetry.RecordDrain(context.WithoutCancel(ctx), jobs.OutcomeUnknown)
		return DrainResult{Err: fmt.Errorf("join drained job attempts: %w", ctx.Err())}
	}
}
