package natsjs

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type pullConsumer interface {
	Consume(handler jetstream.MessageHandler, opts ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, error)
	Info(ctx context.Context) (*jetstream.ConsumerInfo, error)
}

const settlementSchedulingSlack = time.Second

// NewWorker creates or updates the application-owned durable consumer. A Client
// admits one successfully created Worker, whose Run must start before normal
// shutdown. Streams remain operator-owned; the NATS client and server own
// consumer reconciliation.
func (c *Client) NewWorker(ctx context.Context, cfg WorkerConfig, handler Handler) (*Worker, error) {
	if handler == nil {
		return nil, fmt.Errorf("%w: messaging handler is required", ErrRejected)
	}
	if err := ValidateWorkerConfig(cfg, c.cfg.MaxPayloadBytes); err != nil {
		return nil, err
	}
	if !c.workerClaimed.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("%w: messaging client already owns a worker", ErrRejected)
	}
	accepted := false
	defer func() {
		if !accepted {
			c.workerClaimed.Store(false)
		}
	}()

	probeCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	dlqStream, err := c.js.StreamNameBySubject(probeCtx, cfg.DeadLetterSubject)
	if err != nil {
		return nil, fmt.Errorf("%w: dead-letter stream is unavailable", ErrRejected)
	}
	if dlqStream == c.cfg.Stream {
		return nil, fmt.Errorf("%w: source and dead-letter streams must differ", ErrRejected)
	}
	consumer, err := c.js.CreateOrUpdateConsumer(probeCtx, c.cfg.Stream, desiredConsumerConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("%w: durable consumer admission failed", ErrRejected)
	}

	c.probeMu.Lock()
	c.consumer = consumer
	c.probeMu.Unlock()
	if err := c.Check(probeCtx); err != nil {
		c.probeMu.Lock()
		c.consumer = nil
		c.probeMu.Unlock()
		return nil, err
	}

	accepted = true
	return &Worker{
		client: c, cfg: cfg, consumer: consumer, dlqStream: dlqStream, handler: handler,
		terminal: make(chan error, 1), runDone: make(chan struct{}), drain: make(chan struct{}),
	}, nil
}

// desiredConsumerConfig is the durable consumer settlement depends on.
//
// AckWait covers one handler run, the up to two broker operations that settle
// it — each bounded by operationTimeout — and scheduling slack, so the broker
// does not redeliver a message whose settlement is still in flight. MaxDeliver
// is unlimited because the worker counts attempts itself and dead-letters at
// its attempt limit; a broker-side cap would instead stop redelivering and
// leave the message unsettled. MaxAckPending is MaxConcurrency: one in-flight
// message per serial consume context.
func desiredConsumerConfig(cfg WorkerConfig) jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		Name:          cfg.Consumer,
		Durable:       cfg.Consumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       cfg.HandlerTimeout + 2*operationTimeout + settlementSchedulingSlack,
		MaxDeliver:    -1,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
		MaxAckPending: cfg.MaxConcurrency,
		FilterSubject: cfg.FilterSubject,
	}
}
