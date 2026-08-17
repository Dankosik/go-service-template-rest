// The RPC time budget through a real server: does a handler that respects
// cancellation get cut at the configured bound, does a caller's own earlier
// deadline still win, and does the admission slot the RPC held come back?
//
// interceptors_test.go drives deadlineAround directly, so a failure there names
// the rule that broke. These cases need the whole chain, because slot release
// and the replacement context a streaming handler observes are only visible once
// it has run.

package grpcx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// The streaming method this file drives. Each file registering a stream names
// its own, for the reason correlation_service_test.go records.
const deadlineStreamFullMethod = "/grpcx.test.DeadlineService/Wait"

func TestUnaryDeadlineCutsTheHandlerAndReleasesItsSlot(t *testing.T) {
	t.Parallel()
	const budget = 150 * time.Millisecond

	cfg := testServerConfig()
	cfg.UnaryTimeout = budget
	// One slot, so a slot that did not come back answers the probe below with
	// ResourceExhausted instead of OK.
	cfg.MaxConcurrentRPCs = 1

	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			})
		registerUnaryTestService(registrar, testPayloadFullMethod,
			func(context.Context, *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error) {
				return &wrapperspb.BytesValue{}, nil
			})
	}
	_, connection := startTestServer(t, cfg, register)

	// No caller deadline: the server's own bound is the only thing that can end
	// this RPC, which is the case the bound exists for.
	started := time.Now()
	err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	assertStatusCode(t, err, codes.DeadlineExceeded)
	if elapsed := time.Since(started); elapsed < budget {
		t.Fatalf("RPC ended after %s, before its %s budget", elapsed, budget)
	}

	if err := connection.Invoke(
		t.Context(),
		testPayloadFullMethod,
		&wrapperspb.BytesValue{},
		&wrapperspb.BytesValue{},
	); err != nil {
		t.Fatalf("RPC after the budget expired: %v", err)
	}
}

func TestCallerDeadlineEarlierThanTheBudgetWins(t *testing.T) {
	t.Parallel()
	const callerBudget = 200 * time.Millisecond

	cfg := testServerConfig()
	// Far enough out that a handler seeing anything near it proves the cap
	// replaced the caller's deadline rather than bounding it.
	cfg.UnaryTimeout = time.Hour

	observed := make(chan time.Duration, 1)
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
				deadline, ok := ctx.Deadline()
				if !ok {
					observed <- 0
					return &emptypb.Empty{}, nil
				}
				observed <- time.Until(deadline)
				return &emptypb.Empty{}, nil
			})
	}
	_, connection := startTestServer(t, cfg, register)

	ctx, cancel := context.WithTimeout(t.Context(), callerBudget)
	defer cancel()
	if err := connection.Invoke(ctx, testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatalf("ClientConn.Invoke() error = %v", err)
	}

	remaining := <-observed
	if remaining <= 0 || remaining > callerBudget {
		t.Fatalf("handler deadline was %s away, want at most the caller's %s", remaining, callerBudget)
	}
}

func TestStreamDeadlineIsOffByDefaultAndCutsWhenConfigured(t *testing.T) {
	t.Parallel()
	t.Run("off by default", func(t *testing.T) {
		t.Parallel()
		cfg := testServerConfig()
		// The unary bound is short and the stream bound is the shipped zero, so
		// a stream cut here would mean the two chains share one value.
		cfg.UnaryTimeout = 100 * time.Millisecond

		register := func(registrar grpc.ServiceRegistrar) {
			registerStreamTestService(registrar, deadlineStreamFullMethod,
				func(stream grpc.ServerStream) error {
					var request emptypb.Empty
					if err := stream.RecvMsg(&request); err != nil {
						return fmt.Errorf("receive deadline stream request: %w", err)
					}
					select {
					case <-time.After(4 * cfg.UnaryTimeout):
					case <-stream.Context().Done():
						return stream.Context().Err()
					}
					return stream.SendMsg(&emptypb.Empty{})
				})
		}
		_, connection := startTestServer(t, cfg, register)

		if err := driveDeadlineStream(t, connection, &emptypb.Empty{}); err != nil {
			t.Fatalf("stream outliving the unary budget ended with %v", err)
		}
	})

	t.Run("configured", func(t *testing.T) {
		t.Parallel()
		cfg := testServerConfig()
		cfg.StreamTimeout = 150 * time.Millisecond

		register := func(registrar grpc.ServiceRegistrar) {
			registerStreamTestService(registrar, deadlineStreamFullMethod,
				func(stream grpc.ServerStream) error {
					var request emptypb.Empty
					if err := stream.RecvMsg(&request); err != nil {
						return fmt.Errorf("receive deadline stream request: %w", err)
					}
					<-stream.Context().Done()
					return stream.Context().Err()
				})
		}
		_, connection := startTestServer(t, cfg, register)

		assertStatusCode(t, driveDeadlineStream(t, connection, &emptypb.Empty{}), codes.DeadlineExceeded)
	})
}

// driveDeadlineStream opens the stream, sends one message, and returns the
// status the caller receives. The caller sets no deadline of its own, so
// whatever ends the stream came from the server.
func driveDeadlineStream(t *testing.T, connection *grpc.ClientConn, request *emptypb.Empty) error {
	t.Helper()

	stream, err := connection.NewStream(
		t.Context(),
		&grpc.StreamDesc{ServerStreams: true},
		deadlineStreamFullMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(request); err != nil {
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("ClientStream.CloseSend() error = %v", err)
	}

	//nolint:wrapcheck // The status this returns is exactly what the caller asserts on.
	return stream.RecvMsg(&emptypb.Empty{})
}
