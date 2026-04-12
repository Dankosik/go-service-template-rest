package bootstrap

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

func initPostgresWithRetry(ctx context.Context, cfg config.PostgresConfig) (*postgres.Pool, error) {
	options := postgres.Options{
		DSN:                cfg.DSN,
		ConnectTimeout:     cfg.ConnectTimeout,
		HealthcheckTimeout: cfg.HealthcheckTimeout,
		MaxOpenConns:       cfg.MaxOpenConns,
		MaxIdleConns:       cfg.MaxIdleConns,
		ConnMaxLifetime:    cfg.ConnMaxLifetime,
	}

	var lastErr error
	for attempt := 1; attempt <= postgresStartupAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: postgres init canceled: %w", errDependencyInit, err)
		}

		pg, err := postgres.New(ctx, options)
		if err == nil {
			return pg, nil
		}

		lastErr = err
		if !shouldRetryPostgresStartup(err, attempt) {
			break
		}

		delay := fullJitterDelay(attempt)
		if err := sleepWithContext(ctx, delay); err != nil {
			return nil, fmt.Errorf("%w: postgres retry wait canceled: %w", errDependencyInit, err)
		}
	}

	return nil, fmt.Errorf("%w: postgres init failed after retries: %w", errDependencyInit, lastErr)
}

func shouldRetryPostgresStartup(err error, attempt int) bool {
	if attempt >= postgresStartupAttempts {
		return false
	}
	return errors.Is(err, postgres.ErrConnect) || errors.Is(err, postgres.ErrHealthcheck)
}

func fullJitterDelay(attempt int) time.Duration {
	backoff := startupRetryBaseDelay << (attempt - 1)
	if backoff > startupRetryMaxDelay {
		backoff = startupRetryMaxDelay
	}
	if backoff <= 0 {
		return 0
	}

	jitter, err := crand.Int(crand.Reader, big.NewInt(int64(backoff)+1))
	if err != nil {
		return backoff
	}
	return time.Duration(jitter.Int64())
}

func withStageBudget(parent context.Context, stageBudget time.Duration) (context.Context, context.CancelFunc) {
	if stageBudget <= 0 {
		return context.WithCancel(parent) // #nosec G118 -- cancel function is returned to caller.
	}
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < stageBudget {
			stageBudget = remaining
		}
	}
	if stageBudget <= 0 {
		return context.WithCancel(parent) // #nosec G118 -- cancel function is returned to caller.
	}
	return context.WithTimeout(parent, stageBudget) // #nosec G118 -- cancel function is returned to caller.
}

func ensureRemainingStartupBudget(ctx context.Context, minRemaining time.Duration, stage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil
	}
	remaining := time.Until(deadline)
	if remaining < minRemaining {
		return fmt.Errorf(
			"%w: %s aborted due to low remaining startup budget (%s < %s)",
			errDependencyInit,
			stage,
			remaining,
			minRemaining,
		)
	}
	return nil
}

func probeRedisWithContext(ctx context.Context, cfg config.RedisConfig) error {
	timeout := cfg.DialTimeout
	if timeout <= 0 {
		timeout = redisProbeBudget
	}
	return probeTCPDependency(ctx, cfg.Addr, timeout)
}

func probeRedisWithRetry(ctx context.Context, cfg config.RedisConfig) error {
	return probeWithRetry(ctx, redisStoreProbeAttempts, func(probeCtx context.Context) error {
		return probeRedisWithContext(probeCtx, cfg)
	})
}

func probeMongoWithContext(ctx context.Context, cfg config.MongoConfig) error {
	addr, err := config.MongoProbeAddress(cfg.URI)
	if err != nil {
		return fmt.Errorf("%w: resolve mongo probe address: %w", errDependencyInit, err)
	}
	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = mongoProbeBudget
	}
	return probeTCPDependency(ctx, addr, timeout)
}

func probeMongoWithRetry(ctx context.Context, cfg config.MongoConfig) error {
	return probeWithRetry(ctx, mongoProbeAttempts, func(probeCtx context.Context) error {
		return probeMongoWithContext(probeCtx, cfg)
	})
}

func probeWithRetry(ctx context.Context, maxAttempts int, probe func(context.Context) error) error {
	if maxAttempts <= 1 {
		return probe(ctx)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := probe(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if !shouldRetryStartupProbe(err, attempt, maxAttempts) {
			break
		}

		delay := fullJitterDelay(attempt)
		if waitErr := sleepWithContext(ctx, delay); waitErr != nil {
			return waitErr
		}
	}

	return lastErr
}

func shouldRetryStartupProbe(err error, attempt int, maxAttempts int) bool {
	if attempt >= maxAttempts {
		return false
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

func probeTCPDependency(ctx context.Context, address string, timeout time.Duration) error {
	trimmedAddress := strings.TrimSpace(address)
	if trimmedAddress == "" {
		return fmt.Errorf("%w: empty probe address", errDependencyInit)
	}

	dialCtx, dialCancel := withStageBudget(ctx, timeout)
	defer dialCancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", trimmedAddress)
	if err != nil {
		return fmt.Errorf("%w: dial %s: %w", errDependencyInit, trimmedAddress, err)
	}
	_ = conn.Close()
	return nil
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
