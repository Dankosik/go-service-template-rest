package grpcx

import (
	"testing"

	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

func TestAuthnHealthState(t *testing.T) {
	server, connection := startTestServer(t, testServerConfig(), nil)
	healthClient := healthgrpc.NewHealthClient(connection)

	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_NOT_SERVING)
	server.SetAuthnReady(false)
	server.MarkServing()
	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_NOT_SERVING)

	server.SetAuthnReady(true)
	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_SERVING)
	server.SetAuthnReady(false)
	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_NOT_SERVING)
	server.SetAuthnReady(true)
	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_SERVING)

	server.StartDrain()
	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_NOT_SERVING)
	server.SetAuthnReady(true)
	server.MarkServing()
	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_NOT_SERVING)

	if err := server.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
