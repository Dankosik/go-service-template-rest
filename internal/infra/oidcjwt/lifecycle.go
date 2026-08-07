package oidcjwt

// The Verifier's background half: which JWKS fetches run, the deadlines an
// installed key set implies, the one Run that drives both and publishes
// readiness, and the Close that retires them. Verification itself is in
// verifier.go and needs none of this — a Verifier answers requests whether or
// not anyone ever calls Run.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// refreshTrigger names why a JWKS fetch started. It is a named type rather than
// a plain string for the two reasons [Transport] is: the accepted set is closed
// by the compiler, and each value is published verbatim as the
// authn.refresh.trigger metric attribute, so a fourth trigger is a fourth metric
// series rather than a label. TestDocumentedTriggersMatchTheGuide fails until
// docs/authentication.md publishes it.
type refreshTrigger string

const (
	triggerStartup   refreshTrigger = "startup"
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

// errRefreshFailed is the whole result of a failed JWKS refresh, by the
// redaction rule errProviderFetchFailed in provider.go owns. A failure mode
// added to fetchAndInstall below — its sole producer — wants this value rather
// than a message of its own.
var errRefreshFailed = errors.New("JWKS refresh failed")

// refreshCall is one admitted JWKS fetch and the handle every caller waiting on
// that fetch shares. err is written before done is closed and read only after,
// so the channel is what publishes it.
type refreshCall struct {
	done chan struct{}
	err  error
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
}

// begin joins the in-flight JWKS fetch or starts one. It reports false in
// exactly one case: triggerKeyMiss arriving while its cooldown is still active.
// Every other trigger always gets a call to wait on.
func (r *refreshAdmission) begin(trigger refreshTrigger) (*refreshCall, bool) {
	now := r.owner.now()
	rateLimited := trigger.rateLimited()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil {
		return r.active, true
	}
	if rateLimited && now.Before(r.cooldownTill) {
		return nil, false
	}
	call := &refreshCall{done: make(chan struct{})}
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
	// hanging up would cancel the fetch every other waiter is blocked on. Close
	// cancels that context, which is the stop signal for a fetch still in flight.
	go func(refreshCtx context.Context) {
		call.err = r.owner.fetchAndInstall(refreshCtx)
		r.owner.metrics.recordRefresh(context.WithoutCancel(refreshCtx), trigger, call.err)
		// Retiring the call and closing it are one step under the lock, and the
		// close belongs inside rather than after: join reads active under this same
		// lock and waits only on what it found there. Closing after the unlock would
		// leave a window where join sees no active call and returns while this
		// goroutine has not finished, so "join returned" would stop meaning "the
		// fetch goroutine is done" — which is what Close relies on before releasing
		// the provider client.
		r.mu.Lock()
		if r.active == call {
			r.active = nil
		}
		close(call.done)
		r.mu.Unlock()
	}(r.owner.baseCtx)
	return call, true
}

// join waits for an admitted fetch to finish, if one is in flight. It is the
// shutdown half of admission: no new call is started, and the goroutine begin
// launched is given until baseCtx cancellation to return.
func (r *refreshAdmission) join() {
	r.mu.Lock()
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
// not take the process down.
func (v *Verifier) fetchAndInstall(ctx context.Context) (err error) {
	defer func() {
		if recover() != nil {
			err = errRefreshFailed
		}
	}()
	body, err := fetchDocument(ctx, v.client.request, v.jwksURI)
	if err != nil {
		return errRefreshFailed
	}
	candidate, err := parseKeySet(body, v.now())
	if err != nil {
		return errRefreshFailed
	}
	v.keys.Store(candidate)
	select {
	case v.installed <- struct{}{}:
	default:
	}
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

// refresh starts or joins the one JWKS fetch a key miss is owed and waits for
// it. A refusal by the cooldown is reported as success, not as an error: no
// fetch was owed, and the caller answers from the set it already has.
//
// It is the blocking route into begin. Run takes the other: it selects on the
// call it got back, because it has readiness and its own deadlines to serve
// while a fetch runs.
func (v *Verifier) refresh(ctx context.Context) error {
	call, admitted := v.admission.begin(triggerKeyMiss)
	if !admitted {
		return nil
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
// armed together, which is what this type makes hard to get wrong — arming one
// and forgetting the other is silent, and the symptom is a set that goes stale
// under requests or a cadence that never fires again. [Verifier.Run] is its sole
// user and owns when each deadline is restated.
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
// successful scheduled refresh reaches Run twice, on installed and on call.done,
// while a purely request-driven one reaches it once, on installed — which is why
// the retry cadence lives on the call.done arm, the only one a refused or failed
// fetch reaches.
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
		// baseCtx is the Verifier's own lifetime, so this arm is Close retiring
		// the Verifier rather than the caller giving up on Run. Both arms name
		// their cause for the same reason: the two are one cancellation to
		// errors.Is, so the message is all an operator has to tell them apart.
		case <-v.baseCtx.Done():
			return fmt.Errorf("retire OIDC verifier: %w", v.baseCtx.Err())
		// Every installed set re-arms the cadence, whoever fetched it: a
		// request-driven refresh that lands here is as good as a scheduled one.
		// A successful scheduled refresh therefore re-arms twice, once here from
		// the new fetchedAt and once below from now. The two agree to within the
		// time it took to hand the set over, because a successful install stamps
		// fetchedAt with the moment it parsed that set.
		case <-v.installed:
			schedule.rearm(v.now(), v.keys.Load().fetchedAt)
			publishCurrent()
		case <-schedule.due.C:
			call, admitted := v.admission.begin(triggerScheduled)
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
