package natsjs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/example/go-service-template-rest/internal/domainevent"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/propagation"
)

// NewOutboxAppender builds the service-owned event routing table once. Domain
// code appends events without ever seeing the selected NATS subject.
func NewOutboxAppender(maxPayloadBytes int, routes ...Route) (*postgresoutbox.Appender, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("%w: at least one outbox route is required", ErrRejected)
	}
	subjects, err := buildRoutes(routes)
	if err != nil {
		return nil, err
	}
	appender, err := postgresoutbox.NewAppender(maxPayloadBytes, func(event domainevent.Event) (string, error) {
		subject, ok := subjects[routeKey{typeName: event.Type, version: event.Version}]
		if !ok {
			return "", fmt.Errorf("no outbox route for %s v%d", event.Type, event.Version)
		}
		return subject, nil
	})
	if err != nil {
		return nil, fmt.Errorf("initialize PostgreSQL outbox appender: %w", err)
	}
	return appender, nil
}

type OutboxWorker struct {
	river.WorkerDefaults[postgresoutbox.PublishJob]

	producer *Producer
}

func NewOutboxWorker(producer *Producer) (*OutboxWorker, error) {
	if producer == nil {
		return nil, fmt.Errorf("%w: outbox producer is required", ErrRejected)
	}
	return &OutboxWorker{producer: producer}, nil
}

func (w *OutboxWorker) Work(
	ctx context.Context,
	job *river.Job[postgresoutbox.PublishJob],
) error {
	if job == nil {
		return fmt.Errorf("%w: outbox job is required", ErrRejected)
	}
	ctx = outboxCreationContext(ctx, job.Metadata)
	args := job.Args
	_, err := w.producer.Publish(ctx, Event{
		Subject:       args.Subject,
		MessageID:     args.ID,
		PublicationID: args.ID,
		Type:          args.Type,
		Schema:        "v" + strconv.FormatUint(uint64(args.Version), 10),
		CreatedAt:     args.OccurredAt,
		Payload:       args.Payload,
	})
	if err != nil {
		return fmt.Errorf("publish domain event: %w", err)
	}
	return nil
}

// outboxCreationContext restores the request context stored by otelriver.
// Malformed or absent telemetry metadata never blocks publication.
func outboxCreationContext(ctx context.Context, metadata []byte) context.Context {
	var carrier propagation.MapCarrier
	if len(metadata) == 0 || json.Unmarshal(metadata, &carrier) != nil {
		return ctx
	}
	return propagation.TraceContext{}.Extract(ctx, carrier)
}
