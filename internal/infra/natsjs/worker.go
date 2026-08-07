package natsjs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
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

const settlementSchedulingSlack = time.Second

func (c *Client) NewWorker(ctx context.Context, cfg WorkerConfig, handler Handler) (*Worker, error) {
	if handler == nil {
		return nil, fmt.Errorf("%w: messaging handler is required", ErrRejected)
	}
	if err := ValidateWorkerConfig(cfg, c.cfg.MaxPayloadBytes); err != nil {
		return nil, err
	}
	c.probeMu.Lock()
	if c.workerClaimed || c.consumer != nil {
		c.probeMu.Unlock()
		return nil, fmt.Errorf("%w: messaging client already owns a worker", ErrRejected)
	}
	c.workerClaimed = true
	c.probeMu.Unlock()
	defer func() {
		c.probeMu.Lock()
		c.workerClaimed = false
		c.probeMu.Unlock()
	}()
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
		c.probeMu.Lock()
		c.consumer = nil
		c.probeMu.Unlock()
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
		// Cover the handler, DLQ publish, source acknowledgement, and a small
		// scheduling margin without exposing another operator setting.
		AckWait:      cfg.HandlerTimeout + 2*operationTimeout + settlementSchedulingSlack,
		MaxDeliver:   -1,
		ReplayPolicy: jetstream.ReplayInstantPolicy,
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
			if runErr := w.recoverFetch(fetchCtx, err); runErr != nil {
				return active, runErr
			}
			continue
		}
		for msg := range batch.Messages() {
			if !w.startHandler(ctx, handlerRoot, msg, completion) {
				break
			}
			active++
		}
		batchErr := batch.Error()
		cancel()
		if batchErr != nil && !pullExhausted(batchErr) && fetchCtx.Err() == nil {
			if runErr := w.recoverFetch(fetchCtx, batchErr); runErr != nil {
				return active, runErr
			}
		}
	}
	return active, nil
}

// recoverFetch decides what a failed pull means. It returns nil to pull again,
// having waited for the connection to come back, and a terminal error when the
// failure is not a reconnect this client can ride out.
func (w *Worker) recoverFetch(fetchCtx context.Context, err error) error {
	if w.client.waitForReconnect(fetchCtx, err) {
		return nil
	}
	runErr := fmt.Errorf("%w: fetch source message", ErrTerminal)
	w.fail(runErr)
	return runErr
}

// pullExhausted is the ordinary end of a pull that delivered nothing: no
// message arrived within the request's own expiry.
func pullExhausted(err error) bool {
	return errors.Is(err, jetstream.ErrNoMessages) || errors.Is(err, context.DeadlineExceeded)
}

// startHandler launches one handler goroutine and reports whether it did.
//
// The check and the launch share one critical section because StartDrain sets
// draining under the same lock: that is what stops a handler from starting
// after drain began. A handler started later would not be counted in fetchLoop's
// active total, so Run would return without ever joining it.
func (w *Worker) startHandler(ctx, handlerRoot context.Context, msg jetstream.Msg, completion chan<- struct{}) bool {
	w.mu.Lock()
	if w.draining.Load() {
		w.mu.Unlock()
		return false
	}
	// Only past the drain check is msg known to be a real message: a batch that
	// was already in flight when drain began can still yield a nil one.
	messageBytes := wireSize(msg)
	go func() {
		defer func() { completion <- struct{}{} }()
		if err := w.handle(handlerRoot, msg); err != nil {
			w.fail(err)
		}
	}()
	w.mu.Unlock()
	w.client.telemetry.fetchMessages.Add(ctx, 1)
	w.client.telemetry.fetchBytes.Add(ctx, int64(messageBytes))
	return true
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
	// fail publishes the terminal error before the handler publishes its
	// completion. The final completion can still win a select when both channels
	// are ready, so inspect the already-published terminal result once more before
	// reporting a graceful stop.
	if runErr == nil {
		select {
		case runErr = <-w.fatal:
		default:
		}
	}
	return runErr
}

func (w *Worker) StartDrain() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.draining.Load() {
		return
	}
	w.draining.Store(true)
	w.client.StopPublish()
	if w.fetchCancel != nil {
		w.fetchCancel()
	}
}

func (w *Worker) Shutdown(ctx context.Context) error {
	w.StartDrain()
	select {
	case <-w.runDone:
		if err := w.client.Shutdown(ctx); err != nil {
			w.client.telemetry.drainOperations.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "failed")))
			return err
		}
		w.client.telemetry.drainOperations.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "graceful")))
		return nil
	case <-ctx.Done():
		w.forceClose()
		metricCtx := context.WithoutCancel(ctx)
		w.client.telemetry.drainOperations.Add(metricCtx, 1, metric.WithAttributes(attribute.String("outcome", "forced")))
		w.client.telemetry.forcedShutdowns.Add(metricCtx, 1)
		return fmt.Errorf("forced messaging shutdown: %w", ctx.Err())
	}
}

