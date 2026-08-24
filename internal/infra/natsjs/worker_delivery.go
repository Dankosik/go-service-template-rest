package natsjs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

// delivery is one message as this worker sees it: the broker's view, its
// delivery metadata, and — once admitted — the decoded envelope handed to the
// feature handler.
type delivery struct {
	source   jetstream.Msg
	metadata *jetstream.MsgMetadata
	message  Message
}

// handlerPanic is what a recovered feature-handler panic contributes to its
// terminal-delivery record: the bounded class an operator alerts on, and the
// frames naming the defect behind it. Never the recovered value — it carries
// whatever the handler was working on, and internal/observability/logctx owns
// that argument.
type handlerPanic struct {
	class  string
	frames []string
}

// handlerResult is one feature-handler invocation: the panic it raised, what it
// returned, whether its own timeout fired, and when it started.
type handlerResult struct {
	panicked   *handlerPanic
	err        error
	contextErr error
	started    time.Time
}

// handle runs one delivery end to end: admit it, invoke the feature handler
// under its own span and timeout, then settle the source message with the
// broker. A returned error stops the worker; an ordinary retry, dead-letter, or
// shutdown cancellation returns nil.
func (w *Worker) handle(handlerRoot context.Context, source jetstream.Msg) error {
	metadata, err := source.Metadata()
	if err != nil {
		w.client.telemetry.logTerminalDelivery(handlerRoot, source.Subject(), nil, reasonMetadataUnavailable, nil)
		return fmt.Errorf("%w: source metadata unavailable", ErrTerminal)
	}
	current := delivery{source: source, metadata: metadata}
	decoded, base, err := w.admit(handlerRoot, current)
	if errors.Is(err, errSettledWithoutHandler) {
		return nil
	}
	if err != nil {
		return err
	}
	current.message = decoded

	ctx, span := w.client.telemetry.tracer.Start(
		base, consumeSpanName(w.cfg.FilterSubject), consumeSpanOptions(decoded, w.cfg.FilterSubject)...,
	)
	defer span.End()
	handlerCtx, cancel := context.WithTimeout(ctx, w.cfg.HandlerTimeout)
	result := handlerResult{started: time.Now()}
	result.panicked, result.err = w.invokeHandler(handlerCtx, decoded)
	result.contextErr = handlerCtx.Err()
	cancel()

	return w.settle(ctx, handlerRoot, current, result)
}

// errSettledWithoutHandler reports that a gate in admit already settled the
// delivery with the broker — it was dead-lettered — so the feature handler must
// not run and the worker must not fault. Worker.handle is the only reader; it
// never reaches a caller.
var errSettledWithoutHandler = errors.New("delivery settled without the handler")

// admit runs every gate that applies before the feature handler sees a message:
// the delivery bounds this worker was configured with, envelope decoding, and
// the attempt budget.
//
// A nil error means the handler may run, and the results are its envelope and
// its context. [errSettledWithoutHandler] means a gate dead-lettered the
// delivery itself, which finishes the message without the handler and without
// faulting the worker. Every other error stops the worker.
func (w *Worker) admit(handlerRoot context.Context, current delivery) (Message, context.Context, error) {
	source, metadata := current.source, current.metadata
	if wireSize(source) > w.cfg.MaxDeliveryBytes || len(source.Data()) > w.client.cfg.MaxPayloadBytes {
		w.client.telemetry.logTerminalDelivery(handlerRoot, source.Subject(), metadata, reasonDeliveryBound, nil)
		return Message{}, nil, fmt.Errorf("%w: retained source exceeds admitted message bound", ErrTerminal)
	}
	if encodedHeaderBytes(source.Headers()) > HeaderLimitBytes {
		return Message{}, nil, settled(w.deadLetter(handlerRoot, source, metadata, Message{}, deadLetterMalformed))
	}
	decoded, remote, decodeErr := decodeMessage(source, metadata)
	if decodeErr != nil {
		return Message{}, nil, settled(w.deadLetter(handlerRoot, source, metadata, Message{}, deadLetterMalformed))
	}
	if metadata.NumDelivered > w.attemptLimit() {
		return Message{}, nil, settled(w.deadLetter(handlerRoot, source, metadata, decoded, deadLetterExhausted))
	}
	return decoded, contextWithRemoteParent(handlerRoot, remote), nil
}

// settled turns a completed dead-letter into the signal that the handler must
// not run. A dead-letter that failed keeps its own error, which stops the
// worker; that is the one case a gate must not swallow.
func settled(deadLetterErr error) error {
	if deadLetterErr != nil {
		return deadLetterErr
	}
	return errSettledWithoutHandler
}

