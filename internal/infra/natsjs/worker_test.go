package natsjs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestExplicitAckPolicy(t *testing.T) {
	cfg := DefaultWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	desired := desiredConsumerConfig(cfg)
	if desired.AckPolicy != jetstream.AckExplicitPolicy {
		t.Fatalf("AckPolicy = %v, want AckExplicitPolicy", desired.AckPolicy)
	}
	if desired.MaxDeliver != -1 || desired.MaxRequestBatch != 1 || desired.MaxWaiting != 2 {
		t.Fatalf("consumer bounds = MaxDeliver %d, MaxRequestBatch %d, MaxWaiting %d", desired.MaxDeliver, desired.MaxRequestBatch, desired.MaxWaiting)
	}
	mutated := desired
	mutated.HeadersOnly = true
	if consumerConfigEqual(mutated, desired) {
		t.Fatal("consumerConfigEqual accepted delivery-affecting mutation")
	}
	metadataOnly := desired
	metadataOnly.Description = "operator note"
	metadataOnly.Metadata = map[string]string{"owner": "platform"}
	if !consumerConfigEqual(metadataOnly, desired) {
		t.Fatal("consumerConfigEqual rejected description/metadata-only difference")
	}
}

func TestSingleMessageFetch(t *testing.T) {
	consumer := &recordingConsumer{fetchErr: errors.New("stop")}
	w := &Worker{
		client:   &Client{},
		cfg:      WorkerConfig{MaxConcurrency: 1},
		consumer: consumer,
		fatal:    make(chan error, 1),
		runDone:  make(chan struct{}),
	}
	err := w.Run(context.Background())
	if !errors.Is(err, ErrTerminal) {
		t.Fatalf("Run() error = %v, want ErrTerminal", err)
	}
	if consumer.batch != 1 {
		t.Fatalf("Fetch batch = %d, want 1", consumer.batch)
	}
}

type recordingConsumer struct {
	batch    int
	fetchErr error
}

func (c *recordingConsumer) Fetch(batch int, _ ...jetstream.FetchOpt) (jetstream.MessageBatch, error) { //nolint:ireturn // The test double implements jetstream's interface-returning contract.
	c.batch = batch
	return nil, c.fetchErr
}

func (c *recordingConsumer) Info(context.Context) (*jetstream.ConsumerInfo, error) {
	return &jetstream.ConsumerInfo{}, nil
}

var _ = time.Second
