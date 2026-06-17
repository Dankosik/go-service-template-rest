package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

func TestStartupProbeHelperBasics(t *testing.T) {
	t.Parallel()

	t.Run("shouldRetryPostgresStartup", func(t *testing.T) {
		t.Parallel()

		if shouldRetryPostgresStartup(postgres.ErrConnect, postgresStartupAttempts) {
			t.Fatal("shouldRetryPostgresStartup() = true at last attempt, want false")
		}
		if !shouldRetryPostgresStartup(postgres.ErrHealthcheck, 1) {
			t.Fatal("shouldRetryPostgresStartup() = false, want true")
		}
		if shouldRetryPostgresStartup(errors.New("other"), 1) {
			t.Fatal("shouldRetryPostgresStartup() = true for unrelated error, want false")
		}
	})

	t.Run("fullJitterDelay bounded", func(t *testing.T) {
		t.Parallel()

		d := fullJitterDelay(1)
		if d < 0 || d > startupRetryBaseDelay {
			t.Fatalf("fullJitterDelay(1) = %s, want in [0,%s]", d, startupRetryBaseDelay)
		}
		d = fullJitterDelay(10)
		if d < 0 || d > startupRetryMaxDelay {
			t.Fatalf("fullJitterDelay(10) = %s, want in [0,%s]", d, startupRetryMaxDelay)
		}
	})

	t.Run("withStageBudget clamps to parent deadline", func(t *testing.T) {
		t.Parallel()

		parent, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		child, childCancel := withStageBudget(parent, time.Second)
		defer childCancel()
		if _, ok := child.Deadline(); !ok {
			t.Fatal("child context has no deadline")
		}
	})

	t.Run("ensureRemainingStartupBudget", func(t *testing.T) {
		t.Parallel()

		if err := ensureRemainingStartupBudget(context.Background(), time.Second, "stage"); err != nil {
			t.Fatalf("ensureRemainingStartupBudget(no deadline) error = %v, want nil", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if !errors.Is(ensureRemainingStartupBudget(ctx, time.Second, "stage"), context.Canceled) {
			t.Fatal("ensureRemainingStartupBudget(canceled) did not return context.Canceled")
		}
		shortCtx, shortCancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer shortCancel()
		if err := ensureRemainingStartupBudget(shortCtx, time.Second, "stage"); err == nil {
			t.Fatal("ensureRemainingStartupBudget() error = nil, want non-nil")
		}
	})
}

func TestSleepWithContext(t *testing.T) {
	t.Parallel()

	if err := sleepWithContext(context.Background(), 0); err != nil {
		t.Fatalf("sleepWithContext(0) err = %v, want nil", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(sleepWithContext(ctx, time.Second), context.Canceled) {
		t.Fatal("sleepWithContext(canceled) did not return context.Canceled")
	}
}
