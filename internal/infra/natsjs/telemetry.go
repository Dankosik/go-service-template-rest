package natsjs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	// infratelemetry is aliased because this package declares its own telemetry
	// type, which an unaliased import would collide with.
	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationScope = "service.messaging.nats"

// Span attributes come from semconv rather than from string literals of this
// package's own, for the reason the literals they replaced did not survive: the
// convention renamed the producer's messaging.operation.type from "publish" to
// "send", and a hardcoded key or value cannot be carried across that by the
// version bump that performs it. postgresoutbox pins the same semconv version,
// so the two packs describe one publication in one vocabulary.
//
// The metric and log label vocabulary is vocabulary.go; the dead-letter reasons
// that travel on the wire are message_deadletter.go.
const (
	// attributeOutcome has no semconv equivalent at the pinned version. It stays
	// a literal because it is this repository's own attribute, bounded by
	// boundedOutcome, and naming it beside the semconv calls is what keeps that
	// distinction visible.
	attributeOutcome = "messaging.operation.outcome"

	// systemNATS is a literal because messaging.system is an open enum and the
	// pinned semconv ships no NATS member; "nats" is the value NATS
	// instrumentations already publish.
	systemNATS = "nats"

	// The messaging.operation.name values. They are the broker's own verbs
	// rather than the normalized messaging.operation.type enum, and each is also
	// the first word of its span name — semconv names a messaging span
	// "{operation name} {destination}".
	operationNamePublish = "publish"
	operationNameProcess = "process"
)

// publishSpanName and consumeSpanName build the two span names.
//
// The publish name carries the concrete subject, which is the publisher's own
// choice and matches what postgresoutbox names a publication. The consume name
// carries the worker's configured filter rather than the delivered subject:
// semconv asks for the destination template when a subscription has one, and a
// filter with a wildcard is exactly the case where the delivered subject would
// put an unbounded value in the one field a tracing backend groups on.
func publishSpanName(subject string) string { return operationNamePublish + " " + subject }
func consumeSpanName(filter string) string  { return operationNameProcess + " " + filter }

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
	obs.Meter = infratelemetry.MeterOrGlobal(obs.Meter, instrumentationScope)
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
	set := infratelemetry.NewInstrumentSet(meter)
	set.Int64Counter(&s.publishOperations, "messaging.publish.operations")
	set.Float64Histogram(&s.publishDuration, "messaging.publish.duration", metric.WithUnit("s"))
	set.Int64Counter(&s.connectionEvents, "messaging.connection.events")
	set.Int64ObservableGauge(&s.readiness, "messaging.readiness")
	set.Int64Counter(&s.fetchMessages, "messaging.fetch.messages")
	set.Int64Counter(&s.fetchBytes, "messaging.fetch.bytes", metric.WithUnit("By"))
	set.Int64UpDownCounter(&s.consumeActive, "messaging.consume.active")
	set.Int64Counter(&s.handlerOperations, "messaging.handler.operations")
	set.Float64Histogram(&s.handlerDuration, "messaging.handler.duration", metric.WithUnit("s"))
	set.Int64Counter(&s.redeliveries, "messaging.redeliveries")
	set.Int64Counter(&s.retries, "messaging.retries")
	set.Int64Counter(&s.dlqTransfers, "messaging.dlq.transfers")
	set.Int64Counter(&s.drainOperations, "messaging.drain.operations")
	set.Int64Counter(&s.forcedShutdowns, "messaging.forced_shutdowns")
	return set.Err()
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

// publishSpanOptions and consumeSpanOptions return options rather than a started
// span so tracer.Start stays at the call site, where spancheck can still prove
// the span is ended.
//
// They are spelled out separately rather than sharing a constructor because the
// values that differ are adjacent strings, and a helper taking them as parameters
// would let a call site transpose the destination and the message id without the
// compiler noticing.
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

// consumeSpanOptions takes the worker's configured filter beside the message
// because the two answer different questions: the delivered subject is what
// arrived, and the filter is what this consumer subscribes to. Both are
// recorded, and only the filter reaches the span name.
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
		"operation", "publish", "subject", event.Subject,
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
		"operation", "consume", "subject", msg.Subject(), "attempt", msg.Metadata().NumDelivered,
		"outcome", outcome, "duration_seconds", duration, "reason", reason,
	)
}

func (s *telemetry) logTerminalDelivery(ctx context.Context, subject string, metadata *jetstream.MsgMetadata, reason string, panicked *handlerPanic) {
	args := []any{"operation", "consume", "subject", subject, "outcome", outcomeTerminal, "reason", reason}
	if metadata != nil {
		args = append(args, "attempt", metadata.NumDelivered)
	}
	if panicked != nil {
		args = append(args, "panic.class", panicked.class, "handler_frames", panicked.frames)
	}
	s.log.ErrorContext(ctx, "messaging_terminal_delivery", args...)
}

// The counters below carry no log line of their own: each is one number on a
// path recordHandler, recordDrain, or logTerminalDelivery already narrates. They
// are methods anyway, so an attribute added to any of them is added here rather
// than at whichever delivery site noticed first.
func (s *telemetry) countFetch(ctx context.Context, messages int64, messageBytes int64) {
	s.fetchMessages.Add(ctx, messages)
	s.fetchBytes.Add(ctx, messageBytes)
}

func (s *telemetry) countConsumeActive(ctx context.Context, delta int64) {
	s.consumeActive.Add(ctx, delta)
}

func (s *telemetry) countRedelivery(ctx context.Context) {
	s.redeliveries.Add(ctx, 1)
}

func (s *telemetry) countRetry(ctx context.Context) {
	s.retries.Add(ctx, 1)
}

func (s *telemetry) countForcedShutdown(ctx context.Context) {
	s.forcedShutdowns.Add(ctx, 1)
}
