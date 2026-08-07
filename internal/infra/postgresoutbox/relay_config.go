package postgresoutbox

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// This block owns the relay's fixed ceilings, and the reasoning about them that
// the two sites restating these values point back to.
//
// [ValidateRelayConfig] checks a RelayConfig against them at construction, and
// internal/config restates them so an operator's settings are rejected at load
// time instead of at startup. That copy is not a shortcut and cannot be removed
// by importing this package: depguard's config_no_runtime_owners rule forbids
// internal/config from importing any repository runtime package, so that
// loading configuration does not link the PostgreSQL adapter into every binary
// that merely reads it. The copy survives only because the composition root —
// cmd/outbox-relay/internal/bootstrap, the one package that wires both — pins
// it from outside; its tests say which value each of them holds.
//
// Those pins deliberately stop short of the lease relation, because the two
// sides check different ones: internal/config also charges the lease for the
// PostgreSQL acquire and statement budgets a RelayConfig cannot see, so it is
// strictly the stricter of the two.
const (
	// PublisherJoinTimeout is the grace a publisher goroutine gets to stop after
	// cancellation before the relay reports it stuck and declares cleanup unsafe.
	PublisherJoinTimeout = time.Second
	// MaxBatchSize bounds RelayConfig.BatchSize, and so the relay's peak memory
	// in stored envelopes.
	MaxBatchSize = 1000
	// MaxPublishConcurrency bounds RelayConfig.PublishConcurrency.
	MaxPublishConcurrency = 256
	// MaxAttempts bounds RelayConfig.MaxAttempts, the cycle attempt count at
	// which an adapter-proven rejection poisons.
	MaxAttempts = 100
	// MaxCleanupBatchSize bounds RelayConfig.CleanupBatchSize.
	MaxCleanupBatchSize = 10_000
)

// RelayConfig is the relay's runtime budget. [ValidateRelayConfig] rejects the
// combinations that cannot hold: notably LeaseDuration must exceed
// PublishTimeout by more than the publisher join bound, because the batch's
// publication deadline is derived from the lease.
type RelayConfig struct {
	// PollInterval bounds how long an idle relay waits before claiming again.
	// An append notification normally arrives first; this is the fallback.
	PollInterval time.Duration
	// BatchSize is how many events one lease covers, and so the relay's peak
	// memory in stored envelopes.
	BatchSize int
	// PublishConcurrency is how many events of a batch may be in the publisher
	// at once. It never reorders an ordering key, because a batch holds at most
	// one claimable event per key — the partial unique index
	// outbox_events_ordering_ready_key is what enforces that, not this package.
	PublishConcurrency int
	// PublishTimeout budgets one whole batch's publication, not one event's.
	PublishTimeout time.Duration
	// LeaseDuration is how long a claim fences its events against another relay.
	// It also caps the publication budget, so it must exceed PublishTimeout plus
	// the publisher join bound.
	LeaseDuration time.Duration
	// MaxAttempts is the cycle attempt count at which an adapter-proven
	// rejection poisons. Ambiguous failures ignore it and keep retrying.
	MaxAttempts int
	// RetryBase and RetryMax bound full-jitter exponential backoff.
	RetryBase time.Duration
	RetryMax  time.Duration
	// ObservationInterval is how often relay state is sampled. Readiness treats
	// a sample as fresh for two of these intervals.
	ObservationInterval time.Duration
	// CleanupInterval is the normal retention cadence. A full delete batch
	// shortens the next one to PollInterval so a backlog drains.
	CleanupInterval time.Duration
	// PublishedRetention is how long a published row is kept before deletion.
	PublishedRetention time.Duration
	// CleanupBatchSize bounds one retention delete transaction.
	CleanupBatchSize int
}

// ValidateRelayConfig rejects a configuration the relay cannot run under. Every
// message names the field and the value it was given, because the operator who
// reads it is holding a config file rather than this source.
//
// [NewRelay] applies it, and it is exported for the same reason
// [ValidatePublisher] is: the composition root is the only place that can hold
// this package's rules beside internal/config's copies of them, so it is the
// only place that can prove the two still describe the same budget. See the
// ceiling block above for which tests do that.
func ValidateRelayConfig(config RelayConfig) error {
	for _, field := range []struct {
		name  string
		value time.Duration
	}{
		{name: "poll_interval", value: config.PollInterval},
		{name: "publish_timeout", value: config.PublishTimeout},
		{name: "lease_duration", value: config.LeaseDuration},
		{name: "retry_base", value: config.RetryBase},
		{name: "observation_interval", value: config.ObservationInterval},
		{name: "cleanup_interval", value: config.CleanupInterval},
		{name: "published_retention", value: config.PublishedRetention},
	} {
		if field.value <= 0 {
			return fmt.Errorf("%w: %s must be positive, got %s", ErrConfig, field.name, field.value)
		}
	}
	for _, field := range []struct {
		name      string
		value     int
		low, high int
	}{
		{name: "batch_size", value: config.BatchSize, low: 1, high: MaxBatchSize},
		{name: "publish_concurrency", value: config.PublishConcurrency, low: 1, high: MaxPublishConcurrency},
		{name: "max_attempts", value: config.MaxAttempts, low: 1, high: MaxAttempts},
		{name: "cleanup_batch_size", value: config.CleanupBatchSize, low: 1, high: MaxCleanupBatchSize},
	} {
		if field.value < field.low || field.value > field.high {
			return fmt.Errorf(
				"%w: %s must be in range [%d,%d], got %d",
				ErrConfig, field.name, field.low, field.high, field.value,
			)
		}
	}
	if config.RetryMax < config.RetryBase {
		return fmt.Errorf(
			"%w: retry_max (%s) must be at least retry_base (%s)",
			ErrConfig, config.RetryMax, config.RetryBase,
		)
	}
	// The batch's publication deadline is derived from the lease, and a stuck
	// publisher is given PublisherJoinTimeout to stop before the relay gives up
	// on it, so the lease has to outlast both.
	if config.LeaseDuration <= config.PublishTimeout ||
		config.LeaseDuration-config.PublishTimeout <= PublisherJoinTimeout {
		return fmt.Errorf(
			"%w: lease_duration (%s) must exceed publish_timeout (%s) by more than the %s publisher-join timeout",
			ErrConfig, config.LeaseDuration, config.PublishTimeout, PublisherJoinTimeout,
		)
	}
	return nil
}

// retryDelay is full-jitter exponential backoff: the delay is drawn from
// [0, base*2^(attempt-1)] with that limit clamped to maximum. attempt is the
// claim's 1-based cycle attempt count, so the first retry waits within base.
func retryDelay(base, maximum time.Duration, attempt int, jitter func(time.Duration) time.Duration) time.Duration {
	limit := base
	for range max(attempt-1, 0) {
		// Doubling only below half the ceiling keeps the limit from overflowing.
		if limit > maximum/2 {
			return jitter(maximum)
		}
		limit *= 2
	}
	return jitter(min(limit, maximum))
}

// fullJitter draws uniformly from [0,limit]. Spreading the whole interval, not
// a band around it, is what keeps a batch of events that failed together from
// retrying together.
func fullJitter(limit time.Duration) time.Duration {
	if limit == math.MaxInt64 {
		// limit+1 would overflow, and Int64 already covers the whole range.
		// #nosec G404 -- Backoff jitter coordinates retries; it is not a secret or token.
		return time.Duration(rand.Int64())
	}
	// #nosec G404 -- Backoff jitter coordinates retries; it is not a secret or token.
	return time.Duration(rand.Int64N(limit.Nanoseconds() + 1))
}
