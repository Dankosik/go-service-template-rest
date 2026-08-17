package natsjs

import (
	"errors"
	"testing"
	"time"
)

const (
	testMaxConcurrency   = 8
	testMaxDeliveryBytes = 1 << 20
)

func testWorkerConfig() WorkerConfig {
	return WorkerConfig{
		MaxConcurrency:       testMaxConcurrency,
		MaxDeliveryBytes:     testMaxDeliveryBytes,
		HandlerTimeout:       30 * time.Second,
		RetryDelays:          []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute},
		DeadLetterRetryDelay: 30 * time.Second,
	}
}

func TestWorkerAdmissionBound(t *testing.T) {
	t.Parallel()
	valid := testWorkerConfig()
	valid.Consumer = "events-worker"
	valid.FilterSubject = "events.>"
	valid.DeadLetterSubject = "dead.events"
	if err := ValidateWorkerConfig(valid, testMaxPayloadBytes); err != nil {
		t.Fatalf("ValidateWorkerConfig(valid) error = %v", err)
	}

	cases := map[string]func(*WorkerConfig){
		"invalid consumer":        func(cfg *WorkerConfig) { cfg.Consumer = "events worker" },
		"invalid filter":          func(cfg *WorkerConfig) { cfg.FilterSubject = "events.>.invalid" },
		"invalid dead letter":     func(cfg *WorkerConfig) { cfg.DeadLetterSubject = "dead.*" },
		"overlapping dead letter": func(cfg *WorkerConfig) { cfg.DeadLetterSubject = "events.dead" },
		"zero concurrency":        func(cfg *WorkerConfig) { cfg.MaxConcurrency = 0 },
		"zero delivery bound":     func(cfg *WorkerConfig) { cfg.MaxDeliveryBytes = 0 },
		"undersized envelope":     func(cfg *WorkerConfig) { cfg.MaxDeliveryBytes = testMaxPayloadBytes + HeaderLimitBytes - 1 },
		"resident bound":          func(cfg *WorkerConfig) { cfg.MaxConcurrency = ResidentDeliveryLimit/cfg.MaxDeliveryBytes + 1 },
		"zero handler timeout":    func(cfg *WorkerConfig) { cfg.HandlerTimeout = 0 },
		"no retry delays":         func(cfg *WorkerConfig) { cfg.RetryDelays = nil },
		"too many retry delays": func(cfg *WorkerConfig) {
			cfg.RetryDelays = []time.Duration{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		},
		"zero retry delay":       func(cfg *WorkerConfig) { cfg.RetryDelays = []time.Duration{0} },
		"zero dead letter retry": func(cfg *WorkerConfig) { cfg.DeadLetterRetryDelay = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			mutate(&cfg)
			if err := ValidateWorkerConfig(cfg, testMaxPayloadBytes); !errors.Is(err, ErrRejected) {
				t.Fatalf("ValidateWorkerConfig() error = %v, want ErrRejected", err)
			}
		})
	}
}
