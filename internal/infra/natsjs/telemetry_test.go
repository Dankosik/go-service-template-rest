package natsjs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
)

// The outcome attribute is shared by four counters and the connection event by
// one, and only boundedOutcome and boundedConnectionEvent bound them. A value a
// call site invents but those lists forgot would report "other" on the metric
// while the operator log beside it carries the real string, so the two could
// not be correlated. This drives the real producers rather than restating the
// lists, so a classifier that gains a case fails here until the case is named.
func TestDeliveryVocabularyIsBounded(t *testing.T) {
	t.Parallel()

	outcomes := map[string]string{
		"publish broker rejection": outcomeOf(classifyPublishError(nats.ErrNoResponders)),
		"publish api error":        outcomeOf(classifyPublishError(&jetstream.APIError{ErrorCode: 1})),
		"publish no ack":           outcomeOf(classifyPublishError(errors.New("broker detail"))),
		"handler timeout": handlerOutcome(
			handlerResult{contextErr: context.DeadlineExceeded, err: context.DeadlineExceeded}, false),
		"handler canceled":  handlerOutcome(handlerResult{err: context.Canceled}, false),
		"handler retryable": handlerOutcome(handlerResult{err: errors.New("handler detail")}, false),
		"handler exhausted": handlerOutcome(handlerResult{err: context.Canceled}, true),
	}
	// The outcomes no classifier produces: each is named directly at its one
	// settlement or drain site, so the constant is the producer.
	maps.Copy(outcomes, map[string]string{
		"publish accepted":     outcomeAccepted,
		"handler success":      outcomeSuccess,
		"handler panic":        outcomeTerminal,
		"handler shutdown":     outcomeCanceled,
		"handler permanent":    outcomePermanent,
		"dead-letter accepted": outcomeAccepted,
		"dead-letter rejected": outcomeRejected,
		"drain graceful":       outcomeGraceful,
		"drain forced":         outcomeForced,
		"drain failed":         outcomeFailed,
	})
	for source, outcome := range outcomes {
		if bounded := boundedOutcome(outcome); bounded != outcome {
			t.Errorf("boundedOutcome(%q) from %s = %q, want the outcome unchanged", outcome, source, bounded)
		}
	}

	for source, event := range map[string]string{
		"disconnect handler": connectionDisconnected,
		"reconnect handler":  connectionReconnected,
		"closed handler":     connectionClosed,
		"async error":        connectionAsyncError,
	} {
		if bounded := boundedConnectionEvent(event); bounded != event {
			t.Errorf("boundedConnectionEvent(%q) from %s = %q, want the event unchanged", event, source, bounded)
		}
	}

	// An unrecognized value must collapse rather than mint a time series.
	if got := boundedOutcome("invented"); got != boundedOther {
		t.Errorf("boundedOutcome(invented) = %q, want %q", got, boundedOther)
	}
	if got := boundedConnectionEvent("invented"); got != boundedOther {
		t.Errorf("boundedConnectionEvent(invented) = %q, want %q", got, boundedOther)
	}
}

// outcomeOf keeps classifyPublishError's other two results out of the table
// above, which is only about the attribute it produces.
func outcomeOf(outcome, _ string, _ error) string { return outcome }

