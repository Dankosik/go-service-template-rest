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

type OutboxConfig struct {
	Enabled      bool          `koanf:"enabled"`
	PollInterval time.Duration `koanf:"poll_interval"`
	// BatchSize is how many events one lease covers. It trades round trips per
	// event against peak relay memory, which is this many stored envelopes.
	BatchSize int `koanf:"batch_size"`
	// PublishConcurrency is how many events of a batch may be in the publisher
	// at once. The claim query hands out at most one ready event per ordering
	// key, so concurrency never reorders a key.
	PublishConcurrency  int           `koanf:"publish_concurrency"`
	PublishTimeout      time.Duration `koanf:"publish_timeout"`
	LeaseDuration       time.Duration `koanf:"lease_duration"`
	MaxAttempts         int           `koanf:"max_attempts"`
	RetryBase           time.Duration `koanf:"retry_base"`
	RetryMax            time.Duration `koanf:"retry_max"`
	ObservationInterval time.Duration `koanf:"observation_interval"`
	CleanupInterval     time.Duration `koanf:"cleanup_interval"`
	PublishedRetention  time.Duration `koanf:"published_retention"`
	CleanupBatchSize    int           `koanf:"cleanup_batch_size"`
	DrainTimeout        time.Duration `koanf:"drain_timeout"`
}

func outboxDefaults() map[string]any {
	return map[string]any{
		"outbox.enabled":              false,
		"outbox.poll_interval":        "500ms",
		"outbox.batch_size":           500,
		"outbox.publish_concurrency":  16,
		"outbox.publish_timeout":      "10s",
		"outbox.lease_duration":       "30s",
		"outbox.max_attempts":         10,
		"outbox.retry_base":           "1s",
		"outbox.retry_max":            "5m",
		"outbox.observation_interval": "5s",
		"outbox.cleanup_interval":     "1m",
		"outbox.published_retention":  "168h",
		"outbox.cleanup_batch_size":   1000,
		"outbox.drain_timeout":        "20s",
	}
}

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
		if err := validateIntRange(bound.name, bound.value, 1, bound.high); err != nil {
			return err
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
