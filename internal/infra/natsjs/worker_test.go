package natsjs

import (
	"context"
	"errors"
	"fmt"
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

func TestWorkerWaitForHandlersPreservesTerminalError(t *testing.T) {
	w := &Worker{fatal: make(chan error, 1)}
	want := fmt.Errorf("%w: handler failed", ErrTerminal)
	w.fatal <- want

	if err := w.waitForHandlers(make(chan struct{}), 0, nil); !errors.Is(err, want) {
		t.Fatalf("waitForHandlers() error = %v, want %v", err, want)
	}
}

func TestWorkerDrainRejectsMessageAlreadyReturnedByFetch(t *testing.T) {
	consumer := &gatedConsumer{fetchStarted: make(chan struct{}), batch: make(chan jetstream.MessageBatch, 1)}
	sig, err := newTelemetry(Observability{}, RoleWorker, func() bool { return false })
	if err != nil {
		t.Fatalf("newTelemetry() error = %v", err)
	}
	t.Cleanup(sig.close)
	w := &Worker{
		client:   &Client{telemetry: sig},
		cfg:      WorkerConfig{MaxConcurrency: 1},
		consumer: consumer,
		handler: func(context.Context, Message) error {
			t.Fatal("handler entered after drain started")
			return nil
		},
		fatal:   make(chan error, 1),
		runDone: make(chan struct{}),
	}
	runErr := make(chan error, 1)
	go func() { runErr <- w.Run(t.Context()) }()
	<-consumer.fetchStarted
	w.StartDrain()
	messages := make(chan jetstream.Msg, 1)
	messages <- nil
	close(messages)
	consumer.batch <- messageBatch{messages: messages}
	if err := <-runErr; err != nil {
		t.Fatalf("Run() error = %v, want graceful drain", err)
	}
}

type recordingConsumer struct {
	batch    int
	fetchErr error
}

type gatedConsumer struct {
	fetchStarted chan struct{}
	batch        chan jetstream.MessageBatch
}

func (c *gatedConsumer) Fetch(int, ...jetstream.FetchOpt) (jetstream.MessageBatch, error) { //nolint:ireturn // The test double implements jetstream's interface-returning contract.
	close(c.fetchStarted)
	return <-c.batch, nil
}

func (c *gatedConsumer) Info(context.Context) (*jetstream.ConsumerInfo, error) {
	return &jetstream.ConsumerInfo{}, nil
}

type messageBatch struct {
	messages <-chan jetstream.Msg
}

func (b messageBatch) Messages() <-chan jetstream.Msg { return b.messages }
func (messageBatch) Error() error                     { return nil }

func (c *recordingConsumer) Fetch(batch int, _ ...jetstream.FetchOpt) (jetstream.MessageBatch, error) { //nolint:ireturn // The test double implements jetstream's interface-returning contract.
	c.batch = batch
	return nil, c.fetchErr
}

func (c *recordingConsumer) Info(context.Context) (*jetstream.ConsumerInfo, error) {
	return &jetstream.ConsumerInfo{}, nil
}

var _ = time.Second
