package postgresoutbox

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// StartPublish opens the span covering one publication attempt and returns the
// context the adapter is called with, so an adapter's own spans nest under it.
//
// The span is a new root linked to the event's creation context rather than a
// child of it. A publication can happen long after its append, so parenting
// would hold the producing request's trace open past the assembly window of
// tracing backends. The link carries the same join without that lifetime
// coupling.
//
// messaging.system is deliberately absent: this package is broker-neutral and
// only the adapter knows the system. StartPublish and EndPublish stay paired
// around the adapter call so the caller's outcome reaches the same span.
//
//nolint:ireturn,spancheck // trace.Span is OTel's own interface, and EndPublish is this span's end.
func (t *Telemetry) StartPublish(ctx context.Context, event Event) (context.Context, trace.Span) {
	if t == nil || t.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	options := []trace.SpanStartOption{
		trace.WithNewRoot(),
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingOperationName(publishOperationName),
			semconv.MessagingDestinationName(event.Destination),
		),
	}
	// Extracted from a blank context on purpose. Extracting into ctx would leave
	// whatever span the relay is already inside in place when the carrier is
	// empty, and this span would then link to itself rather than to nothing.
	creation := trace.SpanContextFromContext(
		otel.GetTextMapPropagator().Extract(context.Background(), event.CreationContext()),
	)
	if creation.IsValid() {
		options = append(options, trace.WithLinks(trace.Link{SpanContext: creation}))
	}
	return t.tracer.Start(ctx, publishOperationName+" "+event.Destination, options...)
}

// EndPublish closes a publication span with the same bounded class the publish
// metric carries, so a trace and a dashboard name one condition. The broker's
// own error text never reaches it, for the reason LogListenerRetry states.
func (t *Telemetry) EndPublish(span trace.Span, err error, errorClass string) {
	if t == nil || span == nil {
		return
	}
	if err != nil {
		bounded := boundedErrorType(errorClass)
		span.SetAttributes(attribute.String("error.type", bounded))
		span.SetStatus(codes.Error, bounded)
	}
	span.End()
}
