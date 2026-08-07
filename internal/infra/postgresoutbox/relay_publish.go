package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// publishBatch publishes one claimed batch and finalizes every event in it.
// stop is true for a stuck publisher, a finalization failure, an observed
// cancellation, and a publisher panic. Only the cancellation stop carries a nil
// Err, because shutting down is not a failure.
func (r *Relay) publishBatch(
	ctx context.Context,
	batch ClaimedBatch,
	leaseExpiry time.Time,
) (result RelayResult, stop bool) {
	// The whole claimed batch, not the events inside Publish right now — that is
	// bounded by PublishConcurrency and this gauge is deliberately not. What an
	// operator needs from outbox.relay.inflight is how much durable work one
	// lease is holding, which is what a crash would redeliver.
	r.inflight.Store(int64(len(batch.Events)))
	r.reportProcessState()
	defer func() {
		r.inflight.Store(0)
		r.reportProcessState()
	}()

	publications, cleanupSafe := r.publishAll(ctx, batch, leaseExpiry)
	if !cleanupSafe {
		// A batch-scoped verdict, not an event's latency: publishOne already
		// timed whatever each event managed before the join bound expired.
		r.telemetry.CountOperation(ctx, "publish", outcomeError, classStuck)
		r.telemetry.LogPublisherStuck(ctx)
		return RelayResult{CleanupUnsafe: true, Err: ErrPublisherStuck}, true
	}

	// Finalization must reach PostgreSQL even while the process is shutting
	// down: an acknowledged event left unmarked becomes a duplicate after lease
	// recovery, and a failed event left leased blocks its ordering key until the
	// lease expires. The lease this batch already holds bounds the write.
	finalizeCtx, cancel := context.WithDeadline(context.WithoutCancel(ctx), leaseExpiry)
	defer cancel()
	if err := r.finalize(finalizeCtx, batch, publications); err != nil {
		return RelayResult{Err: err}, true
	}
	// Order matters between these two, and shutdown wins. A batch that was being
	// cancelled reports an ordinary drain even when an adapter also panicked in
	// it: the process is already stopping for a reason the operator asked for,
	// and naming a second one would only obscure that. The panic still reached
	// the publish metric, and the event was still released for retry. A stop
	// reason added below this point inherits the same precedence, so put it
	// above if it must outrank a shutdown in progress.
	if ctx.Err() != nil {
		return RelayResult{}, true
	}
	if err := firstPublisherPanic(publications); err != nil {
		return RelayResult{Err: err}, true
	}
	return RelayResult{}, false
}

// publishAll publishes the batch through at most PublishConcurrency workers and
// never past the lease it was claimed under. The returned slice holds one entry
// per event, by its index in the batch: the publication's error, or nil once the
// broker durably acknowledged that exact event. It reports cleanupSafe = false
// when a publisher ignored cancellation, because its goroutine can still touch
// the dependencies the process is about to close.
//
// The deadline computed here is the whole batch's publication budget. Because
// it starts no later than any single publication does, it is also the budget of
// each one, which is why publishOne needs no timeout of its own.
//
// cleanupSafe is stated positively even though [RelayResult.CleanupUnsafe] is
// not: this reports what it observed, and the result field is stated so its
// zero value is the safe one. Both results are named so the polarity is
// readable at the signature rather than a line into the caller.
func (r *Relay) publishAll(
	ctx context.Context,
	batch ClaimedBatch,
	leaseExpiry time.Time,
) (publications []error, cleanupSafe bool) {
	deadline := earliest(time.Now().Add(r.config.PublishTimeout), leaseExpiry.Add(-PublisherJoinTimeout))
	batchCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	// The workers write to a local rather than to the named result. A stuck
	// batch returns nil while they are still running, and assigning that to the
	// named result would hand a live goroutine a nil slice to index into.
	written := make([]error, len(batch.Events))
	var next atomic.Int64
	var workers sync.WaitGroup
	for range min(r.config.PublishConcurrency, len(batch.Events)) {
		workers.Go(func() {
			for {
				index := int(next.Add(1)) - 1
				if index >= len(batch.Events) {
					return
				}
				written[index] = r.publishOne(batchCtx, batch.Events[index].Event)
			}
		})
	}
	finished := make(chan struct{})
	go func() {
		workers.Wait()
		close(finished)
	}()

	select {
	case <-finished:
		return written, true
	case <-batchCtx.Done():
	}
	join := time.NewTimer(PublisherJoinTimeout)
	defer join.Stop()
	select {
	case <-finished:
		return written, true
	case <-join.C:
		return nil, false
	}
}

func (r *Relay) publishOne(ctx context.Context, event Event) (err error) {
	started := time.Now()
	defer func() {
		if recover() != nil {
			err = ErrPublisherPanic
		}
		errorClass := classNone
		if err != nil {
			errorClass = publicationErrorClass(err)
		}
		r.telemetry.RecordOperation(ctx, "publish", operationOutcome(err), errorClass, time.Since(started))
	}()

	err = r.publisher.Publish(ctx, event)
	if err == nil {
		// A publisher that returns nil after its budget expired cannot prove the
		// broker accepted the event, so the attempt stays retryable.
		if budget := ctx.Err(); budget != nil {
			err = fmt.Errorf("publisher reported success after its budget expired: %w", budget)
		}
	}
	return err
}

func firstPublisherPanic(publications []error) error {
	for _, publication := range publications {
		if errors.Is(publication, ErrPublisherPanic) {
			return publication
		}
	}
	return nil
}

func publicationErrorClass(err error) string {
	switch {
	case errors.Is(err, ErrPublisherPanic):
		return classPanic
	case errors.Is(err, ErrPermanentPublication):
		return classPublisherPermanent
	case errors.Is(err, ErrPublicationNotAccepted):
		return classPublisherRejected
	case errors.Is(err, context.DeadlineExceeded):
		return classPublisherTimeout
	case errors.Is(err, context.Canceled):
		return classPublisherCanceled
	default:
		return classPublisherTemporary
	}
}

func operationOutcome(err error) string {
	if err != nil {
		return outcomeError
	}
	return outcomeSuccess
}

func earliest(first, second time.Time) time.Time {
	if second.Before(first) {
		return second
	}
	return first
}
