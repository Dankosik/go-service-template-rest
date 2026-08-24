package natsjs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationScope = "service.messaging.nats"

const (
	systemNATS           = "nats"
	operationNamePublish = "publish"
	operationNameProcess = "process"
)

func publishSpanName(subject string) string { return operationNamePublish + " " + subject }
func consumeSpanName(filter string) string  { return operationNameProcess + " " + filter }

type Observability struct {
	Logger *slog.Logger
	Meter  metric.Meter
	Tracer trace.Tracer
}

type telemetry struct {
	log               *slog.Logger
	tracer            trace.Tracer
	publishOperations metric.Int64Counter
	publishDuration   metric.Float64Histogram
	handlerOperations metric.Int64Counter
	handlerDuration   metric.Float64Histogram
	dlqTransfers      metric.Int64Counter
}

func newTelemetry(obs Observability) (*telemetry, error) {
	if obs.Logger == nil {
		obs.Logger = slog.Default()
	}
	if obs.Meter == nil {
		obs.Meter = otel.GetMeterProvider().Meter(instrumentationScope)
	}
	if obs.Tracer == nil {
		obs.Tracer = otel.GetTracerProvider().Tracer(instrumentationScope)
	}
	t := &telemetry{log: obs.Logger, tracer: obs.Tracer}
	var err error
	if t.publishOperations, err = obs.Meter.Int64Counter("messaging.publish.operations"); err != nil {
		return nil, fmt.Errorf("create messaging.publish.operations metric: %w", err)
	}
	if t.publishDuration, err = obs.Meter.Float64Histogram("messaging.publish.duration", metric.WithUnit("s")); err != nil {
		return nil, fmt.Errorf("create messaging.publish.duration metric: %w", err)
	}
	if t.handlerOperations, err = obs.Meter.Int64Counter("messaging.handler.operations"); err != nil {
		return nil, fmt.Errorf("create messaging.handler.operations metric: %w", err)
	}
	if t.handlerDuration, err = obs.Meter.Float64Histogram("messaging.handler.duration", metric.WithUnit("s")); err != nil {
		return nil, fmt.Errorf("create messaging.handler.duration metric: %w", err)
	}
	if t.dlqTransfers, err = obs.Meter.Int64Counter("messaging.dlq.transfers"); err != nil {
		return nil, fmt.Errorf("create messaging.dlq.transfers metric: %w", err)
	}
	return t, nil
}

//nolint:ireturn // metric.WithAttributes returns OTel's option interface.
func outcomeAttribute(outcome string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String("outcome", boundedOutcome(outcome)))
}

func publishSpanOptions(event Event) []trace.SpanStartOption {
	return []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String(systemNATS),
			semconv.MessagingOperationTypeSend,
			semconv.MessagingOperationName(operationNamePublish),
			semconv.MessagingDestinationName(event.Subject),
		),
	}
}

func consumeSpanOptions(msg Message, filter string) []trace.SpanStartOption {
	return []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String(systemNATS),
			semconv.MessagingOperationTypeProcess,
			semconv.MessagingOperationName(operationNameProcess),
			semconv.MessagingDestinationName(msg.Subject()),
			semconv.MessagingDestinationTemplate(filter),
		),
	}
}

func setSpanOutcome(span trace.Span, outcome string) {
	span.SetAttributes(attribute.String(attributeOutcome, boundedOutcome(outcome)))
}

func (t *telemetry) recordPublish(ctx context.Context, event Event, outcome, reason string, started time.Time) {
	duration := time.Since(started).Seconds()
	attrs := outcomeAttribute(outcome)
	t.publishOperations.Add(ctx, 1, attrs)
	t.publishDuration.Record(ctx, duration, attrs)
	if outcome != outcomeAccepted {
		t.log.WarnContext(ctx, "messaging_publish_failed",
			"operation", "publish", "subject", event.Subject,
			"outcome", outcome, "duration_seconds", duration, "reason", reason,
		)
	}
}

func (t *telemetry) recordDeadLetterTransfer(ctx context.Context, outcome string) {
	t.dlqTransfers.Add(ctx, 1, outcomeAttribute(outcome))
}

func (t *telemetry) recordAsyncError(ctx context.Context, err error) {
	t.log.WarnContext(ctx, "messaging_connection",
		"operation", "connection", "outcome", connectionAsyncError, "reason", asyncErrorReason(err),
	)
}

func asyncErrorReason(err error) string {
	switch {
	case errors.Is(err, nats.ErrSlowConsumer):
		return reasonSlowConsumer
	case errors.Is(err, nats.ErrAuthorization), errors.Is(err, nats.ErrAuthExpired), errors.Is(err, nats.ErrAuthRevoked):
		return reasonAuthentication
	case errors.Is(err, nats.ErrPermissionViolation):
		return reasonPermission
	case errors.Is(err, nats.ErrMaxPayload):
		return reasonMessageBound
	case errors.Is(err, nats.ErrConnectionClosed), errors.Is(err, nats.ErrDisconnected), errors.Is(err, nats.ErrStaleConnection):
		return reasonConnection
	default:
		return boundedOther
	}
}

func (t *telemetry) recordHandler(ctx context.Context, msg Message, outcome, reason string, started time.Time) {
	duration := time.Since(started).Seconds()
	attrs := outcomeAttribute(outcome)
	t.handlerOperations.Add(ctx, 1, attrs)
	t.handlerDuration.Record(ctx, duration, attrs)
	if outcome != outcomeSuccess {
		t.log.WarnContext(ctx, "messaging_delivery_failed",
			"operation", "consume", "subject", msg.Subject(), "attempt", msg.Metadata().NumDelivered,
			"outcome", outcome, "duration_seconds", duration, "reason", reason,
		)
	}
}

func (t *telemetry) logTerminalDelivery(ctx context.Context, subject string, metadata *jetstream.MsgMetadata, reason string, panicked *handlerPanic) {
	args := []any{"operation", "consume", "subject", subject, "outcome", outcomeTerminal, "reason", reason}
	if metadata != nil {
		args = append(args, "attempt", metadata.NumDelivered)
	}
	if panicked != nil {
		args = append(args, "panic.class", panicked.class, "handler_frames", panicked.frames)
	}
	t.log.ErrorContext(ctx, "messaging_terminal_delivery", args...)
}
