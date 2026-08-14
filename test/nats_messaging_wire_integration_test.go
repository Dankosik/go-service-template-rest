//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/baggage"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestNATSOversizedSourceIsRetained(t *testing.T) {
	f := newNATSFixture(t)
	client, _, errCh := f.worker(t, func(context.Context, natsjs.Message) error {
		t.Fatal("oversized source reached handler")
		return nil
	}, func(cfg *natsjs.WorkerConfig) { cfg.Consumer = "oversized-worker" })
	payload := make([]byte, testMaxPayloadBytes+1)
	ack, err := f.js.Publish(t.Context(), sourceSubject, payload)
	if err != nil {
		t.Fatalf("publish oversized source: %v", err)
	}
	if err := waittest.Receive(t, errCh, 5*time.Second, "worker rejecting oversized source"); !errors.Is(err, natsjs.ErrTerminal) {
		t.Fatalf("worker error = %v, want ErrTerminal", err)
	}
	if client.Ready() {
		t.Fatal("client remained ready after terminal oversized delivery")
	}
	consumer, err := f.js.Consumer(t.Context(), sourceStream, "oversized-worker")
	if err != nil {
		t.Fatalf("lookup oversized consumer: %v", err)
	}
	consumerInfo, err := consumer.Info(t.Context())
	if err != nil {
		t.Fatalf("read oversized consumer state: %v", err)
	}
	if consumerInfo.NumAckPending != 1 {
		t.Fatalf("oversized consumer ack pending = %d, want exact source retained", consumerInfo.NumAckPending)
	}
	source, err := f.js.Stream(t.Context(), sourceStream)
	if err != nil {
		t.Fatalf("lookup oversized source stream: %v", err)
	}
	retained, err := source.GetMsg(t.Context(), ack.Sequence)
	if err != nil {
		t.Fatalf("read retained oversized source: %v", err)
	}
	if !slices.Equal(retained.Data, payload) {
		t.Fatalf("retained oversized source = %d bytes, want %d exact bytes", len(retained.Data), len(payload))
	}
	stream, err := f.js.Stream(t.Context(), deadLetterStream)
	if err != nil {
		t.Fatalf("lookup DLQ stream: %v", err)
	}
	info, err := stream.Info(t.Context())
	if err != nil {
		t.Fatalf("read DLQ stream: %v", err)
	}
	if info.State.Msgs != 0 {
		t.Fatalf("DLQ messages = %d, want 0", info.State.Msgs)
	}
}

func TestNATSOversizedHeadersAreDeadLettered(t *testing.T) {
	f := newNATSFixture(t)
	var handlerCalls atomic.Int32
	_, _, _ = f.worker(t, func(context.Context, natsjs.Message) error {
		handlerCalls.Add(1)
		return nil
	}, func(cfg *natsjs.WorkerConfig) { cfg.Consumer = "oversized-header-worker" })
	payload := []byte("oversized-header-payload")
	msg := nats.NewMsg(sourceSubject)
	msg.Header.Set("Message-Id", "oversized-header-message")
	msg.Header.Set("Publication-Id", "oversized-header-publication")
	msg.Header.Set("Event-Type", "test.event")
	msg.Header.Set("Event-Schema", "v1")
	msg.Header.Set("Created-At", time.Now().UTC().Format(time.RFC3339Nano))
	msg.Header.Set("X-Oversized", strings.Repeat("h", natsjs.HeaderLimitBytes))
	msg.Data = payload
	ack, err := f.js.PublishMsg(t.Context(), msg, jetstream.WithMsgID("oversized-header-publication"))
	if err != nil {
		t.Fatalf("publish oversized headers: %v", err)
	}
	var deadLetter *jetstream.RawStreamMsg
	waittest.Until(t, 5*time.Second, func() bool {
		stream, streamErr := f.js.Stream(t.Context(), deadLetterStream)
		if streamErr != nil {
			return false
		}
		deadLetter, streamErr = stream.GetLastMsgForSubject(t.Context(), deadLetterSubject)
		return streamErr == nil
	}, "oversized headers dead-letter transfer")
	if handlerCalls.Load() != 0 {
		t.Fatalf("handler calls for oversized headers = %d, want 0", handlerCalls.Load())
	}
	if !slices.Equal(deadLetter.Data, payload) || deadLetter.Header.Get("Original-Stream-Sequence") != fmt.Sprint(ack.Sequence) {
		t.Fatalf("oversized-header DLQ transfer did not preserve source identity and payload")
	}
	waitConsumerSettled(t, f, "oversized-header-worker")
}

