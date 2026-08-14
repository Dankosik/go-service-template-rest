package postgreswebhook

import (
	"context"
	"errors"
	"fmt"
	"time"
)

//nolint:gocognit // The select owns one worker lifecycle and keeps shutdown ordering visible.
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
	if _, err := worker.store.ReconcileExpired(ctx, worker.config.MaintenanceBatch); err != nil {
		return WorkerResult{Err: err}
	}
	observation, err := worker.store.ObserveReadiness(ctx, worker.manifest)
	if err != nil {
		return WorkerResult{Err: err}
	}
	worker.ready.update()
	worker.telemetry.Update(observation, worker.Ready())
	poll := time.NewTicker(worker.config.PollInterval)
	observe := time.NewTicker(worker.config.ObservationInterval)
	maintain := time.NewTicker(worker.config.MaintenanceInterval)
	defer poll.Stop()
	defer observe.Stop()
	defer maintain.Stop()
	errorsC := make(chan error, cap(worker.slots))
	for {
		select {
		case <-ctx.Done():
			err := worker.drain()
			return WorkerResult{Err: err, CleanupUnsafe: errors.Is(err, ErrDrainUnsafe)}
		case <-errorsC:
			worker.ready.close()
			worker.telemetry.MarkStale()
			worker.telemetry.Record(ctx, "attempt", OutcomeUnknown)
		case <-poll.C:
			if len(worker.slots) == cap(worker.slots) {
				continue
			}
			claim, err := worker.store.Claim(ctx, worker.config.WorkerID, worker.config.ClaimScanPage, worker.leaseDuration(), worker.manifest)
			if err != nil {
				worker.ready.close()
				worker.telemetry.MarkStale()
				worker.telemetry.Record(ctx, "claim", OutcomeUnknown)
				continue
			}
			if claim.Attempt == nil {
				continue
			}
			worker.slots <- struct{}{}
			worker.attempts.Add(1)
			go func(attempt ClaimedAttempt) {
				defer func() { <-worker.slots; worker.attempts.Done() }()
				if err := worker.runAttempt(ctx, attempt); err != nil {
					select {
					case errorsC <- err:
					default:
					}
				}
			}(*claim.Attempt)
		case <-maintain.C:
			if err := worker.maintain(ctx); err != nil {
				worker.ready.close()
				worker.telemetry.MarkStale()
				worker.telemetry.Record(ctx, "maintenance", OutcomeUnknown)
				continue
			}
		case <-observe.C:
			observation, err := worker.store.ObserveReadiness(ctx, worker.manifest)
			if err != nil {
				worker.ready.close()
				worker.telemetry.MarkStale()
				worker.telemetry.Record(ctx, "observation", OutcomeUnknown)
				continue
			}
			worker.ready.update()
			worker.telemetry.Update(observation, worker.Ready())
		}
	}
}

func (worker *Worker) leaseDuration() time.Duration {
	return worker.config.AttemptTimeout + 2*worker.config.StoreOperationTimeout + worker.config.DrainTimeout
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
