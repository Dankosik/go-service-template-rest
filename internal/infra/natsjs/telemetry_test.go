package natsjs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMessagingTelemetryContract(t *testing.T) {
	const (
		payloadCanary    = "PAYLOAD_CANARY"
		credentialCanary = "CREDENTIAL_PATH_CANARY"
		brokerCanary     = "BROKER_ERROR_CANARY"
		eventTypeCanary  = "EVENT_TYPE_CANARY"
		headerCanary     = "HEADER_CANARY"
	)

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	var logs bytes.Buffer
	sig, err := newSignals(Observability{
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)),
		Meter:  provider.Meter(instrumentationScope),
	}, RoleWorker, func() bool { return true })
	if err != nil {
		t.Fatalf("newSignals() error = %v", err)
	}
	t.Cleanup(sig.close)

	ctx := t.Context()
	event := Event{
		Subject: "events.test", MessageID: "message-1", PublicationID: "publication-1",
		Type: eventTypeCanary, Schema: headerCanary, Payload: []byte(payloadCanary),
	}
	msg := Message{
		subject: "events.test", messageID: "message-1", publicationID: "publication-1",
		eventType: eventTypeCanary, schema: headerCanary, payload: []byte(payloadCanary),
		metadata: DeliveryMetadata{Consumer: "events-worker", NumDelivered: 2},
	}
	sig.publish(ctx, event, "accepted", "none", time.Now())
	sig.connection(ctx, "disconnected")
	sig.fetchMessages.Add(ctx, 1)
	sig.fetchBytes.Add(ctx, 64)
	sig.consumeActive.Add(ctx, 1)
	sig.handler(ctx, msg, "retryable", "handler_retry", time.Now())
	sig.terminal(ctx, msg.Subject(), &jetstream.MsgMetadata{
		Stream: "EVENTS", Consumer: "events-worker", NumDelivered: 2,
		Sequence: jetstream.SequencePair{Stream: 3, Consumer: 2},
	}, "handler_panic", []string{"featureHandler handler_test.go:42"})
	sig.redeliveries.Add(ctx, 1)
	sig.retries.Add(ctx, 1)
	sig.dlqTransfers.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "accepted")))
	sig.drainOperations.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", "graceful")))
	sig.forcedShutdowns.Add(ctx, 1)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &collected); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	want := map[string]map[string]string{
		"messaging.publish.operations": {"outcome": "accepted"},
		"messaging.publish.duration":   {"outcome": "accepted"},
		"messaging.connection.events":  {"event": "disconnected"},
		"messaging.readiness":          {"role": "worker"},
		"messaging.fetch.messages":     {},
		"messaging.fetch.bytes":        {},
		"messaging.consume.active":     {},
		"messaging.handler.operations": {"outcome": "retryable"},
		"messaging.handler.duration":   {"outcome": "retryable"},
		"messaging.redeliveries":       {},
		"messaging.retries":            {},
		"messaging.dlq.transfers":      {"outcome": "accepted"},
		"messaging.drain.operations":   {"outcome": "graceful"},
		"messaging.forced_shutdowns":   {},
	}
	got := messagingMetricAttributes(t, collected)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metric attributes = %#v, want %#v", got, want)
	}

	serialized := logs.String()
	for _, forbidden := range []string{payloadCanary, credentialCanary, brokerCanary, eventTypeCanary, headerCanary} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("messaging logs contain forbidden value %q: %s", forbidden, serialized)
		}
	}
	for _, required := range []string{"messaging_publish", "messaging_connection", "messaging_delivery", "messaging_terminal_delivery", "message_id", "subject", "outcome", "reason"} {
		if !strings.Contains(serialized, required) {
			t.Fatalf("messaging logs are missing %q: %s", required, serialized)
		}
	}
	records := decodeMessagingLogs(t, serialized)
	delivery := messagingLogByMessage(t, records, "messaging_delivery")
	if delivery["consumer"] != "events-worker" || delivery["attempt"] != float64(2) {
		t.Fatalf("messaging delivery log = %#v, want consumer and attempt", delivery)
	}
	terminal := messagingLogByMessage(t, records, "messaging_terminal_delivery")
	if terminal["stream"] != "EVENTS" || terminal["consumer"] != "events-worker" ||
		terminal["stream_sequence"] != float64(3) || terminal["attempt"] != float64(2) ||
		terminal["reason"] != "handler_panic" ||
		!reflect.DeepEqual(terminal["handler_frames"], []any{"featureHandler handler_test.go:42"}) {
		t.Fatalf("terminal delivery log = %#v, want safe source identity", terminal)
	}

	_, _, classified := classifyPublishError(errors.New(brokerCanary))
	if classified == nil {
		t.Fatal("classifyPublishError() returned nil for broker failure")
	}
	if strings.Contains(classified.Error(), brokerCanary) {
		t.Fatalf("classified broker error leaked raw text: %v", classified)
	}
	cfg := testConfig()
	cfg.URLs = []string{"tls://127.0.0.1:1"}
	cfg.CredentialsFile = "/" + credentialCanary
	cfg.Stream = "EVENTS"
	connectCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	_, err = Connect(connectCtx, cfg, RoleProducer, Observability{})
	if err == nil || strings.Contains(err.Error(), credentialCanary) {
		t.Fatalf("Connect() error = %v, want sanitized failure", err)
	}
}

