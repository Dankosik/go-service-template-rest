// Health composition under the authn-oidc-jwt profile: does SetAuthnReady
// combine with startup admission and terminal drain the way publishHealthLocked
// promises?
//
// This whole file is removed when a service is generated with AUTHN=none —
// scripts/init-module.sh deletes it by name, the convention every authn_*.go file
// here follows — while the other health tests live in server_test.go because they
// survive that profile. An assertion touching SetAuthnReady therefore belongs
// here: in server_test.go it compiles and passes, then breaks the generated tree,
// where only template-init-check would catch it.

package grpcx

import (
	"testing"

	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

func TestAuthnHealthState(t *testing.T) {
	t.Parallel()
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
