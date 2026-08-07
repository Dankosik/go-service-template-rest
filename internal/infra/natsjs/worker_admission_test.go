package natsjs

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestExplicitAckPolicy(t *testing.T) {
	cfg := testWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	desired := desiredConsumerConfig(cfg)
	if desired.AckPolicy != jetstream.AckExplicitPolicy {
		t.Fatalf("AckPolicy = %v, want AckExplicitPolicy", desired.AckPolicy)
	}
	if desired.MaxDeliver != -1 || desired.MaxRequestBatch != 1 || desired.MaxWaiting != 2 {
		t.Fatalf("consumer bounds = MaxDeliver %d, MaxRequestBatch %d, MaxWaiting %d", desired.MaxDeliver, desired.MaxRequestBatch, desired.MaxWaiting)
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