func TestMessagingTelemetryContract(t *testing.T) {
	t.Parallel()
	const (
		payloadCanary    = "PAYLOAD_CANARY"
		credentialCanary = "CREDENTIAL_PATH_CANARY"
		brokerCanary     = "BROKER_ERROR_CANARY"
		eventTypeCanary  = "EVENT_TYPE_CANARY"
		headerCanary     = "HEADER_CANARY"
	)

	reader, provider := telemetrytest.NewManualMeterProvider(t)
	spanRecorder, traceProvider := telemetrytest.NewRecordingTracerProvider(t)
	var logs bytes.Buffer
	sig, err := newTelemetry(Observability{
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)),
		Meter:  provider.Meter(instrumentationScope),
		Tracer: traceProvider.Tracer(instrumentationScope),
	}, RoleWorker, func() bool { return true })
	if err != nil {
		t.Fatalf("newTelemetry() error = %v", err)
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
	// Every counter that carries an outcome goes through its recorder rather
	// than through the instrument, so this exercises the bounding the
	// production path applies instead of a hand-built attribute set.
	sig.recordPublish(ctx, event, outcomeAccepted, reasonNone, time.Now())
	_, publishSpan := sig.tracer.Start(ctx, "publish", publishSpanOptions(event)...)
	publishSpan.End()
	_, consumeSpan := sig.tracer.Start(ctx, "consume", consumeSpanOptions(msg, "events.>")...)
	consumeSpan.End()
	sig.recordConnection(ctx, connectionDisconnected)
	sig.countFetch(ctx, 1, 64)
	sig.countConsumeActive(ctx, 1)
	sig.recordHandler(ctx, msg, outcomeRetryable, reasonHandlerRetry, time.Now())
	sig.logTerminalDelivery(ctx, msg.Subject(), &jetstream.MsgMetadata{
		Stream: "EVENTS", Consumer: "events-worker", NumDelivered: 2,
		Sequence: jetstream.SequencePair{Stream: 3, Consumer: 2},
	}, reasonHandlerPanic, &handlerPanic{class: "string", frames: []string{"featureHandler handler_test.go:42"}})
	sig.countRedelivery(ctx)
	sig.countRetry(ctx)
	sig.recordDeadLetterTransfer(ctx, outcomeAccepted)
	sig.recordDrain(ctx, outcomeGraceful)
	sig.countForcedShutdown(ctx)
	sig.recordBrokerObservation(time.Unix(500, 0), &jetstream.StreamInfo{
		Config: jetstream.StreamConfig{MaxMsgs: 100, MaxBytes: 1000},
		State: jetstream.StreamState{
			Msgs: 4, Bytes: 200, FirstTime: time.Unix(100, 0),
		},
	}, &jetstream.ConsumerInfo{NumPending: 3, NumAckPending: 2})

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &collected); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	want := map[string]map[string]string{
		"messaging.publish.operations":      {"outcome": "accepted"},
		"messaging.publish.duration":        {"outcome": "accepted"},
		"messaging.connection.events":       {"event": "disconnected"},
		"messaging.readiness":               {"role": "worker"},
		"messaging.fetch.messages":          {},
		"messaging.fetch.bytes":             {},
		"messaging.consume.active":          {},
		"messaging.handler.operations":      {"outcome": "retryable"},
		"messaging.handler.duration":        {"outcome": "retryable"},
		"messaging.redeliveries":            {},
		"messaging.retries":                 {},
		"messaging.dlq.transfers":           {"outcome": "accepted"},
		"messaging.drain.operations":        {"outcome": "graceful"},
		"messaging.forced_shutdowns":        {},
		"messaging.consumer.pending":        {},
		"messaging.consumer.ack_pending":    {},
		"messaging.stream.messages":         {},
		"messaging.stream.messages.limit":   {},
		"messaging.stream.storage":          {},
		"messaging.stream.storage.limit":    {},
		"messaging.stream.oldest.timestamp": {},
		"messaging.observation.timestamp":   {},
	}
	got := messagingMetricAttributes(t, collected)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metric attributes = %#v, want %#v", got, want)
	}
	wantGauges := map[string]int64{
		"messaging.consumer.pending":        3,
		"messaging.consumer.ack_pending":    2,
		"messaging.stream.messages":         4,
		"messaging.stream.messages.limit":   100,
		"messaging.stream.storage":          200,
		"messaging.stream.storage.limit":    1000,
		"messaging.stream.oldest.timestamp": 100,
		"messaging.observation.timestamp":   500,
	}
	if got := messagingGaugeValues(t, collected); !reflect.DeepEqual(got, wantGauges) {
		t.Fatalf("broker gauge values = %#v, want %#v", got, wantGauges)
	}

	serialized := logs.String()
	for _, forbidden := range []string{payloadCanary, credentialCanary, brokerCanary, eventTypeCanary, headerCanary, "message-1", "publication-1", "events-worker"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("messaging logs contain forbidden value %q: %s", forbidden, serialized)
		}
	}
	for _, required := range []string{"messaging_publish", "messaging_connection", "messaging_delivery", "messaging_terminal_delivery", "subject", "outcome", "reason"} {
		if !strings.Contains(serialized, required) {
			t.Fatalf("messaging logs are missing %q: %s", required, serialized)
		}
	}
	records := decodeMessagingLogs(t, serialized)
	delivery := messagingLogByMessage(t, records, "messaging_delivery")
	if delivery["attempt"] != float64(2) {
		t.Fatalf("messaging delivery log = %#v, want attempt", delivery)
	}
	terminal := messagingLogByMessage(t, records, "messaging_terminal_delivery")
	if terminal["attempt"] != float64(2) ||
		terminal["reason"] != "handler_panic" ||
		terminal["panic.class"] != "string" ||
		!reflect.DeepEqual(terminal["handler_frames"], []any{"featureHandler handler_test.go:42"}) {
		t.Fatalf("terminal delivery log = %#v, want bounded diagnostics", terminal)
	}
	for _, span := range spanRecorder.Ended() {
		for _, attr := range span.Attributes() {
			if attr.Key == "messaging.message.id" || strings.Contains(attr.Value.AsString(), "message-1") || strings.Contains(attr.Value.AsString(), "events-worker") {
				t.Fatalf("span %q leaked durable identity through %s=%v", span.Name(), attr.Key, attr.Value)
			}
		}
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

// The span names and their semconv attributes, pinned here because the real
// broker suite cannot see the distinction that matters: it configures one
// worker whose filter equals the subject it publishes to, so a consume span
// named after the delivered subject and one named after the filter are the
// same string there. A wildcard filter separates them, and separating them is
// the point — the delivered subject of a wildcard subscription is what would
// put an unbounded value in a span name.
func TestMessagingSpanNamesFollowSemanticConventions(t *testing.T) {
	t.Parallel()

	const filter = "events.>"
	recorder, provider := telemetrytest.NewRecordingTracerProvider(t)
	tracer := provider.Tracer(instrumentationScope)

	event := Event{Subject: "events.created", MessageID: "message-1", PublicationID: "publication-1"}
	msg := Message{subject: "events.created", messageID: "message-1"}
	_, publishSpan := tracer.Start(t.Context(), publishSpanName(event.Subject), publishSpanOptions(event)...)
	publishSpan.End()
	_, consumeSpan := tracer.Start(t.Context(), consumeSpanName(filter), consumeSpanOptions(msg, filter)...)
	consumeSpan.End()

	// Keyed by span name, so the names are asserted by the comparison itself
	// rather than by a lookup that would have to report a miss of its own.
	got := make(map[string]map[attribute.Key]string)
	for _, span := range recorder.Ended() {
		attrs := make(map[attribute.Key]string, len(span.Attributes()))
		for _, attr := range span.Attributes() {
			attrs[attr.Key] = attr.Value.AsString()
		}
		got[span.Name()] = attrs
	}
	want := map[string]map[attribute.Key]string{
		"publish events.created": {
			"messaging.system":         "nats",
			"messaging.operation.type": "send",
			"messaging.operation.name": "publish",
			// The producer's own subject; there is no template to prefer.
			"messaging.destination.name": "events.created",
		},
		"process events.>": {
			"messaging.system":         "nats",
			"messaging.operation.type": "process",
			"messaging.operation.name": "process",
			// What arrived, beside what this consumer subscribes to. Only the
			// second reaches the span name above.
			"messaging.destination.name":     "events.created",
			"messaging.destination.template": filter,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("messaging spans = %#v, want %#v", got, want)
	}
}

func TestAsyncConnectionErrorIsClassifiedWithoutRawBrokerText(t *testing.T) {
	t.Parallel()
	const brokerCanary = "BROKER_ERROR_CANARY"
	var logs bytes.Buffer
	sig, err := newTelemetry(Observability{Logger: slog.New(slog.NewJSONHandler(&logs, nil))}, RoleWorker, func() bool { return false })
	if err != nil {
		t.Fatalf("newTelemetry() error = %v", err)
	}
	t.Cleanup(sig.close)
	sig.recordAsyncError(t.Context(), fmt.Errorf("%s: %w", brokerCanary, nats.ErrPermissionViolation))
	if strings.Contains(logs.String(), brokerCanary) {
		t.Fatalf("async connection log leaked raw broker error: %s", logs.String())
	}
	record := messagingLogByMessage(t, decodeMessagingLogs(t, logs.String()), "messaging_connection")
	if record["outcome"] != "async_error" || record["reason"] != "permission" {
		t.Fatalf("async connection log = %#v, want classified permission", record)
	}
}

func TestAsyncErrorReason(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	sets := telemetrytest.AttributeSets(t, aggregation)
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

func messagingGaugeValues(t *testing.T, collected metricdata.ResourceMetrics) map[string]int64 {
	t.Helper()
	result := make(map[string]int64)
	for _, scope := range collected.ScopeMetrics {
		if scope.Scope.Name != instrumentationScope {
			continue
		}
		for _, measured := range scope.Metrics {
			if !strings.HasPrefix(measured.Name, "messaging.consumer.") &&
				!strings.HasPrefix(measured.Name, "messaging.stream.") &&
				measured.Name != "messaging.observation.timestamp" {
				continue
			}
			gauge, ok := measured.Data.(metricdata.Gauge[int64])
			if !ok || len(gauge.DataPoints) != 1 {
				t.Fatalf("%s aggregation = %T with %d points, want one int64 gauge", measured.Name, measured.Data, len(gauge.DataPoints))
			}
			result[measured.Name] = gauge.DataPoints[0].Value
		}
	}
	return result
}
