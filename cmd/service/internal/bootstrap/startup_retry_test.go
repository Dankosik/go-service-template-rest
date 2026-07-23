package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

func TestStartupTimingAndRetryBasics(t *testing.T) {
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

func TestInitPostgresWithRetryRetriesThenSucceeds(t *testing.T) {
	t.Parallel()

	wantPool := &postgres.Pool{}
	attempts := 0
	sleeps := 0
	cfg := config.PostgresConfig{
		DSN:                "postgres://user:pass@localhost:5432/app?sslmode=disable",
		ConnectTimeout:     time.Second,
		HealthcheckTimeout: time.Second,
		MaxOpenConns:       4,
		ConnMaxLifetime:    time.Minute,
	}

	gotPool, err := initPostgresWithRetryFunc(
		context.Background(),
		cfg,
		func(_ context.Context, opts postgres.Options) (*postgres.Pool, error) {
			attempts++
			if opts.DSN != cfg.DSN || opts.MaxOpenConns != cfg.MaxOpenConns {
				t.Fatalf("postgres options = %+v, want cfg-derived values", opts)
			}
			if attempts == 1 {
				return nil, postgres.ErrConnect
			}
			return wantPool, nil
		},
		func(attempt int) time.Duration {
			if attempt != 1 {
				t.Fatalf("delay attempt = %d, want 1", attempt)
			}
			return 7 * time.Millisecond
		},
		func(_ context.Context, delay time.Duration) error {
			sleeps++
			if delay != 7*time.Millisecond {
				t.Fatalf("sleep delay = %s, want 7ms", delay)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("initPostgresWithRetryFunc() error = %v, want nil", err)
	}
	if gotPool != wantPool {
		t.Fatalf("pool = %p, want %p", gotPool, wantPool)
	}
	if attempts != 2 || sleeps != 1 {
		t.Fatalf("attempts/sleeps = %d/%d, want 2/1", attempts, sleeps)
	}
}
