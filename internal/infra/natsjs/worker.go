package natsjs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Worker struct {
	client    *Client
	cfg       WorkerConfig
	consumer  pullConsumer
	dlqStream string
	handler   Handler

	draining atomic.Bool
	started  atomic.Bool
	terminal atomic.Bool
	fatal    chan error
	runDone  chan struct{}

	mu            sync.Mutex
	fetchCancel   context.CancelFunc
	handlerCancel context.CancelFunc
}

// pullConsumer is deliberately narrower than jetstream.Consumer: acquisition
// code cannot call continuous consumption or FetchBytes by construction.
type pullConsumer interface {
	Fetch(batch int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error)
	Info(ctx context.Context) (*jetstream.ConsumerInfo, error)
}

func (c *Client) NewWorker(ctx context.Context, cfg WorkerConfig, handler Handler) (*Worker, error) {
	if handler == nil {
		return nil, fmt.Errorf("%w: messaging handler is required", ErrRejected)
	}
	if err := ValidateWorkerConfig(cfg, c.cfg.MaxPayloadBytes); err != nil {
		return nil, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, boundedTimeout(ctx))
	defer cancel()
	source, err := c.js.Stream(probeCtx, c.cfg.Stream)
	if err != nil {
		return nil, fmt.Errorf("%w: source stream is unavailable", ErrRejected)
	}
	sourceInfo, err := source.Info(probeCtx)
	if err != nil {
		return nil, fmt.Errorf("%w: source stream configuration is unavailable", ErrRejected)
	}
	if sourceInfo.Config.MaxMsgSize <= 0 || int(sourceInfo.Config.MaxMsgSize) > cfg.MaxDeliveryBytes {
		return nil, fmt.Errorf("%w: source stream max message size is unbounded or exceeds worker delivery bound", ErrRejected)
	}
	dlqStream, err := c.js.StreamNameBySubject(probeCtx, cfg.DeadLetterSubject)
	if err != nil {
		return nil, fmt.Errorf("%w: dead-letter stream is unavailable", ErrRejected)
	}
	if dlqStream == c.cfg.Stream {
		return nil, fmt.Errorf("%w: source and dead-letter streams must differ", ErrRejected)
	}
	dlq, err := c.js.Stream(probeCtx, dlqStream)
	if err != nil {
		return nil, fmt.Errorf("%w: dead-letter stream is unavailable", ErrRejected)
	}
	dlqInfo, err := dlq.Info(probeCtx)
	if err != nil {
		return nil, fmt.Errorf("%w: dead-letter stream configuration is unavailable", ErrRejected)
	}
	minimumDLQSize := c.cfg.MaxPayloadBytes + HeaderLimitBytes
	if dlqInfo.Config.MaxMsgSize > 0 && int(dlqInfo.Config.MaxMsgSize) < minimumDLQSize {
		return nil, fmt.Errorf("%w: dead-letter stream cannot contain the configured envelope", ErrRejected)
	}
	desired := desiredConsumerConfig(cfg)
	consumer, err := c.js.Consumer(probeCtx, c.cfg.Stream, cfg.Consumer)
	if errors.Is(err, jetstream.ErrConsumerNotFound) {
		consumer, err = c.js.CreateConsumer(probeCtx, c.cfg.Stream, desired)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: durable consumer admission failed", ErrRejected)
	}
	info, err := consumer.Info(probeCtx)
	if err != nil {
		return nil, fmt.Errorf("%w: durable consumer configuration is unavailable", ErrRejected)
	}
	if !consumerConfigEqual(info.Config, desired) {
		return nil, fmt.Errorf("%w: existing durable consumer configuration is incompatible", ErrRejected)
	}
	w := &Worker{
		client:    c,
		cfg:       cfg,
		consumer:  consumer,
		dlqStream: dlqStream,
		handler:   handler,
		fatal:     make(chan error, 1),
		runDone:   make(chan struct{}),
	}
	c.probeMu.Lock()
	c.consumer = consumer
	c.probeMu.Unlock()
	if err := c.Check(ctx); err != nil {
		return nil, err
	}
	return w, nil
}

