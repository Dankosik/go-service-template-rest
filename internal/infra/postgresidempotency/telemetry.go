package postgresidempotency

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const idempotencyMeterName = "service.http.idempotency"

type storeTelemetry struct {
	log                  *slog.Logger
	transitions          metric.Int64Counter
	terminals            metric.Int64Counter
	stageDuration        metric.Float64Histogram
	rows                 metric.Int64ObservableGauge
	relationBytes        metric.Int64ObservableGauge
	resultBytes          metric.Int64ObservableGauge
	oldestExpiry         metric.Int64ObservableGauge
	observationTimestamp metric.Int64ObservableGauge
	headroomBytes        metric.Int64ObservableGauge
	registration         metric.Registration
}

func newStoreTelemetry(store *Store) (*storeTelemetry, error) {
	meter := otel.GetMeterProvider().Meter(idempotencyMeterName)
	t := &storeTelemetry{log: slog.Default()}
	set := infratelemetry.NewInstrumentSet(meter)
	set.Int64Counter(&t.transitions, "http.idempotency.transitions")
	set.Int64Counter(&t.terminals, "http.idempotency.requests")
	set.Float64Histogram(&t.stageDuration, "http.idempotency.stage.duration", metric.WithUnit("s"))
	set.Int64ObservableGauge(&t.rows, "http.idempotency.rows", metric.WithUnit("{row}"))
	set.Int64ObservableGauge(&t.relationBytes, "http.idempotency.relation.bytes", metric.WithUnit("By"))
	set.Int64ObservableGauge(&t.resultBytes, "http.idempotency.result.bytes", metric.WithUnit("By"))
	set.Int64ObservableGauge(&t.oldestExpiry, "http.idempotency.oldest_expiry.timestamp", metric.WithUnit("s"))
	set.Int64ObservableGauge(&t.observationTimestamp, "http.idempotency.observation.timestamp", metric.WithUnit("s"))
	set.Int64ObservableGauge(&t.headroomBytes, "http.idempotency.admission.headroom", metric.WithUnit("By"))
	if err := set.Err(); err != nil {
		return nil, fmt.Errorf("create HTTP idempotency telemetry: %w", err)
	}
	registration, err := meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		snapshot := store.safety.Load()
		if snapshot == nil || snapshot.observedAt.IsZero() {
			return nil
		}
		observer.ObserveInt64(t.rows, snapshot.rows)
		observer.ObserveInt64(t.relationBytes, snapshot.relationBytes)
		observer.ObserveInt64(t.resultBytes, snapshot.resultBytes)
		observer.ObserveInt64(t.oldestExpiry, snapshot.oldestExpiryUnix)
		observer.ObserveInt64(t.observationTimestamp, snapshot.observedAt.UTC().Unix())
		observer.ObserveInt64(t.headroomBytes, max(
			0,
			store.options.MaxRelationBytes-store.options.AdmissionHeadroomBytes-snapshot.relationBytes,
		))
		return nil
	}, t.rows, t.relationBytes, t.resultBytes, t.oldestExpiry, t.observationTimestamp, t.headroomBytes)
	if err != nil {
		return nil, fmt.Errorf("register HTTP idempotency telemetry: %w", err)
	}
	t.registration = registration
	return t, nil
}

func (t *storeTelemetry) recordTransition(ctx context.Context, value string) {
	if t != nil {
		t.transitions.Add(ctx, 1, metric.WithAttributes(attribute.String("event", boundedTransition(value))))
	}
}

func (t *storeTelemetry) recordTerminal(ctx context.Context, value string) {
	if t != nil {
		t.terminals.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", boundedTerminal(value))))
	}
}

func terminalOutcome(decision httpidempotency.Decision, err error) string {
	if err != nil {
		return terminalFailed
	}
	switch decision.Outcome {
	case httpidempotency.OutcomeExecute:
		return terminalExecuted
	case httpidempotency.OutcomeReplay:
		return terminalReplayed
	case httpidempotency.OutcomeMismatch:
		return terminalMismatch
	case httpidempotency.OutcomeInProgress:
		return terminalInProgress
	case httpidempotency.OutcomeUnknown:
		return terminalCommitUnknown
	case httpidempotency.OutcomeExpired,
		httpidempotency.OutcomeRateLimited,
		httpidempotency.OutcomeUnavailable,
		httpidempotency.OutcomeResultTooLarge,
		httpidempotency.OutcomeIntegrityConflict:
		return terminalFailed
	default:
		return terminalFailed
	}
}

func (t *storeTelemetry) recordStage(ctx context.Context, stage string, started time.Time) {
	if t != nil {
		t.stageDuration.Record(ctx, max(0, time.Since(started).Seconds()),
			metric.WithAttributes(attribute.String("stage", boundedStage(stage))))
	}
}

func (t *storeTelemetry) recordMaintenance(ctx context.Context, event string, err error) {
	if t == nil {
		return
	}
	outcome := "failed"
	if event == transitionCleanupRecovered {
		outcome = "recovered"
	}
	if event != "" {
		t.recordTransition(ctx, event)
	}
	t.log.LogAttrs(ctx, slog.LevelWarn, "http_idempotency_maintenance",
		slog.String("component", "http_idempotency"),
		slog.String("operation", "maintenance"),
		slog.String("outcome", outcome),
		slog.String("reason", boundedErrorClass(err)),
	)
}
