package oidcjwt

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeTimerClock struct {
	mu      sync.Mutex
	now     time.Time
	timers  map[*fakeVerifierTimer]struct{}
	created chan struct{}
	resets  chan time.Duration
}

type fakeVerifierTimer struct {
	clock  *fakeTimerClock
	ch     chan time.Time
	due    time.Time
	active bool
}

func newFakeTimerClock(now time.Time) *fakeTimerClock {
	return &fakeTimerClock{
		now:     now,
		timers:  make(map[*fakeVerifierTimer]struct{}),
		created: make(chan struct{}, 16),
		resets:  make(chan time.Duration, 32),
	}
}

func (c *fakeTimerClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeTimerClock) newTimer(duration time.Duration) *fakeVerifierTimer {
	timer := &fakeVerifierTimer{clock: c, ch: make(chan time.Time, 1)}
	c.mu.Lock()
	timer.due = c.now.Add(duration)
	timer.active = true
	c.timers[timer] = struct{}{}
	c.mu.Unlock()
	c.created <- struct{}{}
	return timer
}

func (c *fakeTimerClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	now := c.now
	var due []*fakeVerifierTimer
	for timer := range c.timers {
		if timer.active && !timer.due.After(now) {
			timer.active = false
			due = append(due, timer)
		}
	}
	c.mu.Unlock()
	for _, timer := range due {
		timer.ch <- now
	}
}

func (t *fakeVerifierTimer) C() <-chan time.Time {
	return t.ch
}

func (t *fakeVerifierTimer) Reset(duration time.Duration) {
	t.clock.mu.Lock()
	select {
	case <-t.ch:
	default:
	}
	t.due = t.clock.now.Add(duration)
	t.active = true
	t.clock.mu.Unlock()
	t.clock.resets <- duration
}

func (t *fakeVerifierTimer) Stop() {
	t.clock.mu.Lock()
	t.active = false
	t.clock.mu.Unlock()
}

func TestScheduledRecoveryCadenceResetsFromSuccessfulInstall(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	clock := newFakeTimerClock(now)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	failedOne := make(chan struct{})
	failedTwo := make(chan struct{})
	succeeded := make(chan struct{})
	nextScheduled := make(chan struct{})
	client := &scriptedClient{responses: append(initialResponses(t, first),
		scriptedResponse{err: errors.New("scheduled outage one"), started: failedOne},
		scriptedResponse{err: errors.New("scheduled outage two"), started: failedTwo},
		scriptedResponse{status: http.StatusOK, body: jwksDocument(t, second, "key-2"), started: succeeded},
		scriptedResponse{err: errors.New("next scheduled outage"), started: nextScheduled},
	)}
	verifier := newTestVerifierWithRuntime(t, clock, client)

	runCtx, cancel := context.WithCancel(context.Background())
	runResult := make(chan error, 1)
	current := make(chan bool, 16)
	go func() {
		runResult <- verifier.Run(runCtx, func(ready bool) { current <- ready })
	}()
	requireBoolEvent(t, current, true)
	requireTimerCreations(t, clock, 2)

	clock.Advance(RefreshInterval)
	requireSignal(t, failedOne)
	requireReset(t, clock, RefreshCooldown)
	clock.Advance(RefreshCooldown - time.Second)
	if client.callCount() != 3 {
		t.Fatalf("provider calls before cooldown = %d, want 3", client.callCount())
	}
	clock.Advance(time.Second)
	requireSignal(t, failedTwo)
	requireReset(t, clock, RefreshCooldown)

	clock.Advance(RefreshCooldown)
	requireSignal(t, succeeded)
	requireReset(t, clock, RefreshInterval)
	clock.Advance(RefreshInterval - time.Second)
	if client.callCount() != 5 {
		t.Fatalf("provider calls before post-install interval = %d, want 5", client.callCount())
	}
	clock.Advance(time.Second)
	requireSignal(t, nextScheduled)

	cancel()
	if err := requireErrorEvent(t, runResult); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want cancellation", err)
	}
}

