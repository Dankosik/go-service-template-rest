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
	}, func(context.Context, pgx.Tx) error { return nil }, new(contextWatcherMark))
	if errors.Is(err, context.Canceled) {
		t.Fatalf("runInTx() error = %v, did not want caller cancellation", err)
	}
	if !errors.Is(err, serverErr) || !errors.Is(err, cleanupErr) {
		t.Fatalf("runInTx() error = %v, want server and cleanup errors", err)
	}
	if tx.rollbackCount != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCount)
	}
	if tx.rollbackContextCanceled || !tx.rollbackContextHasDone {
		t.Fatal("rollback context = canceled or unbounded, want bounded non-cancelled cleanup context")
	}
}

func TestRunInTxAttributesMarkedCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverErr := &pgconn.PgError{Code: pgerrcode.QueryCanceled}
	marker := new(contextWatcherMark)
	marker.canceled.Store(true)
	tx := new(trackedTx)

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
		name        string
		err         error
		wantUnknown bool
	}{
		{name: "server rejection", err: &pgconn.PgError{Code: "23505"}},
		{name: "transaction resolution unknown", err: &pgconn.PgError{Code: "08007"}, wantUnknown: true},
		{name: "statement completion unknown", err: &pgconn.PgError{Code: "40003"}, wantUnknown: true},
		{name: "server shutdown during commit", err: &pgconn.PgError{Code: pgerrcode.AdminShutdown}, wantUnknown: true},
		{name: "commit cancellation", err: &pgconn.PgError{Code: pgerrcode.QueryCanceled}, wantUnknown: true},
		{name: "serialization rollback", err: &pgconn.PgError{Code: pgerrcode.SerializationFailure}},
		{name: "commit returned rollback", err: pgx.ErrTxCommitRollback},
		{name: "request was not sent", err: safeToRetryError{}},
		{name: "wrapped request was not sent", err: errors.Join(errors.New("commit"), safeToRetryError{})},
		{name: "opaque transport failure", err: errors.New("connection lost"), wantUnknown: true},
		{name: "context deadline after send", err: context.DeadlineExceeded, wantUnknown: true},
		{name: "nil", err: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ClassifyCommitError(tc.err)
			if errors.Is(got, ErrCommitUnknown) != tc.wantUnknown {
				t.Fatalf("ClassifyCommitError(%v) = %v, unknown=%t want %t", tc.err, got, errors.Is(got, ErrCommitUnknown), tc.wantUnknown)
			}
			if tc.err != nil && !errors.Is(got, tc.err) {
				t.Fatalf("ClassifyCommitError(%v) = %v, want original cause", tc.err, got)
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
