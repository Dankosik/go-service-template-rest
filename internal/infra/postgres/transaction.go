package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InTx runs fn inside one transaction, committing when it returns nil and
// rolling back otherwise.
//
// This is the single transaction seam for generated sqlc queries. The caller
// chooses the atomic boundary and binds its generated Queries with WithTx.
func InTx(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("%w: postgres pool is nil", ErrConfig)
	}
	if fn == nil {
		return fmt.Errorf("%w: transaction function is required", ErrConfig)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("%w: acquire connection: %w", ErrTransaction, err)
	}
	defer conn.Release()

	marker := new(contextWatcherMark)
	contextWatcherMarks.Store(conn.Conn().PgConn(), marker)
	defer contextWatcherMarks.Delete(conn.Conn().PgConn())

	tx, err := conn.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransaction, err)
	}

	if err := runInTx(ctx, tx, fn, commitTx, marker); err != nil {
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
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), postgresConnectTimeout)
		defer cancel()
		rollbackErr := tx.Rollback(rollbackCtx)
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
		return ClassifyCommitError(err)
	}
	return nil
}

func commitTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// ClassifyCommitError preserves failures known to have rejected commit and
// marks every other commit response as an unknown durable outcome.
func ClassifyCommitError(err error) error {
	if err == nil || commitDefinitelyFailed(err) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrCommitUnknown, err)
}

func commitDefinitelyFailed(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) ||
			pgerrcode.IsTransactionRollback(pgErr.Code) && pgErr.Code != pgerrcode.StatementCompletionUnknown
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
