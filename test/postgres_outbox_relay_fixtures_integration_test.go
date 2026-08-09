//go:build integration

package integration_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

type testPublisherFunc func(context.Context, postgresoutbox.Event) error

func (publish testPublisherFunc) Publish(ctx context.Context, event postgresoutbox.Event) error {
	return publish(ctx, event)
}

func testRelayConfig() postgresoutbox.RelayConfig {
	return postgresoutbox.RelayConfig{
		PollInterval:        time.Millisecond,
		BatchSize:           100,
		PublishConcurrency:  16,
		PublishTimeout:      20 * time.Millisecond,
		LeaseDuration:       2 * time.Second,
		MaxAttempts:         10,
		RetryBase:           time.Second,
		RetryMax:            5 * time.Minute,
		ObservationInterval: time.Hour,
		CleanupInterval:     time.Hour,
		PublishedRetention:  7 * 24 * time.Hour,
		CleanupBatchSize:    1000,
	}
}

// singleEventRelayConfig keeps one event per claim so a test can distinguish
// the attempt in flight from the claim that would follow it.
func singleEventRelayConfig() postgresoutbox.RelayConfig {
	config := testRelayConfig()
	config.BatchSize = 1
	return config
}

// gatingPublisher hands the test control of when one publication completes:
// started closes as the call begins, and the call returns once the test closes
// release.
//
// It publishes exactly once: started is a close, not a send, so a second
// concurrent call panics and surfaces as a confusing publisher_panic. Pair it
// with singleEventRelayConfig and one appended event, or with a test that stops
// the relay before it can claim again.
func gatingPublisher() (publisher testPublisherFunc, started <-chan struct{}, release chan<- struct{}, attempts *atomic.Int64) {
	begun := make(chan struct{})
	gate := make(chan struct{})
	var calls atomic.Int64
	return testPublisherFunc(func(context.Context, postgresoutbox.Event) error {
		calls.Add(1)
		close(begun)
		<-gate
		return nil
	}), begun, gate, &calls
}

func mustNewOutboxRelay(
	t *testing.T,
	store *postgresoutbox.Store,
	publisher postgresoutbox.Publisher,
	telemetry *postgresoutbox.Telemetry,
	config postgresoutbox.RelayConfig,
) *postgresoutbox.Relay {
	t.Helper()
	relay, err := postgresoutbox.NewRelay(store, publisher, telemetry, config)
	if err != nil {
		t.Fatalf("NewRelay(): %v", err)
	}
	return relay
}

func runOutboxRelay(ctx context.Context, relay *postgresoutbox.Relay) <-chan postgresoutbox.RelayResult {
	result := make(chan postgresoutbox.RelayResult, 1)
	go func() { result <- relay.Run(ctx) }()
	return result
}

// assertRelayResult is the ordinary stop: whatever the relay reported, its
// dependencies are still safe to close. Only a stuck publisher says otherwise,
// which assertRelayStuckResult asserts instead.
func assertRelayResult(t *testing.T, result <-chan postgresoutbox.RelayResult, wantErr error) {
	t.Helper()
	got := readRelayResult(t, result)
	if got.CleanupUnsafe {
		t.Fatalf("Relay.Run() = %+v, want cleanup to stay safe", got)
	}
	if !errors.Is(got.Err, wantErr) {
		t.Fatalf("Relay.Run() error = %v, want %v", got.Err, wantErr)
	}
}

func assertRelayStuckResult(t *testing.T, result <-chan postgresoutbox.RelayResult, wantErr error) {
	t.Helper()
	got := readRelayResult(t, result)
	if !got.CleanupUnsafe {
		t.Fatalf("Relay.Run() = %+v, want cleanup reported unsafe", got)
	}
	if !errors.Is(got.Err, wantErr) {
		t.Fatalf("Relay.Run() error = %v, want %v", got.Err, wantErr)
	}
}

func readRelayResult(t *testing.T, result <-chan postgresoutbox.RelayResult) postgresoutbox.RelayResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(outboxWaitTimeout):
		t.Fatal("Relay.Run() did not stop")
	}
	return postgresoutbox.RelayResult{}
}

func waitForOutboxReady(t *testing.T, relay *postgresoutbox.Relay) {
	t.Helper()
	waitForOutbox(t, func() string { return "the relay to become ready" }, relay.Ready)
}
