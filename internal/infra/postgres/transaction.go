package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// InTx runs fn inside one transaction, committing when it returns nil and
// rolling back otherwise.
//
// This is the seam that keeps a service from reaching for PGX to compose two
// repository calls atomically. fn receives a pgx.Tx, which satisfies Querier, so a
// method written against Querier works inside and outside a transaction. The
// connection comes from Acquire, under the same acquire budget as anything else.
func (p *Pool) InTx(ctx context.Context, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	if p == nil {
		return fmt.Errorf("%w: postgres pool is nil", ErrConfig)
	}
	if fn == nil {
		return fmt.Errorf("%w: transaction function is required", ErrConfig)
	}

	conn, err := p.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	marker := &contextWatcherMark{}
	p.contextWatcherMarks.Store(conn.Conn().PgConn(), marker)
	defer p.contextWatcherMarks.Delete(conn.Conn().PgConn())

	tx, err := conn.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransaction, err)
	}

	commit := p.commitTx
	if commit == nil {
		commit = commitTx
	}
	if err := runInTx(ctx, tx, fn, commit, marker); err != nil {
		return fmt.Errorf("%w: %w", ErrTransaction, err)
	}
	return nil
}

func runInTx(
	ctx context.Context,
	tx pgx.Tx,
	fn func(pgx.Tx) error,
	commit func(context.Context, pgx.Tx) error,
	marker *contextWatcherMark,
) (err error) {
	defer func() {
		rollbackErr := tx.Rollback(context.WithoutCancel(ctx))
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			if err == nil {
				err = rollbackErr
			} else {
				err = errors.Join(err, rollbackErr)
			}
		}
		if marker != nil && marker.canceled.Load() && ctx.Err() != nil && err != nil {
			err = errors.Join(ctx.Err(), err)
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := commit(ctx, tx); err != nil {
		if commitDefinitelyFailed(err) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrCommitUnknown, err)
	}
	return nil
}

func commitTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func commitDefinitelyFailed(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code != pgerrcode.TransactionResolutionUnknown &&
			pgErr.Code != pgerrcode.StatementCompletionUnknown
	}
	return errors.Is(err, pgx.ErrTxCommitRollback) || pgconn.SafeToRetry(err)
}

// Retryable reports whether err is a PostgreSQL failure that the same request
// could succeed at if it ran again.
//
// There is deliberately no retry loop here: whether a retry is safe depends on
// what the caller already did — a serialization failure in a read-only query is
// free to retry, the same failure after an outbound side effect is not.
func Retryable(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return false
	}
	switch pgErr.Code {
	case pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected:
		return true
	default:
		return false
	}
}
