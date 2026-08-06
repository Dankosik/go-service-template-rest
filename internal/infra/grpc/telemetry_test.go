// What an operator sees: spans, metrics, and access-log records for real RPCs,
// and the guarantee that none of them carries a value from the request.
//
// The secret canary travels through metadata, a handler error, and a panic, so a
// disclosure fails here rather than in production. This file also owns the proof
// that the telemetry filter drops unregistered methods.

package grpcx

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestOTelStatsHandlersTraceAndMeasureUnaryAndStreamingRPCs(t *testing.T) {
	spanRecorder, tracerProvider, metricReader, meterProvider := newRecordingTelemetry(t)

	cfg := testServerConfig()
	cfg.TelemetryHealthChecks = true
	server, err := NewServer(cfg, Options{
		MeterProvider:  meterProvider,
		TracerProvider: tracerProvider,
		Propagators:    propagation.TraceContext{},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	server.MarkServing()

	connection, serveDone := serveOverTCP(t, server, grpcclient.Options{
		MeterProvider:  meterProvider,
		TracerProvider: tracerProvider,
		Propagation:    grpcclient.PropagationTraceContext,
	})
	closeAfterTest(t, server, connection, serveDone)

	parentCtx, parent := tracerProvider.Tracer("grpcx-test").Start(t.Context(), "parent")
	parentTraceID := parent.SpanContext().TraceID()
	healthClient := healthgrpc.NewHealthClient(connection)
	if _, err := healthClient.Check(parentCtx, &healthgrpc.HealthCheckRequest{}); err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}

	watchCtx, cancelWatch := context.WithCancel(parentCtx)
	watch, err := healthClient.Watch(watchCtx, &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health.Watch() error = %v", err)
	}
	if _, err := watch.Recv(); err != nil {
		t.Fatalf("Health.Watch().Recv() initial error = %v", err)
	}
	cancelWatch()
	if _, err := watch.Recv(); status.Code(err) != codes.Canceled {
		t.Fatalf("Health.Watch().Recv() after cancel = %v, want Canceled", err)
	}
	if err := server.Shutdown(t.Context()); err != nil {
		t.Fatalf("Server.Shutdown() after stream cancellation = %v", err)
	}
	parent.End()

	assertRPCSpans(t, spanRecorder.Ended(), parentTraceID)
	assertRPCMetrics(t, metricReader)
}

func TestServerObservabilitySanitizesValuesAndFiltersUnknownMethods(t *testing.T) {
	t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "")

	const secretCanary = "credential=grpc-observability-secret"

	spanRecorder, tracerProvider, metricReader, meterProvider := newRecordingTelemetry(t)

	var logOutput bytes.Buffer
	var handlerCalls atomic.Int64
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testPayloadFullMethod,
			func(
				_ context.Context,
				request *wrapperspb.BytesValue,
			) (*wrapperspb.BytesValue, error) {
				handlerCalls.Add(1)
				if strings.HasPrefix(string(request.GetValue()), "panic:") {
					panic(secretCanary)
				}
				return nil, errors.New(secretCanary)
			})
	}
	rejectOnDemandPolicy := func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		incoming, _ := metadata.FromIncomingContext(ctx)
		if len(incoming.Get("x-test-reject")) > 0 {
			return nil, errors.New(secretCanary)
		}
		return handler(ctx, request)
	}
	server, connection := startTestServerWithOptions(
		t,
		testServerConfig(),
		register,
		Options{
			Logger:         slogJSONLogger(&logOutput),
			MeterProvider:  meterProvider,
			TracerProvider: tracerProvider,
			Propagators:    propagation.TraceContext{},
			UnaryPolicy:    []grpc.UnaryServerInterceptor{rejectOnDemandPolicy},
		},
	)
	server.MarkServing()

	var header metadata.MD
	var trailer metadata.MD
	handlerCtx := metadata.NewOutgoingContext(
		t.Context(),
		metadata.Pairs("x-secret-canary", secretCanary),
	)
	err := connection.Invoke(
		handlerCtx,
		testPayloadFullMethod,
		&wrapperspb.BytesValue{Value: []byte(secretCanary)},
		&wrapperspb.BytesValue{},
		grpc.Header(&header),
		grpc.Trailer(&trailer),
	)
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("handler status detail = %q, want sanitized detail", detail)
	}
	assertMetadataDoesNotContain(t, header, secretCanary)
	assertMetadataDoesNotContain(t, trailer, secretCanary)

	policyCtx := metadata.NewOutgoingContext(
		t.Context(),
		metadata.Pairs("x-test-reject", secretCanary),
	)
	err = connection.Invoke(
		policyCtx,
		testPayloadFullMethod,
		&wrapperspb.BytesValue{},
		&wrapperspb.BytesValue{},
	)
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("policy status detail = %q, want sanitized detail", detail)
	}

	err = connection.Invoke(
		t.Context(),
		testPayloadFullMethod,
		&wrapperspb.BytesValue{Value: []byte("panic:" + secretCanary)},
		&wrapperspb.BytesValue{},
	)
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("panic status detail = %q, want sanitized detail", detail)
	}
	if calls := handlerCalls.Load(); calls != 2 {
		t.Fatalf("handler calls = %d, want raw-error and panic calls only", calls)
	}

	if _, err := healthgrpc.NewHealthClient(connection).Check(
		t.Context(),
		&healthgrpc.HealthCheckRequest{},
	); err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}

	for _, method := range []string{
		"/attacker.Service/credential-grpc-observability-secret-a",
		"/attacker.Service/credential-grpc-observability-secret-b",
	} {
		err := connection.Invoke(t.Context(), method, &emptypb.Empty{}, &emptypb.Empty{})
		if code := status.Code(err); code != codes.Unimplemented {
			t.Fatalf("unknown method %q status = %s, want %s", method, code, codes.Unimplemented)
		}
	}

	finishServerRPCs(t, server)

	assertSanitizedAccessLogs(t, logOutput.Bytes(), secretCanary, 3, 1)
	assertSanitizedServerSpans(t, spanRecorder.Ended(), secretCanary, map[string]bool{
		"grpcx.test.PayloadService/Call": true,
	})
	assertSanitizedServerMetrics(t, metricReader, secretCanary, map[string]bool{
		"grpcx.test.PayloadService/Call": true,
	})
}