func desiredConsumerConfig(cfg WorkerConfig) jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		Name:          cfg.Consumer,
		Durable:       cfg.Consumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       cfg.HandlerTimeout + 11*time.Second,
		MaxDeliver:    -1,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
		// A canceled pull can remain broker-side until its five-second expiry.
		// Two slots let the one local fetch loop recover without treating that
		// bounded stale request as a terminal MaxWaiting failure.
		MaxWaiting:         2,
		MaxAckPending:      cfg.MaxConcurrency,
		MaxRequestBatch:    1,
		MaxRequestExpires:  operationTimeout,
		MaxRequestMaxBytes: cfg.MaxDeliveryBytes,
		FilterSubject:      cfg.FilterSubject,
	}
}

func consumerConfigEqual(actual, desired jetstream.ConsumerConfig) bool {
	actual.Description = ""
	actual.Metadata = nil
	desired.Description = ""
	desired.Metadata = nil
	actual.BackOff = nilIfEmpty(actual.BackOff)
	desired.BackOff = nilIfEmpty(desired.BackOff)
	actual.FilterSubjects = nilIfEmpty(actual.FilterSubjects)
	desired.FilterSubjects = nilIfEmpty(desired.FilterSubjects)
	actual.PriorityGroups = nilIfEmpty(actual.PriorityGroups)
	desired.PriorityGroups = nilIfEmpty(desired.PriorityGroups)
	return reflect.DeepEqual(actual, desired)
}

func nilIfEmpty[T any](values []T) []T {
	if len(values) == 0 {
		return nil
	}
	return values
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.started.CompareAndSwap(false, true) {
		return fmt.Errorf("%w: worker already started", ErrRejected)
	}
	defer close(w.runDone)
	fetchCtx, fetchCancel := context.WithCancel(ctx)
	handlerRoot, handlerCancel := context.WithCancel(context.WithoutCancel(ctx))
	w.mu.Lock()
	w.fetchCancel = fetchCancel
	w.handlerCancel = handlerCancel
	w.mu.Unlock()
	defer fetchCancel()

	completion := make(chan struct{}, w.cfg.MaxConcurrency)
	active, runErr := w.fetchLoop(ctx, fetchCtx, handlerRoot, completion)
	return w.waitForHandlers(completion, active, runErr)
}

func (w *Worker) fetchLoop(ctx, fetchCtx, handlerRoot context.Context, completion chan struct{}) (int, error) {
	active := 0
	for !w.draining.Load() {
		if err := w.pollStop(ctx); err != nil {
			return active, err
		}
		if active >= w.cfg.MaxConcurrency {
			if err := w.waitForHandler(ctx, completion); err != nil {
				return active, err
			}
			active--
			continue
		}
		pullCtx, cancel := context.WithTimeout(fetchCtx, operationTimeout)
		batch, err := w.consumer.Fetch(1, jetstream.FetchContext(pullCtx))
		if err != nil {
			cancel()
			if fetchCtx.Err() != nil {
				return active, nil //nolint:nilerr // Pull cancellation stops acquisition; active handler errors still win during drain.
			}
			if w.client.waitForReconnect(fetchCtx, err) {
				continue
			}
			runErr := fmt.Errorf("%w: fetch source message", ErrTerminal)
			w.fail(runErr)
			return active, runErr
		}
		for msg := range batch.Messages() {
			active++
			w.client.signals.fetchMessages.Add(ctx, 1)
			w.client.signals.fetchBytes.Add(ctx, int64(wireSize(msg)))
			go func() {
				defer func() { completion <- struct{}{} }()
				if err := w.handle(handlerRoot, msg); err != nil {
					w.fail(err)
				}
			}()
		}
		batchErr := batch.Error()
		cancel()
		if batchErr != nil && fetchCtx.Err() == nil && !errors.Is(batchErr, jetstream.ErrNoMessages) && !errors.Is(batchErr, context.DeadlineExceeded) {
			if w.client.waitForReconnect(fetchCtx, batchErr) {
				continue
			}
			runErr := fmt.Errorf("%w: fetch source message", ErrTerminal)
			w.fail(runErr)
			return active, runErr
		}
	}
	return active, nil
}

