//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"strings"
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
	t.Run("append before retirement", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		const key = "append-first"
		mustAppendOutbox(t, ctx, pool, store, orderedEvent("append-first-1", key, 1))
		claim := mustClaimOutbox(t, ctx, store)
		if err := markOutboxPublished(ctx, store, claim); err != nil {
			t.Fatalf("drain ordering key: %v", err)
		}

		appendTx, err := pool.PGX().Begin(ctx)
		if err != nil {
			t.Fatalf("begin append transaction: %v", err)
		}
		defer func() { _ = appendTx.Rollback(context.WithoutCancel(ctx)) }()
		var appendPID int
		if err := appendTx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&appendPID); err != nil {
			t.Fatalf("read append backend PID: %v", err)
		}
		if err := store.Append(ctx, appendTx, orderedEvent("append-first-2", key, 2)); err != nil {
			t.Fatalf("append inside the racing transaction: %v", err)
		}

		retired := make(chan error, 1)
		go func() { retired <- retireOrderingKeys(ctx, pool, store, key) }()
		waitForOutbox(t,
			func() string { return "retirement did not block behind the append head lock" },
			func() bool { return outboxBlockedBy(t, ctx, pool, appendPID) },
		)

		if err := appendTx.Commit(ctx); err != nil {
			t.Fatalf("commit the racing append: %v", err)
		}
		if err := <-retired; !errors.Is(err, postgresoutbox.ErrOrderingKeyActive) {
			t.Fatalf("retirement after committed append = %v, want ErrOrderingKeyActive", err)
		}
		if heads := countOrderingHeads(t, ctx, pool); heads != 1 {
			t.Fatalf("ordering heads after append-first serialization = %d, want the mark intact", heads)
		}
	})

	t.Run("retirement before append", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		const key = "retire-first"
		mustAppendOutbox(t, ctx, pool, store, orderedEvent("retire-first-9", key, 9))
		claim := mustClaimOutbox(t, ctx, store)
		if err := markOutboxPublished(ctx, store, claim); err != nil {
			t.Fatalf("drain ordering key: %v", err)
		}

		retireTx, err := pool.PGX().Begin(ctx)
		if err != nil {
			t.Fatalf("begin retirement transaction: %v", err)
		}
		defer func() { _ = retireTx.Rollback(context.WithoutCancel(ctx)) }()
		var retirePID int
		if err := retireTx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&retirePID); err != nil {
			t.Fatalf("read retirement backend PID: %v", err)
		}
		if err := store.RetireOrderingKeys(ctx, retireTx, key); err != nil {
			t.Fatalf("retire ordering key: %v", err)
		}

		appended := make(chan error, 1)
		go func() {
			appended <- pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
				return store.Append(ctx, tx, orderedEvent("retire-first-1", key, 1))
			})
		}()
		waitForOutbox(t,
			func() string { return "append did not block behind the retirement head lock" },
			func() bool { return outboxBlockedBy(t, ctx, pool, retirePID) },
		)

		if err := retireTx.Commit(ctx); err != nil {
			t.Fatalf("commit retirement: %v", err)
		}
		if err := <-appended; err != nil {
			t.Fatalf("append after committed retirement: %v", err)
		}
		if heads := countOrderingHeads(t, ctx, pool); heads != 1 {
			t.Fatalf("ordering heads after retire-first serialization = %d, want one fresh mark", heads)
		}
	})

	t.Run("simultaneous retirements", func(t *testing.T) {
		ctx, pool, store := newOutboxFixture(t)
		const key = "double-retire"
		mustAppendOutbox(t, ctx, pool, store, orderedEvent("double-retire-1", key, 1))
		claim := mustClaimOutbox(t, ctx, store)
		if err := markOutboxPublished(ctx, store, claim); err != nil {
			t.Fatalf("drain ordering key: %v", err)
		}

		firstTx, err := pool.PGX().Begin(ctx)
		if err != nil {
			t.Fatalf("begin first retirement: %v", err)
		}
		defer func() { _ = firstTx.Rollback(context.WithoutCancel(ctx)) }()
		var firstPID int
		if err := firstTx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&firstPID); err != nil {
			t.Fatalf("read first retirement backend PID: %v", err)
		}
		if err := store.RetireOrderingKeys(ctx, firstTx, key); err != nil {
			t.Fatalf("first retirement: %v", err)
		}
		var headsInsideFirst int
		if err := firstTx.QueryRow(ctx, "SELECT count(*) FROM outbox_ordering_heads").Scan(&headsInsideFirst); err != nil {
			t.Fatalf("count heads inside first retirement: %v", err)
		}
		if headsInsideFirst != 0 {
			t.Fatalf("heads inside first retirement = %d, want its one deletion visible", headsInsideFirst)
		}

		second := make(chan error, 1)
		go func() { second <- retireOrderingKeys(ctx, pool, store, key) }()
		waitForOutbox(t,
			func() string { return "second retirement did not block behind the first" },
			func() bool { return outboxBlockedBy(t, ctx, pool, firstPID) },
		)
		if err := firstTx.Commit(ctx); err != nil {
			t.Fatalf("commit first retirement: %v", err)
		}
		if err := <-second; err != nil {
			t.Fatalf("second idempotent retirement: %v", err)
		}
		if heads := countOrderingHeads(t, ctx, pool); heads != 0 {
			t.Fatalf("ordering heads after simultaneous retirements = %d, want exactly one deletion", heads)
		}
	})
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

