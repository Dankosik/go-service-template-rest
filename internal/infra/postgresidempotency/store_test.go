package postgresidempotency

import (
	"context"
	"testing"
	"time"
)

func TestClassificationBudgetDoesNotReset(t *testing.T) {
	deadline := time.Now().Add(time.Second)
	parent, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()
	first, firstCancel, firstOwn := classificationContext(parent, 2*time.Second)
	defer firstCancel()
	second, secondCancel, secondOwn := classificationContext(first, 2*time.Second)
	defer secondCancel()

	if firstOwn || secondOwn {
		t.Fatalf("classification contexts own budget = %t/%t, want false/false", firstOwn, secondOwn)
	}
	firstDeadline, firstOK := first.Deadline()
	secondDeadline, secondOK := second.Deadline()
	if !firstOK || !secondOK || !firstDeadline.Equal(deadline) || !secondDeadline.Equal(deadline) {
		t.Fatalf("classification deadlines = %v/%v, want parent deadline %v", firstDeadline, secondDeadline, deadline)
	}
}
