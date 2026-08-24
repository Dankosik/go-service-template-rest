package outboxworker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/propagation"
)

type publishFunc func(context.Context, natsjs.Event) (natsjs.PublishResult, error)

// Worker maps one durable outbox job onto the NATS publication boundary.
type Worker struct {
	river.WorkerDefaults[postgresoutbox.PublishJob]

	publish publishFunc
}

// New builds the River worker for one admitted NATS producer.
func New(producer *natsjs.Producer) (*Worker, error) {
	if producer == nil {
		return nil, fmt.Errorf("%w: outbox producer is required", natsjs.ErrRejected)
	}
	return newWorker(producer.Publish), nil
}

func newWorker(publish publishFunc) *Worker {
	return &Worker{publish: publish}
}

func (w *Worker) Work(
	ctx context.Context,
	job *river.Job[postgresoutbox.PublishJob],
) error {
	if job == nil {
		return fmt.Errorf("%w: outbox job is required", natsjs.ErrRejected)
	}
	ctx = creationContext(ctx, job.Metadata)
	args := job.Args
	_, err := w.publish(ctx, natsjs.Event{
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

// creationContext restores the request context stored by otelriver. Malformed
// or absent telemetry metadata never blocks publication.
func creationContext(ctx context.Context, metadata []byte) context.Context {
	var carrier propagation.MapCarrier
	if len(metadata) == 0 || json.Unmarshal(metadata, &carrier) != nil {
		return ctx
	}
	return propagation.TraceContext{}.Extract(ctx, carrier)
}