// The outbox-owned creation context has its own allowance: it neither mutates
// caller metadata nor consumes the caller's 288 KiB envelope budget.
func TestPostgresOutboxTraceContextAllowance(t *testing.T) {
	telemetrytest.InstallSpanRecorder(t)
	ctx, pool, store := newOutboxFixture(t)

	producing, span := otel.GetTracerProvider().Tracer("integration").Start(ctx, "POST /orders")
	origin := trace.SpanContextFromContext(producing)

	const (
		maxTextBytes     = 256
		maxPayloadBytes  = 256 << 10
		maxEnvelopeBytes = 288 << 10
	)
	event := postgresoutbox.Event{
		ID:               strings.Repeat("i", maxTextBytes),
		Type:             strings.Repeat("t", maxTextBytes),
		Source:           strings.Repeat("s", maxTextBytes),
		Destination:      strings.Repeat("d", maxTextBytes),
		Schema:           strings.Repeat("v", maxTextBytes),
		OccurredAt:       outboxEvent("allowance").OccurredAt,
		Payload:          []byte(`"` + strings.Repeat("p", maxPayloadBytes-2) + `"`),
		OrderingKey:      strings.Repeat("k", maxTextBytes),
		OrderingSequence: 1,
	}
	metadataPrefix := `{"traceparent":"caller-owned","padding":"`
	metadataSuffix := `"}`
	metadataBytes := maxEnvelopeBytes - 6*maxTextBytes - maxPayloadBytes
	event.Metadata = []byte(metadataPrefix + strings.Repeat("m", metadataBytes-len(metadataPrefix)-len(metadataSuffix)) + metadataSuffix)
	if got := len(event.ID) + len(event.Type) + len(event.Source) + len(event.Destination) +
		len(event.Schema) + len(event.OrderingKey) + len(event.Payload) + len(event.Metadata); got != maxEnvelopeBytes {
		t.Fatalf("test envelope = %d bytes, want %d", got, maxEnvelopeBytes)
	}

	if err := pool.InTx(producing, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(producing, tx, event)
	}); err != nil {
		t.Fatalf("append traced event at caller envelope limit: %v", err)
	}
	span.End()

	record, err := store.Get(ctx, event.ID)
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if !bytes.Equal(record.Event.Metadata, event.Metadata) {
		t.Fatal("stored creation context mutated caller metadata")
	}
	carried := trace.SpanContextFromContext(
		otel.GetTextMapPropagator().Extract(context.Background(), record.Event.CreationContext()),
	)
	if carried.TraceID() != origin.TraceID() || carried.SpanID() != origin.SpanID() {
		t.Fatalf("stored creation context = %s/%s, want %s/%s", carried.TraceID(), carried.SpanID(), origin.TraceID(), origin.SpanID())
	}
}