func TestAsyncConnectionErrorIsClassifiedWithoutRawBrokerText(t *testing.T) {
	const brokerCanary = "BROKER_ERROR_CANARY"
	var logs bytes.Buffer
	sig, err := newSignals(Observability{Logger: slog.New(slog.NewJSONHandler(&logs, nil))}, RoleWorker, func() bool { return false })
	if err != nil {
		t.Fatalf("newSignals() error = %v", err)
	}
	t.Cleanup(sig.close)
	sig.asyncError(t.Context(), fmt.Errorf("%s: %w", brokerCanary, nats.ErrPermissionViolation))
	if strings.Contains(logs.String(), brokerCanary) {
		t.Fatalf("async connection log leaked raw broker error: %s", logs.String())
	}
	record := messagingLogByMessage(t, decodeMessagingLogs(t, logs.String()), "messaging_connection")
	if record["outcome"] != "async_error" || record["reason"] != "permission" {
		t.Fatalf("async connection log = %#v, want classified permission", record)
	}
}

func TestAsyncErrorReason(t *testing.T) {
	tests := map[string]struct {
		err  error
		want string
	}{
		"slow consumer":  {err: nats.ErrSlowConsumer, want: "slow_consumer"},
		"authentication": {err: fmt.Errorf("wrapped: %w", nats.ErrAuthExpired), want: "authentication"},
		"message bound":  {err: nats.ErrMaxPayload, want: "message_bound"},
		"connection":     {err: nats.ErrDisconnected, want: "connection"},
		"other":          {err: errors.New("unexpected"), want: "other"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := asyncErrorReason(tt.err); got != tt.want {
				t.Fatalf("asyncErrorReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func decodeMessagingLogs(t *testing.T, serialized string) []map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(serialized))
	var records []map[string]any
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode messaging log: %v", err)
		}
		records = append(records, record)
	}
	return records
}

func messagingLogByMessage(t *testing.T, records []map[string]any, message string) map[string]any {
	t.Helper()
	for _, record := range records {
		if record["msg"] == message {
			return record
		}
	}
	t.Fatalf("messaging log %q not found in %#v", message, records)
	return nil
}

func messagingMetricAttributes(t *testing.T, collected metricdata.ResourceMetrics) map[string]map[string]string {
	t.Helper()
	result := make(map[string]map[string]string)
	for _, scope := range collected.ScopeMetrics {
		if scope.Scope.Name != instrumentationScope {
			continue
		}
		for _, measured := range scope.Metrics {
			sets := aggregationAttributeSets(t, measured.Data)
			if len(sets) != 1 {
				t.Fatalf("%s data point attribute sets = %#v, want one", measured.Name, sets)
			}
			result[measured.Name] = sets[0]
		}
	}
	return result
}

func aggregationAttributeSets(t *testing.T, aggregation metricdata.Aggregation) []map[string]string {
	t.Helper()
	var sets []attribute.Set
	switch data := aggregation.(type) {
	case metricdata.Sum[int64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Gauge[int64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	default:
		t.Fatalf("unexpected metric aggregation %T", aggregation)
	}
	result := make([]map[string]string, 0, len(sets))
	for _, set := range sets {
		values := make(map[string]string)
		for _, value := range set.ToSlice() {
			values[string(value.Key)] = value.Value.AsString()
		}
		result = append(result, values)
	}
	return result
}
