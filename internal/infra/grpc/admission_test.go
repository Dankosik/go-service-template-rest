// The process-wide admission budget through a server built by NewServer: does
// filling it from one RPC kind shed the other?
//
// interceptors_test.go proves one limiter value serves both interceptor types.
// That is a different question. NewServer builds the two policy lists
// separately, so whether the server hands both the same limiter is its
// composition rather than a structural guarantee, and only a real server can
// show it. Without this file, passing one limiter to the unary chain and a
// second to the streaming chain would compile, lint, and pass every other test
// while quietly doubling the budget.

package grpcx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