func (w *Worker) pollStop(ctx context.Context) error {
	select {
	case err := <-w.fatal:
		w.StartDrain()
		return err
	case <-ctx.Done():
		w.StartDrain()
		return fmt.Errorf("run durable consumer: %w", ctx.Err())
	default:
		return nil
	}
}

func (w *Worker) waitForHandler(ctx context.Context, completion <-chan struct{}) error {
	select {
	case <-completion:
		return nil
	case err := <-w.fatal:
		w.StartDrain()
		return err
	case <-ctx.Done():
		w.StartDrain()
		return fmt.Errorf("run durable consumer: %w", ctx.Err())
	}
}

func (w *Worker) waitForHandlers(completion <-chan struct{}, active int, runErr error) error {
	for active > 0 {
		select {
		case <-completion:
			active--
		case err := <-w.fatal:
			if runErr == nil {
				runErr = err
			}
		}
	}
	return runErr
}

func (w *Worker) StartDrain() {
	if !w.draining.CompareAndSwap(false, true) {
		return
	}
	w.client.StopPublish()
	w.mu.Lock()
	if w.fetchCancel != nil {
		w.fetchCancel()
	}
	w.mu.Unlock()
}

func (w *Worker) Shutdown(ctx context.Context) error {
	w.StartDrain()
	select {
	case <-w.runDone:
		if err := w.client.Shutdown(ctx); err != nil {
			w.client.signals.drainOperations.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "failed")))
			return err
		}
		w.client.signals.drainOperations.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "graceful")))
		return nil
	case <-ctx.Done():
		w.ForceClose()
		metricCtx := context.WithoutCancel(ctx)
		w.client.signals.drainOperations.Add(metricCtx, 1, metric.WithAttributes(attribute.String("outcome", "forced")))
		w.client.signals.forcedShutdowns.Add(metricCtx, 1)
		return fmt.Errorf("forced messaging shutdown: %w", ctx.Err())
	}
}

func (w *Worker) ForceClose() {
	w.StartDrain()
	w.terminal.Store(true)
	w.mu.Lock()
	if w.handlerCancel != nil {
		w.handlerCancel()
	}
	w.mu.Unlock()
	w.client.Close()
}

