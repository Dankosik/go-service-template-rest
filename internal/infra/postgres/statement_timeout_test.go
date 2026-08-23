package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyStatementTimeoutsPublishesSessionDefaults(t *testing.T) {
	t.Parallel()

	poolConfig, err := parsePoolConfig("postgres://app:app@127.0.0.1:5432/app?sslmode=disable")
	if err != nil {
		t.Fatalf("parsePoolConfig() error = %v", err)
	}
	applyStatementTimeouts(poolConfig.ConnConfig, defaultStatementTimeout)
	for _, name := range []string{"statement_timeout", "idle_in_transaction_session_timeout"} {
		if got := poolConfig.ConnConfig.RuntimeParams[name]; got != "8000ms" {
			t.Fatalf("RuntimeParams[%q] = %q, want 8000ms", name, got)
		}
	}
}

func TestRuntimeParamMillisecondsRoundsUp(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		duration time.Duration
		want     string
	}{
		{duration: 8 * time.Second, want: "8000ms"},
		{duration: 100500 * time.Microsecond, want: "101ms"},
		{duration: 100 * time.Millisecond, want: "100ms"},
	} {
		if got := RuntimeParamMilliseconds(tc.duration); got != tc.want {
			t.Errorf("RuntimeParamMilliseconds(%s) = %q, want %q", tc.duration, got, tc.want)
		}
	}
}

func TestApplyStatementTimeoutsToleratesMissingConfig(t *testing.T) {
	t.Parallel()
	applyStatementTimeouts(nil, time.Second)
}

func TestApplyContextWatcherUsesCancelRequestWithServerFallback(t *testing.T) {
	t.Parallel()

	poolConfig, err := parsePoolConfig("postgres://app:app@127.0.0.1:5432/app?sslmode=disable")
	if err != nil {
		t.Fatalf("parsePoolConfig() error = %v", err)
	}
	applyContextWatcher(poolConfig.ConnConfig, defaultStatementTimeout)
	handler := poolConfig.ConnConfig.BuildContextWatcherHandler(new(pgconn.PgConn))
	watcher, ok := handler.(*contextWatcherHandler)
	if !ok {
		t.Fatalf("context watcher = %T, want *contextWatcherHandler", handler)
	}
	cancel, ok := watcher.handler.(*pgconn.CancelRequestContextWatcherHandler)
	if !ok || cancel.CancelRequestDelay != 0 || cancel.DeadlineDelay != defaultStatementTimeout {
		t.Fatalf("cancel watcher = %#v, want immediate cancel with %s fallback", watcher.handler, defaultStatementTimeout)
	}
}

type watcherStub struct {
	cancel, unwatch int
}

func (s *watcherStub) HandleCancel(context.Context) { s.cancel++ }
func (s *watcherStub) HandleUnwatchAfterCancel()    { s.unwatch++ }

func TestContextWatcherMarksTheActiveTransaction(t *testing.T) {
	conn := new(pgconn.PgConn)
	marker := new(contextWatcherMark)
	contextWatcherMarks.Store(conn, marker)
	t.Cleanup(func() { contextWatcherMarks.Delete(conn) })
	delegate := new(watcherStub)
	handler := &contextWatcherHandler{conn: conn, handler: delegate}

	handler.HandleCancel(t.Context())
	handler.HandleUnwatchAfterCancel()
	if !marker.canceled.Load() || delegate.cancel != 1 || delegate.unwatch != 1 {
		t.Fatalf("watcher state = canceled:%t cancel:%d unwatch:%d", marker.canceled.Load(), delegate.cancel, delegate.unwatch)
	}
}

func TestInTxRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if err := InTx(context.Background(), nil, pgx.TxOptions{}, func(pgx.Tx) error { return nil }); !errors.Is(err, ErrConfig) {
		t.Fatalf("InTx() nil pool error = %v, want ErrConfig", err)
	}
	if err := InTx(context.Background(), new(pgxpool.Pool), pgx.TxOptions{}, nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("InTx() nil function error = %v, want ErrConfig", err)
	}
}

func TestRetryableClassifiesTransientConflicts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "serialization failure", err: &pgconn.PgError{Code: pgerrcode.SerializationFailure}, want: true},
		{name: "deadlock detected", err: &pgconn.PgError{Code: pgerrcode.DeadlockDetected}, want: true},
		{name: "unique violation", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}},
		{name: "statement timeout", err: &pgconn.PgError{Code: pgerrcode.QueryCanceled}},
		{name: "context cancellation", err: context.Canceled},
		{name: "nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Retryable(tc.err); got != tc.want {
				t.Fatalf("Retryable(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}
