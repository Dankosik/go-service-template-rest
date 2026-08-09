package postgresoutbox

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestRelayCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(ctx context.Context, _ Event) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}))
		// No PublishTimeout here on purpose: publishOne carries no budget of its
		// own, so the only deadline it can observe is the one its caller passes.
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() { result <- relay.publishOne(ctx, unitClaim(1).Event) }()
		<-started
		cancel()
		synctest.Wait()
		if got := <-result; !errors.Is(got, context.Canceled) {
			t.Fatalf("publishOne() = %v, want context cancellation", got)
		}
	})
}

func TestRelayReadinessRequiresFreshObservation(t *testing.T) {
	t.Parallel()

	// Built through the constructor like every other relay here, so this test
	// keeps working if Ready starts reading a field only newRelay populates.
	// unitRelayConfig already uses a one-minute observation interval.
	relay := newUnitRelay(&relayStoreStub{}, noopPublisher())
	relay.ready.Store(true)
	relay.observedAtUnixNano.Store(time.Now().UnixNano())
	if !relay.Ready() {
		t.Fatal("Ready() = false after a fresh successful observation")
	}
	relay.observedAtUnixNano.Store(time.Now().Add(-90 * time.Second).UnixNano())
	if !relay.Ready() {
		t.Fatal("Ready() = false within the two-interval freshness window")
	}
	relay.observedAtUnixNano.Store(time.Now().Add(-3 * time.Minute).UnixNano())
	if relay.Ready() {
		t.Fatal("Ready() = true after the observation became stale")
	}
	relay.observedAtUnixNano.Store(time.Now().UnixNano())
	relay.StartDrain()
	if relay.Ready() {
		t.Fatal("Ready() = true while draining")
	}
	if got := observationStaleAfter(time.Duration(1<<63 - 1)); got != time.Duration(1<<63-1) {
		t.Fatalf("observationStaleAfter(max) = %s", got)
	}
}

func TestRelayRunPublishesAndDrains(t *testing.T) {
	t.Parallel()

	store := &relayStoreStub{}
	claimed := false
	store.claim = func(context.Context, time.Duration, int) (ClaimedBatch, error) {
		if claimed {
			return ClaimedBatch{}, nil
		}
		claimed = true
		return unitBatch(3), nil
	}
	var published atomic.Int64
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error {
		published.Add(1)
		return nil
	}))
	store.markUnorderedPublishedBatch = func(_ context.Context, _ string, ids []string) ([]string, error) {
		relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
		return ids, nil
	}

	result := relay.Run(t.Context())
	if result.Err != nil {
		t.Errorf("Run() stopped with an error after an ordinary drain: %v", result.Err)
	}
	if result.CleanupUnsafe {
		t.Error("Run() reported cleanup unsafe though no publisher outlived cancellation")
	}
	if !claimed {
		t.Error("Run() drained without ever claiming a batch")
	}
	if got := published.Load(); got != 3 {
		t.Errorf("published %d events of the claimed batch, want 3", got)
	}
	if relay.Ready() {
		t.Error("Ready() = true after Run() returned; a stopped relay is never ready")
	}
}

// A wake-up signal replaces the poll wait, so an append committed during an
// idle cycle is claimed without waiting out the poll interval.
func TestRelayRunWakesOnNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		claims := 0
		store := &relayStoreStub{}
		relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
		relay.config.PollInterval = time.Hour
		relay.listen = func(ctx context.Context, wake chan<- struct{}) {
			signal(wake)
			<-ctx.Done()
		}
		store.claim = func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			if claims == 2 {
				relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
			}
			return ClaimedBatch{}, nil
		}

		startedAt := time.Now()
		result := relay.Run(context.Background())
		if result.Err != nil || result.CleanupUnsafe {
			t.Fatalf("Run() = %+v", result)
		}
		if claims != 2 || time.Since(startedAt) != 0 {
			t.Fatalf("claims=%d elapsed=%s, want 2 claims without a poll wait", claims, time.Since(startedAt))
		}
	})
}

func TestRelayRunMaintainsWhileEmpty(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		claims := 0
		observations := 0
		cleanups := 0
		store := &relayStoreStub{
			observe: func(context.Context) (StateObservation, error) {
				observations++
				return StateObservation{}, nil
			},
			cleanup: func(context.Context, time.Duration, int) (int, error) {
				cleanups++
				return 0, nil
			},
		}
		relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
		relay.config.PollInterval = 2 * time.Second
		relay.config.ObservationInterval = time.Second
		relay.config.CleanupInterval = time.Second
		store.claim = func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			if claims == 2 {
				relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
			}
			return ClaimedBatch{}, nil
		}

		started := time.Now()
		result := relay.Run(context.Background())
		if result.Err != nil || result.CleanupUnsafe || time.Since(started) != time.Second {
			t.Fatalf("Run() = %+v elapsed=%s", result, time.Since(started))
		}
		if claims != 2 || observations != 2 || cleanups != 1 {
			t.Fatalf("claims=%d observations=%d cleanups=%d, want 2/2/1", claims, observations, cleanups)
		}
	})
}