func (w *Worker) forceClose() {
	w.StartDrain()
	w.mu.Lock()
	if w.handlerCancel != nil {
		w.handlerCancel()
	}
	w.mu.Unlock()
	w.client.Close()
}

// delivery is one message as this worker sees it: the broker's view, its
// delivery metadata, and — once admitted — the decoded envelope handed to the
// feature handler.
type delivery struct {
	source   jetstream.Msg
	metadata *jetstream.MsgMetadata
	message  Message
}

// handlerResult is one feature-handler invocation: the frames if it panicked,
// what it returned, whether its own timeout fired, and when it started.
type handlerResult struct {
	panicFrames []string
	err         error
	contextErr  error
	started     time.Time
}

// handle runs one delivery end to end: admit it, invoke the feature handler
// under its own span and timeout, then settle the source message with the
// broker. A returned error stops the worker; an ordinary retry, dead-letter, or
// shutdown cancellation returns nil.
func (w *Worker) handle(handlerRoot context.Context, source jetstream.Msg) error {
	metadata, err := source.Metadata()
	if err != nil {
		w.client.telemetry.terminal(handlerRoot, source.Subject(), nil, "metadata_unavailable", nil)
		return fmt.Errorf("%w: source metadata unavailable", ErrTerminal)
	}
	current := delivery{source: source, metadata: metadata}
	decoded, base, proceed, err := w.admit(handlerRoot, current)
	if !proceed {
		return err
	}
	current.message = decoded

	ctx, span := w.client.telemetry.tracer.Start(base, "messaging consume",
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
	result := handlerResult{started: time.Now()}
	w.client.telemetry.consumeActive.Add(ctx, 1)
	result.panicFrames, result.err = w.invokeHandler(handlerCtx, decoded)
	result.contextErr = handlerCtx.Err()
	w.client.telemetry.consumeActive.Add(ctx, -1)
	cancel()

	return w.settle(ctx, handlerRoot, current, result)
}

// admit runs every gate that applies before the feature handler sees a message:
// the delivery bounds this worker was configured with, envelope decoding, and
// the attempt budget. proceed is false once admit has already settled the
// delivery, and err is then the whole outcome — nil when the message went to
// the dead-letter stream, terminal when the worker must stop.
func (w *Worker) admit(
	handlerRoot context.Context,
	current delivery,
) (decoded Message, base context.Context, proceed bool, err error) {
	source, metadata := current.source, current.metadata
	if wireSize(source) > w.cfg.MaxDeliveryBytes || len(source.Data()) > w.client.cfg.MaxPayloadBytes {
		w.client.telemetry.terminal(handlerRoot, source.Subject(), metadata, "delivery_bound", nil)
		return Message{}, nil, false, fmt.Errorf("%w: retained source exceeds admitted message bound", ErrTerminal)
	}
	if encodedHeaderBytes(source.Headers()) > HeaderLimitBytes {
		return Message{}, nil, false, w.deadLetter(handlerRoot, source, metadata, Message{}, "malformed")
	}
	if metadata.NumDelivered > 1 {
		w.client.telemetry.redeliveries.Add(handlerRoot, 1)
	}
	decoded, remote, decodeErr := decodeMessage(source, metadata)
	if decodeErr != nil {
		return Message{}, nil, false, w.deadLetter(handlerRoot, source, metadata, Message{}, "malformed")
	}
	if metadata.NumDelivered > w.attemptLimit() {
		return Message{}, nil, false, w.deadLetter(handlerRoot, source, metadata, decoded, "exhausted")
	}
	return decoded, contextWithRemoteParent(handlerRoot, remote), true, nil
}

// settle records the delivery's outcome and gives the broker its instruction:
// acknowledge, dead-letter, or request a delayed redelivery. Only a fault that
// must stop the whole worker returns an error.
func (w *Worker) settle(ctx, handlerRoot context.Context, current delivery, result handlerResult) error {
	telemetry := w.client.telemetry
	if len(result.panicFrames) != 0 {
		telemetry.handler(ctx, current.message, "terminal", "handler_panic", result.started)
		telemetry.terminal(ctx, current.message.Subject(), current.metadata, "handler_panic", result.panicFrames)
		return fmt.Errorf("%w: feature handler panicked", ErrTerminal)
	}
	if result.err == nil {
		return w.acknowledge(ctx, handlerRoot, current, result.started)
	}
	if handlerRoot.Err() != nil {
		// Shutdown cancelled the handler. The source stays unacknowledged, so
		// the broker redelivers it rather than this process retrying it.
		telemetry.handler(ctx, current.message, "canceled", "shutdown", result.started)
		return nil //nolint:nilerr // Shutdown is not a worker fault; the message is left for redelivery.
	}
	if isPermanent(result.err) {
		telemetry.handler(ctx, current.message, "permanent", "handler_permanent", result.started)
		return w.deadLetter(handlerRoot, current.source, current.metadata, current.message, "permanent")
	}
	exhausted := current.metadata.NumDelivered >= w.attemptLimit()
	outcome := handlerOutcome(result, exhausted)
	if exhausted {
		telemetry.handler(ctx, current.message, outcome, "handler_exhausted", result.started)
		return w.deadLetter(handlerRoot, current.source, current.metadata, current.message, "exhausted")
	}
	telemetry.handler(ctx, current.message, outcome, "handler_retry", result.started)
	telemetry.retries.Add(ctx, 1)
	return w.requestRedelivery(
		ctx, current.source, current.metadata,
		w.retryDelayFor(current.metadata.NumDelivered), "handler_redelivery",
	)
}

// acknowledge confirms the delivery with the broker. A lost acknowledgement is
// not a failure of the handler's work: it already succeeded, and the redelivery
// that follows is the duplicate every handler here must already tolerate.
func (w *Worker) acknowledge(ctx, handlerRoot context.Context, current delivery, started time.Time) error {
	ackCtx, cancel := context.WithTimeout(handlerRoot, operationTimeout)
	err := current.source.DoubleAck(ackCtx)
	cancel()
	reason := "none"
	if err != nil {
		reason = "ack_ambiguous"
	}
	w.client.telemetry.handler(ctx, current.message, "success", reason, started)
	return nil
}

// handlerOutcome labels one failed invocation for telemetry. A handler that
// outran its own timeout is a timeout. A handler that returned a cancellation
// of its own is reported as such only while the message still has attempts
// left: an exhausted message goes to the dead-letter stream whatever ended it,
// so that path keeps the plain retryable label.
func handlerOutcome(result handlerResult, exhausted bool) string {
	switch {
	case errors.Is(result.contextErr, context.DeadlineExceeded):
		return "timeout"
	case !exhausted && errors.Is(result.err, context.Canceled):
		return "canceled"
	default:
		return "retryable"
	}
}

// attemptLimit is how many deliveries a message gets before the dead-letter
// stream: the first, plus one per configured retry delay.
func (w *Worker) attemptLimit() uint64 {
	return uint64(1 + len(w.cfg.RetryDelays))
}

// retryDelayFor selects the delay before the redelivery that follows attempt
// numDelivered. settle calls it only below attemptLimit, which is exactly the
// range RetryDelays covers.
func (w *Worker) retryDelayFor(numDelivered uint64) time.Duration {
	return w.cfg.RetryDelays[numDelivered-1]
}

func (w *Worker) invokeHandler(ctx context.Context, msg Message) (panicFrames []string, err error) {
	defer func() {
		if recover() != nil {
			err = nil
			panicFrames = captureHandlerPanicFrames()
		}
	}()
	return nil, w.handler(ctx, msg)
}

func captureHandlerPanicFrames() []string {
	programCounters := make([]uintptr, 16)
	count := runtime.Callers(3, programCounters)
	iterator := runtime.CallersFrames(programCounters[:count])
	frames := make([]string, 0, 8)
	for len(frames) < cap(frames) {
		frame, more := iterator.Next()
		if frame.Function != "" && !strings.HasPrefix(frame.Function, "runtime.") && !strings.Contains(frame.Function, ".invokeHandler") {
			frames = append(frames, fmt.Sprintf("%s %s:%d", frame.Function, filepath.Base(frame.File), frame.Line))
		}
		if !more {
			break
		}
	}
	return frames
}

func (w *Worker) deadLetter(ctx context.Context, source jetstream.Msg, metadata *jetstream.MsgMetadata, decoded Message, reason string) error {
	msg, transferID := deadLetterMessage(source, metadata, decoded, reason)
	msg.Subject = w.cfg.DeadLetterSubject
	if err := validateEncodedMessage(msg, w.client.cfg.MaxPayloadBytes); err != nil {
		w.client.telemetry.dlqTransfers.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "rejected")))
		w.client.telemetry.terminal(ctx, source.Subject(), metadata, "dlq_envelope", nil)
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
		w.client.telemetry.dlqTransfers.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
		if errors.Is(wrapped, ErrAmbiguous) {
			return w.requestRedelivery(ctx, source, metadata, w.cfg.DeadLetterRetryDelay, "dlq_redelivery")
		}
		w.client.telemetry.terminal(ctx, source.Subject(), metadata, "dlq_rejected", nil)
		return fmt.Errorf("%w: dead-letter publish rejected", ErrTerminal)
	}
	w.client.telemetry.dlqTransfers.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "accepted")))
	ackCtx, ackCancel := context.WithTimeout(ctx, operationTimeout)
	err = source.DoubleAck(ackCtx)
	ackCancel()
	if err != nil {
		return w.requestRedelivery(ctx, source, metadata, w.cfg.DeadLetterRetryDelay, "source_ack_redelivery")
	}
	return nil
}

func (w *Worker) requestRedelivery(ctx context.Context, source jetstream.Msg, metadata *jetstream.MsgMetadata, delay time.Duration, reason string) error {
	if err := source.NakWithDelay(delay); err != nil {
		w.client.telemetry.terminal(ctx, source.Subject(), metadata, reason+"_rejected", nil)
		return fmt.Errorf("%w: request delayed source redelivery", ErrTerminal)
	}
	return nil
}

func (w *Worker) fail(err error) {
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
