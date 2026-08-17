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
	name string,
	accessMode pgx.TxAccessMode,
	operation func(context.Context, *sqlcgen.Queries) error,
) (err error) {
	if !s.valid() {
		return fmt.Errorf("%w: Session is required", ErrConfig)
	}
	if operation == nil {
		return fmt.Errorf("%w: operation is required", ErrConfig)
	}
	startedAt := time.Now()
	defer func() {
		s.store.recordOperation(context.WithoutCancel(ctx), name, err, time.Since(startedAt))
	}()

	operationCtx, cancel := context.WithTimeout(ctx, s.store.operationTimeout)
	defer cancel()

	tx, err := s.conn.BeginTx(operationCtx, pgx.TxOptions{AccessMode: accessMode})
	if err != nil {
		return s.classifyOperationError(ctx, operationCtx, err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), s.store.operationTimeout)
		defer cleanupCancel()
		rollbackErr := tx.Rollback(cleanupCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) && err == nil {
			err = s.classifyOperationError(ctx, operationCtx, rollbackErr)
		}
	}()

	deadline, _ := operationCtx.Deadline()
	timerTimeout := operationTimerTimeout(time.Until(deadline), s.store.statementTimeout)
	if timerTimeout == 0 {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("postgres jobs operation: %w", err)
		}
		return fmt.Errorf("%w: transaction setup exhausted operation timeout", ErrOperationTimeout)
	}
	timeoutValue := postgres.RuntimeParamMilliseconds(timerTimeout)
	if _, err := tx.Exec(
		operationCtx,
		"SELECT set_config('statement_timeout', $1, true), set_config('lock_timeout', $1, true)",
		timeoutValue,
	); err != nil {
		return s.classifyOperationError(ctx, operationCtx, err)
	}
	if err := operation(operationCtx, s.queries.WithTx(tx)); err != nil {
		return s.classifyOperationError(ctx, operationCtx, err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return s.classifyOperationError(ctx, operationCtx, postgres.ClassifyCommitError(err))
	}
	return nil
}

func operationTimerTimeout(operationTimeout, statementTimeout time.Duration) time.Duration {
	timeout := min(operationTimeout, statementTimeout)
	if timeout <= 10*time.Millisecond {
		return 0
	}
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
	var classified error
	if parentCtx != nil && parentCtx.Err() != nil {
		classified = errors.Join(fmt.Errorf("postgres jobs operation: %w", parentCtx.Err()), err)
	} else {
		if pgconn.Timeout(err) {
			classified = fmt.Errorf("%w: %w", ErrOperationTimeout, err)
		} else if postgresError, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch postgresError.Code {
			case pgerrcode.QueryCanceled, pgerrcode.LockNotAvailable:
				classified = fmt.Errorf("%w: %w", ErrOperationTimeout, err)
			}
		}
		if classified == nil && operationCtx != nil && operationCtx.Err() != nil {
			classified = errors.Join(fmt.Errorf("%w: %w", ErrOperationTimeout, operationCtx.Err()), err)
		}
	}
	if classified == nil {
		classified = err
	}
	if connectionClosed {
		classified = errors.Join(ErrSessionTerminal, classified)
	}
	return classified
}
