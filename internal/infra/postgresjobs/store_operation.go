package postgresjobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Session) withOperation(
	ctx context.Context,
	accessMode pgx.TxAccessMode,
	operation func(context.Context, *sqlcgen.Queries) error,
) (err error) {
	if !s.valid() {
		return fmt.Errorf("%w: Session is required", ErrConfig)
	}
	if operation == nil {
		return fmt.Errorf("%w: operation is required", ErrConfig)
	}

	setupCtx, setupCancel := context.WithTimeout(ctx, s.store.operationTimeout)
	defer setupCancel()

	tx, err := s.conn.BeginTx(setupCtx, pgx.TxOptions{AccessMode: accessMode})
	if err != nil {
		return s.classifyOperationError(ctx, setupCtx, err)
	}
	operationCtx := setupCtx
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), s.store.operationTimeout)
		defer cleanupCancel()
		rollbackErr := tx.Rollback(cleanupCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) && err == nil {
			err = s.classifyOperationError(ctx, operationCtx, rollbackErr)
		}
	}()

	timeoutValue := postgres.RuntimeParamMilliseconds(operationTimerTimeout(s.store.operationTimeout, s.store.statementTimeout))
	if _, err := tx.Exec(
		setupCtx,
		"SELECT set_config('statement_timeout', $1, true), set_config('lock_timeout', $1, true)",
		timeoutValue,
	); err != nil {
		return s.classifyOperationError(ctx, setupCtx, err)
	}
	operationCtx, cancel := context.WithTimeout(ctx, s.store.operationTimeout)
	defer cancel()
	if err := operation(operationCtx, s.queries.WithTx(tx)); err != nil {
		return s.classifyOperationError(ctx, operationCtx, err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return s.classifyOperationError(ctx, operationCtx, err)
	}
	return nil
}

func operationTimerTimeout(operationTimeout, statementTimeout time.Duration) time.Duration {
	timeout := min(operationTimeout, statementTimeout)
	return timeout - min(timeout/10, 10*time.Millisecond)
}

func (s *Session) classifyOperationError(parentCtx, operationCtx context.Context, err error) error {
	if err == nil {
		return nil
	}
	connectionClosed := s != nil && s.conn != nil && s.conn.Conn() != nil && s.conn.Conn().IsClosed()
	classified := classifyOperationError(parentCtx, operationCtx, connectionClosed, err)
	if errors.Is(classified, ErrSessionTerminal) && s != nil {
		s.terminal = true
	}
	return classified
}

func classifyOperationError(parentCtx, operationCtx context.Context, connectionClosed bool, err error) error {
	if parentCtx != nil && parentCtx.Err() != nil {
		return fmt.Errorf("postgres jobs operation: %w", parentCtx.Err())
	}
	if pgconn.Timeout(err) {
		return fmt.Errorf("%w: %w", ErrOperationTimeout, err)
	}
	if postgresError, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch postgresError.Code {
		case pgerrcode.QueryCanceled, pgerrcode.LockNotAvailable:
			return fmt.Errorf("%w: %w", ErrOperationTimeout, err)
		}
	}
	if operationCtx != nil && operationCtx.Err() != nil {
		return fmt.Errorf("%w: %w", ErrOperationTimeout, operationCtx.Err())
	}
	if connectionClosed {
		return fmt.Errorf("%w: %w", ErrSessionTerminal, err)
	}
	return err
}
