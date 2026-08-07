//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// The whole production path for R2, in one exercise: an event appended inside a
// request's trace, stored through the real column and its CHECK, claimed by a
// real relay, and published through a real adapter — after which the exported
// publish span names the operation that produced it.
//
// Split proof would not close this. A unit test can show the span links to a
// carrier, and another can show the carrier survives PostgreSQL, but only this
// shows the relay hands the claimed event's own carrier to the span it opens.
func TestPostgresOutboxPublicationNamesTheProducingOperation(t *testing.T) {
	// Installed before the telemetry is built: NewTelemetry takes its tracer from
	// the global provider, so a recorder installed afterwards would see nothing.
	recorder := telemetrytest.InstallSpanRecorder(t)
	ctx, pool, store := newOutboxFixture(t)

	producing, span := otel.GetTracerProvider().Tracer("integration").Start(ctx, "POST /orders")
	origin := trace.SpanContextFromContext(producing)
	// Appended with the request's context, which is the only way the creation
	// context is captured — a caller cannot set one.
	if err := pool.InTx(producing, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(producing, tx, outboxEvent("traced-event"))
	}); err != nil {
		t.Fatalf("append inside the producing trace: %v", err)
	}
	span.End()

	published := make(chan postgresoutbox.Event, 1)
	publisher := testPublisherFunc(func(_ context.Context, event postgresoutbox.Event) error {
		published <- event
		return nil
	})
	_, telemetry := newOutboxTelemetry(t)
	relay := mustNewOutboxRelay(t, store, publisher, telemetry, testRelayConfig())
	result := runOutboxRelay(ctx, relay)
	delivered := <-published
	waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", 1)
	relay.StartDrain()
	assertRelayResult(t, result, nil)

	// The adapter can put the origin on its broker's headers, which is what lets
	// a consumer join the same trace.
	carried := trace.SpanContextFromContext(
		otel.GetTextMapPropagator().Extract(context.Background(), delivered.CreationContext()),
	)
	if carried.TraceID() != origin.TraceID() {
		t.Errorf("adapter received trace %s, want the producing trace %s", carried.TraceID(), origin.TraceID())
	}

	publish := findSpan(t, recorder, "publish events")
	if len(publish.Links()) != 1 {
		t.Fatalf("publish span carries %d links, want one to the producing operation", len(publish.Links()))
	}
	if got := publish.Links()[0].SpanContext.TraceID(); got != origin.TraceID() {
		t.Errorf("publish span links to trace %s, want %s", got, origin.TraceID())
	}
	// Linked rather than parented: a publication can happen long after its
	// append, and a child would hold the producing request's trace open until it
	// does.
	if publish.SpanContext().TraceID() == origin.TraceID() {
		t.Error("publish span joined the producing trace directly; it must be a linked root")
	}
}

// R4's state table against the authority that owns it. The precondition, the
// idempotent repeat, and the restarted sequence space are all PostgreSQL's
// behavior, and none of them can be proven against a stubbed driver.
func TestPostgresOutboxRetireOrderingKey(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	const key = "aggregate-1"
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("ordered-1", key, 9))

	// A key with unpublished events keeps its mark: retiring it now would drop
	// the protection those events still rely on.
	err := retireOrderingKeys(ctx, pool, store, key)
	if !errors.Is(err, postgresoutbox.ErrOrderingKeyActive) {
		t.Fatalf("retire with a pending event = %v, want ErrOrderingKeyActive", err)
	}
	if heads := countOrderingHeads(t, ctx, pool); heads != 1 {
		t.Fatalf("ordering heads after a refused retirement = %d, want the mark intact", heads)
	}

	// Drain the key, then retire it.
	claim := mustClaimOutbox(t, ctx, store)
	if err := store.MarkOrderedPublished(ctx, claim.Token, postgresoutbox.OrderedDirective{
		ID: "ordered-1", OrderingKey: key, OrderingSequence: 9,
	}); err != nil {
		t.Fatalf("MarkOrderedPublished(): %v", err)
	}
	if err := retireOrderingKeys(ctx, pool, store, key); err != nil {
		t.Fatalf("retire a drained key: %v", err)
	}
	if heads := countOrderingHeads(t, ctx, pool); heads != 0 {
		t.Fatalf("ordering heads after retirement = %d, want 0", heads)
	}

	// Repeating it, and retiring a key that never existed, both change nothing.
	if err := retireOrderingKeys(ctx, pool, store, key, "never-appended"); err != nil {
		t.Fatalf("repeated retirement: %v", err)
	}

	// The sequence space restarts, which is exactly the protection the caller
	// traded away by asserting the key was terminal.
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("ordered-2", key, 1))
	if heads := countOrderingHeads(t, ctx, pool); heads != 1 {
		t.Fatalf("ordering heads after a post-retirement append = %d, want 1", heads)
	}
}

// A retirement racing an append for the same key cannot interleave with the
// precondition: both take the same head lock, so the append is either visible
// as pending work that refuses the retirement, or lands after it.
func TestPostgresOutboxRetireSerializesWithAppend(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	const key = "aggregate-2"
	mustAppendOutbox(t, ctx, pool, store, orderedEvent("racer-1", key, 1))

	// An uncommitted append holds the head lock. The retirement below must wait
	// for it rather than reading around it.
	appendTx, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatalf("begin append transaction: %v", err)
	}
	defer func() { _ = appendTx.Rollback(ctx) }()
	if err := store.Append(ctx, appendTx, orderedEvent("racer-2", key, 2)); err != nil {
		t.Fatalf("append inside the racing transaction: %v", err)
	}

	retired := make(chan error, 1)
	go func() { retired <- retireOrderingKeys(ctx, pool, store, key) }()

	if err := appendTx.Commit(ctx); err != nil {
		t.Fatalf("commit the racing append: %v", err)
	}
	if err := <-retired; !errors.Is(err, postgresoutbox.ErrOrderingKeyActive) {
		t.Fatalf("retirement racing an append = %v, want ErrOrderingKeyActive", err)
	}
	if heads := countOrderingHeads(t, ctx, pool); heads != 1 {
		t.Fatalf("ordering heads after the race = %d, want the mark intact", heads)
	}
}

// An event appended outside any trace still stores and publishes. The stored
// context is the column default rather than absent, so the CHECK holds.
func TestPostgresOutboxAppendWithoutTraceContext(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("untraced"))

	record, err := store.Get(ctx, "untraced")
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if carrier := record.Event.CreationContext(); len(carrier) != 0 {
		t.Errorf("untraced event carries %v, want no creation context", carrier)
	}
}

func retireOrderingKeys(
	ctx context.Context,
	pool *postgres.Pool,
	store *postgresoutbox.Store,
	keys ...string,
) error {
	return pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.RetireOrderingKeys(ctx, tx, keys...)
	})
}

//nolint:ireturn // The SDK's recorder reports spans as its own interface.
func findSpan(t *testing.T, recorder *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, span := range recorder.Ended() {
		if span.Name() == name {
			return span
		}
	}
	t.Fatalf("no span named %q was recorded", name)
	return nil
}

func countOrderingHeads(t *testing.T, ctx context.Context, pool *postgres.Pool) int {
	t.Helper()
	var count int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM outbox_ordering_heads").Scan(&count); err != nil {
		t.Fatalf("count ordering heads: %v", err)
	}
	return count
}
