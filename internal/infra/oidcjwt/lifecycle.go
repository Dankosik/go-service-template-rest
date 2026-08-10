package oidcjwt

// The Verifier's background half: which JWKS fetches run, the deadlines an
// installed key set implies, the one Run that drives both and publishes
// readiness, and the Close that retires them. Verification itself is in
// verifier.go and needs none of this — a Verifier answers requests whether or
// not anyone ever calls Run. The installed set itself belongs to [trustStore],
// in keyset.go.

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

var errRefreshInProgress = errors.New("JWKS refresh is already in progress")

// refreshTrigger names why a JWKS fetch started. It is a named type rather than
// a plain string for the two reasons transport is: the accepted set is closed by
// the compiler, and each value is published verbatim as the
// authn.refresh.trigger metric attribute, so a fourth trigger is a fourth metric
// series rather than a label. TestDocumentedTriggersMatchTheGuide fails until
// docs/authentication.md publishes it.
type refreshTrigger string

const (
	triggerStartup refreshTrigger = "startup"
	// triggerKeyMiss is every verification a refresh is owed, which is wider than
	// its name: keySet.verifies answers one question — does an installed key sign
	// this token — and a token naming an unknown key id and a token whose key id
	// was rotated under the same name are both a no. Both are recovered by the
	// same fetch, so both belong to this trigger; what an operator reads it as is
	// "the installed set could not answer", not "the id was absent".
	triggerKeyMiss   refreshTrigger = "key_miss"
	triggerScheduled refreshTrigger = "scheduled"
)

// rateLimited reports whether this trigger's fetches are held to
// RefreshCooldown. Only the token-driven one is, because an unknown key id is
// attacker-reachable while startup and the scheduled cadence are the service's
// own.
//
// A trigger added above has to be classified here, and the mandatory exhaustive
// linter is what says so rather than a reader noticing.
func (t refreshTrigger) rateLimited() bool {
	switch t {
	case triggerKeyMiss:
		return true
	case triggerStartup, triggerScheduled:
		return false
	default:
		// Unreachable for a declared trigger, because the exhaustive linter holds
		// the arms above to the full set. Rate limiting an unclassified one is the
		// direction that protects the provider.
		return true
	}
}

// refreshCall is one admitted JWKS fetch and the handle its request owner and
// lifecycle observers use. err is written before done is closed and read only
// after, so the channel is what publishes it.
type refreshCall struct {
	done      chan struct{}
	startedAt time.Time
	err       error
}

// refreshAdmission decides which JWKS fetches actually run: it coalesces
// concurrent requests into one call and rate-limits the token-driven trigger.
// Deciding when a refresh is due belongs to its callers.
//
// Its mutex guards admission and nothing else, so a change to the cooldown or to
// coalescing can be read here without also holding the installed key set or the
// Run/Close progression in mind.
type refreshAdmission struct {
	// owner runs the admitted fetch, records its outcome, and owns both the clock
	// admission reads and the lifetime a fetch runs under. There is exactly one
	// refreshAdmission per [Verifier] and newVerifier builds it, so this points
	// back at the Verifier that built it rather than at copies of its parts.
	owner *Verifier

	mu           sync.Mutex
	active       *refreshCall
	cooldownTill time.Time
	retired      bool
}

// begin joins the in-flight JWKS fetch or starts one. admitted is false when a
// key miss is cooling down or the Verifier is retired; started distinguishes the
// owner of a new fetch from a caller that found one already running.
func (r *refreshAdmission) begin(trigger refreshTrigger) (call *refreshCall, admitted, started bool) {
	now := r.owner.now()
	rateLimited := trigger.rateLimited()
	r.mu.Lock()
	defer r.mu.Unlock()
	// Joining a fetch already in flight is admitted even after retire: it is
	// running either way, retire is waiting for exactly it, and refusing here
	// would discard an answer the caller is about to need.
	if r.active != nil {
		return r.active, true, false
	}
	if r.retired {
		return nil, false, false
	}
	if rateLimited && now.Before(r.cooldownTill) {
		return nil, false, false
	}
	call = &refreshCall{done: make(chan struct{}), startedAt: time.Now()}
	r.active = call
	if rateLimited {
		r.cooldownTill = now.Add(RefreshCooldown)
	}

	// Started under the lock, which costs nothing while fetchAndInstall does not
	// take this mutex. A fetch that called back into the refreshAdmission would
	// break that, so the launch would have to move below the unlock.
	//
	// It runs under the Verifier's own lifetime and deliberately not any caller's:
	// a refresh has to outlive the verification that triggered it, or one client
	// hanging up would prevent the completed trust replacement every later request
	// needs. Close cancels that context, which is the stop signal for a fetch still
	// in flight.
	go func(refreshCtx context.Context) {
		call.err = r.owner.fetchAndInstall(refreshCtx)
		r.owner.metrics.recordRefresh(
			context.WithoutCancel(refreshCtx),
			trigger,
			call.err,
			time.Since(call.startedAt),
		)
		// Clearing the call and closing it are one step under the lock. join and
		// retire both read active under this same lock and wait only on what they
		// found there, so closing after the unlock would leave a window where
		// either sees no active call and returns while this goroutine is still
		// running — and "the wait returned" would stop meaning "the fetch is
		// done", which is what Close relies on before releasing the client.
		r.mu.Lock()
		if r.active == call {
			r.active = nil
		}
		close(call.done)
		r.mu.Unlock()
	}(r.owner.baseCtx)
	return call, true, true
}

