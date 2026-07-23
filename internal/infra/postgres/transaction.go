package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrTransaction = errors.New("postgres transaction")

// InTx runs fn once inside one pgx transaction: commit on nil, rollback on error.
// It performs no retries; keep network calls and other external side effects out of fn.
// Repositories that use generated sqlc queries bind them to the transaction with
// sqlcgen.New(...).WithTx(tx) inside fn. Remove InTx only when the service has
// no transactional use case at all.
func (p *Pool) InTx(ctx context.Context, fn func(pgx.Tx) error) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("%w: postgres pool is nil", ErrTransaction)
	}
	if fn == nil {
		return fmt.Errorf("%w: callback is nil", ErrTransaction)
	}

	if err := pgx.BeginFunc(ctx, p.pool, fn); err != nil {
		return fmt.Errorf("%w: %w", ErrTransaction, err)
	}
	return nil
}
