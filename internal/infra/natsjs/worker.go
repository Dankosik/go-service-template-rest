package natsjs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Worker runs MaxConcurrency serial JetStream consume contexts over one
// durable consumer. The NATS client owns pulling, buffering, reconnect, and callback
// drain; this type only aggregates those contexts into the process lifecycle.
type Worker struct {
	client    *Client
	cfg       WorkerConfig
	consumer  pullConsumer
	dlqStream string
	handler   Handler

	draining atomic.Bool
	started  atomic.Bool
	fatal    chan error
	runDone  chan struct{}
	drain    chan struct{}

	mu            sync.Mutex
	consumers     []jetstream.ConsumeContext
	handlerCancel context.CancelFunc
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.started.CompareAndSwap(false, true) {
		return fmt.Errorf("%w: worker already started", ErrRejected)
	}
	defer close(w.runDone)

	handlerRoot, handlerCancel := context.WithCancel(context.WithoutCancel(ctx))
	w.mu.Lock()
	w.handlerCancel = handlerCancel
	w.mu.Unlock()

	consumers, err := w.startConsumers(handlerRoot)
	if err != nil {
		handlerCancel()
		return err
	}
	w.mu.Lock()
	w.consumers = consumers
	draining := w.draining.Load()
	w.mu.Unlock()
	if draining {
		drainConsumers(consumers)
	}

	var runErr error
	select {
	case <-w.drain:
	case runErr = <-w.fatal:
		w.StartDrain()
	case <-ctx.Done():
		runErr = fmt.Errorf("run durable consumer: %w", ctx.Err())
		w.StartDrain()
	}

	waitConsumers(consumers)
	if runErr == nil {
		select {
		case runErr = <-w.fatal:
		default:
		}
	}
	return runErr
}

func (w *Worker) startConsumers(handlerRoot context.Context) ([]jetstream.ConsumeContext, error) {
	consumers := make([]jetstream.ConsumeContext, 0, w.cfg.MaxConcurrency)
	for range w.cfg.MaxConcurrency {
		consumer, err := w.consumer.Consume(
			func(msg jetstream.Msg) {
				if handleErr := w.handle(handlerRoot, msg); handleErr != nil {
					w.fail(handleErr)
				}
			},
			jetstream.PullMaxMessages(1),
			jetstream.PullExpiry(operationTimeout),
			jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, consumeErr error) {
				if w.draining.Load() || !terminalConsumeError(consumeErr) {
					return
				}
				w.fail(fmt.Errorf("%w: consume source message: %w", ErrTerminal, consumeErr))
			}),
		)
		if err != nil {
			stopConsumers(consumers)
			waitConsumers(consumers)
			return nil, fmt.Errorf("%w: start durable consumer: %w", ErrRejected, err)
		}
		consumers = append(consumers, consumer)
	}
	return consumers, nil
}

func terminalConsumeError(err error) bool {
	// The native consume error callback receives both errors it recovers from
	// and the small set for which it stops itself. Mirror that terminal set;
	// everything else remains library-owned recovery.
	return errors.Is(err, jetstream.ErrConsumerDeleted) ||
		errors.Is(err, jetstream.ErrBadRequest) ||
		errors.Is(err, jetstream.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrConnectionClosed)
}

func (w *Worker) StartDrain() {
	if !w.draining.CompareAndSwap(false, true) {
		return
	}
	w.client.StopPublish()
	close(w.drain)
	w.mu.Lock()
	consumers := append([]jetstream.ConsumeContext(nil), w.consumers...)
	w.mu.Unlock()
	drainConsumers(consumers)
}

func (w *Worker) Shutdown(ctx context.Context) error {
	w.StartDrain()
	select {
	case <-w.runDone:
		if err := w.client.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	case <-ctx.Done():
		w.forceClose()
		return fmt.Errorf("forced messaging shutdown: %w", ctx.Err())
	}
}

func (w *Worker) forceClose() {
	w.StartDrain()
	w.mu.Lock()
	if w.handlerCancel != nil {
		w.handlerCancel()
	}
	consumers := append([]jetstream.ConsumeContext(nil), w.consumers...)
	w.mu.Unlock()
	stopConsumers(consumers)
	w.client.Close()
}

func (w *Worker) fail(err error) {
	select {
	case w.fatal <- err:
	default:
	}
	w.StartDrain()
}

func drainConsumers(consumers []jetstream.ConsumeContext) {
	for _, consumer := range consumers {
		consumer.Drain()
	}
}

func stopConsumers(consumers []jetstream.ConsumeContext) {
	for _, consumer := range consumers {
		consumer.Stop()
	}
}

func waitConsumers(consumers []jetstream.ConsumeContext) {
	for _, consumer := range consumers {
		<-consumer.Closed()
	}
}
