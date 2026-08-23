package natsjs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type fakeConsumeContext struct {
	once   sync.Once
	closed chan struct{}
}

func (c *fakeConsumeContext) Stop()                   { c.once.Do(func() { close(c.closed) }) }
func (c *fakeConsumeContext) Drain()                  { c.Stop() }
func (c *fakeConsumeContext) Closed() <-chan struct{} { return c.closed }

type fakePullConsumer struct {
	started chan *fakeConsumeContext
}

//nolint:ireturn // The fake must implement the narrowed production interface.
func (c *fakePullConsumer) Consume(jetstream.MessageHandler, ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, error) {
	consumer := &fakeConsumeContext{closed: make(chan struct{})}
	c.started <- consumer
	return consumer, nil
}

func (*fakePullConsumer) Info(context.Context) (*jetstream.ConsumerInfo, error) {
	return &jetstream.ConsumerInfo{}, nil
}

func TestWorkerUsesNativeBoundedConsumeContextsAndJoinsDrain(t *testing.T) {
	worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
	worker.cfg.MaxConcurrency = 3
	consumer := &fakePullConsumer{started: make(chan *fakeConsumeContext, worker.cfg.MaxConcurrency)}
	worker.consumer = consumer
	runErr := make(chan error, 1)
	go func() { runErr <- worker.Run(t.Context()) }()
	for range worker.cfg.MaxConcurrency {
		select {
		case <-consumer.started:
		case <-time.After(time.Second):
			t.Fatal("native consume context did not start")
		}
	}
	worker.StartDrain()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not join drained consume contexts")
	}
}

func TestWorkerStopsWhenNativeConsumeContextCloses(t *testing.T) {
	worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
	worker.cfg.MaxConcurrency = 1
	consumer := &fakePullConsumer{started: make(chan *fakeConsumeContext, 1)}
	worker.consumer = consumer
	runErr := make(chan error, 1)
	go func() { runErr <- worker.Run(t.Context()) }()
	(<-consumer.started).Stop()

	select {
	case err := <-runErr:
		if !errors.Is(err, ErrTerminal) {
			t.Fatalf("Run() error = %v, want ErrTerminal", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not report an unexpectedly closed native consume context")
	}
}
