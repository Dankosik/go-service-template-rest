package natsjs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationScope = "service.messaging.nats"

// The span attributes, which are OpenTelemetry's messaging convention rather
// than names this package chose. They are named beside the two functions below
// that write them, because a publish span and a consume span must carry the
// same keys to be comparable, and two call sites spelling them out separately
// is how one gains a key the other never learns about.
//
// The metric and log label vocabulary is vocabulary.go; the dead-letter reasons
// that travel on the wire are message_wire.go.
const (
	attributeSystem        = "messaging.system"
	attributeOperationType = "messaging.operation.type"
	attributeDestination   = "messaging.destination.name"
	attributeMessageID     = "messaging.message.id"
	attributeOutcome       = "messaging.operation.outcome"

	systemNATS           = "nats"
	operationTypePublish = "publish"
	operationTypeProcess = "process"

	spanNamePublish = "messaging publish"
	spanNameConsume = "messaging consume"
)

type Role string

const (
	RoleProducer Role = "producer"
	RoleWorker   Role = "worker"
)

type Observability struct {
	Logger *slog.Logger
	Meter  metric.Meter
	Tracer trace.Tracer
}

type telemetry struct {
	log                   *slog.Logger
	tracer                trace.Tracer
	publishOperations     metric.Int64Counter
	publishDuration       metric.Float64Histogram
	connectionEvents      metric.Int64Counter
	readiness             metric.Int64ObservableGauge
	fetchMessages         metric.Int64Counter
	fetchBytes            metric.Int64Counter
	consumeActive         metric.Int64UpDownCounter
	handlerOperations     metric.Int64Counter
	handlerDuration       metric.Float64Histogram
	redeliveries          metric.Int64Counter
	retries               metric.Int64Counter
	dlqTransfers          metric.Int64Counter
	drainOperations       metric.Int64Counter
	forcedShutdowns       metric.Int64Counter
	readinessRegistration metric.Registration
	closeOnce             sync.Once
}

func newTelemetry(obs Observability, role Role, readiness func() bool) (*telemetry, error) {
	if role != RoleProducer && role != RoleWorker {
		return nil, fmt.Errorf("%w: invalid messaging role", ErrRejected)
	}
	if obs.Logger == nil {
		obs.Logger = slog.Default()
	}
	if obs.Meter == nil {
		obs.Meter = otel.GetMeterProvider().Meter(instrumentationScope)
	}
	if obs.Tracer == nil {
		obs.Tracer = otel.GetTracerProvider().Tracer(instrumentationScope)
	}
	s := &telemetry{log: obs.Logger, tracer: obs.Tracer}
	if err := s.registerMetrics(obs.Meter); err != nil {
		return nil, err
	}
	registration, err := obs.Meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		value := int64(0)
		if readiness() {
			value = 1
		}
		observer.ObserveInt64(s.readiness, value, metric.WithAttributes(attribute.String("role", string(role))))
		return nil
	}, s.readiness)
	if err != nil {
		return nil, fmt.Errorf("register messaging readiness metric: %w", err)
	}
	s.readinessRegistration = registration
	return s, nil
}

func (s *telemetry) registerMetrics(meter metric.Meter) error {
	var err error
	if s.publishOperations, err = meter.Int64Counter("messaging.publish.operations"); err != nil {
		return fmt.Errorf("create messaging publish operations metric: %w", err)
	}
	if s.publishDuration, err = meter.Float64Histogram("messaging.publish.duration", metric.WithUnit("s")); err != nil {
		return fmt.Errorf("create messaging publish duration metric: %w", err)
	}
	if s.connectionEvents, err = meter.Int64Counter("messaging.connection.events"); err != nil {
		return fmt.Errorf("create messaging connection events metric: %w", err)
	}
	if s.readiness, err = meter.Int64ObservableGauge("messaging.readiness"); err != nil {
		return fmt.Errorf("create messaging readiness metric: %w", err)
	}
	if s.fetchMessages, err = meter.Int64Counter("messaging.fetch.messages"); err != nil {
		return fmt.Errorf("create messaging fetch messages metric: %w", err)
	}
	if s.fetchBytes, err = meter.Int64Counter("messaging.fetch.bytes", metric.WithUnit("By")); err != nil {
		return fmt.Errorf("create messaging fetch bytes metric: %w", err)
	}
	if s.consumeActive, err = meter.Int64UpDownCounter("messaging.consume.active"); err != nil {
		return fmt.Errorf("create messaging active consumers metric: %w", err)
	}
	if s.handlerOperations, err = meter.Int64Counter("messaging.handler.operations"); err != nil {
		return fmt.Errorf("create messaging handler operations metric: %w", err)
	}
	if s.handlerDuration, err = meter.Float64Histogram("messaging.handler.duration", metric.WithUnit("s")); err != nil {
		return fmt.Errorf("create messaging handler duration metric: %w", err)
	}
	if s.redeliveries, err = meter.Int64Counter("messaging.redeliveries"); err != nil {
		return fmt.Errorf("create messaging redeliveries metric: %w", err)
	}
	if s.retries, err = meter.Int64Counter("messaging.retries"); err != nil {
		return fmt.Errorf("create messaging retries metric: %w", err)
	}
	if s.dlqTransfers, err = meter.Int64Counter("messaging.dlq.transfers"); err != nil {
		return fmt.Errorf("create messaging dead-letter transfers metric: %w", err)
	}
	if s.drainOperations, err = meter.Int64Counter("messaging.drain.operations"); err != nil {
		return fmt.Errorf("create messaging drain operations metric: %w", err)
	}
	if s.forcedShutdowns, err = meter.Int64Counter("messaging.forced_shutdowns"); err != nil {
		return fmt.Errorf("create messaging forced shutdowns metric: %w", err)
	}
	return nil
}

