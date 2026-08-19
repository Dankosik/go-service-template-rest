//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/domainevent"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestPostgresOutboxPublishesThroughRiverWithOriginalIdentityAndTrace(t *testing.T) {
	telemetrytest.InstallSpanRecorder(t)
	ctx, pool, appender := newOutboxFixture(t)
	fixture := newNATSFixture(t)

	type delivery struct {
		message natsjs.Message
		traceID trace.TraceID
	}
	received := make(chan delivery, 1)
	_, _, _ = fixture.worker(t, func(ctx context.Context, message natsjs.Message) error {
		received <- delivery{message: message, traceID: trace.SpanContextFromContext(ctx).TraceID()}
		return nil
	}, func(config *natsjs.WorkerConfig) {
		config.Consumer = "river-outbox-conformance"
	})

	producerClient := fixture.client(t, natsjs.RoleProducer)
	outboxWorker, err := natsjs.NewOutboxWorker(producerClient.Producer())
	if err != nil {
		t.Fatalf("natsjs.NewOutboxWorker(): %v", err)
	}
	workers := river.NewWorkers()
	if err := river.AddWorkerSafely(workers, outboxWorker); err != nil {
		t.Fatalf("river.AddWorkerSafely(): %v", err)
	}
	plugin := otelriver.NewMiddleware(&otelriver.MiddlewareConfig{
		EnableSemanticMetrics:  true,
		EnableTracePropagation: true,
	})
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		CancelledJobRetentionPeriod: -1,
		DiscardedJobRetentionPeriod: -1,
		Logger:                      slog.New(slog.DiscardHandler),
		PollOnly:                    true,
		Plugins:                     []rivertype.Plugin{plugin},
		Queues: map[string]river.QueueConfig{
			postgresoutbox.Queue: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("river.NewClient(): %v", err)
	}
	if err := riverClient.Start(ctx); err != nil {
		t.Fatalf("river.Client.Start(): %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := riverClient.Stop(stopCtx); err != nil {
			t.Errorf("river.Client.Stop(): %v", err)
		}
	})

	producing, span := otel.GetTracerProvider().Tracer("integration").Start(ctx, "change example")
	origin := trace.SpanContextFromContext(producing)
	event, err := domainevent.New(
		"event-river-nats",
		"example.changed",
		1,
		time.Now().UTC(),
		map[string]string{"id": "domain-1"},
	)
	if err != nil {
		t.Fatalf("domainevent.New(): %v", err)
	}
	if err := postgres.InTx(producing, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return appender.Append(producing, tx, event)
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	span.End()

	delivered := waittest.Receive(t, received, 10*time.Second, "River outbox delivery")
	message := delivered.message
	if message.MessageID() != event.ID ||
		message.PublicationID() != event.ID ||
		message.Subject() != sourceSubject ||
		message.Type() != event.Type ||
		message.Schema() != "v1" ||
		string(message.Payload()) != string(event.Payload) {
		t.Fatalf(
			"message = id %q publication %q subject %q type %q schema %q payload %q",
			message.MessageID(),
			message.PublicationID(),
			message.Subject(),
			message.Type(),
			message.Schema(),
			message.Payload(),
		)
	}
	if delivered.traceID != origin.TraceID() {
		t.Fatalf("consumer trace = %s, want producing trace %s", delivered.traceID, origin.TraceID())
	}
	waittest.Until(t, 10*time.Second, func() bool {
		var state string
		return pool.QueryRow(ctx, "SELECT state::text FROM river_job WHERE args->>'id' = $1", event.ID).Scan(&state) == nil &&
			state == "completed"
	}, "River job completion")
}
