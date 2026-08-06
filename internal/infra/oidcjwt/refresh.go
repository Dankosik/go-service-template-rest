package oidcjwt

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

// refresher decides which JWKS fetches actually run: it coalesces concurrent
// requests into one call and rate-limits the token-driven trigger. Admission is
// all it owns. Fetching and installing a key set is the fetch function it is
// built with — newVerifier supplies [Verifier.fetchAndInstall], the only
// implementation — and deciding when a refresh is due belongs to its callers.
//
// Keeping it out of [Verifier] is what holds one mutex to one question. This
// type's mutex guards admission and nothing else, so a change to the cooldown or
// to coalescing can be read here without also holding the installed key set or
// the Run/Close progression in mind.
type refresher struct {
	// baseCtx is what a fetch runs under, and it is deliberately not any
	// caller's: a refresh has to outlive the verification that triggered it, or
	// one client hanging up would cancel the fetch every other waiter is
	// blocked on. [Verifier] owns this context and cancels it in Close, which is
	// the stop signal for a fetch still in flight.
	baseCtx context.Context //nolint:containedctx // a refresh outlives its caller by design; Verifier.Close cancels this.
	now     func() time.Time
	fetch   func(context.Context) error
	record  func(ctx context.Context, trigger refreshTrigger, err error)

	mu           sync.Mutex
	active       *refreshCall
	cooldownTill time.Time
}

// begin joins the in-flight JWKS fetch or starts one.
//
// It returns nil in exactly one case: triggerKeyMiss arriving while its cooldown
// is still active. Every other trigger always gets a call to wait on.
func (r *refresher) begin(trigger refreshTrigger) *refreshCall {
	now := r.now()
	rateLimited := trigger.rateLimited()
	r.mu.Lock()
	if r.active != nil {
		call := r.active
		r.mu.Unlock()
		return call
	}
	if rateLimited && now.Before(r.cooldownTill) {
		r.mu.Unlock()
		return nil
	}
	call := &refreshCall{done: make(chan struct{})}
	r.active = call
	if rateLimited {
		r.cooldownTill = now.Add(RefreshCooldown)
	}
	r.mu.Unlock()

	// baseCtx is handed to the goroutine as a parameter rather than read from the
	// receiver inside it, so the context this fetch runs under is visible at the
	// point the goroutine starts.
	go func(refreshCtx context.Context) {
		call.err = r.fetch(refreshCtx)
		r.record(context.WithoutCancel(refreshCtx), trigger, call.err)
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
	}(r.baseCtx)
	return call
}

// fetchAndInstall is the fetch every admitted call runs: newVerifier supplies it
// as the refresher's fetch function, and nothing else calls it. It is a
// [Verifier] method living in this file for the same reason Run and Close live
// in lifecycle.go — a method belongs with the concern it implements rather than
// with the struct it hangs off — and keeping it beside errRefreshFailed is what
// puts every cause of that one value in view of the comment that explains why
// they share it.
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
	case v.install <- struct{}{}:
	default:
	}
	return nil
}

// pendingDone is the completion channel of a refresh a caller is holding on to,
// or nil when it holds none. A nil channel blocks forever in a select, which is
// how one variable states both "waiting on this fetch" and "not waiting", with
// no second variable to keep in step.
func pendingDone(call *refreshCall) <-chan struct{} {
	if call == nil {
		return nil
	}
	return call.done
}

// join waits for an admitted fetch to finish, if one is in flight. It is the
// shutdown half of admission: no new call is started, and the goroutine begin
// launched is given until baseCtx cancellation to return.
func (r *refresher) join() {
	r.mu.Lock()
	call := r.active
	r.mu.Unlock()
	if call != nil {
		<-call.done
	}
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
// It is one of the two routes into begin and the only one that blocks. Run takes
// the other: it calls begin directly and then selects on the call it got back,
// because it has readiness and its own deadlines to serve while a fetch runs.
func (v *Verifier) refresh(ctx context.Context) error {
	call := v.admission.begin(triggerKeyMiss)
	if call == nil {
		return nil
	}
	return waitRefresh(ctx, call)
}
