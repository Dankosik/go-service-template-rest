package natsjs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/nats-io/nats.go/jetstream"
)

func TestFetchAsksForEveryFreeSlot(t *testing.T) {
	t.Parallel()
	consumer := &recordingConsumer{fetchErr: errors.New("stop")}
	w := &Worker{
		client:   &Client{},
		cfg:      WorkerConfig{MaxConcurrency: testMaxConcurrency},
		consumer: consumer,
		fatal:    make(chan error, 1),
		runDone:  make(chan struct{}),
	}
	err := w.Run(context.Background())
	if !errors.Is(err, ErrTerminal) {
		t.Fatalf("Run() error = %v, want ErrTerminal", err)
	}
	if consumer.batch != testMaxConcurrency {
		t.Fatalf("Fetch batch = %d, want every free handler slot (%d)", consumer.batch, testMaxConcurrency)
	}
}

// TestFetchBatchShrinksToFreeSlots pins the bound that keeps batching from
// widening concurrency: a pull never asks for a slot a running handler holds,
// so the resident wire data stays inside MaxConcurrency deliveries however
// large a batch the broker could return.
func TestFetchBatchShrinksToFreeSlots(t *testing.T) {
	t.Parallel()
	const occupied = 5

	sources := make([]jetstream.Msg, occupied)
	for index := range sources {
		sources[index] = unitSource(t, 1)
	}
	release := make(chan struct{})
	consumer := &slotConsumer{sources: sources, requested: make(chan int, 4)}
	worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error {
		<-release
		return nil
	})
	worker.cfg.MaxConcurrency = testMaxConcurrency
	worker.consumer = consumer

	ctx, cancel := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	go func() { runErr <- worker.Run(ctx) }()

	if got := <-consumer.requested; got != testMaxConcurrency {
		t.Fatalf("first pull asked for %d, want every free slot (%d)", got, testMaxConcurrency)
	}
	if got, want := <-consumer.requested, testMaxConcurrency-occupied; got != want {
		t.Fatalf("second pull asked for %d, want the %d slots the first batch left", got, want)
	}
	close(release)
	cancel()
	if err := <-runErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}

func TestDrainedHandlersReclaimsEveryFinishedSlot(t *testing.T) {
	t.Parallel()
	completion := make(chan struct{}, 4)
	if got := drainedHandlers(completion); got != 0 {
		t.Fatalf("drainedHandlers(none finished) = %d, want 0", got)
	}
	for range 3 {
		completion <- struct{}{}
	}
	if got := drainedHandlers(completion); got != 3 {
		t.Fatalf("drainedHandlers(three finished) = %d, want 3", got)
	}
	if got := drainedHandlers(completion); got != 0 {
		t.Fatalf("drainedHandlers(already reclaimed) = %d, want 0", got)
	}
}

func TestWorkerWaitForHandlersPreservesTerminalError(t *testing.T) {
	t.Parallel()
	w := &Worker{fatal: make(chan error, 1)}
	want := fmt.Errorf("%w: handler failed", ErrTerminal)
	w.fatal <- want

	if err := w.waitForHandlers(make(chan struct{}), 0, nil); !errors.Is(err, want) {
		t.Fatalf("waitForHandlers() error = %v, want %v", err, want)
	}
}

func TestWorkerDrainRejectsMessageAlreadyReturnedByFetch(t *testing.T) {
	t.Parallel()
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

// slotConsumer hands out a fixed set of messages on its first pull and empty
// batches after that, recording what each pull asked for. The record never
// blocks the fetch loop: the pulls worth asserting on are the first two, and a
// full channel must not stop the loop from reaching cancellation.
type slotConsumer struct {
	sources   []jetstream.Msg
	requested chan int
}

func (c *slotConsumer) Fetch(batch int, _ ...jetstream.FetchOpt) (jetstream.MessageBatch, error) { //nolint:ireturn // The test double implements jetstream's interface-returning contract.
	select {
	case c.requested <- batch:
	default:
	}
	messages := make(chan jetstream.Msg, batch)
	for _, source := range c.sources[:min(batch, len(c.sources))] {
		messages <- source
	}
	c.sources = nil
	close(messages)
	return messageBatch{messages: messages}, nil
}

func (c *slotConsumer) Info(context.Context) (*jetstream.ConsumerInfo, error) {
	return &jetstream.ConsumerInfo{}, nil
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

func TestWorkerShutdownStateTransitions(t *testing.T) {
	t.Parallel()
	worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
	close(worker.runDone)
	if err := worker.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown(graceful) error = %v", err)
	}

	forced := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := forced.Shutdown(canceled); !errors.Is(err, context.Canceled) || !forced.client.draining.Load() {
		t.Fatalf("Shutdown(forced) error = %v, draining = %t", err, forced.client.draining.Load())
	}
}

func TestWorkerShutdownWaitsForRunCompletion(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
		result := make(chan error, 1)
		go func() { result <- worker.Shutdown(t.Context()) }()
		synctest.Wait()
		select {
		case err := <-result:
			t.Fatalf("Shutdown() returned before Run completion: %v", err)
		default:
		}
		close(worker.runDone)
		synctest.Wait()
		if err := <-result; err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})
}