func TestRelayRunFailures(t *testing.T) {
	t.Parallel()

	if result := (*Relay)(nil).Run(t.Context()); !errors.Is(result.Err, ErrConfig) || result.CleanupUnsafe {
		t.Fatalf("nil Run() = %+v", result)
	}
	observeErr := errors.New("observe failed")
	relay := newUnitRelay(&relayStoreStub{
		observe: func(context.Context) (StateObservation, error) { return StateObservation{}, observeErr },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	if result := relay.Run(t.Context()); !errors.Is(result.Err, observeErr) {
		t.Fatalf("observe failure Run() = %+v", result)
	}
	claimErr := errors.New("claim failed")
	relay = newUnitRelay(&relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) { return ClaimedBatch{}, claimErr },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	if result := relay.Run(t.Context()); !errors.Is(result.Err, claimErr) {
		t.Fatalf("claim failure Run() = %+v", result)
	}
}

// A typed nil satisfies the interface but cannot publish, so construction has
// to reject it as firmly as an untyped nil.
func TestRelayRejectsMissingDependencies(t *testing.T) {
	t.Parallel()

	var typedNil *pointerPublisher
	if !holdsTypedNil(typedNil) {
		t.Error("holdsTypedNil(typed nil pointer) = false, want true")
	}
	if holdsTypedNil(1) {
		t.Error("holdsTypedNil(int) = true, want false")
	}
	if _, err := NewRelay(nil, noopPublisher(), nil, RelayConfig{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewRelay(nil) error = %v", err)
	}
	var nilStore *relayStoreStub
	if _, err := newRelay(nilStore, noopPublisher(), nil, unitRelayConfig()); !errors.Is(err, ErrConfig) {
		t.Fatalf("newRelay(typed nil store) error = %v", err)
	}
	if _, err := newRelay(&relayStoreStub{}, nil, nil, unitRelayConfig()); !errors.Is(err, ErrConfig) {
		t.Fatalf("newRelay(nil publisher) error = %v", err)
	}
}

// The next observation is measured from when the slow one finished, not from
// when it came due, so a store slower than the interval cannot queue up.
func TestRelayMaintainReschedulesFromEndOfSlowObservation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newUnitRelay(&relayStoreStub{
			observe: func(context.Context) (StateObservation, error) {
				time.Sleep(2 * time.Second)
				return StateObservation{}, errors.New("observe")
			},
		}, noopPublisher())
		relay.config.ObservationInterval = time.Second
		now := time.Now()
		due, err := relay.runDueMaintenance(t.Context(), schedule{observation: now, cleanup: now.Add(time.Hour)})
		if err != nil {
			t.Fatalf("maintain(periodic observation failure) error = %v", err)
		}
		if elapsed := time.Since(now); elapsed != 2*time.Second {
			t.Fatalf("observation elapsed = %s, want 2s", elapsed)
		}
		if delay := due.observation.Sub(now); delay != 3*time.Second {
			t.Fatalf("next observation = %s after start, want 3s", delay)
		}
	})
}

// Retention is how the table stays bounded, so a failed cleanup stops the loop.
func TestRelayMaintainStopsOnCleanupFailure(t *testing.T) {
	t.Parallel()

	cleanupErr := errors.New("cleanup")
	relay := newUnitRelay(&relayStoreStub{
		cleanup: func(context.Context, time.Duration, int) (int, error) { return 0, cleanupErr },
	}, noopPublisher())
	now := time.Now()
	if _, err := relay.runDueMaintenance(
		t.Context(), schedule{observation: now.Add(time.Hour), cleanup: now},
	); !errors.Is(err, cleanupErr) {
		t.Fatalf("maintain(cleanup) error = %v", err)
	}
}

// A full delete batch means more is waiting, so the next cleanup comes at poll
// speed rather than after the whole interval.
func TestRelayMaintainShortensDelayAfterFullCleanupBatch(t *testing.T) {
	t.Parallel()

	relay := newUnitRelay(&relayStoreStub{
		cleanup: func(_ context.Context, _ time.Duration, batch int) (int, error) { return batch, nil },
	}, noopPublisher())
	now := time.Now()
	due, err := relay.runDueMaintenance(t.Context(), schedule{observation: now.Add(time.Hour), cleanup: now})
	if err != nil {
		t.Fatalf("maintain(full cleanup batch) error = %v", err)
	}
	if delay := time.Until(due.cleanup); delay <= 0 || delay > relay.config.PollInterval {
		t.Fatalf("full cleanup batch delay = %s, want (0,%s]", delay, relay.config.PollInterval)
	}
}

func TestRelayWaitStopsOnCancellationAndDrain(t *testing.T) {
	t.Parallel()

	relay := newUnitRelay(&relayStoreStub{}, noopPublisher())
	wake := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if !relay.wait(ctx, wake, time.Hour) {
		t.Fatal("wait(canceled) did not report stop")
	}
	if relay.wait(t.Context(), wake, -time.Second) {
		t.Fatal("wait(negative duration) reported stop")
	}
	relay.StartDrain()
	if !relay.wait(t.Context(), wake, time.Hour) {
		t.Fatal("wait(drain) did not report stop")
	}
}

