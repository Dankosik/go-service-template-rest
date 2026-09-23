package natsjs

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

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
	terminal chan error
	runDone  chan struct{}
	drain    chan struct{}

	mu              sync.Mutex
	consumeContexts []jetstream.ConsumeContext
	handlerCancel   context.CancelFunc
}

// Run starts this single-use worker and must begin before normal shutdown.
// Canceling ctx starts the drain, but active handlers use a separate context
// and keep running until their delivery settles or a forced shutdown cancels it.
func (w *Worker) Run(ctx context.Context) error {
	if !w.started.CompareAndSwap(false, true) {
		return fmt.Errorf("%w: worker already started", ErrRejected)
	}
	defer close(w.runDone)

	handlerRoot, handlerCancel := context.WithCancel(context.WithoutCancel(ctx))
	w.mu.Lock()
	w.handlerCancel = handlerCancel
	w.mu.Unlock()

	consumeContexts, err := w.startConsumeContexts(handlerRoot)
	if err != nil {
		handlerCancel()
		return err
	}
	w.mu.Lock()
	w.consumeContexts = consumeContexts
	draining := w.draining.Load()
	w.mu.Unlock()
	if draining {
		drainConsumeContexts(consumeContexts)
	}
	unexpectedClose := make(chan struct{}, 1)
	var watchers sync.WaitGroup
	for _, consumeContext := range consumeContexts {
		watchers.Go(func() {
			<-consumeContext.Closed()
			if !w.draining.Load() {
				select {
				case unexpectedClose <- struct{}{}:
				default:
				}
			}
		})
	}

	var runErr error
	select {
	case <-w.drain:
	case runErr = <-w.terminal:
		w.StartDrain()
	case <-unexpectedClose:
		runErr = fmt.Errorf("%w: native consume context closed unexpectedly", ErrTerminal)
		w.StartDrain()
	case <-ctx.Done():
		runErr = fmt.Errorf("run durable consumer: %w", ctx.Err())
		w.StartDrain()
	}

	waitConsumeContexts(consumeContexts)
	watchers.Wait()
	if runErr == nil {
		select {
		case runErr = <-w.terminal:
		default:
		}
	}
	return runErr
}

func (w *Worker) startConsumeContexts(handlerRoot context.Context) ([]jetstream.ConsumeContext, error) {
	consumeContexts := make([]jetstream.ConsumeContext, 0, w.cfg.MaxConcurrency)
	for range w.cfg.MaxConcurrency {
		consumeContext, err := w.consumer.Consume(
			func(msg jetstream.Msg) {
				if handleErr := w.handle(handlerRoot, msg); handleErr != nil {
					w.failTerminal(handleErr)
				}
			},
			jetstream.PullMaxMessages(1),
			jetstream.PullExpiry(operationTimeout),
		)
		if err != nil {
			stopConsumeContexts(consumeContexts)
			waitConsumeContexts(consumeContexts)
			return nil, fmt.Errorf("%w: start consume context: %w", ErrRejected, err)
		}
		consumeContexts = append(consumeContexts, consumeContext)
	}
	return consumeContexts, nil
}

// StartDrain prevents new worker deliveries and publications through the
// shared Client. It does not cancel active handler contexts.
func (w *Worker) StartDrain() {
	if !w.draining.CompareAndSwap(false, true) {
		return
	}
	close(w.drain)
	w.mu.Lock()
	consumeContexts := append([]jetstream.ConsumeContext(nil), w.consumeContexts...)
	w.mu.Unlock()
	drainConsumeContexts(consumeContexts)
	w.client.StopPublish()
}

// Shutdown starts the drain and waits for Run before shutting down the Client.
// A forced-shutdown error only reports that its context expired; it does not
// prove a handler observed cancellation or finished.
func (w *Worker) Shutdown(ctx context.Context) error {
	w.StartDrain()
	select {
	case <-w.runDone:
		return w.client.Shutdown(ctx)
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
	consumeContexts := append([]jetstream.ConsumeContext(nil), w.consumeContexts...)
	w.mu.Unlock()
	stopConsumeContexts(consumeContexts)
	w.client.Close()
}

func (w *Worker) failTerminal(err error) {
	select {
	case w.terminal <- err:
	default:
	}
	w.StartDrain()
}

func drainConsumeContexts(consumeContexts []jetstream.ConsumeContext) {
	for _, consumeContext := range consumeContexts {
		consumeContext.Drain()
	}
}

func stopConsumeContexts(consumeContexts []jetstream.ConsumeContext) {
	for _, consumeContext := range consumeContexts {
		consumeContext.Stop()
	}
}

func waitConsumeContexts(consumeContexts []jetstream.ConsumeContext) {
	for _, consumeContext := range consumeContexts {
		<-consumeContext.Closed()
	}
}
