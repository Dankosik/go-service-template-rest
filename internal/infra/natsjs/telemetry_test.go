package natsjs

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMessagingTelemetryKeepsOnlyBusinessOperations(t *testing.T) {
	reader, provider := telemetrytest.NewManualMeterProvider(t)
	recorder, tracerProvider := telemetrytest.NewRecordingTracerProvider(t)
	var logs bytes.Buffer
	sig, err := newTelemetry(Observability{
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)),
		Meter:  provider.Meter(instrumentationScope),
	})
	if err != nil {
		t.Fatalf("newTelemetry() error = %v", err)
	}
	event := validTestEvent()
	message := Message{subject: event.Subject, metadata: DeliveryMetadata{NumDelivered: 1}}
	sig.recordPublish(t.Context(), event, outcomeAccepted, reasonNone, time.Now())
	sig.recordHandler(t.Context(), message, outcomeRetryable, reasonHandlerRetry, time.Now())
	sig.recordDeadLetterTransfer(t.Context(), outcomeAccepted)
	_, span := tracerProvider.Tracer(instrumentationScope).Start(t.Context(), "outcome-test")
	setSpanOutcome(span, outcomeRetryable)
	span.End()

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
			for _, attributes := range telemetrytest.AttributeSets(t, measured.Data) {
				if telemetrytest.Attribute(t, attributes, attributeOutcome) == "" {
					t.Fatalf("%s has an empty %s attribute", measured.Name, attributeOutcome)
				}
			}
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
	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	if attributes := spans[0].Attributes(); len(attributes) != 1 || string(attributes[0].Key) != attributeOutcome || attributes[0].Value.AsString() != outcomeRetryable {
		t.Fatalf("span attributes = %v, want %s=%s", attributes, attributeOutcome, outcomeRetryable)
	}
}

// TestHandlerOutcomeMetric drives one failed delivery through the worker and
// reads the outcome label an operator alerts on from the handler operations
// counter.
func TestHandlerOutcomeMetric(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		delivered uint64
		handler   Handler
		want      string
	}{
		{
			name:      "handler cancellation with attempts left",
			delivered: 1,
			handler:   func(context.Context, Message) error { return context.Canceled },
			want:      "canceled",
		},
		{
			name:      "handler cancellation on the last attempt",
			delivered: 5,
			handler:   func(context.Context, Message) error { return context.Canceled },
			want:      "retryable",
		},
		{
			name:      "handler outlives its timeout",
			delivered: 1,
			handler: func(ctx context.Context, _ Message) error {
				<-ctx.Done()
				return ctx.Err()
			},
			want: "timeout",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				reader, provider := telemetrytest.NewManualMeterProvider(t)
				sig, err := newTelemetry(Observability{
					Logger: slog.New(slog.DiscardHandler),
					Meter:  provider.Meter(instrumentationScope),
				})
				if err != nil {
					t.Fatalf("newTelemetry() error = %v", err)
				}
				worker := unitWorker(t, &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS_DLQ", Sequence: 1}}, tc.handler)
				worker.client.telemetry = sig
				if err := worker.handle(t.Context(), unitSource(t, tc.delivered)); err != nil {
					t.Fatalf("handle() error = %v", err)
				}

				var points []metricdata.DataPoint[int64]
				telemetrytest.ForEachMetric(t.Context(), t, reader, func(measured metricdata.Metrics) {
					if measured.Name == "messaging.handler.operations" {
						points = telemetrytest.Int64Sum(t, measured).DataPoints
					}
				})
				if len(points) != 1 || points[0].Value != 1 {
					t.Fatalf("messaging.handler.operations points = %#v, want one operation", points)
				}
				if got := telemetrytest.Attribute(t, points[0].Attributes, attributeOutcome); got != tc.want {
					t.Fatalf("%s = %q, want %q", attributeOutcome, got, tc.want)
				}
			})
		})
	}
}