// Retry, lease recovery, reconstruction, and redrive all reuse the immutable
// creation carrier captured by the original append.
func TestPostgresOutboxCreationContextSurvivesRecovery(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)
	ctx, pool, store := newOutboxFixture(t)

	producing, producingSpan := otel.GetTracerProvider().Tracer("integration").Start(ctx, "POST /orders")
	origin := trace.SpanContextFromContext(producing)
	if err := pool.InTx(producing, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return store.Append(producing, tx, outboxEvent("recover-trace"))
	}); err != nil {
		t.Fatalf("append inside the producing trace: %v", err)
	}
	producingSpan.End()

	first := mustClaimOutbox(t, ctx, store)
	wantCarrier := maps.Clone(first.Event.CreationContext())
	assertCarrier := func(stage string, event postgresoutbox.Event) {
		t.Helper()
		if event.ID != first.Event.ID {
			t.Fatalf("%s event = %q, want %q", stage, event.ID, first.Event.ID)
		}
		if !maps.Equal(event.CreationContext(), wantCarrier) {
			t.Fatalf("%s creation carrier = %v, want %v", stage, event.CreationContext(), wantCarrier)
		}
		carried := trace.SpanContextFromContext(
			otel.GetTextMapPropagator().Extract(context.Background(), event.CreationContext()),
		)
		if carried.TraceID() != origin.TraceID() || carried.SpanID() != origin.SpanID() {
			t.Fatalf("%s creation context = %s/%s, want %s/%s", stage, carried.TraceID(), carried.SpanID(), origin.TraceID(), origin.SpanID())
		}
	}
	assertCarrier("first claim", first.Event)
	if err := scheduleOutboxRetry(ctx, store, first.Event.ID, first.Token, "publisher_temporary", 0); err != nil {
		t.Fatalf("ScheduleRetryBatch(): %v", err)
	}

	abandoned, err := claimOutboxEvent(ctx, store, shortOutboxLease)
	if err != nil {
		t.Fatalf("retry Claim(): %v", err)
	}
	assertCarrier("retry claim", abandoned.Event)
	expireOutboxLease(t, ctx, pool)

	restarted, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("NewStore() after abandoned claim: %v", err)
	}
	published := make(chan postgresoutbox.Event, 1)
	release := make(chan struct{})
	publisher := testPublisherFunc(func(_ context.Context, event postgresoutbox.Event) error {
		published <- event
		<-release
		return postgresoutbox.ErrPermanentPublication
	})
	_, telemetry := newOutboxTelemetry(t)
	relay := mustNewOutboxRelay(t, restarted, publisher, telemetry, testRelayConfig())
	recoveryCtx, recoverySpan := otel.GetTracerProvider().Tracer("integration").Start(ctx, "recovery process")
	recoveryTrace := trace.SpanContextFromContext(recoveryCtx).TraceID()
	result := runOutboxRelay(recoveryCtx, relay)
	assertCarrier("recovered publication", <-published)
	relay.StartDrain()
	close(release)
	assertRelayResult(t, result, nil)
	recoverySpan.End()
	waitForOutboxCount(t, ctx, pool, "poisoned_at IS NOT NULL", 1)

	if err := restarted.RedriveUnknown(ctx, first.Event.ID, "trace-recovery-audit"); err != nil {
		t.Fatalf("RedriveUnknown(): %v", err)
	}
	restarted, err = postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("NewStore() after redrive: %v", err)
	}
	published = make(chan postgresoutbox.Event, 1)
	publisher = testPublisherFunc(func(_ context.Context, event postgresoutbox.Event) error {
		published <- event
		return nil
	})
	_, telemetry = newOutboxTelemetry(t)
	relay = mustNewOutboxRelay(t, restarted, publisher, telemetry, testRelayConfig())
	redriveCtx, redriveSpan := otel.GetTracerProvider().Tracer("integration").Start(ctx, "redrive process")
	redriveTrace := trace.SpanContextFromContext(redriveCtx).TraceID()
	result = runOutboxRelay(redriveCtx, relay)
	assertCarrier("redriven publication", <-published)
	waitForOutboxCount(t, ctx, pool, "published_at IS NOT NULL", 1)
	relay.StartDrain()
	assertRelayResult(t, result, nil)
	redriveSpan.End()

	var publishSpans []sdktrace.ReadOnlySpan
	for _, span := range recorder.Ended() {
		if span.Name() == "publish events" {
			publishSpans = append(publishSpans, span)
		}
	}
	if len(publishSpans) != 2 {
		t.Fatalf("publish spans = %d, want one recovered and one redriven attempt", len(publishSpans))
	}
	for index, span := range publishSpans {
		if len(span.Links()) != 1 || span.Links()[0].SpanContext.TraceID() != origin.TraceID() {
			t.Fatalf("publish span %d links = %v, want only the producing trace %s", index, span.Links(), origin.TraceID())
		}
		if got := span.SpanContext().TraceID(); got == origin.TraceID() || got == recoveryTrace || got == redriveTrace {
			t.Fatalf("publish span %d trace = %s, want a new linked root", index, got)
		}
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
