package natsjs

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type outboxPublisher struct {
	producer *Producer
}

// NewOutboxPublisher adapts this package's durable producer to the PostgreSQL
// outbox. A nil producer deliberately returns a nil interface so relay startup
// keeps failing closed when no usable transport was constructed.
func NewOutboxPublisher(producer *Producer) postgresoutbox.Publisher { //nolint:ireturn // This adapter intentionally returns the outbox's sole publication interface.
	if producer == nil {
		return nil
	}
	return outboxPublisher{producer: producer}
}

func (p outboxPublisher) Publish(ctx context.Context, event postgresoutbox.Event) error {
	return p.publish(ctx, Event{
		Subject:       event.Destination,
		MessageID:     event.ID,
		PublicationID: event.ID,
		Type:          event.Type,
		Schema:        event.Schema,
		OrderingKey:   event.OrderingKey,
		CreatedAt:     event.OccurredAt,
		Payload:       event.Payload,
	}, event.CreationContext())
}

func (p outboxPublisher) publish(ctx context.Context, mapped Event, creationContext propagation.MapCarrier) error {
	if err := validateEvent(mapped, p.producer.maxPayloadBytes); err != nil {
		return fmt.Errorf("%w: invalid NATS outbox envelope", postgresoutbox.ErrPermanentPublication)
	}

	// Keep cancellation and the batch deadline, but remove the relay-attempt
	// span. The producer span must descend from the stored creation context.
	ctx = trace.ContextWithSpanContext(ctx, trace.SpanContext{})
	ctx = propagation.TraceContext{}.Extract(ctx, creationContext)
	_, err := p.producer.Publish(ctx, mapped)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrRejected):
		return fmt.Errorf("%w: NATS publish was not accepted", postgresoutbox.ErrPublicationNotAccepted)
	default:
		return err
	}
}