// Maintenance owns the cycle: a failed cleanup stops the relay before it claims,
// and a drain raised during maintenance starts no claim at all.
func TestRelayLoopStopsOnMaintenanceFailureAndDrain(t *testing.T) {
	t.Parallel()

	cleanupErr := errors.New("cleanup failed")
	claims := 0
	store := &relayStoreStub{
		cleanup: func(context.Context, time.Duration, int) (int, error) { return 0, cleanupErr },
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			return ClaimedBatch{}, nil
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.config.CleanupInterval = 0
	if result := relay.runLoop(t.Context(), nil); !errors.Is(result.Err, cleanupErr) || result.CleanupUnsafe {
		t.Fatalf("runLoop(cleanup failure) = %+v", result)
	}
	if claims != 0 {
		t.Fatalf("claims after maintenance failure = %d, want 0", claims)
	}

	store.cleanup = func(context.Context, time.Duration, int) (int, error) { return 0, nil }
	relay = newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.config.CleanupInterval = 0
	store.cleanup = func(context.Context, time.Duration, int) (int, error) {
		relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
		return 0, nil
	}
	if result := relay.runLoop(t.Context(), nil); result.Err != nil || result.CleanupUnsafe {
		t.Fatalf("runLoop(drain during maintenance) = %+v", result)
	}
	if claims != 0 {
		t.Fatalf("claims after a drain raised during maintenance = %d, want 0", claims)
	}
}

// A finalization fault is absorbed until it repeats. Neither fault it covers is
// something the stop prevents — another relay owns those events, or lease
// recovery republishes one — so stopping on the first occurrence spends a whole
// process restart on one ambiguous event and turns a lease that is persistently
// too short into a halt instead of slower delivery. Repetition is the signal, so
// the count has to reset on a cycle that finalized cleanly.
//
// The tolerated sample is asserted here rather than in telemetry_test.go: it is
// the operator-visible half of this one behavior, and proving it from a second
// copy of this loop setup would only let the two drift.
func TestRelayToleratesFinalizationFaultsUntilTheyRepeat(t *testing.T) {
	t.Parallel()

	reader, telemetry := newTestTelemetry(t, slog.New(slog.DiscardHandler))
	claims := 0
	// One fault short of the tolerance, then a cycle that finalizes cleanly. Those
	// faults must not count toward the run that follows, so the relay stops on the
	// last cycle of a full tolerance after the clean one.
	const (
		absorbedBeforeClean = finalizationFaultTolerance - 1
		cleanCycle          = absorbedBeforeClean + 1
	)
	store := &relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			return unitBatch(1), nil
		},
		scheduleRetryBatch: func(context.Context, string, []RetryDirective) error {
			if claims == cleanCycle {
				return nil
			}
			return ErrLeaseLost
		},
	}
	relay, err := newRelay(store,
		publisherFunc(func(context.Context, Event) error { return ErrPublicationNotAccepted }),
		telemetry, unitRelayConfig())
	if err != nil {
		t.Fatalf("newRelay() error = %v", err)
	}
	relay.config.PollInterval = time.Millisecond

	result := relay.runLoop(t.Context(), nil)
	if !errors.Is(result.Err, ErrLeaseLost) || result.CleanupUnsafe {
		t.Fatalf("runLoop(repeated lease loss) = %+v, want a lost-lease stop", result)
	}
	wantClaims := cleanCycle + finalizationFaultTolerance
	if claims != wantClaims {
		t.Fatalf("claims = %d, want %d: absorbed faults, one clean cycle, then a full tolerance",
			claims, wantClaims)
	}

	tolerated := int64(0)
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != "outbox.relay.operations" {
			return
		}
		for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
			if telemetrytest.Attribute(t, point.Attributes, "operation") == "finalize" &&
				telemetrytest.Attribute(t, point.Attributes, "outcome") == "tolerated" &&
				telemetrytest.Attribute(t, point.Attributes, "error.type") == classLostLease {
				tolerated = point.Value
			}
		}
	})
	if want := int64(absorbedBeforeClean + finalizationFaultTolerance - 1); tolerated != want {
		t.Errorf("tolerated finalize samples = %d, want %d: every fault but the one that stopped the relay",
			tolerated, want)
	}
}

// Run and StartDrain stay safe on a nil relay and on a relay already draining,
// because the process lifecycle calls both without owning that state.
func TestRelayDrainBeforeRunStartsNoLoop(t *testing.T) {
	t.Parallel()

	(*Relay)(nil).StartDrain()

	claims := 0
	relay := newUnitRelay(&relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			return ClaimedBatch{}, nil
		},
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.StartDrain()
	if result := relay.Run(t.Context()); result.Err != nil || result.CleanupUnsafe {
		t.Fatalf("Run(already draining) = %+v", result)
	}
	if claims != 0 || relay.Ready() {
		t.Fatalf("claims=%d ready=%t after running an already-drained relay", claims, relay.Ready())
	}
}
