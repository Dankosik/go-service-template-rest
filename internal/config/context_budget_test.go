package config

import (
	"context"
	"testing"
	"time"
)

func TestWithContextBudgetRespectsShorterParentDeadline(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	ctx, budgetCancel := withContextBudget(parent, time.Hour)
	defer budgetCancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("withContextBudget() context has no deadline, want parent deadline")
	}
	parentDeadline, ok := parent.Deadline()
	if !ok {
		t.Fatal("parent context has no deadline")
	}
	if deadline.After(parentDeadline) {
		t.Fatalf("budget deadline %v extends past parent deadline %v", deadline, parentDeadline)
	}
}

func TestWithContextBudgetExpiredParentDeadlineStaysExpired(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	ctx, budgetCancel := withContextBudget(parent, time.Hour)
	defer budgetCancel()

	if ctx.Err() == nil {
		t.Fatal("withContextBudget() context err = nil, want expired parent deadline to propagate")
	}
}
