package postgresoutbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// TelemetryScope names this package's instrumentation. Callers that build the
// meter themselves pass it to Meter so the scope stays one string; NewTelemetry
// uses it for the global-provider fallback.
const TelemetryScope = "service.outbox.postgres"

// This file owns the instruments, the snapshot the scrape callback reads, and
// the recorders that write it. The closed vocabularies every attribute here is
// bounded through are vocabulary.go.

// Telemetry records this package's metrics and operator logs. A nil *Telemetry
// is a working no-op: every method below returns on a nil receiver, because
// telemetry is optional at both NewStore and NewRelay. Call these methods
// directly rather than guarding at the call site — a second nil check reads as
// if one of the two were load-bearing, and neither is. A method added here
// carries the same guard.
type Telemetry struct {
	log                  *slog.Logger
	tracer               trace.Tracer
	messages             metric.Int64ObservableGauge
	oldestTimestamp      metric.Float64ObservableGauge
	observationTimestamp metric.Float64ObservableGauge
	lastProgress         metric.Float64ObservableGauge
	orderingHeads        metric.Int64ObservableGauge
	storageBytes         metric.Int64ObservableGauge
	inflight             metric.Int64ObservableGauge
	readiness            metric.Int64ObservableGauge
	operations           metric.Int64Counter
	duration             metric.Float64Histogram
	registration         metric.Registration
	closeOnce            sync.Once
	mu                   sync.RWMutex
	snapshot             telemetrySnapshot
}

type telemetrySnapshot struct {
	observation  StateObservation
	observedAt   time.Time
	lastProgress time.Time
	running      bool
	inflight     int64
	staleAfter   time.Duration
}

func NewTelemetry(meter metric.Meter, logger *slog.Logger) (*Telemetry, error) {
	if meter == nil {
		meter = otel.GetMeterProvider().Meter(TelemetryScope)
	}
	if logger == nil {
		logger = slog.Default()
	}
	// The tracer comes from the global provider rather than a parameter, because
	// the only process that publishes — cmd/outbox-relay — already installs one
	// and the global W3C propagator during bootstrap. A second constructor
	// argument would let a caller wire a meter and forget the tracer, and there
	// is no composition in this repository that wants them from different
	// providers.
	telemetry := &Telemetry{log: logger, tracer: otel.GetTracerProvider().Tracer(TelemetryScope)}
	set := infratelemetry.NewInstrumentSet(meter)
	set.Int64ObservableGauge(&telemetry.messages, "outbox.relay.messages")
	set.Float64ObservableGauge(&telemetry.oldestTimestamp, "outbox.relay.oldest.timestamp", metric.WithUnit("s"))
	set.Float64ObservableGauge(&telemetry.observationTimestamp, "outbox.relay.observation.timestamp", metric.WithUnit("s"))
	set.Float64ObservableGauge(&telemetry.lastProgress, "outbox.relay.last_progress.timestamp", metric.WithUnit("s"))
	set.Int64ObservableGauge(&telemetry.orderingHeads, "outbox.relay.ordering_heads")
	set.Int64ObservableGauge(&telemetry.storageBytes, "outbox.relay.storage.bytes", metric.WithUnit("By"))
	set.Int64ObservableGauge(&telemetry.inflight, "outbox.relay.inflight")
	set.Int64ObservableGauge(&telemetry.readiness, "outbox.relay.readiness")
	set.Int64Counter(&telemetry.operations, "outbox.relay.operations")
	set.Float64Histogram(&telemetry.duration, "outbox.relay.operation.duration", metric.WithUnit("s"))
	if err := set.Err(); err != nil {
		return nil, err
	}

	registration, err := meter.RegisterCallback(telemetry.collect,
		telemetry.messages,
		telemetry.oldestTimestamp,
		telemetry.observationTimestamp,
		telemetry.lastProgress,
		telemetry.orderingHeads,
		telemetry.storageBytes,
		telemetry.inflight,
		telemetry.readiness,
	)
	if err != nil {
		return nil, fmt.Errorf("register outbox metrics callback: %w", err)
	}
	telemetry.registration = registration
	return telemetry, nil
}

func (t *Telemetry) Close() {
	if t == nil || t.registration == nil {
		return
	}
	t.closeOnce.Do(func() { _ = t.registration.Unregister() })
}

func (t *Telemetry) RecordObservation(observation StateObservation, observedAt time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.observation = observation
	t.snapshot.observedAt = observedAt.UTC()
	t.mu.Unlock()
}

