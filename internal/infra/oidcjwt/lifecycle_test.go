package oidcjwt

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestScheduledRecoveryCadenceResetsFromSuccessfulInstall(t *testing.T) {
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	synctest.Test(t, func(t *testing.T) {
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
		verifier := newTestVerifierWithRuntime(t, client)

		runCtx, cancel := context.WithCancel(context.Background())
		runResult := make(chan error, 1)
		current := make(chan bool, 16)
		go func() {
			runResult <- verifier.Run(runCtx, func(ready bool) { current <- ready })
		}()
		requireBoolEvent(t, current, true)

		time.Sleep(RefreshInterval)
		requireSignal(t, failedOne)
		synctest.Wait()
		time.Sleep(RefreshCooldown - time.Second)
		if client.callCount() != 3 {
			t.Fatalf("provider calls before cooldown = %d, want 3", client.callCount())
		}
		time.Sleep(time.Second)
		requireSignal(t, failedTwo)
		synctest.Wait()

		time.Sleep(RefreshCooldown)
		requireSignal(t, succeeded)
		synctest.Wait()
		time.Sleep(RefreshInterval - time.Second)
		if client.callCount() != 5 {
			t.Fatalf("provider calls before post-install interval = %d, want 5", client.callCount())
		}
		time.Sleep(time.Second)
		requireSignal(t, nextScheduled)

		cancel()
		if err := requireErrorEvent(t, runResult); !errors.Is(err, context.Canceled) {
			t.Fatalf("Run() error = %v, want cancellation", err)
		}
	})
}

func TestTrustCurrentness(t *testing.T) {
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	synctest.Test(t, func(t *testing.T) {
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
		verifier := newTestVerifierWithRuntime(t, client)
		keys := *verifier.keys.Load()
		keys.fetchedAt = time.Now().Add(-MaxKeySetAge + time.Second)
		verifier.keys.Store(&keys)

		runCtx, cancel := context.WithCancel(context.Background())
		runResult := make(chan error, 1)
		current := make(chan bool, 16)
		go func() {
			runResult <- verifier.Run(runCtx, func(ready bool) { current <- ready })
		}()
		requireBoolEvent(t, current, true)

		time.Sleep(time.Second)
		requireSignal(t, refreshStarted)
		requireKind(t, verifier.CheckReady(), KindUnavailable)
		requireBoolEvent(t, current, false)
		close(releaseFailure)

		synctest.Wait()
		time.Sleep(RefreshCooldown)
		requireSignal(t, recoveryStarted)
		requireBoolEvent(t, current, true)
		if err := verifier.CheckReady(); err != nil {
			t.Fatalf("CheckReady() after recovery = %v", err)
		}

		cancel()
		if err := requireErrorEvent(t, runResult); !errors.Is(err, context.Canceled) {
			t.Fatalf("Run() error = %v, want cancellation", err)
		}
	})
}

func TestVerifierLifecycleClosesExactlyOnce(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
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
		func() time.Time { return now },
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

func newTestVerifierWithRuntime(t *testing.T, client *scriptedClient) *Verifier {
	t.Helper()
	verifier, err := newVerifier(
		t.Context(),
		testPolicy(t),
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		time.Now,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)
	return verifier
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
