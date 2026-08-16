package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type trackedTx struct {
	pgx.Tx

	rollbackContextCanceled bool
	rollbackContextHasDone  bool
	rollbackCount           int
	rollbackErr             error
	commitErr               error
}

func (tx *trackedTx) Commit(context.Context) error { return tx.commitErr }

func (tx *trackedTx) Rollback(ctx context.Context) error {
	tx.rollbackContextCanceled = ctx.Err() != nil
	tx.rollbackContextHasDone = ctx.Done() != nil
	tx.rollbackCount++
	return tx.rollbackErr
}

func TestRunInTxPreservesServerErrorAfterLaterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverErr := &pgconn.PgError{Code: pgerrcode.QueryCanceled}
	cleanupErr := errors.New("rollback cleanup failed")
	tx := &trackedTx{rollbackErr: cleanupErr}

	err := runInTx(ctx, tx, func(pgx.Tx) error {
		cancel()
		return serverErr
	}, func(context.Context, pgx.Tx) error { return nil }, &contextWatcherMark{})
	if errors.Is(err, context.Canceled) {
		t.Fatalf("runInTx() error = %v, did not want caller cancellation", err)
	}
	if !errors.Is(err, serverErr) || !errors.Is(err, cleanupErr) {
		t.Fatalf("runInTx() error = %v, want server and cleanup errors", err)
	}
	if tx.rollbackCount != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCount)
	}
	if tx.rollbackContextCanceled || tx.rollbackContextHasDone {
		t.Fatal("rollback context = canceled or done, want non-cancelled cleanup context")
	}
}

func TestRunInTxAttributesMarkedCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverErr := &pgconn.PgError{Code: pgerrcode.QueryCanceled}
	marker := &contextWatcherMark{}
	marker.canceled.Store(true)
	tx := &trackedTx{}

	err := runInTx(ctx, tx, func(pgx.Tx) error {
		cancel()
		return serverErr
	}, func(context.Context, pgx.Tx) error { return nil }, marker)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, serverErr) {
		t.Fatalf("runInTx() error = %v, want caller cancellation and server error", err)
	}
	if tx.rollbackCount != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCount)
	}
}

type safeToRetryError struct{}

func (safeToRetryError) Error() string     { return "request was not sent" }
func (safeToRetryError) SafeToRetry() bool { return true }

func TestClassifyCommitOutcome(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "server rejection", err: &pgconn.PgError{Code: "23505"}, want: true},
		{name: "transaction resolution unknown", err: &pgconn.PgError{Code: "08007"}},
		{name: "statement completion unknown", err: &pgconn.PgError{Code: "40003"}},
		{name: "commit returned rollback", err: pgx.ErrTxCommitRollback, want: true},
		{name: "request was not sent", err: safeToRetryError{}, want: true},
		{name: "wrapped request was not sent", err: errors.Join(errors.New("commit"), safeToRetryError{}), want: true},
		{name: "opaque transport failure", err: errors.New("connection lost")},
		{name: "context deadline after send", err: context.DeadlineExceeded},
		{name: "nil", err: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := commitDefinitelyFailed(tc.err); got != tc.want {
				t.Fatalf("commitDefinitelyFailed(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

func TestCommitTxWrapsCommitFailure(t *testing.T) {
	want := errors.New("connection lost")
	if err := commitTx(t.Context(), &trackedTx{commitErr: want}); !errors.Is(err, want) {
		t.Fatalf("commitTx() error = %v, want wrapped commit failure", err)
	}
	if err := commitTx(t.Context(), &trackedTx{}); err != nil {
		t.Fatalf("commitTx() error = %v", err)
	}
}

func TestCommitTxClassifiesUnknownOutcome(t *testing.T) {
	err := CommitTx(t.Context(), &trackedTx{commitErr: errors.New("connection lost")})
	if !errors.Is(err, ErrCommitUnknown) {
		t.Fatalf("CommitTx() error = %v, want ErrCommitUnknown", err)
	}
	err = CommitTx(t.Context(), &trackedTx{commitErr: pgx.ErrTxCommitRollback})
	if errors.Is(err, ErrCommitUnknown) || !errors.Is(err, pgx.ErrTxCommitRollback) {
		t.Fatalf("CommitTx(definite rollback) error = %v", err)
	}
}