func TestStreamingPolicyObservabilitySanitizesRawError(t *testing.T) {
	t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "")

	const secretCanary = "credential=grpc-stream-policy-secret"

	spanRecorder, tracerProvider, metricReader, meterProvider := newRecordingTelemetry(t)

	var logOutput bytes.Buffer
	cfg := testServerConfig()
	cfg.LogHealthChecks = true
	cfg.TelemetryHealthChecks = true
	server, connection := startTestServerWithOptions(t, cfg, nil, Options{
		Logger:         slogJSONLogger(&logOutput),
		MeterProvider:  meterProvider,
		TracerProvider: tracerProvider,
		Propagators:    propagation.TraceContext{},
		StreamPolicy: []grpc.StreamServerInterceptor{
			func(any, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
				return errors.New(secretCanary)
			},
		},
	})
	server.MarkServing()

	watch, err := healthgrpc.NewHealthClient(connection).Watch(
		metadata.NewOutgoingContext(
			t.Context(),
			metadata.Pairs("x-secret-canary", secretCanary),
		),
		&healthgrpc.HealthCheckRequest{Service: secretCanary},
	)
	if err == nil {
		_, err = watch.Recv()
	}
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("stream policy status detail = %q, want sanitized detail", detail)
	}

	finishServerRPCs(t, server)

	assertSanitizedAccessLogs(t, logOutput.Bytes(), secretCanary, 1, 0)
	assertSanitizedServerSpans(t, spanRecorder.Ended(), secretCanary, map[string]bool{
		"grpc.health.v1.Health/Watch": true,
	})
	assertSanitizedServerMetrics(t, metricReader, secretCanary, map[string]bool{
		"grpc.health.v1.Health/Watch": true,
	})
}

