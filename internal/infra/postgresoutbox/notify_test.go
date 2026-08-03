package postgresoutbox

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestSignalCollapsesPendingWakeups(t *testing.T) {
	t.Parallel()

	wake := make(chan struct{}, 1)
	signal(wake)
	// A second signal must not block: one pending wake-up already tells the
	// relay to claim again, and that claim sees every prior commit.
	signal(wake)
	if len(wake) != 1 {
		t.Fatalf("pending wake-ups = %d, want 1", len(wake))
	}
	<-wake
	if len(wake) != 0 {
		t.Fatalf("wake-ups after receive = %d, want 0", len(wake))
	}
}

func TestSleepReturnsOnCancellationAndOnExpiry(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	started := time.Now()
	sleep(ctx, time.Hour)
	if elapsed := time.Since(started); elapsed > time.Minute {
		t.Fatalf("canceled sleep took %s, want an immediate return", elapsed)
	}
	sleep(t.Context(), time.Millisecond)
}

func TestConsumeAppendsReportsConnectFailure(t *testing.T) {
	t.Parallel()

	err := consumeAppends(t.Context(), unreachableConnConfig(t), make(chan struct{}, 1))
	if err == nil || !strings.Contains(err.Error(), "connect outbox listener") {
		t.Fatalf("consumeAppends(unreachable) error = %v, want a connect failure", err)
	}
}

// A listener that cannot reach PostgreSQL reports the degraded state and keeps
// retrying, because pickup falls back to the poll interval rather than
// stopping. It returns only when its context is done.
func TestListenForAppendsRetriesUntilContextDone(t *testing.T) {
	t.Parallel()

	logs := &syncBuffer{}
	telemetry, err := NewTelemetry(nil, slog.New(slog.NewJSONHandler(logs, nil)))
	if err != nil {
		t.Fatalf("NewTelemetry(): %v", err)
	}
	t.Cleanup(telemetry.Close)

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		listenForAppends(unreachableConnConfig(t), telemetry)(ctx, make(chan struct{}, 1))
	}()

	deadline := time.After(30 * time.Second)
	for !logs.contains("outbox_listener_retry") {
		select {
		case <-deadline:
			t.Fatal("listener did not report the failed subscription")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		t.Fatal("listener did not return after cancellation")
	}
}

// unreachableConnConfig points at a closed local port so connecting fails
// immediately instead of waiting on a network timeout.
func unreachableConnConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	config, err := pgx.ParseConfig("postgres://outbox:outbox@127.0.0.1:1/outbox?sslmode=disable")
	if err != nil {
		t.Fatalf("pgx.ParseConfig(): %v", err)
	}
	config.ConnectTimeout = time.Second
	return config
}

type syncBuffer struct {
	mutex   sync.Mutex
	written bytes.Buffer
}

func (buffer *syncBuffer) Write(payload []byte) (int, error) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	written, err := buffer.written.Write(payload)
	if err != nil {
		return written, fmt.Errorf("buffer log record: %w", err)
	}
	return written, nil
}

func (buffer *syncBuffer) contains(value string) bool {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return strings.Contains(buffer.written.String(), value)
}
