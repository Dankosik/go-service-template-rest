package bootstrap

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/background"
)

// TestSupervisedWorkOutlivesTheHTTPDrain pins the shutdown order run.go
// documents, in the one direction that fails silently.
//
// A supervisor derived from the signal context is canceled the moment SIGTERM
// arrives — while the drain below it keeps serving for readiness propagation
// plus the remaining shutdown budget. Every request admitted in that window runs
// without the background work it depends on, and the only artifact is an INFO
// line inside an otherwise orderly shutdown. This walks the three moments that
// distinguish the two orderings.
func TestSupervisedWorkOutlivesTheHTTPDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		signalCtx, signal := context.WithCancel(context.Background())
		supervisor := newSupervisedBackground(signalCtx, shutdownTestLogger())

		started := make(chan context.Context, 1)
		supervisor.Go(background.Task{
			Name: "outbox_publisher",
			Run: func(ctx context.Context) error {
				started <- ctx
				<-ctx.Done()
				return nil
			},
		})
		taskCtx := <-started
		synctest.Wait()

		// SIGTERM. Nothing has drained yet, so a request admitted right now still
		// needs this task for the whole drain that follows.
		signal()
		synctest.Wait()
		if err := taskCtx.Err(); err != nil {
			t.Fatalf("supervised work was canceled by SIGTERM before the HTTP drain that depends on it: %v", err)
		}

		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}
		server := &fakeShutdownServer{events: events}
		if err := drainAndShutdown(signalCtx, shutdownTestLogger(), 0, time.Second, drainer, server); err != nil {
			t.Fatalf("drainAndShutdown() error = %v", err)
		}
		synctest.Wait()
		if err := taskCtx.Err(); err != nil {
			t.Fatalf("supervised work stopped during the HTTP drain: %v", err)
		}

		// Only now is nothing left that could depend on it.
		if err := supervisor.Shutdown(testShutdownBudget().stage(signalCtx, backgroundShutdownTimeout)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
		if taskCtx.Err() == nil {
			t.Fatal("supervised work outlived the ordered teardown that owns stopping it")
		}
	})
}

// TestSupervisedBackgroundShutdownIsBoundedByItsOwnBudget keeps the join from
// inheriting the already-canceled signal context, which would make Shutdown
// return instantly and report a task that had not been given its budget.
func TestSupervisedBackgroundShutdownIsBoundedByItsOwnBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		signalCtx, signal := context.WithCancel(context.Background())
		signal()

		ctx := testShutdownBudget().stage(signalCtx, backgroundShutdownTimeout)

		if err := ctx.Err(); err != nil {
			t.Fatalf("background shutdown context is already done: %v", err)
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("background shutdown context carries no deadline")
		}
		if remaining := time.Until(deadline); remaining != backgroundShutdownTimeout {
			t.Fatalf("remaining budget = %s, want %s", remaining, backgroundShutdownTimeout)
		}
	})
}

// profile:authn-oidc-jwt:start
func TestAuthnCloseDoesNotOutliveTheBackgroundBudget(t *testing.T) {
	authn := &countingAuthnRuntime{}
	closeAuthnWithinBudget(authn, true)
	if authn.closes != 1 {
		t.Fatalf("Close calls with budget remaining = %d, want 1", authn.closes)
	}

	expired, cancel := context.WithCancel(context.Background())
	cancel()
	closeAuthnWithinBudget(authn, expired.Err() == nil)
	if authn.closes != 1 {
		t.Fatalf("Close calls after budget expiration = %d, want still 1", authn.closes)
	}
}

type countingAuthnRuntime struct {
	fakeAuthnRuntime

	closes int
}

func (a *countingAuthnRuntime) Close() {
	a.closes++
}

// profile:authn-oidc-jwt:end
