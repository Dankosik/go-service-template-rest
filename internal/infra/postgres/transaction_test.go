package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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
