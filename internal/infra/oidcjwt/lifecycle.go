package oidcjwt

// The Verifier's own lifetime: the one Run that drives scheduled refresh and
// readiness, and the Close that retires it. Verification itself is in
// verifier.go and needs none of this — a Verifier answers requests whether or
// not anyone ever calls Run.
//
// Refresh is the one behaviour no single file holds. refresh.go admits fetches
// through refresher.begin and runs them; verifier.go owns keys and install; this
// file owns the two deadlines an installed set implies. Run below is the only
// owner that sees all three, and the only publisher of readiness.
//
//	Run: due fires ───▶ begin(scheduled) ─┐
//	                                      ├─▶ one fetchAndInstall goroutine
//	Verify: key miss ─▶ begin(key_miss) ──┘             │
//	                                                    ├─ on success: keys.Store,
//	                                                    │  then install ─▶ Run: rearm + publish
//	                                                    └─ always: close(call.done)
//	                                                       ─▶ Run: retryAfter + publish
//
// Run reacts to call.done only for a fetch its own begin returned; a key-miss
// refresh it never asked for is awaited by the verification that caused it. So a
// successful scheduled refresh reaches Run twice, on install and on call.done,
// while a purely request-driven one reaches it once, on install — which is why
// the retry cadence lives on the call.done arm, the only one a refused or failed
// fetch reaches.

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// readinessPublisher tells Run's caller whether trust is current, once at the
// start and then only on a change.
//
// It is a type rather than two variables in Run because its fields have to move
// together: recording a state without publishing the edge, or the reverse, loses
// a transition silently. [refreshSchedule] exists against the same hazard.
type readinessPublisher struct {
	publish func(bool)
	started bool
	last    bool
}

// observe publishes current when it is the first answer or a change from the
// last one. A nil publish function makes this a no-op, which is what lets Run be
// driven without a readiness consumer.
func (p *readinessPublisher) observe(current bool) {
	if p.publish != nil && (!p.started || current != p.last) {
		p.publish(current)
	}
	p.started = true
	p.last = current
}

// refreshSchedule holds the two deadlines one key set implies: due is when the
// service should have replaced the set, stale is when verification and readiness
// must start failing closed. Both derive from the same fetchedAt and are always
// armed together, which is what this type exists to make hard to get wrong —
// arming one and forgetting the other is silent, and the symptom is a set that
// goes stale under requests or a cadence that never fires again.
//
// The type owns timer policy only. [Verifier.Run] is its sole user and owns when
// each deadline is restated.
type refreshSchedule struct {
	due   *time.Timer
	stale *time.Timer
}

func newRefreshSchedule(now, fetchedAt time.Time) refreshSchedule {
	return refreshSchedule{
		due:   time.NewTimer(until(now, fetchedAt.Add(RefreshInterval))),
		stale: time.NewTimer(until(now, fetchedAt.Add(MaxKeySetAge))),
	}
}

// rearm restates both deadlines against a newly installed set.
func (s refreshSchedule) rearm(now, fetchedAt time.Time) {
	s.due.Reset(until(now, fetchedAt.Add(RefreshInterval)))
	s.stale.Reset(until(now, fetchedAt.Add(MaxKeySetAge)))
}

// retryAfter moves only the due deadline. The stale deadline deliberately stays
// where the installed set put it: retrying sooner changes when the service will
// next try to replace that set, not how long the set it still holds remains
// usable.
func (s refreshSchedule) retryAfter(delay time.Duration) {
	s.due.Reset(delay)
}

func (s refreshSchedule) stop() {
	s.due.Stop()
	s.stale.Stop()
}

func until(now, future time.Time) time.Duration {
	if !future.After(now) {
		return 0
	}
	return future.Sub(now)
}

// Run owns scheduled refresh and exact trust-current readiness transitions.
func (v *Verifier) Run(ctx context.Context, onTrustCurrent func(bool)) error {
	v.lifecycleMu.Lock()
	if v.runStarted || v.retired {
		v.lifecycleMu.Unlock()
		return errors.New("OIDC verifier lifecycle is invalid")
	}
	v.runStarted = true
	v.lifecycleMu.Unlock()
	// runDone is closed without the lock because Run is admitted at most once,
	// so this is the only close, and runDone is assigned once in newVerifier.
	defer func() {
		v.cancel()
		v.admission.join()
		close(v.runDone)
	}()

	readiness := readinessPublisher{publish: onTrustCurrent}
	publishCurrent := func() { readiness.observe(v.CheckReady() == nil) }
	publishCurrent()

	// The [Verifier] field comment owns why keys is non-nil here and below.
	schedule := newRefreshSchedule(v.now(), v.keys.Load().fetchedAt)
	defer schedule.stop()
	var scheduled *refreshCall

	for {
		select {
		case <-ctx.Done():
			v.cancel()
			return fmt.Errorf("run OIDC verifier: %w", ctx.Err())
		case <-v.baseCtx.Done():
			return context.Canceled
		// Every installed set re-arms the cadence, whoever fetched it: a
		// request-driven refresh that lands here is as good as a scheduled one.
		// A successful scheduled refresh therefore re-arms twice, once here from
		// the new fetchedAt and once below from now. The two agree to within the
		// time it took to hand the set over, because a successful install stamps
		// fetchedAt with the moment it parsed that set.
		case <-v.install:
			schedule.rearm(v.now(), v.keys.Load().fetchedAt)
			publishCurrent()
		case <-schedule.due.C:
			scheduled = v.admission.begin(triggerScheduled)
			// Defensive: the scheduled trigger is not rate limited, so today this
			// is never nil. The guard is what keeps that a local property of
			// refresher.begin rather than something Run has to be edited alongside.
			if scheduled == nil {
				schedule.retryAfter(RefreshCooldown)
			}
		// pendingDone answers nil while no scheduled refresh is in flight, and
		// receiving from a nil channel blocks, so this case is reachable only
		// when scheduled is non-nil. The guard restates that for the nil checker,
		// which reads the channel and the pointer as unrelated values.
		case <-pendingDone(scheduled):
			if scheduled == nil {
				continue
			}
			next := RefreshCooldown
			if scheduled.err == nil {
				next = RefreshInterval
			}
			scheduled = nil
			schedule.retryAfter(next)
			publishCurrent()
		case <-schedule.stale.C:
			publishCurrent()
		}
	}
}

// Close cancels and joins owned work and releases the JWKS connection pool.
func (v *Verifier) Close() {
	if v == nil {
		return
	}
	// Retiring under the lock is what stops a Run from starting inside the window
	// below, where Close has released the lock and is still cancelling and
	// joining. Whether Run ever started is read in the same critical section,
	// because once retired is set no later Run can change the answer.
	v.lifecycleMu.Lock()
	v.retired = true
	runStarted := v.runStarted
	v.lifecycleMu.Unlock()

	// sync.Once holds a second caller here until the first has returned, so
	// every Close reports a Verifier whose work is joined and whose owned client
	// and gauge were released exactly once.
	v.closeOnce.Do(func() {
		v.cancel()
		if runStarted {
			<-v.runDone
		}
		// Run's defer joins the fetches it saw, and this covers the rest: the
		// ones a verification admitted, whether or not Run ever ran, including
		// one admitted after Run left.
		v.admission.join()
		v.client.close()
		if v.unregisterAge != nil {
			v.unregisterAge()
		}
	})
}
