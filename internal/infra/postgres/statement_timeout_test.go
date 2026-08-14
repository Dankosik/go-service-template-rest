package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestApplyStatementTimeoutsPublishesSessionDefaults pins the wire-level effect:
// these arrive as startup parameters, so every connection the pool opens later
// carries them without any query path having to remember.
func TestApplyStatementTimeoutsPublishesSessionDefaults(t *testing.T) {
	t.Parallel()

	poolConfig, err := parsePoolConfig("postgres://app:app@127.0.0.1:5432/app?sslmode=disable")
	if err != nil {
		t.Fatalf("parsePoolConfig() error = %v", err)
	}

	applyStatementTimeouts(poolConfig.ConnConfig, 8*time.Second)

	for _, name := range []string{"statement_timeout", "idle_in_transaction_session_timeout"} {
		if got := poolConfig.ConnConfig.RuntimeParams[name]; got != "8000ms" {
			t.Fatalf("RuntimeParams[%q] = %q, want %q", name, got, "8000ms")
		}
	}
}

// TestRuntimeParamMillisecondsRoundsUp pins the rounding direction rather than
// the arithmetic. A configured timeout with a sub-millisecond remainder is
// reachable — postgres.statement_timeout is floored at 100ms, and 100500us is
// above that floor — and truncating it would publish less time to PostgreSQL
// than the operator configured.
func TestRuntimeParamMillisecondsRoundsUp(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		duration time.Duration
		want     string
	}{
		{duration: 8 * time.Second, want: "8000ms"},
		{duration: 100500 * time.Microsecond, want: "101ms"},
		{duration: 100 * time.Millisecond, want: "100ms"},
	} {
		if got := RuntimeParamMilliseconds(testCase.duration); got != testCase.want {
			t.Errorf("RuntimeParamMilliseconds(%s) = %q, want %q", testCase.duration, got, testCase.want)
		}
	}
}

func TestApplyStatementTimeoutsToleratesMissingConfig(t *testing.T) {
	t.Parallel()

	// Must not panic: New guards its inputs, but this helper is called before the
	// pool exists and a nil config is the one shape it can receive.
	applyStatementTimeouts(nil, time.Second)
}

func TestApplyStatementTimeoutsInstallsImmediateCancelHandler(t *testing.T) {
	t.Parallel()

	poolConfig, err := parsePoolConfig("postgres://app:app@127.0.0.1:5432/app?sslmode=disable")
	if err != nil {
		t.Fatalf("parsePoolConfig() error = %v", err)
	}

	var marks sync.Map
	applyContextWatcher(poolConfig.ConnConfig, 8*time.Second, &marks)
	handler, ok := poolConfig.ConnConfig.BuildContextWatcherHandler(&pgconn.PgConn{}).(*contextWatcherHandler)
	if !ok {
		t.Fatalf("BuildContextWatcherHandler() = %T, want *contextWatcherHandler", handler)
	}
	cancelHandler, ok := handler.handler.(*pgconn.CancelRequestContextWatcherHandler)
	if !ok {
		t.Fatalf("watcher handler = %T, want *pgconn.CancelRequestContextWatcherHandler", handler.handler)
	}
	if cancelHandler.CancelRequestDelay != 0 || cancelHandler.DeadlineDelay != 8*time.Second {
		t.Fatalf("watcher delays = (%s, %s), want (0s, 8s)", cancelHandler.CancelRequestDelay, cancelHandler.DeadlineDelay)
	}
}

type watcherStub struct {
	cancel, unwatch int
}

func (s *watcherStub) HandleCancel(context.Context) { s.cancel++ }
func (s *watcherStub) HandleUnwatchAfterCancel()    { s.unwatch++ }

func TestContextWatcherMarksCanceledConnection(t *testing.T) {
	conn := &pgconn.PgConn{}
	marker := &contextWatcherMark{}
	var marks sync.Map
	marks.Store(conn, marker)
	delegate := &watcherStub{}
	handler := &contextWatcherHandler{marks: &marks, conn: conn, handler: delegate}

	handler.HandleCancel(t.Context())
	handler.HandleUnwatchAfterCancel()
	if !marker.canceled.Load() || delegate.cancel != 1 || delegate.unwatch != 1 {
		t.Fatalf("watcher state = canceled:%t cancel:%d unwatch:%d", marker.canceled.Load(), delegate.cancel, delegate.unwatch)
	}
}

func TestNewRejectsMissingStatementTimeout(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Options{
		DSN:                "postgres://app:app@127.0.0.1:5432/app?sslmode=disable",
		ConnectTimeout:     time.Second,
		HealthcheckTimeout: time.Second,
		MaxOpenConns:       1,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Minute,
	})
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("New() error = %v, want ErrConfig", err)
	}
}

func TestInTxRejectsUnusableReceiver(t *testing.T) {
	t.Parallel()

	var nilPool *Pool
	if err := nilPool.InTx(context.Background(), pgx.TxOptions{}, func(pgx.Tx) error { return nil }); !errors.Is(err, ErrConfig) {
		t.Fatalf("InTx() on nil pool error = %v, want ErrConfig", err)
	}
	if err := (&Pool{}).InTx(context.Background(), pgx.TxOptions{}, nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("InTx() with nil function error = %v, want ErrConfig", err)
	}
}

// TestRetryableClassifiesTransientConflicts is the classification a service would
// otherwise learn from a production 500 spike.
func TestRetryableClassifiesTransientConflicts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "serialization failure", err: &pgconn.PgError{Code: pgerrcode.SerializationFailure}, want: true},
		{name: "deadlock detected", err: &pgconn.PgError{Code: pgerrcode.DeadlockDetected}, want: true},
		{name: "wrapped serialization failure", err: errors.Join(errors.New("repository: create order"), &pgconn.PgError{Code: pgerrcode.SerializationFailure}), want: true},
		{name: "unique violation is a real conflict", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}},
		{name: "statement timeout is not retryable", err: &pgconn.PgError{Code: pgerrcode.QueryCanceled}},
		{name: "context cancellation", err: context.Canceled},
		{name: "nil", err: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := Retryable(tc.err); got != tc.want {
				t.Fatalf("Retryable(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}
