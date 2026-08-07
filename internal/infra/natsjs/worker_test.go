package natsjs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/nats-io/nats.go/jetstream"
)

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

func TestWorkerShutdownStateTransitions(t *testing.T) {
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
