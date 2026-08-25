package grpcx

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServerTelemetryUsesRegisteredBusinessMethodsOnly(t *testing.T) {
	recorder, provider := telemetrytest.NewRecordingTracerProvider(t)
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, nil
			})
	}
	server, connection := startTestServerWithOptions(t, testServerConfig(), register, Options{
		TracerProvider: provider,
	})
	server.SetServing(true)

	if err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatalf("business Invoke() error = %v", err)
	}
	if _, err := healthgrpc.NewHealthClient(connection).Check(
		t.Context(),
		&healthgrpc.HealthCheckRequest{},
	); err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}
	_ = connection.Invoke(t.Context(), "/peer.controlled.Unknown/Method", &emptypb.Empty{}, &emptypb.Empty{})

	serverSpans := serverSpanNames(recorder)
	if !serverSpans[testUnaryFullMethod[1:]] {
		t.Fatalf("business server span missing: %v", serverSpans)
	}
	if serverSpans["grpc.health.v1.Health/Check"] || serverSpans["peer.controlled.Unknown/Method"] {
		t.Fatalf("health or unknown method reached server telemetry: %v", serverSpans)
	}
}

func TestUnhandledFailureLogOmitsHandlerText(t *testing.T) {
	const secret = "credential=grpc-handler-secret"

	var output bytes.Buffer
	_, provider := telemetrytest.NewRecordingTracerProvider(t)
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return nil, fmt.Errorf("load dependency: %w", errors.New(secret))
			})
	}
	_, connection := startTestServerWithOptions(t, testServerConfig(), register, Options{
		Logger:         logctx.NewProcessLogger(&output, slog.LevelError),
		TracerProvider: provider,
	})

	err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != failure.SanitizedDetail {
		t.Fatalf("status detail = %q, want %q", detail, failure.SanitizedDetail)
	}
	if bytes.Contains(output.Bytes(), []byte(secret)) {
		t.Fatalf("unhandled failure log disclosed handler text: %q", output.String())
	}

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode unhandled failure log %q: %v", output.String(), err)
	}
	for key, want := range map[string]string{
		"msg":         "grpc_unhandled_failure",
		"rpc.method":  testUnaryFullMethod,
		"error_chain": "*fmt.wrapError -> *errors.errorString",
	} {
		if got := record[key]; got != want {
			t.Fatalf("%s = %v, want %q", key, got, want)
		}
	}
	for _, key := range []string{"trace_id", "span_id"} {
		if got, _ := record[key].(string); got == "" {
			t.Fatalf("%s = %v, want correlation", key, record[key])
		}
	}
}

func serverSpanNames(recorder *tracetest.SpanRecorder) map[string]bool {
	names := make(map[string]bool)
	for _, span := range recorder.Ended() {
		if span.SpanKind() == trace.SpanKindServer {
			names[span.Name()] = true
		}
	}
	return names
}
