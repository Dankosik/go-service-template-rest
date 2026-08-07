// Shared test harness for package grpcx: the bounded default config, the two
// ways to stand a real server up, and the one way each to register a unary and a
// streaming service on it. A test needing an RPC through the full server composes
// these rather than hand-rolling another descriptor.
//
// Pick startTestServer for anything that only needs an RPC to reach a handler:
// it is bufconn-backed, owns teardown, and touches no port. Pick serveOverTCP
// when the client has to be grpcclient.New — it takes a target string and
// exposes no dialer, so it cannot reach a bufconn. Both carry every connection
// bound, so keepalive_test.go proves rotation over the bufconn path.

package grpcx

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

// The method names more than one test file drives. They live here rather than
// beside their first caller because a constant at the tail of a sibling file is
// one a reader has to go looking for; a name used by one file only stays in that
// file.
const (
	testUnaryFullMethod   = "/grpcx.test.Service/Call"
	testStreamFullMethod  = "/grpcx.test.Service/Stream"
	testPayloadFullMethod = "/grpcx.test.PayloadService/Call"
)

// testServerConfig is the bounded default every test starts from. Its liveness
// bounds are far enough out that none of them fires during a test, and rotation
// and both RPC timeouts are left off, so a test that cares about one sets it
// rather than inheriting a cut it did not ask for.
//
// The liveness values are not decoration: config_parity_test.go runs this
// fixture through validateConfig, which refuses a zero there.
func testServerConfig() Config {
	return Config{
		MaxConcurrentRPCs:          4,
		MaxConcurrentStreams:       4,
		MaxHeaderListBytes:         16 << 10,
		MaxReceiveMessageBytes:     4 << 20,
		MaxSendMessageBytes:        4 << 20,
		AccessLogSuccessSampleRate: 1,
		MaxConnectionIdle:          time.Minute,
		ServerPingInterval:         time.Minute,
		ServerPingTimeout:          20 * time.Second,
		MinClientPingInterval:      10 * time.Second,
		PermitPingWithoutStream:    true,
	}
}

func startTestServer(
	t *testing.T,
	cfg Config,
	register RegisterService,
) (*Server, *grpc.ClientConn) {
	t.Helper()

	return startTestServerWithOptions(t, cfg, register, Options{})
}

func startTestServerWithOptions(
	t *testing.T,
	cfg Config,
	register RegisterService,
	options Options,
) (*Server, *grpc.ClientConn) {
	t.Helper()

	services := append([]RegisterService(nil), options.Services...)
	if register != nil {
		services = append(services, register)
	}
	options.Services = services
	server, err := NewServer(cfg, options)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	listener := bufconn.Listen(1 << 20)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	connection, err := grpc.NewClient(
		"passthrough:///grpcx-test",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		if closeErr := server.Close(); closeErr != nil {
			t.Errorf("Server.Close() after client setup failure = %v", closeErr)
		}
		_ = listener.Close()
		t.Fatalf("grpc.NewClient() error = %v", err)
	}

	closeAfterTest(t, server, connection, serveDone)

	return server, connection
}

// closeAfterTest registers this package's standard teardown: close the client,
// stop the server, then drain Serve's result, in that order. A test that does
// not assert its own shutdown sequence wants exactly this and should not retype
// it.
func closeAfterTest(
	t *testing.T,
	server *Server,
	connection *grpc.ClientConn,
	serveDone <-chan error,
) {
	t.Helper()

	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
		}
		if err := server.Close(); err != nil {
			t.Errorf("Server.Close() error = %v", err)
		}
		if err := <-serveDone; err != nil {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})
}

// registerUnaryTestService registers one unary method named by fullMethod
// ("/package.Service/Method") and backed by call.
//
// Package grpcx has no generated service of its own, so a test wanting a real
// RPC through the full server needs a hand-written descriptor;
// internal/infra/grpc/grpctest owns that shim for the three packages in this
// position. This wrapper exists so the call sites here stay one line each.
func registerUnaryTestService[Request any, RequestPtr interface {
	*Request
	proto.Message
}](
	registrar grpc.ServiceRegistrar,
	fullMethod string,
	call func(context.Context, RequestPtr) (RequestPtr, error),
) {
	grpctest.Register(registrar, grpctest.Unary[Request, RequestPtr](fullMethod, call))
}

// registerStreamTestService registers one streaming method named by fullMethod
// and backed by call, for the same reason registerUnaryTestService exists.
//
// Every caller supplies its own handler and its own method name: the handlers
// close over the channels their test observes, so what is shared here is the
// registration, not the service.
func registerStreamTestService(
	registrar grpc.ServiceRegistrar,
	fullMethod string,
	call func(grpc.ServerStream) error,
) {
	grpctest.Register(registrar, grpctest.ServerStream(fullMethod, call))
}

// serveOverTCP serves an already-built server on a loopback listener and dials
// it through grpcclient.New, returning the connection and the channel carrying
// Serve's result.
//
// Teardown stays with the caller because one call site needs it to:
// correlation_service_test.go consumes serveDone itself while asserting the
// shutdown sequence, so a helper owning cleanup unconditionally would race it for
// that channel. Any other caller passes the three values to closeAfterTest.
//
// clientOptions.TransportCredentials defaults to insecure because the listener
// is loopback; a test proving transport trust passes its own.
func serveOverTCP(
	t *testing.T,
	server *Server,
	clientOptions grpcclient.Options,
) (*grpc.ClientConn, <-chan error) {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	if clientOptions.TransportCredentials == nil {
		clientOptions.TransportCredentials = insecure.NewCredentials()
	}
	connection, err := grpcclient.New(
		grpcclient.DefaultConfig("passthrough:///"+listener.Addr().String()),
		clientOptions,
	)
	if err != nil {
		if closeErr := server.Close(); closeErr != nil {
			t.Errorf("Server.Close() after client setup failure = %v", closeErr)
		}
		t.Fatalf("grpcclient.New() error = %v", err)
	}

	return connection, serveDone
}

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if got := status.Code(err); got != want {
		t.Fatalf("status code = %s, want %s (error %v)", got, want, err)
	}
}