// settle records the delivery's outcome and gives the broker its instruction:
// acknowledge, dead-letter, or request a delayed redelivery. Only a fault that
// must stop the whole worker returns an error.
func (w *Worker) settle(ctx, handlerRoot context.Context, current delivery, result handlerResult) error {
	telemetry := w.client.telemetry
	if result.panicked != nil {
		telemetry.recordHandler(ctx, current.message, outcomeTerminal, reasonHandlerPanic, result.started)
		telemetry.logTerminalDelivery(ctx, current.message.Subject(), current.metadata, reasonHandlerPanic, result.panicked)
		return fmt.Errorf("%w: feature handler panicked", ErrTerminal)
	}
	if result.err == nil {
		return w.acknowledge(ctx, handlerRoot, current, result.started)
	}
	if handlerRoot.Err() != nil {
		// Shutdown cancelled the handler. The source stays unacknowledged, so
		// the broker redelivers it rather than this process retrying it.
		telemetry.recordHandler(ctx, current.message, outcomeCanceled, reasonShutdown, result.started)
		return nil //nolint:nilerr // Shutdown is not a worker fault; the message is left for redelivery.
	}
	if IsPermanent(result.err) {
		telemetry.recordHandler(ctx, current.message, outcomePermanent, reasonHandlerPermanent, result.started)
		return w.deadLetter(handlerRoot, current.source, current.metadata, current.message, deadLetterPermanent)
	}
	exhausted := current.metadata.NumDelivered >= w.attemptLimit()
	outcome := handlerOutcome(result, exhausted)
	if exhausted {
		telemetry.recordHandler(ctx, current.message, outcome, reasonHandlerExhausted, result.started)
		return w.deadLetter(handlerRoot, current.source, current.metadata, current.message, deadLetterExhausted)
	}
	telemetry.recordHandler(ctx, current.message, outcome, reasonHandlerRetry, result.started)
	return w.requestRedelivery(
		ctx, current.source, current.metadata,
		w.retryDelayFor(current.metadata.NumDelivered), redeliveryHandler,
	)
}

// acknowledge confirms the delivery with the broker. A lost acknowledgement is
// not a failure of the handler's work: it already succeeded, and the redelivery
// that follows is the duplicate every handler here must already tolerate.
func (w *Worker) acknowledge(ctx, handlerRoot context.Context, current delivery, started time.Time) error {
	ackCtx, cancel := context.WithTimeout(handlerRoot, operationTimeout)
	err := current.source.DoubleAck(ackCtx)
	cancel()
	reason := reasonNone
	if err != nil {
		reason = reasonAckAmbiguous
	}
	w.client.telemetry.recordHandler(ctx, current.message, outcomeSuccess, reason, started)
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
		return outcomeTimeout
	case !exhausted && errors.Is(result.err, context.Canceled):
		return outcomeCanceled
	default:
		return outcomeRetryable
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

func (w *Worker) invokeHandler(ctx context.Context, msg Message) (panicked *handlerPanic, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = nil
			panicked = &handlerPanic{
				class:  logctx.PanicClass(recovered),
				frames: captureHandlerPanicFrames(),
			}
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
		w.client.telemetry.recordDeadLetterTransfer(ctx, outcomeRejected)
		w.client.telemetry.logTerminalDelivery(ctx, source.Subject(), metadata, reasonDeadLetterEnvelope, nil)
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
		w.client.telemetry.recordDeadLetterTransfer(ctx, outcome)
		if errors.Is(wrapped, ErrAmbiguous) {
			return w.requestRedelivery(ctx, source, metadata, w.cfg.DeadLetterRetryDelay, redeliveryDeadLetter)
		}
		w.client.telemetry.logTerminalDelivery(ctx, source.Subject(), metadata, reasonDeadLetterRejected, nil)
		return fmt.Errorf("%w: dead-letter publish rejected", ErrTerminal)
	}
	w.client.telemetry.recordDeadLetterTransfer(ctx, outcomeAccepted)
	ackCtx, ackCancel := context.WithTimeout(ctx, operationTimeout)
	err = source.DoubleAck(ackCtx)
	ackCancel()
	if err != nil {
		return w.requestRedelivery(ctx, source, metadata, w.cfg.DeadLetterRetryDelay, redeliverySourceAck)
	}
	return nil
}

func (w *Worker) requestRedelivery(ctx context.Context, source jetstream.Msg, metadata *jetstream.MsgMetadata, delay time.Duration, reason string) error {
	if err := source.NakWithDelay(delay); err != nil {
		w.client.telemetry.logTerminalDelivery(ctx, source.Subject(), metadata, reason+"_rejected", nil)
		return fmt.Errorf("%w: request delayed source redelivery", ErrTerminal)
	}
	return nil
}
