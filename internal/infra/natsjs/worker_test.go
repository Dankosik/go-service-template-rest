package natsjs

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type fakeConsumeContext struct {
	once   sync.Once
	closed chan struct{}
}

func TestConsumeErrorClassification(t *testing.T) {
	for _, err := range []error{
		context.DeadlineExceeded,
		jetstream.ErrNoMessages,
		jetstream.ErrNoHeartbeat,
		nats.ErrNoResponders,
		jetstream.ErrConsumerLeadershipChanged,
		nats.ErrDisconnected,
		nats.ErrConnectionReconnecting,
		nats.ErrReconnectBufExceeded,
		jetstream.ErrServerShutdown,
	} {
		if terminalConsumeError(err) {
			t.Errorf("terminalConsumeError(%v) = true", err)
		}
	}
	for _, err := range []error{
		jetstream.ErrConsumerDeleted,
		jetstream.ErrBadRequest,
		jetstream.ErrConnectionClosed,
		nats.ErrConnectionClosed,
	} {
		if !terminalConsumeError(err) {
			t.Errorf("terminalConsumeError(%v) = false", err)
		}
	}
}

func (c *fakeConsumeContext) Stop()                   { c.once.Do(func() { close(c.closed) }) }
func (c *fakeConsumeContext) Drain()                  { c.Stop() }
func (c *fakeConsumeContext) Closed() <-chan struct{} { return c.closed }

type fakePullConsumer struct {
	started chan struct{}
}

//nolint:ireturn // The fake must implement the narrowed production interface.
func (c *fakePullConsumer) Consume(jetstream.MessageHandler, ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, error) {
	c.started <- struct{}{}
	return &fakeConsumeContext{closed: make(chan struct{})}, nil
}

func (*fakePullConsumer) Info(context.Context) (*jetstream.ConsumerInfo, error) {
	return &jetstream.ConsumerInfo{}, nil
}

func TestWorkerUsesNativeBoundedConsumeContextsAndJoinsDrain(t *testing.T) {
	worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
	worker.cfg.MaxConcurrency = 3
	consumer := &fakePullConsumer{started: make(chan struct{}, worker.cfg.MaxConcurrency)}
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

func TestConsumerPolicyKeepsOnlySettlementCoupledFields(t *testing.T) {
	cfg := testWorkerConfig()
	desired := desiredConsumerConfig(cfg)
	if desired.AckPolicy != jetstream.AckExplicitPolicy || desired.MaxDeliver != -1 {
		t.Fatalf("consumer settlement = ack %v, max deliver %d", desired.AckPolicy, desired.MaxDeliver)
	}
	if desired.MaxAckPending != cfg.MaxConcurrency || desired.FilterSubject != cfg.FilterSubject {
		t.Fatalf("consumer bound = pending %d, filter %q", desired.MaxAckPending, desired.FilterSubject)
	}
	if desired.MaxWaiting != 0 || desired.MaxRequestBatch != 0 || desired.MaxRequestMaxBytes != 0 {
		t.Fatalf("consumer retained client pull policy: %#v", desired)
	}
}