func TestTrustCurrentness(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	clock := newFakeTimerClock(now)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	refreshStarted := make(chan struct{})
	releaseFailure := make(chan struct{})
	recoveryStarted := make(chan struct{})
	client := &scriptedClient{responses: append(initialResponses(t, first),
		scriptedResponse{
			err:     errors.New("stale outage"),
			started: refreshStarted,
			wait:    releaseFailure,
		},
		scriptedResponse{
			status:  http.StatusOK,
			body:    jwksDocument(t, second, "key-2"),
			started: recoveryStarted,
		},
	)}
	verifier := newTestVerifierWithRuntime(t, clock, client)

	runCtx, cancel := context.WithCancel(context.Background())
	runResult := make(chan error, 1)
	current := make(chan bool, 16)
	go func() {
		runResult <- verifier.Run(runCtx, func(ready bool) { current <- ready })
	}()
	requireBoolEvent(t, current, true)
	requireTimerCreations(t, clock, 2)

	clock.Advance(MaxKeySetAge)
	requireSignal(t, refreshStarted)
	close(releaseFailure)
	requireBoolEvent(t, current, false)
	requireReset(t, clock, RefreshCooldown)
	requireKind(t, verifier.CheckReady(), KindUnavailable)

	clock.Advance(RefreshCooldown)
	requireSignal(t, recoveryStarted)
	requireBoolEvent(t, current, true)
	if err := verifier.CheckReady(); err != nil {
		t.Fatalf("CheckReady() after recovery = %v", err)
	}

	cancel()
	if err := requireErrorEvent(t, runResult); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want cancellation", err)
	}
}

func TestVerifierLifecycleClosesExactlyOnce(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	clock := newFakeTimerClock(now)
	key := loadTestRSAKey(t, "test-key-1.pem")
	client := &scriptedClient{responses: initialResponses(t, key)}
	var closes atomic.Int64
	policy := testPolicy(t)
	verifier, err := newVerifier(
		t.Context(),
		policy,
		func(string) (providerClient, error) {
			return providerClient{
				request: client,
				close:   func() { closes.Add(1) },
			}, nil
		},
		clock.Now,
		func(duration time.Duration) verifierTimer {
			return clock.newTimer(duration)
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	if closes.Load() != 1 {
		t.Fatalf("discovery closes after bootstrap = %d, want 1", closes.Load())
	}
	verifier.Close()
	verifier.Close()
	if closes.Load() != 2 {
		t.Fatalf("total provider closes = %d, want discovery + JWKS once", closes.Load())
	}
	if err := verifier.Run(t.Context(), nil); err == nil {
		t.Fatal("Run() after Close error = nil")
	}
}

func newTestVerifierWithRuntime(
	t *testing.T,
	clock *fakeTimerClock,
	client *scriptedClient,
) *Verifier {
	t.Helper()
	verifier, err := newVerifier(
		t.Context(),
		testPolicy(t),
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		clock.Now,
		func(duration time.Duration) verifierTimer {
			return clock.newTimer(duration)
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)
	return verifier
}

func requireTimerCreations(t *testing.T, clock *fakeTimerClock, count int) {
	t.Helper()
	for range count {
		requireSignal(t, clock.created)
	}
}

func requireReset(t *testing.T, clock *fakeTimerClock, want time.Duration) {
	t.Helper()
	for {
		select {
		case got := <-clock.resets:
			if got == want {
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for timer reset to %s", want)
		}
	}
}

func requireSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for owned event")
	}
}

func requireBoolEvent(t *testing.T, events <-chan bool, want bool) {
	t.Helper()
	select {
	case got := <-events:
		if got != want {
			t.Fatalf("readiness event = %v, want %v", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for readiness event")
	}
}

func requireErrorEvent(t *testing.T, events <-chan error) error {
	t.Helper()
	select {
	case err := <-events:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for lifecycle result")
		return nil
	}
}