func (w *Worker) handle(handlerRoot context.Context, source jetstream.Msg) error {
	metadata, err := source.Metadata()
	if err != nil {
		return fmt.Errorf("%w: source metadata unavailable", ErrTerminal)
	}
	if wireSize(source) > w.cfg.MaxDeliveryBytes || len(source.Data()) > w.client.cfg.MaxPayloadBytes {
		return fmt.Errorf("%w: retained source exceeds admitted message bound", ErrTerminal)
	}
	if metadata.NumDelivered > 1 {
		w.client.signals.redeliveries.Add(handlerRoot, 1)
	}
	extracted, decoded, decodeErr := decodeMessage(source, metadata)
	if decodeErr != nil {
		return w.deadLetter(handlerRoot, source, metadata, Message{}, "malformed")
	}
	attemptLimit := uint64(1 + len(w.cfg.RetryDelays))
	if metadata.NumDelivered > attemptLimit {
		return w.deadLetter(handlerRoot, source, metadata, decoded, "exhausted")
	}
	base := contextWithRemoteParent(handlerRoot, extracted)
	ctx, span := w.client.signals.tracer.Start(base, "messaging consume",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.operation.type", "process"),
			attribute.String("messaging.destination.name", decoded.Subject()),
			attribute.String("messaging.message.id", decoded.MessageID()),
		),
	)
	defer span.End()
	handlerCtx, cancel := context.WithTimeout(ctx, w.cfg.HandlerTimeout)
	started := time.Now()
	w.client.signals.consumeActive.Add(ctx, 1)
	handlerErr, handlerPanicked := w.invokeHandler(handlerCtx, decoded)
	handlerContextErr := handlerCtx.Err()
	w.client.signals.consumeActive.Add(ctx, -1)
	cancel()
	if handlerPanicked {
		w.client.signals.handler(ctx, decoded, "terminal", "handler_panic", started)
		return fmt.Errorf("%w: feature handler panicked", ErrTerminal)
	}
	if handlerErr == nil {
		ackCtx, ackCancel := context.WithTimeout(handlerRoot, operationTimeout)
		ackErr := source.DoubleAck(ackCtx)
		ackCancel()
		if ackErr != nil {
			w.client.signals.handler(ctx, decoded, "success", "ack_ambiguous", started)
			return nil
		}
		w.client.signals.handler(ctx, decoded, "success", "none", started)
		return nil
	}
	if handlerRoot.Err() != nil {
		w.client.signals.handler(ctx, decoded, "canceled", "shutdown", started)
		return nil
	}
	if isPermanent(handlerErr) {
		w.client.signals.handler(ctx, decoded, "permanent", "handler_permanent", started)
		return w.deadLetter(handlerRoot, source, metadata, decoded, "permanent")
	}
	if metadata.NumDelivered >= attemptLimit {
		outcome := "retryable"
		if errors.Is(handlerContextErr, context.DeadlineExceeded) {
			outcome = "timeout"
		}
		w.client.signals.handler(ctx, decoded, outcome, "handler_exhausted", started)
		return w.deadLetter(handlerRoot, source, metadata, decoded, "exhausted")
	}
	outcome := "retryable"
	if errors.Is(handlerContextErr, context.DeadlineExceeded) {
		outcome = "timeout"
	} else if errors.Is(handlerErr, context.Canceled) {
		outcome = "canceled"
	}
	w.client.signals.handler(ctx, decoded, outcome, "handler_retry", started)
	w.client.signals.retries.Add(ctx, 1)
	_ = source.NakWithDelay(w.cfg.RetryDelays[metadata.NumDelivered-1])
	return nil
}

func (w *Worker) invokeHandler(ctx context.Context, msg Message) (err error, panicked bool) {
	defer func() {
		if recover() != nil {
			err = nil
			panicked = true
		}
	}()
	return w.handler(ctx, msg), false
}

func (w *Worker) deadLetter(ctx context.Context, source jetstream.Msg, metadata *jetstream.MsgMetadata, decoded Message, reason string) error {
	msg, transferID := deadLetterMessage(source, metadata, decoded, reason)
	msg.Subject = w.cfg.DeadLetterSubject
	if err := validateEncodedMessage(msg, w.client.cfg.MaxPayloadBytes); err != nil {
		w.client.signals.dlqTransfers.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "rejected")))
		return fmt.Errorf("%w: retained source cannot fit dead-letter envelope", ErrTerminal)
	}
	publishCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	_, err := w.client.js.PublishMsg(
		publishCtx,
		msg,
		jetstream.WithMsgID(transferID),
		jetstream.WithExpectStream(w.dlqStream),
		jetstream.WithRetryAttempts(0),
	)
	cancel()
	if err != nil {
		outcome, _, wrapped := classifyPublishError(err)
		w.client.signals.dlqTransfers.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
		if errors.Is(wrapped, ErrAmbiguous) {
			_ = source.NakWithDelay(w.cfg.DeadLetterRetryDelay)
			return nil
		}
		return fmt.Errorf("%w: dead-letter publish rejected", ErrTerminal)
	}
	w.client.signals.dlqTransfers.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "accepted")))
	ackCtx, ackCancel := context.WithTimeout(ctx, operationTimeout)
	err = source.DoubleAck(ackCtx)
	ackCancel()
	if err != nil {
		_ = source.NakWithDelay(w.cfg.DeadLetterRetryDelay)
	}
	return nil
}

func (w *Worker) fail(err error) {
	w.terminal.Store(true)
	w.StartDrain()
	w.client.signalTerminal(err)
	select {
	case w.fatal <- err:
	default:
	}
}

func wireSize(msg jetstream.Msg) int {
	return (&nats.Msg{Subject: msg.Subject(), Header: msg.Headers(), Data: msg.Data()}).Size()
}