// finishServerRPCs blocks until the server has completed every in-flight RPC.
// A client observes an RPC status as soon as the server sends it, which is
// strictly earlier than the server recording that RPC in its access log, span
// processor, and metric instruments. Assertions on server-recorded telemetry
// synchronize on this completion rather than on the client-observed result.
func finishServerRPCs(t *testing.T, server *Server) {
	t.Helper()

	if err := server.Shutdown(t.Context()); err != nil {
		t.Fatalf("Server.Shutdown() before telemetry assertions = %v", err)
	}
}

// newRecordingTelemetry returns in-memory tracing and metric providers that shut
// down with the test.
//
// They are local values passed explicitly into Options rather than the
// process-wide providers telemetrytest installs: these tests assert what the
// explicit wiring produces, and a local provider keeps them independent of
// global state and safe to run alongside anything else.
func newRecordingTelemetry(t *testing.T) (
	*tracetest.SpanRecorder,
	*sdktrace.TracerProvider,
	*sdkmetric.ManualReader,
	*sdkmetric.MeterProvider,
) {
	t.Helper()

	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
	})

	metricReader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))
	t.Cleanup(func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("MeterProvider.Shutdown() error = %v", err)
		}
	})

	return spanRecorder, tracerProvider, metricReader, meterProvider
}

func assertRPCSpans(t *testing.T, spans []sdktrace.ReadOnlySpan, parentTraceID trace.TraceID) {
	t.Helper()

	methods := []string{
		"grpc.health.v1.Health/Check",
		"grpc.health.v1.Health/Watch",
	}
	for _, method := range methods {
		for _, kind := range []trace.SpanKind{trace.SpanKindClient, trace.SpanKindServer} {
			found := false
			for _, span := range spans {
				if span.Name() != method || span.SpanKind() != kind {
					continue
				}
				if span.SpanContext().TraceID() != parentTraceID {
					t.Fatalf(
						"%s %s span trace ID = %s, want propagated %s",
						kind,
						method,
						span.SpanContext().TraceID(),
						parentTraceID,
					)
				}
				found = true
				break
			}
			if !found {
				t.Fatalf("missing %s span for %s; spans = %v", kind, method, spanNames(spans))
			}
		}
	}
}

func slogJSONLogger(output *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, nil))
}

func assertMetadataDoesNotContain(t *testing.T, values metadata.MD, secret string) {
	t.Helper()

	for key, entries := range values {
		if strings.Contains(key, secret) {
			t.Fatalf("response metadata key %q contains secret canary", key)
		}
		for _, entry := range entries {
			if strings.Contains(entry, secret) {
				t.Fatalf("response metadata %q contains secret canary", key)
			}
		}
	}
}

func assertSanitizedAccessLogs(
	t *testing.T,
	encoded []byte,
	secret string,
	wantRequests int,
	wantPanics int,
) {
	t.Helper()

	if strings.Contains(string(encoded), secret) {
		t.Fatalf("gRPC logs disclose secret canary: %s", encoded)
	}
	requests := 0
	panics := 0
	scanner := bufio.NewScanner(bytes.NewReader(encoded))
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("decode gRPC log record: %v", err)
		}
		switch record["msg"] {
		case "grpc_request":
			requests++
			if got := record["rpc.status"]; got != codes.Internal.String() {
				t.Fatalf("access-log status = %v, want %s", got, codes.Internal)
			}
			if got := record["rpc.method"]; strings.HasPrefix(asString(got), healthMethodPrefix+"Check") {
				t.Fatalf("routine health check reached access log: %v", record)
			}
		case "grpc_panic_recovered":
			panics++
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan gRPC log records: %v", err)
	}
	if requests != wantRequests {
		t.Fatalf("grpc_request records = %d, want %d; logs = %s", requests, wantRequests, encoded)
	}
	if panics != wantPanics {
		t.Fatalf(
			"grpc_panic_recovered records = %d, want %d; logs = %s",
			panics,
			wantPanics,
			encoded,
		)
	}
}

