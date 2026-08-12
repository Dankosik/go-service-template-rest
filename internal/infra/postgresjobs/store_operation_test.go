package postgresjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestStoreOperationErrorClassification(t *testing.T) {
	baseErr := errors.New("database failed")

	parentCtx, cancelParent := context.WithCancel(context.Background())
	cancelParent()
	if err := classifyOperationError(parentCtx, context.Background(), false, baseErr); !errors.Is(err, context.Canceled) {
		t.Fatalf("caller cancellation error = %v", err)
	}

	operationCtx, cancelOperation := context.WithCancel(context.Background())
	cancelOperation()
	if err := classifyOperationError(context.Background(), operationCtx, false, baseErr); !errors.Is(err, ErrOperationTimeout) {
		t.Fatalf("operation timeout error = %v", err)
	}

	if err := classifyOperationError(context.Background(), context.Background(), true, baseErr); !errors.Is(err, ErrSessionTerminal) {
		t.Fatalf("terminal Session error = %v", err)
	}
	if err := classifyOperationError(context.Background(), context.Background(), false, baseErr); !errors.Is(err, baseErr) {
		t.Fatalf("ordinary database error = %v", err)
	}
}

func TestStoreOperationErrorClassificationPrefersOperationOutcomeToClosedSession(t *testing.T) {
	parentCtx, cancelParent := context.WithCancel(context.Background())
	cancelParent()
	operationCtx, cancelOperation := context.WithCancel(context.Background())
	cancelOperation()

	for _, test := range []struct {
		name         string
		parentCtx    context.Context
		operationCtx context.Context
		err          error
		want         error
	}{
		{"parent cancellation", parentCtx, context.Background(), errors.New("connection closed"), context.Canceled},
		{"operation timeout", context.Background(), operationCtx, errors.New("connection closed"), ErrOperationTimeout},
		{"lock timeout", context.Background(), context.Background(), &pgconn.PgError{Code: pgerrcode.LockNotAvailable}, ErrOperationTimeout},
		{"statement timeout", context.Background(), context.Background(), &pgconn.PgError{Code: pgerrcode.QueryCanceled}, ErrOperationTimeout},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := classifyOperationError(test.parentCtx, test.operationCtx, true, test.err); !errors.Is(err, test.want) {
				t.Fatalf("classifyOperationError() error = %v, want %v", err, test.want)
			}
		})
	}
}
