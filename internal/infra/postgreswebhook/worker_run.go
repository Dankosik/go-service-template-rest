package postgreswebhook

import (
	"context"
	"errors"
	"fmt"
	"time"
)

//nolint:gocognit,cyclop // The select owns one worker lifecycle and keeps shutdown ordering visible.
func (worker *Worker) Run(ctx context.Context) WorkerResult {
	if worker == nil {
		return WorkerResult{Err: fmt.Errorf("%w: worker is required", ErrConfig)}
	}
	if err := worker.store.CheckSchema(ctx); err != nil {
		return WorkerResult{Err: err}
	}
	if err := worker.store.EnsureCapacity(ctx); err != nil {
		return WorkerResult{Err: err}
	}
	maintenance, err := worker.maintain(ctx)
	worker.recordMaintenanceDurations(ctx, maintenance)
	if err != nil {
		return WorkerResult{Err: err}
	}
	worker.telemetry.RecordN(ctx, "reconciliation", OutcomeHTTPAccepted, maintenance.reconciled)
	worker.telemetry.RecordN(ctx, "privacy_retirement", OutcomeHTTPAccepted, maintenance.retired)
	worker.telemetry.RecordN(ctx, "cleanup", OutcomeHTTPAccepted, maintenance.cleaned)
	worker.telemetry.MarkMaintenance()
	worker.ready.maintained()
	observation, err := worker.store.ObserveReadiness(ctx, worker.manifest)
	if err != nil {
		return WorkerResult{Err: err}
	}
	worker.ready.observed()
	worker.telemetry.Update(observation, worker.Ready())
	poll := time.NewTicker(worker.config.PollInterval)
	observe := time.NewTicker(worker.config.ObservationInterval)
	maintain := time.NewTicker(worker.config.MaintenanceInterval)
	defer poll.Stop()
	defer observe.Stop()
	defer maintain.Stop()
	errorsC := make(chan error, cap(worker.slots))
	startAttempt := func(attempt ClaimedAttempt) {
		worker.slots <- struct{}{}
		worker.attempts.Add(1)
		go func() {
			defer func() { <-worker.slots; worker.attempts.Done() }()
			outcome, failure, err := worker.runAttempt(ctx, attempt)
			if outcome != "" {
				worker.telemetry.RecordFailure(ctx, "attempt", outcome, failure)
			}
			if err != nil {
				select {
				case errorsC <- err:
				default:
				}
			}
		}()
	}
	for {
		select {
		case <-ctx.Done():
			err := worker.drain()
			return WorkerResult{Err: err, CleanupUnsafe: errors.Is(err, ErrDrainUnsafe)}
		case err := <-errorsC:
			worker.ready.close()
			worker.telemetry.MarkStale()
			worker.telemetry.RecordFailure(ctx, "attempt", OutcomeUnknown, failureClass(err))
		case <-poll.C:
			if ctx.Err() != nil {
				continue
			}
			available := cap(worker.slots) - len(worker.slots)
			for range available {
				if ctx.Err() != nil {
					break
				}
				claim, err := worker.store.Claim(ctx, worker.config.WorkerID, worker.config.ClaimScanPage, worker.leaseDuration(), worker.manifest)
				if err != nil {
					worker.ready.close()
					worker.telemetry.MarkStale()
					worker.telemetry.RecordFailure(ctx, "claim", OutcomeUnknown, failureClass(err))
					break
				}
				worker.telemetry.MarkClaim()
				if claim.Attempt == nil {
					if claim.Progress {
						worker.telemetry.Record(ctx, "claim_progress", OutcomeHTTPAccepted)
					}
					break
				}
				if ctx.Err() != nil {
					break
				}
				startAttempt(*claim.Attempt)
			}
		case <-maintain.C:
			maintenance, err := worker.maintain(ctx)
			worker.recordMaintenanceDurations(ctx, maintenance)
			if err != nil {
				worker.ready.close()
				worker.telemetry.MarkStale()
				worker.telemetry.RecordFailure(ctx, "maintenance", OutcomeUnknown, failureClass(err))
				continue
			}
			worker.telemetry.RecordN(ctx, "reconciliation", OutcomeHTTPAccepted, maintenance.reconciled)
			worker.telemetry.RecordN(ctx, "privacy_retirement", OutcomeHTTPAccepted, maintenance.retired)
			worker.telemetry.RecordN(ctx, "cleanup", OutcomeHTTPAccepted, maintenance.cleaned)
			worker.telemetry.MarkMaintenance()
			worker.ready.maintained()
		case <-observe.C:
			observation, err := worker.store.ObserveReadiness(ctx, worker.manifest)
			if err != nil {
				worker.ready.close()
				worker.telemetry.MarkStale()
				worker.telemetry.RecordFailure(ctx, "observation", OutcomeUnknown, failureClass(err))
				continue
			}
			worker.ready.observed()
			worker.telemetry.Update(observation, worker.Ready())
		}
	}
}

func (worker *Worker) recordMaintenanceDurations(ctx context.Context, result maintenanceResult) {
	if result.reconcileDuration > 0 {
		worker.telemetry.RecordDuration(ctx, "reconciliation", result.reconcileDuration)
	}
	if result.retireDuration > 0 {
		worker.telemetry.RecordDuration(ctx, "privacy_retirement", result.retireDuration)
	}
	if result.cleanupDuration > 0 {
		worker.telemetry.RecordDuration(ctx, "cleanup", result.cleanupDuration)
	}
}

func (worker *Worker) leaseDuration() time.Duration {
	return worker.config.AttemptTimeout + 2*worker.config.StoreOperationTimeout
}

func (worker *Worker) drain() error {
	worker.ready.close()
	worker.telemetry.MarkStale()
	done := make(chan struct{})
	go func() { worker.attempts.Wait(); close(done) }()
	timer := time.NewTimer(worker.config.DrainTimeout)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
		return fmt.Errorf("%w: hard bound exceeded", ErrDrainUnsafe)
	}
}
