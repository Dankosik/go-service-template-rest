// Connection lifetime: the liveness bounds cannot end work in progress, and
// rotation can.
//
// That split is the whole reason the liveness bounds ship on and rotation ships
// off, so both halves are driven here under shortened bounds. The values prove
// the mechanism; the shipped ones are minutes long and would prove the same
// thing far more slowly.

package grpcx

import (
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const keepaliveStreamFullMethod = "/grpcx.test.KeepaliveService/Hold"

func TestLivenessBoundsDoNotEndAnRPCInProgress(t *testing.T) {
	t.Parallel()
	cfg := testServerConfig()
	// Every liveness clock is set to fire several times over while one stream is
	// outstanding. None of them may end it: the idle clock only runs when
	// nothing is outstanding, and the ping bound closes only on a ping the peer
	// did not answer.
	cfg.MaxConnectionIdle = 100 * time.Millisecond
	cfg.ServerPingInterval = 50 * time.Millisecond
	cfg.ServerPingTimeout = 50 * time.Millisecond
	cfg.MinClientPingInterval = time.Millisecond

	const held = 500 * time.Millisecond
	register := func(registrar grpc.ServiceRegistrar) {
		registerStreamTestService(registrar, keepaliveStreamFullMethod,
			func(stream grpc.ServerStream) error {
				var request emptypb.Empty
				if err := stream.RecvMsg(&request); err != nil {
					return fmt.Errorf("receive keepalive stream request: %w", err)
				}
				select {
				case <-time.After(held):
				case <-stream.Context().Done():
					return stream.Context().Err()
				}
				return stream.SendMsg(&emptypb.Empty{})
			})
	}
	_, connection := startTestServer(t, cfg, register)

	if err := driveKeepaliveStream(t, connection); err != nil {
		t.Fatalf("a stream outstanding across every liveness bound ended with %v", err)
	}
}

func TestRotationEndsAnRPCInProgress(t *testing.T) {
	t.Parallel()
	cfg := testServerConfig()
	// Rotation is the one bound that cuts live work, which is what makes it the
	// one this repository ships disabled. The grace clears the unary timeout,
	// which testServerConfig leaves at zero.
	cfg.MaxConnectionAge = 200 * time.Millisecond
	cfg.MaxConnectionAgeGrace = 100 * time.Millisecond

	register := func(registrar grpc.ServiceRegistrar) {
		registerStreamTestService(registrar, keepaliveStreamFullMethod,
			func(stream grpc.ServerStream) error {
				var request emptypb.Empty
				if err := stream.RecvMsg(&request); err != nil {
					return fmt.Errorf("receive keepalive stream request: %w", err)
				}
				<-stream.Context().Done()
				return stream.Context().Err()
			})
	}
	_, connection := startTestServer(t, cfg, register)

	err := driveKeepaliveStream(t, connection)
	if err == nil {
		t.Fatal("stream outlived a connection whose rotation age had passed")
	}
	// The caller sees the transport ending, not a deadline: rotation is not a
	// budget the stream was given.
	if code := grpcstatus.Code(err); code != codes.Unavailable {
		t.Fatalf("rotated stream status = %s, want %s (error %v)", code, codes.Unavailable, err)
	}
}

func driveKeepaliveStream(t *testing.T, connection *grpc.ClientConn) error {
	t.Helper()

	stream, err := connection.NewStream(
		t.Context(),
		&grpc.StreamDesc{ServerStreams: true},
		keepaliveStreamFullMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("ClientStream.CloseSend() error = %v", err)
	}

	//nolint:wrapcheck // The status this returns is exactly what the caller asserts on.
	return stream.RecvMsg(&emptypb.Empty{})
}
