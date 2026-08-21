package grpcx

import (
	"context"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
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

func serverSpanNames(recorder *tracetest.SpanRecorder) map[string]bool {
	names := make(map[string]bool)
	for _, span := range recorder.Ended() {
		if span.SpanKind() == trace.SpanKindServer {
			names[span.Name()] = true
		}
	}
	return names
}
