package postgresjobs

import (
	"context"
	"errors"
	"testing"
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
