package postgresidempotency

import (
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestAcquireLockUnavailable(t *testing.T) {
	t.Parallel()
	if !isLockUnavailable(&pgconn.PgError{Code: pgerrcode.LockNotAvailable}) {
		t.Fatal("lock-not-available PostgreSQL error was not classified")
	}
	if isLockUnavailable(errors.New("different error")) {
		t.Fatal("non-PostgreSQL error was classified as a row-lock conflict")
	}
}
