package config

// profile:outbox-postgres:start

import (
	"fmt"
	"time"
)

// Copies of the outbox ceilings, restated so an operator is rejected at load time
// instead of at relay startup. postgresoutbox owns the values and the reasoning;
// its ceiling block in relay_config.go says why depguard makes this a copy rather
// than an import, and cmd/outbox-relay/internal/bootstrap holds the two sides
// together.
//
// A relay budget added to validateOutbox below must be added to
// ValidateRelayConfig too. Nothing here can check that; the composition root's
// walk over OutboxConfig is what fails when only one side learns about it.
//
// The lease relation below is this package's own, not a copy: it also charges
// the acquire and statement budgets a RelayConfig never carries, so it stays
// the stricter of the two by design and is pinned to nothing.
const (
	outboxPublisherJoinTimeout  = time.Second
	outboxMaxBatchSize          = 1000
	outboxMaxPublishConcurrency = 256
	outboxMaxAttempts           = 100
	outboxMaxCleanupBatchSize   = 10_000
)

func validateOutbox(cfg OutboxConfig, postgres PostgresConfig) error {
	if cfg.Enabled && !postgres.Enabled {
		return fmt.Errorf("%w: outbox.enabled requires postgres.enabled", ErrValidate)
	}
	// Claim, finalization, and periodic maintenance are separate statements that
	// share the pool with health checks and, in an API process, with request
	// traffic. A single-connection pool leaves none of them any headroom.
	if cfg.Enabled && postgres.MaxOpenConns < 2 {
		return fmt.Errorf("%w: outbox.enabled requires postgres.max_open_conns >= 2", ErrValidate)
	}
	for _, duration := range []struct {
		name  string
		value time.Duration
	}{
		{name: "outbox.poll_interval", value: cfg.PollInterval},
		{name: "outbox.publish_timeout", value: cfg.PublishTimeout},
		{name: "outbox.lease_duration", value: cfg.LeaseDuration},
		{name: "outbox.retry_base", value: cfg.RetryBase},
		{name: "outbox.retry_max", value: cfg.RetryMax},
		{name: "outbox.observation_interval", value: cfg.ObservationInterval},
		{name: "outbox.cleanup_interval", value: cfg.CleanupInterval},
		{name: "outbox.published_retention", value: cfg.PublishedRetention},
		{name: "outbox.drain_timeout", value: cfg.DrainTimeout},
	} {
		if duration.value <= 0 {
			return fmt.Errorf("%w: %s must be positive", ErrValidate, duration.name)
		}
	}
	for _, bound := range []struct {
		name string
		value,
		high int
	}{
		{name: "outbox.max_attempts", value: cfg.MaxAttempts, high: outboxMaxAttempts},
		{name: "outbox.batch_size", value: cfg.BatchSize, high: outboxMaxBatchSize},
		{name: "outbox.publish_concurrency", value: cfg.PublishConcurrency, high: outboxMaxPublishConcurrency},
		{name: "outbox.cleanup_batch_size", value: cfg.CleanupBatchSize, high: outboxMaxCleanupBatchSize},
	} {
		if bound.value < 1 || bound.value > bound.high {
			return fmt.Errorf("%w: %s must be in range [1,%d]", ErrValidate, bound.name, bound.high)
		}
	}
	if cfg.RetryMax < cfg.RetryBase {
		return fmt.Errorf("%w: outbox.retry_max must be >= outbox.retry_base", ErrValidate)
	}
	if !durationExceedsSum(
		cfg.LeaseDuration,
		cfg.PublishTimeout,
		outboxPublisherJoinTimeout,
		postgres.AcquireTimeout,
		postgres.StatementTimeout,
	) {
		return fmt.Errorf(
			"%w: outbox.lease_duration must exceed publish, publisher-join, postgres acquire, and statement budgets",
			ErrValidate,
		)
	}
	return nil
}

func durationExceedsSum(total time.Duration, parts ...time.Duration) bool {
	for _, part := range parts {
		if total <= part {
			return false
		}
		total -= part
	}
	return true
}

// profile:outbox-postgres:end
