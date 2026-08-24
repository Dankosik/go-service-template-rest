package outboxworker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel/trace"
)

func TestWorkerPublishesStableIdentityAndTrace(t *testing.T) {
	t.Parallel()

	const traceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	metadata, err := json.Marshal(map[string]string{"traceparent": traceparent})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	args := postgresoutbox.PublishJob{
		ID: "event-1", Type: "order.updated", Version: 1,
		OccurredAt: time.Unix(1, 0).UTC(), Payload: json.RawMessage(`{"order_id":"order-1"}`),
		Subject: "events.orders",
	}
	var published natsjs.Event
	var observedTrace trace.TraceID
	worker := newWorker(func(ctx context.Context, event natsjs.Event) (natsjs.PublishResult, error) {
		published = event
		observedTrace = trace.SpanContextFromContext(ctx).TraceID()
		return natsjs.PublishResult{}, nil
	})
	if err := worker.Work(t.Context(), &river.Job[postgresoutbox.PublishJob]{
		JobRow: &rivertype.JobRow{Metadata: metadata},
		Args:   args,
	}); err != nil {
		t.Fatalf("Work() error = %v", err)
	}

	wantTrace, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("parse expected trace ID: %v", err)
	}
	if published.Subject != args.Subject || published.MessageID != args.ID ||
		published.PublicationID != args.ID || published.Type != args.Type ||
		published.Schema != "v1" || !bytes.Equal(published.Payload, args.Payload) ||
		observedTrace != wantTrace {
		t.Fatalf("published event = %#v, trace = %s", published, observedTrace)
	}

	if _, err := New(nil); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("New(nil) error = %v, want ErrRejected", err)
	}
	if err := worker.Work(t.Context(), nil); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Work(nil) error = %v, want ErrRejected", err)
	}
}
