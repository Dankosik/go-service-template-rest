package postgresoutbox

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	t.Parallel()

	const (
		base    = time.Second
		maximum = 5 * time.Second
	)
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: time.Second},
		{attempt: 2, want: 2 * time.Second},
		{attempt: 3, want: 4 * time.Second},
		{attempt: 4, want: maximum},
		{attempt: 100, want: maximum},
	}
	for _, test := range tests {
		got := retryDelay(base, maximum, test.attempt, func(limit time.Duration) time.Duration { return limit })
		if got != test.want {
			t.Errorf("attempt %d delay cap = %s, want %s", test.attempt, got, test.want)
		}
	}
	if got := retryDelay(base, maximum, 3, func(time.Duration) time.Duration { return 0 }); got != 0 {
		t.Fatalf("full-jitter lower bound = %s, want 0", got)
	}
	maxDuration := time.Duration(1<<63 - 1)
	if got := fullJitter(maxDuration); got < 0 || got > maxDuration {
		t.Fatalf("maximum full jitter = %s, want [0,%s]", got, maxDuration)
	}
}

// The lease has to outlast the publication budget plus the publisher-join
// bound, and saying so must survive both a too-short lease and durations near
// the overflow edge.
func TestRelayConfigRejectsLeaseThatCannotHoldPublication(t *testing.T) {
	t.Parallel()

	valid := unitRelayConfig()
	if err := ValidateRelayConfig(valid); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(RelayConfig) RelayConfig
	}{
		{name: "lease equals publish plus join", mutate: func(c RelayConfig) RelayConfig {
			c.LeaseDuration = c.PublishTimeout + PublisherJoinTimeout
			return c
		}},
		{name: "lease and publish both saturate", mutate: func(c RelayConfig) RelayConfig {
			c.PublishTimeout = math.MaxInt64
			c.LeaseDuration = math.MaxInt64
			return c
		}},
	}
	for _, test := range tests {
		if err := ValidateRelayConfig(test.mutate(valid)); !errors.Is(err, ErrConfig) {
			t.Errorf("ValidateRelayConfig(%s) error = %v, want ErrConfig", test.name, err)
		}
	}
}

// Backoff at the maximum representable duration must stay inside its own bound
// rather than overflowing into a negative delay.
func TestRelayRetryDelayHoldsAtMaximumDuration(t *testing.T) {
	t.Parallel()

	config := unitRelayConfig()
	config.RetryBase = math.MaxInt64
	config.RetryMax = math.MaxInt64
	if err := ValidateRelayConfig(config); err != nil {
		t.Fatalf("maximum retry config: %v", err)
	}
	if got := retryDelay(config.RetryBase, config.RetryMax, 1, fullJitter); got < 0 || got > config.RetryMax {
		t.Fatalf("maximum retry delay = %s, want [0,%s]", got, config.RetryMax)
	}
}

func TestRelayConfigRejectsOutOfRangeFields(t *testing.T) {
	t.Parallel()

	valid := unitRelayConfig()
	invalid := []struct {
		name   string
		mutate func(RelayConfig) RelayConfig
	}{
		{name: "zero value", mutate: func(RelayConfig) RelayConfig { return RelayConfig{} }},
		{name: "zero publish timeout", mutate: func(c RelayConfig) RelayConfig { c.PublishTimeout = 0; return c }},
		{name: "zero max attempts", mutate: func(c RelayConfig) RelayConfig { c.MaxAttempts = 0; return c }},
		{name: "zero retry base", mutate: func(c RelayConfig) RelayConfig { c.RetryBase = 0; return c }},
		{name: "retry max below base", mutate: func(c RelayConfig) RelayConfig { c.RetryMax = c.RetryBase - 1; return c }},
		{name: "zero observation interval", mutate: func(c RelayConfig) RelayConfig { c.ObservationInterval = 0; return c }},
		{name: "zero cleanup interval", mutate: func(c RelayConfig) RelayConfig { c.CleanupInterval = 0; return c }},
		{name: "zero published retention", mutate: func(c RelayConfig) RelayConfig { c.PublishedRetention = 0; return c }},
		{name: "zero cleanup batch size", mutate: func(c RelayConfig) RelayConfig { c.CleanupBatchSize = 0; return c }},
		{name: "zero batch size", mutate: func(c RelayConfig) RelayConfig { c.BatchSize = 0; return c }},
		{name: "batch size over maximum", mutate: func(c RelayConfig) RelayConfig { c.BatchSize = MaxBatchSize + 1; return c }},
		{name: "zero publish concurrency", mutate: func(c RelayConfig) RelayConfig { c.PublishConcurrency = 0; return c }},
		{name: "publish concurrency over maximum", mutate: func(c RelayConfig) RelayConfig {
			c.PublishConcurrency = MaxPublishConcurrency + 1
			return c
		}},
	}
	for _, test := range invalid {
		config := test.mutate(valid)
		if err := ValidateRelayConfig(config); !errors.Is(err, ErrConfig) {
			t.Errorf("ValidateRelayConfig(%s) error = %v, want ErrConfig", test.name, err)
		}
		if _, err := newRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error { return nil }), nil, config); !errors.Is(err, ErrConfig) {
			t.Errorf("newRelay(%s) error = %v, want ErrConfig", test.name, err)
		}
	}
}
