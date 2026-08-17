// Does a client reach every address its target resolves to?
//
// grpc-go's own default sends every RPC to the first address that connects, so
// without a selected policy a headless DNS name behind this client is one
// backend wearing N addresses. Both policies are driven here, because the
// default is only meaningful against the alternative.

package grpcclient_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
	"google.golang.org/protobuf/types/known/emptypb"
)

const loadBalancingProbeMethod = "/grpcclient.test.Backend/Probe"

// This test is deliberately sequential. resolver.Register writes the
// package-global registry that resolver.Get reads on every grpcclient.New, and
// this package's parallel tests reach that read; a top-level sequential test is
// ordered before all of them. It registers an additional scheme rather than
// relocating the default one, so it needs none of the child-process isolation
// resolver_selection_test.go uses.
func TestLoadBalancingPolicyDecidesHowManyBackendsAreReached(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name            string
		scheme          string
		policy          grpcclient.LoadBalancingPolicy
		wantBackends    int
		probesToAttempt int
	}{
		{
			name:            "round robin reaches both",
			scheme:          "grpcclientlbrr",
			policy:          grpcclient.LoadBalancingRoundRobin,
			wantBackends:    2,
			probesToAttempt: 8,
		},
		{
			name:            "first-address selection reaches one",
			scheme:          "grpcclientlbpf",
			policy:          grpcclient.LoadBalancingPickFirst,
			wantBackends:    1,
			probesToAttempt: 8,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var first, second atomic.Int64
			addresses := []resolver.Address{
				{Addr: startCountingBackend(t, &first)},
				{Addr: startCountingBackend(t, &second)},
			}

			builder := manual.NewBuilderWithScheme(testCase.scheme)
			resolver.Register(builder)
			builder.InitialState(resolver.State{Addresses: addresses})

			cfg := grpcclient.DefaultConfig(testCase.scheme + ":///backends")
			cfg.LoadBalancing = testCase.policy
			cfg.HealthCheck = false
			connection, err := grpcclient.New(cfg, grpcclient.Options{
				TransportCredentials: insecure.NewCredentials(),
			})
			if err != nil {
				t.Fatalf("grpcclient.New() error = %v", err)
			}
			t.Cleanup(func() {
				if err := connection.Close(); err != nil {
					t.Errorf("ClientConn.Close() error = %v", err)
				}
			})
			connection.Connect()

			for range testCase.probesToAttempt {
				if err := connection.Invoke(
					t.Context(),
					loadBalancingProbeMethod,
					&emptypb.Empty{},
					&emptypb.Empty{},
				); err != nil {
					t.Fatalf("ClientConn.Invoke() error = %v", err)
				}
			}

			reached := 0
			for _, counter := range []*atomic.Int64{&first, &second} {
				if counter.Load() > 0 {
					reached++
				}
			}
			if reached != testCase.wantBackends {
				t.Fatalf(
					"%d of 2 backends served an RPC (%d and %d), want %d",
					reached,
					first.Load(),
					second.Load(),
					testCase.wantBackends,
				)
			}
		})
	}
}

func TestHealthConfigurationCallability(t *testing.T) {
	t.Parallel()
	backend := startHealthBackend(
		t,
		healthgrpc.HealthCheckResponse_NOT_SERVING,
		nil,
	)

	for _, testCase := range []struct {
		name        string
		policy      grpcclient.LoadBalancingPolicy
		healthCheck bool
	}{
		{name: "round robin with health disabled", healthCheck: false},
		{
			name:        "pick first with health disabled",
			policy:      grpcclient.LoadBalancingPickFirst,
			healthCheck: false,
		},
		{
			name:        "direct pick first with health configured",
			policy:      grpcclient.LoadBalancingPickFirst,
			healthCheck: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := grpcclient.DefaultConfig("passthrough:///" + backend.address)
			cfg.LoadBalancing = testCase.policy
			cfg.HealthCheck = testCase.healthCheck
			connection, err := grpcclient.New(cfg, grpcclient.Options{
				TransportCredentials: insecure.NewCredentials(),
			})
			if err != nil {
				t.Fatalf("grpcclient.New() error = %v", err)
			}
			t.Cleanup(func() { _ = connection.Close() })

			before := backend.calls.Load()
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			err = invokeHealthProbe(ctx, connection)
			cancel()
			if err != nil {
				t.Fatalf("ClientConn.Invoke() error = %v", err)
			}
			if got := backend.calls.Load() - before; got != 1 {
				t.Fatalf("backend calls = %d, want 1", got)
			}
		})
	}
}

// startCountingBackend serves one loopback peer that answers any method and
// counts what it served, and returns its bare address for a resolver to hand
// the balancer.
func startCountingBackend(t *testing.T, served *atomic.Int64) string {
	t.Helper()

	handler := func(_ any, stream grpc.ServerStream) error {
		var request emptypb.Empty
		if err := stream.RecvMsg(&request); err != nil {
			return err //nolint:wrapcheck // The client asserts on its own call, not on this.
		}
		served.Add(1)
		return stream.SendMsg(&emptypb.Empty{})
	}

	return serveTestServer(
		t,
		listenLoopback(t),
		func(*grpc.Server) {},
		grpc.UnknownServiceHandler(handler),
	)
}
