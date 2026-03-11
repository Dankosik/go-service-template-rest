package bootstrap

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/Dankosik/search-service/internal/config"
	"github.com/Dankosik/search-service/internal/infra/postgres"
)

func TestStartupProbeHelperBasics(t *testing.T) {
	t.Parallel()

	t.Run("shouldRetryPostgresStartup", func(t *testing.T) {
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
		parent, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		child, childCancel := withStageBudget(parent, time.Second)
		defer childCancel()
		if _, ok := child.Deadline(); !ok {
			t.Fatal("child context has no deadline")
		}
	})

	t.Run("ensureRemainingStartupBudget", func(t *testing.T) {
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

	t.Run("redisStartupMode", func(t *testing.T) {
		if got := redisStartupMode(" STORE "); got != "store" {
			t.Fatalf("redisStartupMode(STORE) = %q, want store", got)
		}
		if got := redisStartupMode("cache"); got != "cache" {
			t.Fatalf("redisStartupMode(cache) = %q, want cache", got)
		}
		if got := redisStartupMode("unexpected"); got != "cache" {
			t.Fatalf("redisStartupMode(unexpected) = %q, want cache", got)
		}
	})

	t.Run("shouldRetryStartupProbe", func(t *testing.T) {
		if shouldRetryStartupProbe(nil, 3, 3) {
			t.Fatal("shouldRetryStartupProbe() = true at max attempts, want false")
		}
		if shouldRetryStartupProbe(context.Canceled, 1, 3) {
			t.Fatal("shouldRetryStartupProbe() = true for canceled, want false")
		}
		if !shouldRetryStartupProbe(errors.New("boom"), 1, 3) {
			t.Fatal("shouldRetryStartupProbe() = false for retryable error, want true")
		}
	})
}

func TestProbeWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("single attempt", func(t *testing.T) {
		calls := 0
		err := probeWithRetry(context.Background(), 1, func(context.Context) error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatalf("probeWithRetry() error = %v, want nil", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	t.Run("retry then success", func(t *testing.T) {
		calls := 0
		err := probeWithRetry(context.Background(), 3, func(context.Context) error {
			calls++
			if calls < 2 {
				return errors.New("transient")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("probeWithRetry() error = %v, want nil", err)
		}
		if calls != 2 {
			t.Fatalf("calls = %d, want 2", calls)
		}
	})

	t.Run("ctx canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := probeWithRetry(ctx, 3, func(context.Context) error { return nil })
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("probeWithRetry() err = %v, want %v", err, context.Canceled)
		}
	})
}

func TestProbeTCPAndDependencyWrappers(t *testing.T) {
	t.Parallel()

	t.Run("empty address", func(t *testing.T) {
		err := probeTCPDependency(context.Background(), "  ", 10*time.Millisecond)
		if err == nil {
			t.Fatal("probeTCPDependency() error = nil, want non-nil")
		}
		if !errors.Is(err, config.ErrDependencyInit) {
			t.Fatalf("err = %v, want wrapped %v", err, config.ErrDependencyInit)
		}
	})

	t.Run("tcp success", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Listen() error = %v", err)
		}
		defer func() {
			if closeErr := ln.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
				t.Errorf("ln.Close() error = %v", closeErr)
			}
		}()
		go func() {
			conn, acceptErr := ln.Accept()
			if acceptErr == nil {
				_ = conn.Close()
			}
		}()

		err = probeTCPDependency(context.Background(), ln.Addr().String(), 100*time.Millisecond)
		if err != nil {
			t.Fatalf("probeTCPDependency() error = %v, want nil", err)
		}
	})

	t.Run("redis wrapper", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Listen() error = %v", err)
		}
		defer func() {
			if closeErr := ln.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
				t.Errorf("ln.Close() error = %v", closeErr)
			}
		}()
		go func() {
			conn, acceptErr := ln.Accept()
			if acceptErr == nil {
				_ = conn.Close()
			}
		}()
		err = probeRedisWithContext(context.Background(), config.RedisConfig{Addr: ln.Addr().String(), DialTimeout: 100 * time.Millisecond})
		if err != nil {
			t.Fatalf("probeRedisWithContext() error = %v, want nil", err)
		}
	})

	t.Run("mongo invalid uri", func(t *testing.T) {
		err := probeMongoWithContext(context.Background(), config.MongoConfig{URI: "::bad-uri"})
		if err == nil {
			t.Fatal("probeMongoWithContext() error = nil, want non-nil")
		}
		if !errors.Is(err, config.ErrDependencyInit) {
			t.Fatalf("err = %v, want wrapped %v", err, config.ErrDependencyInit)
		}
	})

	t.Run("sleepWithContext", func(t *testing.T) {
		if err := sleepWithContext(context.Background(), 0); err != nil {
			t.Fatalf("sleepWithContext(0) err = %v, want nil", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if !errors.Is(sleepWithContext(ctx, time.Second), context.Canceled) {
			t.Fatal("sleepWithContext(canceled) did not return context.Canceled")
		}
	})
}
