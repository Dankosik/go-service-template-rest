package grpcclient_test

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

func TestGeneratedClientUsesSharedInstrumentedConnection(t *testing.T) {
	received, target := startMetadataCaptureServer(t)
	recorder, tracerProvider := telemetrytest.NewRecordingTracerProvider(t)
	connection, err := grpcclient.New(
		target,
		grpcclient.Options{
			TransportCredentials: insecure.NewCredentials(),
			TracerProvider:       tracerProvider,
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer retained"))
	ctx, span := tracerProvider.Tracer("grpcclient-test").Start(ctx, "parent")
	defer span.End()
	if _, err := healthgrpc.NewHealthClient(connection).Check(ctx, &healthgrpc.HealthCheckRequest{}); err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}
	incoming := <-received
	if got := incoming.Get("authorization"); len(got) != 1 || got[0] != "Bearer retained" {
		t.Fatalf("authorization metadata = %v", got)
	}
	if got := incoming.Get("traceparent"); len(got) != 1 {
		t.Fatalf("traceparent metadata = %v", got)
	}
	if got := len(recorder.Ended()); got == 0 {
		t.Fatal("client span was not recorded")
	}
}
