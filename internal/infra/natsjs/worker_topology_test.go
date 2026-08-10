package natsjs

import (
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestStreamContract(t *testing.T) {
	cfg := testConfig()
	valid := jetstream.StreamConfig{
		Storage: jetstream.FileStorage, Replicas: 1,
		Retention: jetstream.LimitsPolicy, Discard: jetstream.DiscardNew,
		MaxAge: 24 * time.Hour, Duplicates: 2 * time.Minute,
	}
	if err := validateStreamContract(valid, cfg); err != nil {
		t.Fatalf("validateStreamContract(valid) error = %v", err)
	}

	unlimited := valid
	unlimited.MaxAge = 0
	unlimited.Discard = jetstream.DiscardOld
	if err := validateStreamContract(unlimited, cfg); err != nil {
		t.Fatalf("validateStreamContract(unlimited) error = %v", err)
	}

	cases := map[string]func(*jetstream.StreamConfig){
		"memory storage":     func(stream *jetstream.StreamConfig) { stream.Storage = jetstream.MemoryStorage },
		"too few replicas":   func(stream *jetstream.StreamConfig) { stream.Replicas = 0 },
		"interest retention": func(stream *jetstream.StreamConfig) { stream.Retention = jetstream.InterestPolicy },
		"evict on bytes": func(stream *jetstream.StreamConfig) {
			stream.Discard = jetstream.DiscardOld
			stream.MaxBytes = 1
		},
		"evict on messages": func(stream *jetstream.StreamConfig) {
			stream.Discard = jetstream.DiscardOld
			stream.MaxMsgs = 1
		},
		"short retention": func(stream *jetstream.StreamConfig) { stream.MaxAge = time.Hour },
		"no publish ack":  func(stream *jetstream.StreamConfig) { stream.NoAck = true },
		"no dedup window": func(stream *jetstream.StreamConfig) { stream.Duplicates = 0 },
		"per-message TTL": func(stream *jetstream.StreamConfig) { stream.AllowMsgTTL = true },
		"sealed":          func(stream *jetstream.StreamConfig) { stream.Sealed = true },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			stream := valid
			caseCfg := cfg
			mutate(&stream)
			if err := validateStreamContract(stream, caseCfg); !errors.Is(err, ErrRejected) {
				t.Fatalf("validateStreamContract() error = %v, want ErrRejected", err)
			}
		})
	}
}

func TestExplicitAckPolicy(t *testing.T) {
	cfg := testWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	desired := desiredConsumerConfig(cfg)
	if desired.AckPolicy != jetstream.AckExplicitPolicy {
		t.Fatalf("AckPolicy = %v, want AckExplicitPolicy", desired.AckPolicy)
	}
	if desired.MaxDeliver != -1 || desired.MaxWaiting != 2 {
		t.Fatalf("consumer bounds = MaxDeliver %d, MaxWaiting %d", desired.MaxDeliver, desired.MaxWaiting)
	}
	// The broker's copy of the worker's own bounds. Stating them apart from the
	// block above is what catches a request cap that would silently cut every
	// batch back to one message.
	if desired.MaxRequestBatch != cfg.MaxConcurrency {
		t.Fatalf("MaxRequestBatch = %d, want one message per handler slot (%d)", desired.MaxRequestBatch, cfg.MaxConcurrency)
	}
	if want := cfg.MaxConcurrency * cfg.MaxDeliveryBytes; desired.MaxRequestMaxBytes != want {
		t.Fatalf("MaxRequestMaxBytes = %d, want the resident wire-data bound %d", desired.MaxRequestMaxBytes, want)
	}
	if want := cfg.HandlerTimeout + 2*operationTimeout + time.Second; desired.AckWait != want {
		t.Fatalf("AckWait = %v, want handler timeout plus settlement budget %v", desired.AckWait, want)
	}
	mutations := map[string]func(*jetstream.ConsumerConfig){
		"durable":                   func(cfg *jetstream.ConsumerConfig) { cfg.Durable = "other" },
		"deliver policy":            func(cfg *jetstream.ConsumerConfig) { cfg.DeliverPolicy = jetstream.DeliverLastPolicy },
		"ack policy":                func(cfg *jetstream.ConsumerConfig) { cfg.AckPolicy = jetstream.AckNonePolicy },
		"ack wait":                  func(cfg *jetstream.ConsumerConfig) { cfg.AckWait++ },
		"max deliver":               func(cfg *jetstream.ConsumerConfig) { cfg.MaxDeliver = 1 },
		"replay policy":             func(cfg *jetstream.ConsumerConfig) { cfg.ReplayPolicy = jetstream.ReplayOriginalPolicy },
		"max waiting":               func(cfg *jetstream.ConsumerConfig) { cfg.MaxWaiting++ },
		"max ack pending":           func(cfg *jetstream.ConsumerConfig) { cfg.MaxAckPending++ },
		"max request batch":         func(cfg *jetstream.ConsumerConfig) { cfg.MaxRequestBatch++ },
		"max request expires":       func(cfg *jetstream.ConsumerConfig) { cfg.MaxRequestExpires++ },
		"max request bytes":         func(cfg *jetstream.ConsumerConfig) { cfg.MaxRequestMaxBytes++ },
		"filter subject":            func(cfg *jetstream.ConsumerConfig) { cfg.FilterSubject = "events.other" },
		"headers only":              func(cfg *jetstream.ConsumerConfig) { cfg.HeadersOnly = true },
		"push delivery":             func(cfg *jetstream.ConsumerConfig) { cfg.DeliverSubject = "deliver.events" },
		"server redelivery backoff": func(cfg *jetstream.ConsumerConfig) { cfg.BackOff = []time.Duration{time.Second} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			mutated := desired
			mutate(&mutated)
			if consumerConfigEqual(mutated, desired) {
				t.Fatal("consumerConfigEqual accepted delivery-affecting mutation")
			}
		})
	}
	metadataOnly := desired
	metadataOnly.Description = "operator note"
	metadataOnly.Metadata = map[string]string{"owner": "platform"}
	if !consumerConfigEqual(metadataOnly, desired) {
		t.Fatal("consumerConfigEqual rejected description/metadata-only difference")
	}
}
