package bootstrap

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

type (
	postgresConnectFunc func(context.Context, postgres.Options) (*postgres.Pool, error)
	startupDelayFunc    func(int) time.Duration
	startupSleepFunc    func(context.Context, time.Duration) error
)

func initPostgresWithRetry(ctx context.Context, cfg config.PostgresConfig) (*postgres.Pool, error) {
	return initPostgresWithRetryFunc(ctx, cfg, postgres.New, fullJitterDelay, sleepWithContext)
}

func initPostgresWithRetryFunc(
	ctx context.Context,
	cfg config.PostgresConfig,
	connect postgresConnectFunc,
	delayFor startupDelayFunc,
	sleep startupSleepFunc,
) (*postgres.Pool, error) {
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

		pg, err := connect(ctx, options)
		if err == nil {
			return pg, nil
		}

		lastErr = err
		if !shouldRetryPostgresStartup(err, attempt) {
			break
		}

		delay := delayFor(attempt)
		if err := sleep(ctx, delay); err != nil {
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
	backoff = min(backoff, startupRetryMaxDelay)
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
		return fmt.Errorf("%w: %s aborted before probe: %w", errDependencyInit, stage, err)
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

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("sleep canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