// join waits for an admitted fetch to finish, if one is in flight. [Verifier.Run]
// uses it on the way out to leave nothing of its own running; a later
// verification may still admit a fetch, which is why this does not close
// admission.
func (r *refreshAdmission) join() {
	r.mu.Lock()
	call := r.active
	r.mu.Unlock()
	if call != nil {
		<-call.done
	}
}

// retire closes admission and waits for the fetch already in flight, if any.
//
// It is what makes [Verifier.Close] safe to release the provider client after:
// once it returns, no fetch is running and none can start, so nothing can reach
// that client again. join alone could not promise the second half — a
// verification racing Close could admit a fetch in the window between the wait
// and the release, and the only thing making that harmless today is that the
// client's release happens to be CloseIdleConnections, which is safe to call
// beside a request in flight. Making admission answer instead means a release
// that ever becomes a real teardown does not turn into a use-after-close on the
// one path only a shutdown under load reaches.
func (r *refreshAdmission) retire() {
	r.mu.Lock()
	r.retired = true
	call := r.active
	r.mu.Unlock()
	if call != nil {
		<-call.done
	}
}

// fetchAndInstall is the fetch every admitted call runs: refreshAdmission.begin
// calls it on its owner, and nothing else calls it.
//
// A panic is converted rather than propagated because the goroutine begin
// launched has no caller to recover it, and one malformed provider response must
// not take the process down. logRecoveredPanic owns why converting it silently
// would be worse than not converting it at all.
func (v *Verifier) fetchAndInstall(ctx context.Context) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = errProviderPanic
			logRecoveredPanic(ctx, v.log, "jwks_refresh", recovered)
		}
	}()
	body, err := fetchDocument(ctx, v.client.request, v.jwksURI)
	if err != nil {
		return err
	}
	candidate, err := parseKeySet(body, v.now())
	if err != nil {
		return errProviderInvalidDocument
	}
	v.trust.install(candidate)
	return nil
}

// waitRefresh waits for an admitted call under the caller's own context, so a
// caller that gives up reports its own cancellation rather than the fetch's
// outcome. The fetch itself keeps running for whoever else is waiting.
func waitRefresh(ctx context.Context, call *refreshCall) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("wait for JWKS refresh: %w", ctx.Err())
	case <-call.done:
		return call.err
	}
}

// refresh starts the one JWKS fetch a key miss is owed and waits for it. A
// refusal by the cooldown is reported as success, not as an error: no fetch was
// owed, and the caller answers from the set it already has. A concurrent miss
// does not wait behind the fetch: it fails retryably, bounding request-path
// waiters without changing the one provider call already in flight.
//
// It is the blocking route into begin. Run takes the other: it selects on the
// call it got back, because it has readiness and its own deadlines to serve
// while a fetch runs. Successful installs rearm Run's cadence through
// [trustStore.replaced] alone; its completion arm only schedules a retry after
// failure, so one success cannot draw two competing jitter values.
//
// Waiting here puts a provider call on the request path, and the trigger is
// reachable without a credential: parseToken has accepted the claims by this
// point but nothing has checked a signature, so an unsigned token carrying the
// configured issuer and audience, an unexpired exp, and an unknown key id gets
// this far. RefreshCooldown bounds what that costs the provider — one fetch per
// cooldown whatever the request rate — and only the request that starts it
// waits up to ProviderTimeout. Concurrent misses return unavailable immediately
// instead of occupying the service's shared admission capacity.
func (v *Verifier) refresh(ctx context.Context) error {
	call, admitted, started := v.admission.begin(triggerKeyMiss)
	if !admitted {
		return nil
	}
	// begin owns the invariant that an admitted refresh has a call; keep the
	// pointer guard explicit because nilaway cannot derive it from the flag.
	if call == nil {
		return nil
	}
	if !started {
		return errRefreshInProgress
	}
	return waitRefresh(ctx, call)
}

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

