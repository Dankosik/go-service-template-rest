// The admission budgets through a server built by NewServer: does filling one
// from one RPC kind shed what the other owns?
//
// interceptors_test.go proves one policy value serves both interceptor types and
// routes each method to the budget that owns it. Those are different questions.
// NewServer builds the two policy lists separately and sizes the two budgets from
// separate configuration, so whether the server actually hands both chains the
// same policy, and whether it gave the health service a budget of its own, are
// its composition rather than structural guarantees. Only a real server can show
// either. Without this file, passing one limiter to the unary chain and a second
// to the streaming chain, or sizing the health budget from MaxConcurrentRPCs,
// would compile, lint, and pass every other test.

package grpcx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

const admissionStreamFullMethod = "/grpcx.test.AdmissionService/Hold"

func TestAdmissionBudgetIsProcessWide(t *testing.T) {
	cfg := testServerConfig()
	// One slot for the whole process, so a second budget is the only way both
	// RPCs below can be admitted.
	cfg.MaxConcurrentRPCs = 1

	occupied := make(chan struct{})
	release := make(chan struct{})
	register := func(registrar grpc.ServiceRegistrar) {
		registerStreamTestService(registrar, admissionStreamFullMethod,
			func(stream grpc.ServerStream) error {
				var request emptypb.Empty
				if err := stream.RecvMsg(&request); err != nil {
					return fmt.Errorf("receive admission stream request: %w", err)
				}
				close(occupied)
				<-release
				return nil
			})
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, nil
			})
	}
	_, connection := startTestServer(t, cfg, register)
	// Registered after the harness's teardown so it runs before it: the handler
	// must leave its slot before the server is closed, or the test leaks the
	// goroutine it is holding.
	t.Cleanup(func() { close(release) })

	stream, err := connection.NewStream(
		t.Context(),
		&grpc.StreamDesc{ServerStreams: true},
		admissionStreamFullMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	select {
	case <-occupied:
	case <-time.After(5 * time.Second):
		t.Fatal("streaming handler never reached the admission slot")
	}

	// The unary chain answers from the same budget, or it has one of its own.
	err = connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	assertStatusCode(t, err, codes.ResourceExhausted)
}

// TestHealthServiceSurvivesBusinessSaturation drives the consequence the separate
// health budget exists for.
//
// grpc-go's client-side health checker holds one Health/Watch open per subchannel
// for the connection's whole life, and this repository's own client enables it by
// default. Shedding that watch does not cost the caller one RPC: its balancer
// stops selecting the backend entirely, so business saturation on one replica
// would remove it from every health-checking caller at once. Check answering is
// the second half — a platform that cannot probe a saturated instance restarts a
// healthy one.
func TestHealthServiceSurvivesBusinessSaturation(t *testing.T) {
	cfg := testServerConfig()
	// One business slot, held for the whole test by the stream below.
	cfg.MaxConcurrentRPCs = 1

	occupied := make(chan struct{})
	release := make(chan struct{})
	register := func(registrar grpc.ServiceRegistrar) {
		registerStreamTestService(registrar, admissionStreamFullMethod,
			func(stream grpc.ServerStream) error {
				var request emptypb.Empty
				if err := stream.RecvMsg(&request); err != nil {
					return fmt.Errorf("receive admission stream request: %w", err)
				}
				close(occupied)
				<-release
				return nil
			})
	}
	_, connection := startTestServer(t, cfg, register)
	t.Cleanup(func() { close(release) })

	stream, err := connection.NewStream(
		t.Context(),
		&grpc.StreamDesc{ServerStreams: true},
		admissionStreamFullMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	select {
	case <-occupied:
	case <-time.After(5 * time.Second):
		t.Fatal("streaming handler never reached the admission slot")
	}

	health := healthgrpc.NewHealthClient(connection)
	if _, err := health.Check(t.Context(), &healthgrpc.HealthCheckRequest{}); err != nil {
		t.Fatalf("Health.Check() against a full business budget = %v", err)
	}

	// Canceled with the test, which is what ends the watch this opens.
	watchCtx, cancelWatch := context.WithCancel(t.Context())
	t.Cleanup(cancelWatch)
	watch, err := health.Watch(watchCtx, &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health.Watch() against a full business budget = %v", err)
	}
	// A shed watch fails here rather than at the call above: grpc-go returns the
	// stream before the server has answered.
	if _, err := watch.Recv(); err != nil {
		t.Fatalf("Health.Watch() first status against a full business budget = %v", err)
	}
}