func (s *telemetry) close() {
	if s != nil && s.readinessRegistration != nil {
		s.closeOnce.Do(func() { _ = s.readinessRegistration.Unregister() })
	}
}

// outcomeAttribute is the only producer of the outcome attribute. Every
// counter that carries one goes through it, so no call site can skip the
// bounding by building its own metric.WithAttributes.
//
//nolint:ireturn // metric.WithAttributes returns OTel's own option interface.
func outcomeAttribute(outcome string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String("outcome", boundedOutcome(outcome)))
}

// publishSpanOptions and consumeSpanOptions own the attributes of the two spans
// this package opens. They return the options rather than the started span on
// purpose: tracer.Start stays at the call site, where spancheck can still prove
// the span is ended, while the attribute set — the part that has to agree
// between a publish and a consume to be comparable — is written once here.
//
// They are spelled out separately rather than sharing a constructor because the
// values that differ are adjacent strings, and a helper taking them as
// parameters would let a call site transpose the destination and the message id
// without the compiler noticing.
func publishSpanOptions(event Event) []trace.SpanStartOption {
	return []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String(attributeSystem, systemNATS),
			attribute.String(attributeOperationType, operationTypePublish),
			attribute.String(attributeDestination, event.Subject),
			attribute.String(attributeMessageID, event.MessageID),
		),
	}
}

func consumeSpanOptions(msg Message) []trace.SpanStartOption {
	return []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String(attributeSystem, systemNATS),
			attribute.String(attributeOperationType, operationTypeProcess),
			attribute.String(attributeDestination, msg.Subject()),
			attribute.String(attributeMessageID, msg.MessageID()),
		),
	}
}

// setSpanOutcome records how one operation ended. It bounds the value through
// the same vocabulary the outcome metric attribute uses, so a span and the
// counter that recorded the same operation cannot disagree about its name.
func setSpanOutcome(span trace.Span, outcome string) {
	span.SetAttributes(attribute.String(attributeOutcome, boundedOutcome(outcome)))
}

func (s *telemetry) recordPublish(ctx context.Context, event Event, outcome, reason string, started time.Time) {
	duration := time.Since(started).Seconds()
	attrs := outcomeAttribute(outcome)
	s.publishOperations.Add(ctx, 1, attrs)
	s.publishDuration.Record(ctx, duration, attrs)
	s.log.InfoContext(ctx, "messaging_publish",
		"operation", "publish", "message_id", event.MessageID, "subject", event.Subject,
		"outcome", outcome, "duration_seconds", duration, "reason", reason,
	)
}

func (s *telemetry) recordDeadLetterTransfer(ctx context.Context, outcome string) {
	s.dlqTransfers.Add(ctx, 1, outcomeAttribute(outcome))
}

func (s *telemetry) recordDrain(ctx context.Context, outcome string) {
	s.drainOperations.Add(ctx, 1, outcomeAttribute(outcome))
}

// countConnectionEvent is the only producer of the event attribute; the two
// recorders below differ in log level and fields, not in what they count.
func (s *telemetry) countConnectionEvent(ctx context.Context, event string) {
	s.connectionEvents.Add(ctx, 1, metric.WithAttributes(attribute.String("event", boundedConnectionEvent(event))))
}

func (s *telemetry) recordConnection(ctx context.Context, event string) {
	s.countConnectionEvent(ctx, event)
	s.log.InfoContext(ctx, "messaging_connection", "operation", "connection", "outcome", event)
}

func (s *telemetry) recordAsyncError(ctx context.Context, err error) {
	s.countConnectionEvent(ctx, connectionAsyncError)
	s.log.WarnContext(ctx, "messaging_connection",
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

func (s *telemetry) recordHandler(ctx context.Context, msg Message, outcome, reason string, started time.Time) {
	duration := time.Since(started).Seconds()
	attrs := outcomeAttribute(outcome)
	s.handlerOperations.Add(ctx, 1, attrs)
	s.handlerDuration.Record(ctx, duration, attrs)
	s.log.InfoContext(ctx, "messaging_delivery",
		"operation", "consume", "message_id", msg.MessageID(), "subject", msg.Subject(),
		"consumer", msg.Metadata().Consumer, "attempt", msg.Metadata().NumDelivered,
		"outcome", outcome, "duration_seconds", duration, "reason", reason,
	)
}

func (s *telemetry) logTerminalDelivery(ctx context.Context, subject string, metadata *jetstream.MsgMetadata, reason string, handlerFrames []string) {
	args := []any{"operation", "consume", "subject", subject, "outcome", outcomeTerminal, "reason", reason}
	if metadata != nil {
		args = append(args,
			"stream", metadata.Stream,
			"consumer", metadata.Consumer,
			"stream_sequence", metadata.Sequence.Stream,
			"attempt", metadata.NumDelivered,
		)
	}
	if len(handlerFrames) != 0 {
		args = append(args, "handler_frames", handlerFrames)
	}
	s.log.ErrorContext(ctx, "messaging_terminal_delivery", args...)
}
