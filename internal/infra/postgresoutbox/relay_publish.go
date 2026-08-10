package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// errPublicationNotAttempted marks an event of a claimed batch that the relay
// never handed to [Publisher], because the batch's shared budget was already
// spent when its turn came. It stays unexported and never leaves this package:
// Relay.publishAll is its only producer and Relay.classify its only consumer,
// which is the same reason notify.go keeps its stage sentinels beside the
// failures they name.
//
// It is not a publication failure. classify turns it into an ordinary retry
// that adds no uncertainty and gives back the attempt the claim charged, so a
// broker slow enough to leave every batch with a tail costs throughput rather
// than a growing quarantine of events nobody tried to publish.
var errPublicationNotAttempted = errors.New("outbox publication was not attempted")

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
// per event, by its index in the batch: the publication's error, or nil once
// the broker acknowledged that exact event under the publisher's configured
// durability contract. It reports cleanupSafe = false when a publisher ignored
// cancellation, because its goroutine can still touch the dependencies the
// process is about to close.
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
				if batchCtx.Err() != nil {
					// The budget belongs to the whole batch, so its tail can find
					// it already spent — by an earlier event, or by the shutdown
					// that cancelled ctx. Handing the publisher a dead context
					// would come back as a timeout, and a timeout is an ambiguous
					// publication: it would set sticky uncertainty and charge an
					// attempt for a call that never left this process. Reporting
					// the non-attempt as itself is what keeps the attempt cap
					// measured against attempts actually made.
					written[index] = errPublicationNotAttempted
					r.telemetry.CountOperation(ctx, publishOperationName, outcomeSkipped, classPublisherNotAttempted)
					continue
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
	// The adapter is called with the span's context, so whatever spans it opens
	// for its own broker call nest under this publication rather than floating
	// as roots of their own.
	ctx, span := r.telemetry.StartPublish(ctx, event)
	defer func() {
		if value := recover(); value != nil {
			err = ErrPublisherPanic
			// Taken here rather than left to the runtime: recovering the panic is
			// what keeps the rest of the batch finalizing, and it also throws away
			// the only description of the fault. The process is about to exit over
			// a deployment fault, so the stack an operator needs to fix it has to
			// be logged before the value goes out of scope. debug.Stack inside a
			// deferred call still reaches the panicking frames, because the stack
			// is not unwound until this returns.
			r.telemetry.LogPublisherPanic(ctx, value, debug.Stack())
		}
		errorClass := classNone
		if err != nil {
			errorClass = publicationErrorClass(err)
		}
		r.telemetry.EndPublish(span, err, errorClass)
		r.telemetry.RecordOperation(ctx, publishOperationName, operationOutcome(err), errorClass, time.Since(started))
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
	case errors.Is(err, errPublicationNotAttempted):
		return classPublisherNotAttempted
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