func TestNATSTraceCorrelation(t *testing.T) {
	const (
		payloadCanary = "TRACE_PAYLOAD_CANARY"
		typeCanary    = "TRACE_EVENT_TYPE_CANARY"
		schemaCanary  = "TRACE_SCHEMA_CANARY"
		baggageCanary = "TRACE_BAGGAGE_CANARY"
	)
	f := newNATSFixture(t)
	recorder, provider := telemetrytest.NewRecordingTracerProvider(t)
	observed := make(chan struct {
		requestID string
		traceID   string
		spanID    string
		baggage   int
	}, 1)
	cfg := testClientConfig()
	cfg.URLs = []string{f.url}
	cfg.AllowPlaintext = true
	cfg.AllowUnauthenticated = true
	cfg.Stream = sourceStream
	client, err := natsjs.Connect(t.Context(), cfg, natsjs.RoleWorker, natsjs.Observability{Tracer: provider.Tracer("test")})
	if err != nil {
		t.Fatalf("connect messaging client: %v", err)
	}
	t.Cleanup(client.Close)
	workerCfg := testWorkerConfig()
	workerCfg.Consumer = "trace-worker"
	workerCfg.FilterSubject = sourceSubject
	workerCfg.DeadLetterSubject = deadLetterSubject
	worker, err := client.NewWorker(t.Context(), workerCfg, func(ctx context.Context, _ natsjs.Message) error {
		observed <- struct {
			requestID string
			traceID   string
			spanID    string
			baggage   int
		}{reqctx.RequestID(ctx), traceID(ctx), trace.SpanContextFromContext(ctx).SpanID().String(), baggage.FromContext(ctx).Len()}
		return nil
	})
	if err != nil {
		t.Fatalf("create trace worker: %v", err)
	}
	runCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { _ = worker.Run(runCtx) }()
	parentCtx, parent := provider.Tracer("test").Start(t.Context(), "parent")
	parentCtx = reqctx.ContextWithRequestID(parentCtx, "request-123")
	member, err := baggage.NewMember("untrusted", baggageCanary)
	if err != nil {
		t.Fatalf("create baggage canary: %v", err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatalf("create baggage: %v", err)
	}
	parentCtx = baggage.ContextWithBaggage(parentCtx, bag)
	event := testEvent(payloadCanary)
	event.Type = typeCanary
	event.Schema = schemaCanary
	if _, err := client.Producer().Publish(parentCtx, event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	got := waittest.Receive(t, observed, 5*time.Second, "trace delivery")
	if got.requestID != "request-123" || got.traceID != parent.SpanContext().TraceID().String() || got.baggage != 0 {
		t.Fatalf("handler correlation = %+v, want request-123/%s and no baggage", got, parent.SpanContext().TraceID())
	}
	parent.End()
	waittest.Until(t, 5*time.Second, func() bool { return len(recorder.Ended()) >= 3 }, "producer, consumer, and parent spans")
	spansByName := make(map[string]sdktrace.ReadOnlySpan)
	for _, span := range recorder.Ended() {
		if _, duplicate := spansByName[span.Name()]; duplicate {
			t.Fatalf("duplicate span %q", span.Name())
		}
		spansByName[span.Name()] = span
	}
	// semconv names a messaging span "{operation name} {destination}". The
	// consume span carries the worker's filter rather than the delivered
	// subject; both are sourceSubject here, so this fixture cannot tell them
	// apart — TestMessagingSpanNamesFollowSemanticConventions does.
	publishSpan, publishOK := spansByName["publish "+sourceSubject]
	consumeSpan, consumeOK := spansByName["process "+sourceSubject]
	parentSpan, parentOK := spansByName["parent"]
	if len(spansByName) != 3 || !publishOK || !consumeOK || !parentOK {
		t.Fatalf("ended spans = %#v, want exactly parent, publish, and process spans", spansByName)
	}
	if publishSpan.SpanKind() != trace.SpanKindProducer || consumeSpan.SpanKind() != trace.SpanKindConsumer {
		t.Fatalf("messaging span kinds = publish:%s consume:%s, want Producer/Consumer", publishSpan.SpanKind(), consumeSpan.SpanKind())
	}
	if publishSpan.Parent().SpanID() != parentSpan.SpanContext().SpanID() || consumeSpan.Parent().SpanID() != publishSpan.SpanContext().SpanID() {
		t.Fatalf("span parents = publish:%s consume:%s, want parent:%s publish:%s", publishSpan.Parent().SpanID(), consumeSpan.Parent().SpanID(), parentSpan.SpanContext().SpanID(), publishSpan.SpanContext().SpanID())
	}
	if got.spanID != consumeSpan.SpanContext().SpanID().String() {
		t.Fatalf("handler span ID = %s, want consumer span %s", got.spanID, consumeSpan.SpanContext().SpanID())
	}
	var spanData strings.Builder
	for _, span := range recorder.Ended() {
		fmt.Fprintln(&spanData, span.Name())
		for _, value := range span.Attributes() {
			fmt.Fprintln(&spanData, value.Key, value.Value.AsInterface())
		}
		for _, event := range span.Events() {
			fmt.Fprintln(&spanData, event.Name)
			for _, value := range event.Attributes {
				fmt.Fprintln(&spanData, value.Key, value.Value.AsInterface())
			}
		}
	}
	for _, forbidden := range []string{payloadCanary, typeCanary, schemaCanary, baggageCanary} {
		if strings.Contains(spanData.String(), forbidden) {
			t.Fatalf("messaging spans contain forbidden value %q: %s", forbidden, spanData.String())
		}
	}
}

func traceID(ctx context.Context) string {
	return trace.SpanContextFromContext(ctx).TraceID().String()
}
