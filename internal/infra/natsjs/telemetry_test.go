package natsjs

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMessagingTelemetryKeepsOnlyBusinessOperations(t *testing.T) {
	reader, provider := telemetrytest.NewManualMeterProvider(t)
	var logs bytes.Buffer
	sig, err := newTelemetry(Observability{
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)),
		Meter:  provider.Meter(instrumentationScope),
	}, RoleWorker, func() bool { return true })
	if err != nil {
		t.Fatalf("newTelemetry() error = %v", err)
	}
	event := validTestEvent()
	message := Message{subject: event.Subject, metadata: DeliveryMetadata{NumDelivered: 1}}
	sig.recordPublish(t.Context(), event, outcomeAccepted, reasonNone, time.Now())
	sig.recordHandler(t.Context(), message, outcomeRetryable, reasonHandlerRetry, time.Now())
	sig.recordDeadLetterTransfer(t.Context(), outcomeAccepted)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	got := make(map[string]struct{})
	for _, scope := range collected.ScopeMetrics {
		if scope.Scope.Name != instrumentationScope {
			continue
		}
		for _, measured := range scope.Metrics {
			got[measured.Name] = struct{}{}
		}
	}
	for _, name := range []string{
		"messaging.publish.operations", "messaging.publish.duration",
		"messaging.handler.operations", "messaging.handler.duration", "messaging.dlq.transfers",
	} {
		if _, ok := got[name]; !ok {
			t.Errorf("missing metric %q in %#v", name, got)
		}
	}
	if len(got) != 5 {
		t.Fatalf("messaging metrics = %#v, want five", got)
	}
	if strings.Contains(logs.String(), "messaging_publish_failed") || !strings.Contains(logs.String(), "messaging_delivery_failed") {
		t.Fatalf("operation logs = %s", logs.String())
	}
}

func TestHandlerOutcomeClassification(t *testing.T) {
	if got := handlerOutcome(handlerResult{err: context.Canceled}, false); got != outcomeCanceled {
		t.Fatalf("handlerOutcome(canceled) = %q", got)
	}
}
