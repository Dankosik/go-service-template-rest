package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

var ErrTransaction = errors.New("postgres transaction")

// InTx runs fn once with sqlc queries bound to one pgx transaction.
// It performs no retries; keep network calls and other external side effects out of fn.
func (p *Pool) InTx(ctx context.Context, fn func(*sqlcgen.Queries) error) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("%w: postgres pool is nil", ErrTransaction)
	}
	if fn == nil {
		return fmt.Errorf("%w: callback is nil", ErrTransaction)
	}

	if err := pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		return fn(sqlcgen.New(p.pool).WithTx(tx))
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrTransaction, err)
	}
	return nil
}
