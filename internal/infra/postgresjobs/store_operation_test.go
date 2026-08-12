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
	for _, test := range []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "parent cancellation",
			run: func() error {
				parentCtx, cancel := context.WithCancel(context.Background())
				cancel()
				return classifyOperationError(parentCtx, context.Background(), true, errors.New("connection closed"))
			},
			want: context.Canceled,
		},
		{
			name: "operation timeout",
			run: func() error {
				operationCtx, cancel := context.WithCancel(context.Background())
				cancel()
				return classifyOperationError(context.Background(), operationCtx, true, errors.New("connection closed"))
			},
			want: ErrOperationTimeout,
		},
		{
			name: "lock timeout",
			run: func() error {
				return classifyOperationError(context.Background(), context.Background(), true, &pgconn.PgError{Code: pgerrcode.LockNotAvailable})
			},
			want: ErrOperationTimeout,
		},
		{
			name: "statement timeout",
			run: func() error {
				return classifyOperationError(context.Background(), context.Background(), true, &pgconn.PgError{Code: pgerrcode.QueryCanceled})
			},
			want: ErrOperationTimeout,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, test.want) {
				t.Fatalf("classifyOperationError() error = %v, want %v", err, test.want)
			}
		})
	}
}
