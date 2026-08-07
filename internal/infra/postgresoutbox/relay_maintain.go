package postgresoutbox

import (
	"context"
	"fmt"
	"time"
)

// This file owns the two periodic duties that run beside the claim-publish-
// finalize cycle: sampling relay state, and deleting retained published rows.
// Neither is part of delivering an event, which is why they live apart from
// relay.go — but a failure in one of them can still stop the relay, and the two
// signatures below are where that difference is stated.

// schedule is when the relay's two periodic duties come due. Each is measured
// from the moment its work finished rather than from when it was due, so a slow
// observation or cleanup cannot queue up behind itself.
type schedule struct {
	observation time.Time
	cleanup     time.Time
}

func newSchedule(config RelayConfig) schedule {
	now := time.Now()
	return schedule{
		observation: now.Add(config.ObservationInterval),
		cleanup:     now.Add(config.CleanupInterval),
	}
}

// runDueMaintenance runs whichever of the two periodic duties are due and
// returns the schedule they earned. The duties differ in what a failure means,
// and the two signatures below are where that difference is stated: an
// observation failure is absorbed, a cleanup failure stops the relay.
//
// now is read here and again inside each duty, because a duty's next due time is
// measured from the end of its work rather than from when it came due.
func (r *Relay) runDueMaintenance(ctx context.Context, due schedule) (schedule, error) {
	now := time.Now()
	due.observation = r.runDueObservation(ctx, now, due.observation)
	cleanupDue, err := r.runDueCleanup(ctx, now, due.cleanup)
	if err != nil {
		return due, err
	}
	due.cleanup = cleanupDue
	return due, nil
}

// runDueObservation samples relay state when it is due and returns when the next
// sample falls. A failed sample is deliberately absorbed rather than returned:
// it leaves the freshness clock untouched, so readiness goes stale on its own
// and the next interval simply tries again.
func (r *Relay) runDueObservation(ctx context.Context, now, due time.Time) time.Time {
	if now.Before(due) {
		return due
	}
	_ = r.sampleState(ctx)
	return time.Now().Add(r.config.ObservationInterval)
}

// runDueCleanup deletes one retention batch when it is due and returns when the
// next delete falls. Its error stops the relay, because retention is how the
// table stays bounded.
func (r *Relay) runDueCleanup(ctx context.Context, now, due time.Time) (time.Time, error) {
	if now.Before(due) {
		return due, nil
	}
	fullBatch, err := r.cleanup(ctx)
	if err != nil {
		return due, err
	}
	delay := r.config.CleanupInterval
	if fullBatch {
		// A full batch means more is waiting; come back at poll speed.
		delay = min(delay, r.config.PollInterval)
	}
	return time.Now().Add(delay), nil
}

// sampleState takes one state observation and publishes it: the relay's
// freshness clock for readiness, and the gauges Telemetry.collect reports.
func (r *Relay) sampleState(ctx context.Context) error {
	observation, err := r.store.Observe(ctx)
	if err != nil {
		return fmt.Errorf("observe outbox: %w", err)
	}
	observedAt := time.Now()
	r.observedAtUnixNano.Store(observedAt.UnixNano())
	r.telemetry.RecordObservation(observation, observedAt)
	return nil
}

// cleanup deletes one bounded batch of retained published rows. fullBatch means
// the delete hit its own limit, so more is waiting and the next run is due
// sooner than the ordinary cadence.
func (r *Relay) cleanup(ctx context.Context) (fullBatch bool, err error) {
	deleted, err := r.store.CleanupPublished(ctx, r.config.PublishedRetention, r.config.CleanupBatchSize)
	if err != nil {
		return false, fmt.Errorf("cleanup outbox: %w", err)
	}
	return deleted == r.config.CleanupBatchSize, nil
}