func assertSanitizedServerSpans(
	t *testing.T,
	spans []sdktrace.ReadOnlySpan,
	secret string,
	allowedMethods map[string]bool,
) {
	t.Helper()

	seen := make(map[string]bool)
	for _, span := range spans {
		if span.SpanKind() != trace.SpanKindServer {
			continue
		}
		if !allowedMethods[span.Name()] {
			t.Fatalf("server span name = %q, want registered method from %v", span.Name(), allowedMethods)
		}
		seen[span.Name()] = true
		assertStringDoesNotContain(t, "span name", span.Name(), secret)
		assertStringDoesNotContain(t, "span status", span.Status().Description, secret)
		for _, attr := range span.Attributes() {
			assertStringDoesNotContain(t, "span attribute "+string(attr.Key), attr.Value.String(), secret)
		}
		for _, event := range span.Events() {
			assertStringDoesNotContain(t, "span event name", event.Name, secret)
			for _, attr := range event.Attributes {
				assertStringDoesNotContain(t, "span event attribute "+string(attr.Key), attr.Value.String(), secret)
			}
		}
	}
	for method := range allowedMethods {
		if !seen[method] {
			t.Fatalf("missing server span for registered method %q; spans = %v", method, spanNames(spans))
		}
	}
}

func assertSanitizedServerMetrics(
	t *testing.T,
	reader *sdkmetric.ManualReader,
	secret string,
	allowedMethods map[string]bool,
) {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &resourceMetrics); err != nil {
		t.Fatalf("ManualReader.Collect() error = %v", err)
	}
	seen := make(map[string]bool)
	for _, scope := range resourceMetrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name != "rpc.server.call.duration" && metric.Name != "rpc.server.duration" {
				continue
			}
			histogram, ok := metric.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("%s data type = %T, want float64 histogram", metric.Name, metric.Data)
			}
			for _, point := range histogram.DataPoints {
				for _, attr := range point.Attributes.ToSlice() {
					assertStringDoesNotContain(
						t,
						"metric attribute "+string(attr.Key),
						attr.Value.String(),
						secret,
					)
				}
				value, ok := point.Attributes.Value(attribute.Key("rpc.method"))
				if !ok {
					t.Fatalf("%s data point has no rpc.method: %v", metric.Name, point.Attributes)
				}
				method := value.AsString()
				if !allowedMethods[method] {
					t.Fatalf("rpc.method metric attribute = %q, want registered method from %v", method, allowedMethods)
				}
				seen[method] = true
			}
		}
	}
	for method := range allowedMethods {
		if !seen[method] {
			t.Fatalf("missing server duration metric for registered method %q", method)
		}
	}
}

func assertStringDoesNotContain(t *testing.T, sink, value, secret string) {
	t.Helper()

	if strings.Contains(value, secret) {
		t.Fatalf("%s discloses secret canary: %q", sink, value)
	}
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func assertRPCMetrics(t *testing.T, reader *sdkmetric.ManualReader) {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &resourceMetrics); err != nil {
		t.Fatalf("ManualReader.Collect() error = %v", err)
	}
	for _, prefix := range []string{"rpc.client.", "rpc.server."} {
		points := 0
		for _, scope := range resourceMetrics.ScopeMetrics {
			for _, metric := range scope.Metrics {
				if metric.Name != prefix+"call.duration" && metric.Name != prefix+"duration" {
					continue
				}
				histogram, ok := metric.Data.(metricdata.Histogram[float64])
				if ok {
					points += len(histogram.DataPoints)
				}
			}
		}
		if points < 2 {
			t.Fatalf("%s duration metric data points = %d, want unary and streaming series", prefix, points)
		}
	}
}

func spanNames(spans []sdktrace.ReadOnlySpan) []string {
	names := make([]string, 0, len(spans))
	for _, span := range spans {
		names = append(names, span.Name())
	}
	return names
}
