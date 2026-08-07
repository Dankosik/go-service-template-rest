package postgresoutbox

import (
	"math"
	"time"
)

// This file owns the readiness policy and nothing else. It lives apart from the
// cycle because it has two readers rather than one: [Relay.Ready] answers the
// diagnostics probe from live relay state, and Telemetry.collect answers a
// scrape from the published snapshot. Both ask readyWithFreshObservation, so a
// readiness input added here reaches the probe and the gauge together.

func (r *Relay) Ready() bool {
	if r == nil {
		return false
	}
	return readyWithFreshObservation(
		r.readyToServe(),
		r.lastObservation(),
		observationStaleAfter(r.config.ObservationInterval),
	)
}

// readyToServe reports the lifecycle half of readiness: the loop started and
// drain has not begun. The freshness half is observation state, which
// readyWithFreshObservation owns.
//
// It is not the negation of loopMustStop and the two are not interchangeable: a
// relay that has not started its loop yet is neither serving nor stopping.
func (r *Relay) readyToServe() bool {
	return r.ready.Load() && !isClosed(r.drain)
}

// lastObservation is when the last state observation succeeded, or the zero
// time when none has. Those are different conditions — readiness requires a
// sample, so "never observed" must not read as an old one.
func (r *Relay) lastObservation() time.Time {
	nanos := r.observedAtUnixNano.Load()
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

// reportProcessState hands the metric callback the readiness inputs it reads.
// Only the relay knows when they change, so every lifecycle edge calls it. It
// reports the inputs rather than the verdict: the callback runs on scrape and
// re-derives readiness itself, so an observation that goes stale between two
// lifecycle edges still turns the gauge off.
func (r *Relay) reportProcessState() {
	r.telemetry.SetProcessState(r.readyToServe(), r.inflight.Load(), observationStaleAfter(r.config.ObservationInterval))
}

// readyWithFreshObservation is the whole readiness policy, and the only copy of
// it. Its two callers reach it from different state — see this file's header —
// so a readiness input added here cannot silently disagree between them.
func readyWithFreshObservation(running bool, observedAt time.Time, staleAfter time.Duration) bool {
	return running && !observedAt.IsZero() && time.Since(observedAt) <= staleAfter
}

// observationStaleAfter is how long a state observation still counts as fresh
// for readiness: two intervals, so one missed sample is tolerated and two are
// not. The guard keeps the doubling from overflowing.
func observationStaleAfter(interval time.Duration) time.Duration {
	if interval > math.MaxInt64/2 {
		return math.MaxInt64
	}
	return 2 * interval
}