func (t *Telemetry) RecordProgress(at time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.lastProgress = at.UTC()
	t.mu.Unlock()
}

// SetProcessState publishes the relay's lifecycle state for the metric
// callback. running is the lifecycle half of readiness, not the whole verdict:
// the callback combines it with observation freshness on every scrape, because
// the relay only calls this at lifecycle edges and an idle relay can go stale
// between two of them.
func (t *Telemetry) SetProcessState(running bool, inflight int64, staleAfter time.Duration) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.snapshot.running = running
	t.snapshot.inflight = inflight
	t.snapshot.staleAfter = staleAfter
	t.mu.Unlock()
}

// RecordOperation counts one operation and records how long it took. Use it
// only where duration measures the labeled operation end to end; anything
// counted without a span of its own belongs in CountOperation.
func (t *Telemetry) RecordOperation(
	ctx context.Context,
	operation string,
	outcome string,
	errorType string,
	duration time.Duration,
) {
	if t == nil {
		return
	}
	attributes := operationAttributes(operation, outcome, errorType)
	t.operations.Add(ctx, 1, attributes)
	t.duration.Record(ctx, duration.Seconds(), attributes)
}

// CountOperation counts one operation that has no duration of its own: a state
// transition, or a verdict reached partway through a longer operation. Such a
// site has nothing honest to pass as a duration, and the placeholders it would
// otherwise pass — a zero span, or the enclosing operation's elapsed time —
// land in the same histogram as the real measurements and make it unreadable.
// They only ever reach the counter.
func (t *Telemetry) CountOperation(ctx context.Context, operation, outcome, errorType string) {
	if t == nil {
		return
	}
	t.operations.Add(ctx, 1, operationAttributes(operation, outcome, errorType))
}

//nolint:ireturn // metric.WithAttributes returns OTel's own option interface.
func operationAttributes(operation, outcome, errorType string) metric.MeasurementOption {
	return metric.WithAttributes(
		attribute.String("operation", boundedOperation(operation)),
		attribute.String("outcome", boundedOutcome(outcome)),
		attribute.String("error.type", boundedErrorType(errorType)),
	)
}

// StartPublish opens the span covering one publication attempt and returns the
// context the adapter is called with, so an adapter's own spans nest under it.
//
// The span is a new root linked to the event's creation context rather than a
// child of it, which is the one decision here worth stating. A publication can
// happen long after its append — after a backlog drains, or after an operator
// redrives days later — and parenting would hold the producing request's trace
// open for that whole horizon, past the assembly window of every backend that
// has one. The link carries the same join without that lifetime coupling, and it
// is what the OpenTelemetry messaging convention prescribes for a send that has
// a separate creation context.
//
// messaging.system is deliberately absent: this package is broker-neutral and
// only the adapter knows the system. The ordering key is absent for the reason
// it is absent everywhere else in this package.
// The span is returned rather than ended here: StartPublish and EndPublish are
// one pair around the adapter call, so that the publication's outcome — which
// only the caller learns — reaches the span it belongs to.
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

func (t *Telemetry) LogPoison(ctx context.Context, errorClass string, attempt int) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_event_poisoned", "error.type", errorClass, "attempt", attempt)
	}
}

func (t *Telemetry) LogPublisherStuck(ctx context.Context) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_publisher_stuck", "error.type", classStuck)
	}
}

// LogPublisherPanic reports an adapter panic with the description the recover
// consumed. It is the one place in this package that logs an unbounded string,
// and it is deliberate: the process is exiting over a deployment fault, and the
// class alone — publisher_panic on the exit line and the publish metric — names
// the category without naming the line of code. LogListenerRetry's rule still
// holds either way, because a panic value comes from the adapter rather than
// from a driver that formats DSN material into it.
//
// The event is not named. A panicking adapter is reproducible from its stack,
// while an event id on an ERROR line is the identity this package keeps off
// telemetry everywhere else.
func (t *Telemetry) LogPublisherPanic(ctx context.Context, value any, stack []byte) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_publisher_panic",
			"error.type", classPanic, "panic", fmt.Sprint(value), "stack", string(stack))
	}
}

func (t *Telemetry) LogRecovery(ctx context.Context, attempt int) {
	if t != nil {
		t.log.WarnContext(ctx, "outbox_lease_recovered", "attempt", attempt)
	}
}

