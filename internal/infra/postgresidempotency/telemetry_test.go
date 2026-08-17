package postgresidempotency

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestHTTPIdempotencyTelemetryAndVocabulary(t *testing.T) {
	t.Parallel()
	reader := telemetrytest.InstallManualReader(t)
	store := maintenanceUnitStore()
	store.safety.Store(&maintenanceSnapshot{
		observedAt:       time.Unix(100, 0),
		writer:           true,
		rows:             2,
		relationBytes:    300,
		resultBytes:      40,
		oldestExpiryUnix: 90,
	})
	telemetry, err := newStoreTelemetry(store)
	if err != nil {
		t.Fatalf("newStoreTelemetry() error = %v", err)
	}
	t.Cleanup(func() { _ = telemetry.registration.Unregister() })
	var logs bytes.Buffer
	telemetry.log = slog.New(slog.NewJSONHandler(&logs, nil))

	ctx := context.Background()
	telemetry.recordTransition(ctx, transitionFirstExecution)
	telemetry.recordTransition(ctx, "sentinel-transition")
	telemetry.recordTerminal(ctx, terminalExecuted)
	telemetry.recordTerminal(ctx, "sentinel-terminal")
	telemetry.recordStage(ctx, stageLookup, time.Now().Add(-time.Millisecond))
	telemetry.recordStage(ctx, "sentinel-stage", time.Now())
	telemetry.recordMaintenance(ctx, transitionCleanupFailed, ErrUnavailable)
	telemetry.recordMaintenance(ctx, "", errors.New("sentinel-error"))
	telemetry.recordMaintenance(ctx, transitionCleanupRecovered, nil)
	store.telemetry = telemetry
	store.recordReserve(ctx, httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil, time.Now())
	store.ObserveTerminal(ctx, httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil)

	required := map[string]bool{
		"http.idempotency.transitions":             false,
		"http.idempotency.requests":                false,
		"http.idempotency.stage.duration":          false,
		"http.idempotency.rows":                    false,
		"http.idempotency.relation.bytes":          false,
		"http.idempotency.result.bytes":            false,
		"http.idempotency.oldest_expiry.timestamp": false,
		"http.idempotency.observation.timestamp":   false,
		"http.idempotency.admission.headroom":      false,
	}
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if _, ok := required[measured.Name]; !ok {
			return
		}
		required[measured.Name] = true
		for _, set := range telemetrytest.AttributeSets(t, measured.Data) {
			assertIdempotencyMetricAttributes(t, measured.Name, set)
		}
	})
	for name, found := range required {
		if !found {
			t.Errorf("metric %s was not collected", name)
		}
	}
	if strings.Contains(logs.String(), "sentinel-error") {
		t.Fatalf("maintenance log leaked error text: %s", logs.String())
	}
	telemetrytest.AssertNoAttributeContains(t, reader, "sentinel")
}

func TestTerminalObservationFollowsReservation(t *testing.T) {
	t.Parallel()
	reader := telemetrytest.InstallManualReader(t)
	store := maintenanceUnitStore()
	telemetry, err := newStoreTelemetry(store)
	if err != nil {
		t.Fatalf("newStoreTelemetry() error = %v", err)
	}
	t.Cleanup(func() { _ = telemetry.registration.Unregister() })
	store.telemetry = telemetry

	store.recordReserve(t.Context(), httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil, time.Now())
	if got := terminalMetricCount(t, reader); got != 0 {
		t.Fatalf("reservation terminal count = %d, want 0", got)
	}

	store.ObserveTerminal(t.Context(), httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil)
	if got := terminalMetricCount(t, reader); got != 1 {
		t.Fatalf("post-transaction terminal count = %d, want 1", got)
	}
}

func TestTerminalOutcome(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name     string
		decision httpidempotency.Decision
		err      error
		want     string
	}{
		{name: "reconciliation unknown", decision: httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnknown}, want: terminalCommitUnknown},
		{name: "render loss", err: errors.New("render failed"), want: terminalFailed},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := terminalOutcome(testCase.decision, testCase.err); got != testCase.want {
				t.Fatalf("terminal outcome = %q, want %q", got, testCase.want)
			}
		})
	}
}

func terminalMetricCount(t *testing.T, reader *sdkmetric.ManualReader) int64 {
	t.Helper()
	var count int64
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != "http.idempotency.requests" {
			return
		}
		for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
			count += point.Value
		}
	})
	return count
}

func assertIdempotencyMetricAttributes(t *testing.T, name string, set attribute.Set) {
	t.Helper()
	for _, pair := range set.ToSlice() {
		key := string(pair.Key)
		if key != "event" && key != "outcome" && key != "stage" {
			t.Fatalf("%s has forbidden attribute %q", name, key)
		}
		if strings.Contains(pair.Value.AsString(), "sentinel") {
			t.Fatalf("%s retained unbounded value %q", name, pair.Value.AsString())
		}
	}
}
