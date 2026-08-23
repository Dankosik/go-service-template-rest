package natsjs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestWorkerConfigBoundsAndConsumerPolicy(t *testing.T) {
	valid := testWorkerConfig()
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
			cfg := valid
			mutate(&cfg)
			if err := ValidateWorkerConfig(cfg, testMaxPayloadBytes); !errors.Is(err, ErrRejected) {
				t.Fatalf("ValidateWorkerConfig() error = %v, want ErrRejected", err)
			}
		})
	}

	desired := desiredConsumerConfig(valid)
	if desired.Name != valid.Consumer || desired.Durable != valid.Consumer ||
		desired.AckPolicy != jetstream.AckExplicitPolicy || desired.MaxDeliver != -1 ||
		desired.MaxAckPending != valid.MaxConcurrency || desired.FilterSubject != valid.FilterSubject {
		t.Fatalf("desired consumer policy = %#v", desired)
	}
	if want := valid.HandlerTimeout + 2*operationTimeout + settlementSchedulingSlack; desired.AckWait != want {
		t.Fatalf("AckWait = %v, want %v", desired.AckWait, want)
	}
}

func TestNewWorkerRejectsBeforeBrokerMutation(t *testing.T) {
	client := unitClient(t, &recordingJetStream{})
	if _, err := client.NewWorker(t.Context(), testWorkerConfig(), nil); !errors.Is(err, ErrRejected) {
		t.Fatalf("NewWorker(nil handler) error = %v", err)
	}
	invalid := testWorkerConfig()
	invalid.Consumer = "bad consumer"
	if _, err := client.NewWorker(t.Context(), invalid, func(context.Context, Message) error { return nil }); !errors.Is(err, ErrRejected) {
		t.Fatalf("NewWorker(invalid config) error = %v", err)
	}
	client.workerClaimed.Store(true)
	if _, err := client.NewWorker(t.Context(), testWorkerConfig(), func(context.Context, Message) error { return nil }); !errors.Is(err, ErrRejected) {
		t.Fatalf("NewWorker(second worker) error = %v", err)
	}
}
