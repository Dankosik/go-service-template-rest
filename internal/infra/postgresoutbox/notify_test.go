package postgresoutbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
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

// A canceled sleep returns without waiting at all, and an uncanceled one waits
// exactly its duration. Under synctest both are exact, so "immediate" is zero
// rather than a wall-clock bound loose enough to pass while hanging.
func TestSleepReturnsOnCancellationAndOnExpiry(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		started := time.Now()
		sleep(ctx, time.Hour)
		if elapsed := time.Since(started); elapsed != 0 {
			t.Fatalf("canceled sleep took %s, want an immediate return", elapsed)
		}

		started = time.Now()
		sleep(context.Background(), listenerRetryDelay)
		if elapsed := time.Since(started); elapsed != listenerRetryDelay {
			t.Fatalf("expired sleep took %s, want %s", elapsed, listenerRetryDelay)
		}
	})
}

func TestConsumeAppendsReportsConnectFailure(t *testing.T) {
	t.Parallel()

	err := consumeAppends(t.Context(), unreachableConnConfig(t), make(chan struct{}, 1))
	if !errors.Is(err, errListenerConnect) {
		t.Fatalf("consumeAppends(unreachable) error = %v, want a connect failure", err)
	}
	if stage := listenerStage(err); stage != "connect" {
		t.Fatalf("listenerStage(connect failure) = %q, want %q", stage, "connect")
	}
}

// A listener that cannot reach PostgreSQL reports the degraded state and keeps
// retrying, because pickup falls back to the poll interval rather than
// stopping. It returns only when its context is done.
//
// The report names the failing stage and carries no DSN material: pgx formats the
// user, database, and host into its connect error, so logging that error would
// break the package's own promise, and the assertion below is what fails if
// LogListenerRetry ever takes an error again.
//
// Under synctest the retry sleep is the bubble's only durable block point, so
// synctest.Wait is an owned event rather than a polled wall-clock bound.
func TestListenForAppendsRetriesUntilContextDone(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
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

		synctest.Wait()
		if !logs.contains(`"stage":"connect"`) {
			t.Fatalf("listener did not report a connect-stage retry: %s", logs.String())
		}
		for _, material := range []string{dsnUser, dsnPassword, dsnDatabase, dsnHost} {
			if logs.contains(material) {
				t.Errorf("listener log leaked DSN material %q: %s", material, logs.String())
			}
		}

		cancel()
		synctest.Wait()
		select {
		case <-stopped:
		default:
			t.Fatal("listener did not return after cancellation")
		}
	})
}

// The DSN unreachableConnConfig dials, split out so the redaction assertion
// above matches on values that appear nowhere else in a log record. Reusing the
// service name for all of them would make that assertion pass on a substring of
// the event name itself.
const (
	dsnUser     = "listeneruser"
	dsnPassword = "listenersecret"
	dsnDatabase = "listenerdb"
	dsnHost     = "127.0.0.1:1"
)

// unreachableConnConfig points at a closed local port so connecting fails
// immediately instead of waiting on a network timeout.
func unreachableConnConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	config, err := pgx.ParseConfig(fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable", dsnUser, dsnPassword, dsnHost, dsnDatabase,
	))
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
	return strings.Contains(buffer.String(), value)
}

func (buffer *syncBuffer) String() string {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return buffer.written.String()
}