// observe publishes only first answers and changes. A nil publish function makes
// it a no-op, which is what lets Run be driven without a readiness consumer.
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
// armed together, which is what this type makes hard to get wrong — arming one
// and forgetting the other is silent, and the symptom is a set that goes stale
// under requests or a cadence that never fires again. [Verifier.Run] is its sole
// user and owns when each deadline is restated.
type refreshSchedule struct {
	due    *time.Timer
	stale  *time.Timer
	jitter func(time.Duration) time.Duration
}

func newRefreshSchedule(
	now, fetchedAt time.Time,
	jitter func(time.Duration) time.Duration,
) refreshSchedule {
	return refreshSchedule{
		due:    time.NewTimer(until(now, fetchedAt.Add(jitter(RefreshInterval)))),
		stale:  time.NewTimer(until(now, fetchedAt.Add(MaxKeySetAge))),
		jitter: jitter,
	}
}

func (s refreshSchedule) rearm(now, fetchedAt time.Time) {
	s.due.Reset(until(now, fetchedAt.Add(s.jitter(RefreshInterval))))
	s.stale.Reset(until(now, fetchedAt.Add(MaxKeySetAge)))
}

// retryAfter moves only the due deadline. The stale deadline deliberately stays
// where the installed set put it: retrying sooner changes when the service will
// next try to replace that set, not how long the set it still holds remains
// usable.
func (s refreshSchedule) retryAfter(delay time.Duration) {
	s.due.Reset(s.jitter(delay))
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

// refreshJitter spreads replica refreshes by ten percent in either direction.
// The stale deadline is deliberately not passed through it: trust expiry stays
// exact even while provider load is decorrelated.
func refreshJitter(delay time.Duration) time.Duration {
	spread := delay / 10
	if spread <= 0 {
		return delay
	}
	return delay - spread + time.Duration(rand.Int64N(int64(2*spread+1))) // #nosec G404 -- provider-load jitter is not security randomness.
}

// pendingDone is the completion channel of the scheduled refresh Run is holding
// on to, or nil when it holds none. A nil channel blocks forever in a select,
// which is how one variable states both "waiting on this fetch" and "not
// waiting", with no second variable to keep in step.
func pendingDone(call *refreshCall) <-chan struct{} {
	if call == nil {
		return nil
	}
	return call.done
}

// Run owns scheduled refresh and exact trust-current readiness transitions.
//
// Run reacts to call.done only for a fetch its own begin returned; a key-miss
// refresh it never asked for is awaited by the verification that caused it. So a
// successful scheduled refresh reaches Run twice, on [trustStore.replaced] and
// on call.done, while a purely request-driven one reaches it once. The
// replacement arm owns the next normal cadence; call.done only schedules the
// shorter failure retry.
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

	schedule := newRefreshSchedule(v.now(), v.trust.current().fetchedAt, v.jitter)
	defer schedule.stop()
	var scheduled *refreshCall

	for {
		select {
		case <-ctx.Done():
			v.cancel()
			return fmt.Errorf("run OIDC verifier: %w", ctx.Err())
		// baseCtx is the Verifier's own lifetime, so this arm is Close retiring
		// the Verifier rather than the caller giving up on Run. Both arms name
		// their cause for the same reason: the two are one cancellation to
		// errors.Is, so the message is all an operator has to tell them apart.
		case <-v.baseCtx.Done():
			return fmt.Errorf("retire OIDC verifier: %w", v.baseCtx.Err())
			// Every installed set re-arms the cadence, whoever fetched it: a
			// request-driven refresh that lands here is as good as a scheduled one.
		case <-v.trust.replaced():
			schedule.rearm(v.now(), v.trust.current().fetchedAt)
			publishCurrent()
		case <-schedule.due.C:
			call, admitted, _ := v.admission.begin(triggerScheduled)
			if !admitted {
				// Unreachable today: the scheduled trigger is not rate limited.
				// Retrying keeps that a local property of begin rather than
				// something Run has to be edited alongside.
				schedule.retryAfter(RefreshCooldown)
				continue
			}
			scheduled = call
		// Reachable only when scheduled is non-nil; pendingDone above owns why.
		// The guard restates that for nilaway, which reads the channel and the
		// pointer as unrelated values.
		case <-pendingDone(scheduled):
			if scheduled == nil {
				continue
			}
			if scheduled.err != nil {
				schedule.retryAfter(RefreshCooldown)
			}
			scheduled = nil
			publishCurrent()
		case <-schedule.stale.C:
			publishCurrent()
		}
	}
}

// Close cancels and joins owned work and releases the JWKS connection pool.
func (v *Verifier) Close() {
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
		// Run's defer joins the fetches it saw. This covers the rest — the ones a
		// verification admitted, whether or not Run ever ran — and then closes
		// admission, so the release below cannot race a fetch admitted after the
		// wait. retire owns why that second half matters.
		v.admission.retire()
		v.client.close()
		if v.unregisterAge != nil {
			v.unregisterAge()
		}
	})
}