// LogListenerRetry reports that wake-up notifications are unavailable. Pickup
// latency falls back to the poll interval until the listener reconnects.
//
// stage is a bounded class from listenerStage, never the driver's error text.
// A pgx connect failure formats the DSN's user, database, and host into its
// message, and this package promises never to log DSN material. Losing that
// detail costs little: the listener shares the pool's DSN, so a real
// connectivity fault also fails the pool, and that path exits the process with
// a postgres_unavailable class instead of degrading quietly.
func (t *Telemetry) LogListenerRetry(ctx context.Context, stage string) {
	if t != nil {
		t.log.WarnContext(ctx, "outbox_listener_retry", "error.type", classDatabase, "stage", stage)
	}
}

// collect answers one scrape from the published snapshot. It is the callback
// registered in NewTelemetry, and OTel's own term for that role — the two verbs
// nearby belong elsewhere: Store.Observe runs the observation statement and
// Relay.sampleState runs it and records the result.
func (t *Telemetry) collect(_ context.Context, observer metric.Observer) error {
	t.mu.RLock()
	snapshot := t.snapshot
	t.mu.RUnlock()

	states := []struct {
		name   string
		count  int64
		oldest time.Time
	}{
		{name: "eligible", count: snapshot.observation.EligibleCount, oldest: snapshot.observation.EligibleOldestAt},
		{name: "in_progress", count: snapshot.observation.InProgressCount, oldest: snapshot.observation.InProgressOldestAt},
		{name: "retry_wait", count: snapshot.observation.RetryWaitCount, oldest: snapshot.observation.RetryWaitOldestAt},
		{name: "recovery_due", count: snapshot.observation.RecoveryDueCount, oldest: snapshot.observation.RecoveryDueOldestAt},
		{name: "ordering_blocked", count: snapshot.observation.OrderingBlockedCount, oldest: snapshot.observation.OrderingBlockedOldestAt},
		{name: "poison", count: snapshot.observation.PoisonCount, oldest: snapshot.observation.PoisonOldestAt},
		{name: "outcome_unknown", count: snapshot.observation.OutcomeUnknownCount, oldest: snapshot.observation.OutcomeUnknownOldestAt},
		{name: "published_retained", count: snapshot.observation.PublishedRetainedEstimate, oldest: snapshot.observation.PublishedRetainedOldestAt},
	}
	for _, state := range states {
		attributes := metric.WithAttributes(attribute.String("state", state.name))
		observer.ObserveInt64(t.messages, state.count, attributes)
		observer.ObserveFloat64(t.oldestTimestamp, unixSecondsFromTime(state.oldest), attributes)
	}
	observer.ObserveFloat64(t.observationTimestamp, unixSecondsFromTime(snapshot.observedAt))
	observer.ObserveFloat64(t.lastProgress, unixSecondsFromTime(snapshot.lastProgress))
	observer.ObserveInt64(t.orderingHeads, snapshot.observation.OrderingHeadCount)

	for _, relation := range []struct {
		name         string
		total, index int64
	}{
		{name: "events", total: snapshot.observation.EventsBytes, index: snapshot.observation.EventsIndexBytes},
		{
			name:  "ordering_heads",
			total: snapshot.observation.OrderingHeadsBytes,
			index: snapshot.observation.OrderingHeadsIndexBytes,
		},
		{name: "redrives", total: snapshot.observation.RedrivesBytes, index: snapshot.observation.RedrivesIndexBytes},
		{name: "receipts", total: snapshot.observation.ReceiptsBytes, index: snapshot.observation.ReceiptsIndexBytes},
	} {
		name := attribute.String("relation", relation.name)
		observer.ObserveInt64(t.storageBytes, relation.total,
			metric.WithAttributes(name, attribute.String("kind", "total")))
		observer.ObserveInt64(t.storageBytes, relation.index,
			metric.WithAttributes(name, attribute.String("kind", "indexes")))
	}

	observer.ObserveInt64(t.inflight, snapshot.inflight)
	// The snapshot carries the relay's lifecycle state as of its last
	// SetProcessState, which only happens at lifecycle edges. This callback runs
	// on scrape, so it asks the shared readiness predicate again against the
	// current clock; without that, a relay that stopped observing between two
	// edges would keep reporting ready=1 until its next edge.
	ready := int64(0)
	if readyWithFreshObservation(snapshot.running, snapshot.observedAt, snapshot.staleAfter) {
		ready = 1
	}
	observer.ObserveInt64(t.readiness, ready)
	return nil
}

// unixSecondsFromTime is the inverse of timeFromUnixSeconds in store_rows.go,
// which reads the same values off the observation statement.
func unixSecondsFromTime(value time.Time) float64 {
	if value.IsZero() {
		return 0
	}
	return float64(value.UnixNano()) / float64(time.Second)
}
