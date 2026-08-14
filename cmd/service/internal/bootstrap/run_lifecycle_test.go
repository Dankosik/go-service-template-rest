package bootstrap

import (
	"context"
	// profile:outbound-auth-oauth2-client-credentials:start
	"errors"
	"log/slog"
	"slices"
	"strings"

	// profile:outbound-auth-oauth2-client-credentials:end
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/background"
	// profile:outbound-auth-oauth2-client-credentials:start
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	// profile:outbound-auth-oauth2-client-credentials:end
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

// profile:http-idempotency-postgres:start
func TestHTTPIdempotencyMaintenanceJoinsAfterHTTPDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		signalCtx, signal := context.WithCancel(context.Background())
		supervisor := newSupervisedBackground(signalCtx, shutdownTestLogger())
		started := make(chan context.Context, 1)
		runtime := httpIdempotencyRuntime{
			interval: time.Second,
			maintain: func(ctx context.Context) error {
				started <- ctx
				<-ctx.Done()
				return ctx.Err()
			},
		}
		supervisor.Go(background.Task{Name: "http_idempotency_maintenance", Run: runtime.Run})
		maintenanceCtx := <-started

		signal()
		events := &eventRecorder{}
		if err := drainAndShutdown(
			signalCtx,
			shutdownTestLogger(),
			0,
			time.Second,
			&fakeDrainer{events: events},
			&fakeShutdownServer{events: events},
		); err != nil {
			t.Fatalf("drainAndShutdown() error = %v", err)
		}
		if err := maintenanceCtx.Err(); err != nil {
			t.Fatalf("maintenance stopped before HTTP drain completed: %v", err)
		}
		if err := supervisor.Shutdown(testShutdownBudget().stage(signalCtx, backgroundShutdownTimeout)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
		if maintenanceCtx.Err() == nil {
			t.Fatal("maintenance was not canceled and joined after HTTP drain")
		}
	})
}

// profile:http-idempotency-postgres:end

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

// profile:outbound-auth-oauth2-client-credentials:start
func TestOutboundAuthRuntimeCloseOrder(t *testing.T) {
	t.Run("normal shutdown", func(t *testing.T) {
		resetShutdownConfigEnv(t)
		var events []string
		runtime := &recordingOutboundAuthRuntime{onClose: func() {
			events = append(events, "outbound_auth_closed")
		}}
		wiring := outboundAuthTestWiring(runtime)
		wiring.lifecycle = func(stage runtimeLifecycleStage) { events = append(events, string(stage)) }
		wiring.initOutboundAuth = func(
			config.OutboundAuthConfig,
			*telemetry.Metrics,
			*slog.Logger,
		) (outboundAuthRuntime, error) {
			events = append(events, "outbound_auth_constructed")
			return runtime, nil
		}
		stopServing := errors.New("stop serving")
		wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error {
			events = append(events, "serving")
			return stopServing
		}

		if err := runWithRuntime(nil, wiring); !errors.Is(err, stopServing) {
			t.Fatalf("runWithRuntime() error = %v, want %v", err, stopServing)
		}
		want := []string{
			string(runtimeLifecycleMemoryPublished),
			// profile:object-storage:start
			string(runtimeLifecycleObjectStorageConstructed),
			// profile:object-storage:end
			"outbound_auth_constructed",
			"serving",
			string(runtimeLifecycleHTTPDrained),
			string(runtimeLifecycleBackgroundJoined),
			// profile:object-storage:start
			string(runtimeLifecycleObjectStorageClosed),
			// profile:object-storage:end
			"outbound_auth_closed",
			string(runtimeLifecycleDependenciesClosed),
			string(runtimeLifecycleTelemetryFlushed),
		}
		assertOutboundAuthCloseOrder(t, runtime, events, want)
	})

	// profile:authn-oidc-jwt:start
	t.Run("partial startup", func(t *testing.T) {
		resetShutdownConfigEnv(t)
		var events []string
		runtime := &recordingOutboundAuthRuntime{onClose: func() {
			events = append(events, "outbound_auth_closed")
		}}
		wiring := outboundAuthTestWiring(runtime)
		wiring.lifecycle = func(stage runtimeLifecycleStage) { events = append(events, string(stage)) }
		wiring.initOutboundAuth = func(
			config.OutboundAuthConfig,
			*telemetry.Metrics,
			*slog.Logger,
		) (outboundAuthRuntime, error) {
			events = append(events, "outbound_auth_constructed")
			return runtime, nil
		}
		startupErr := errors.New("authentication startup failed")
		wiring.initAuthn = func(
			context.Context,
			config.Config,
			*telemetry.Metrics,
			*slog.Logger,
		) (authnRuntime, error) {
			events = append(events, "later_startup_failed")
			return nil, startupErr
		}

		if err := runWithRuntime(nil, wiring); !errors.Is(err, startupErr) {
			t.Fatalf("runWithRuntime() error = %v, want %v", err, startupErr)
		}
		want := []string{
			string(runtimeLifecycleMemoryPublished),
			// profile:object-storage:start
			string(runtimeLifecycleObjectStorageConstructed),
			// profile:object-storage:end
			"outbound_auth_constructed",
			"later_startup_failed",
			// profile:object-storage:start
			string(runtimeLifecycleObjectStorageClosed),
			// profile:object-storage:end
			"outbound_auth_closed",
			string(runtimeLifecycleDependenciesClosed),
			string(runtimeLifecycleTelemetryFlushed),
		}
		assertOutboundAuthCloseOrder(t, runtime, events, want)
	})
	// profile:authn-oidc-jwt:end
}

func TestOutboundAuthCloseFailureIsJoinedOnce(t *testing.T) {
	for _, partialStartup := range []bool{
		false,
		// profile:authn-oidc-jwt:start
		true,
		// profile:authn-oidc-jwt:end
	} {
		name := "normal shutdown"
		if partialStartup {
			name = "partial startup"
		}
		t.Run(name, func(t *testing.T) {
			resetShutdownConfigEnv(t)
			closeErr := errors.New("outbound authentication provider is unavailable")
			runtime := &recordingOutboundAuthRuntime{closeErr: closeErr}
			wiring := outboundAuthTestWiring(runtime)
			pathErr := errors.New("stop serving")
			if partialStartup {
				pathErr = errors.New("later startup failed")
				// profile:authn-oidc-jwt:start
				wiring.initAuthn = func(
					context.Context,
					config.Config,
					*telemetry.Metrics,
					*slog.Logger,
				) (authnRuntime, error) {
					return nil, pathErr
				}
				// profile:authn-oidc-jwt:end
				wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error {
					t.Fatal("partial startup reached serving")
					return nil
				}
			} else {
				wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error { return pathErr }
			}

			err := runWithRuntime(nil, wiring)
			if !errors.Is(err, pathErr) || !errors.Is(err, closeErr) {
				t.Fatalf("runWithRuntime() error = %v, want path and close failures", err)
			}
			if strings.Count(err.Error(), closeErr.Error()) != 1 {
				t.Fatalf("close failure occurrences in %q = %d, want 1", err, strings.Count(err.Error(), closeErr.Error()))
			}
			if runtime.closeCalls != 1 {
				t.Fatalf("Close calls = %d, want 1", runtime.closeCalls)
			}
		})
	}
}

func assertOutboundAuthCloseOrder(
	t *testing.T,
	runtime *recordingOutboundAuthRuntime,
	got, want []string,
) {
	t.Helper()
	if runtime.closeCalls != 1 {
		t.Fatalf("Close calls = %d, want 1", runtime.closeCalls)
	}
	deadline, ok := runtime.closeContext.Deadline()
	if !ok {
		t.Fatal("Close context carries no deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > dependencyCloseTimeout {
		t.Fatalf("Close deadline remaining = %s, want (0,%s]", remaining, dependencyCloseTimeout)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("lifecycle events = %v, want %v", got, want)
	}
}

// profile:outbound-auth-oauth2-client-credentials:end

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
