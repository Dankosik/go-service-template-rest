// Shared bufconn harness and service-descriptor helpers for package grpcx.

package grpcx

import (
	"context"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
func testServerConfig() serverConfig {
	cfg := defaultServerConfig()
	cfg.maxConcurrentRPCs = 4
	cfg.maxConcurrentHealthRPCs = 8
	cfg.maxConcurrentStreams = 4
	return cfg
}

func startTestServer(
	t *testing.T,
	cfg serverConfig,
	register RegisterService,
) (*Server, *grpc.ClientConn) {
	t.Helper()

	return startTestServerWithOptions(t, cfg, register, Options{})
}

func startTestServerWithOptions(
	t *testing.T,
	cfg serverConfig,
	register RegisterService,
	options Options,
) (*Server, *grpc.ClientConn) {
	t.Helper()

	services := append([]RegisterService(nil), options.Services...)
	if register != nil {
		services = append(services, register)
	}
	options.Services = services
	server, err := newServer(cfg, options)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	// The listener, the dialer, and the teardown order are grpctest.ServeBufconn's,
	// which this package's tests are meant to use rather than restate: *Server
	// already satisfies its BufconnServer interface, and its cleanup additionally
	// joins Serve on the client-setup failure path this one only abandoned.
	return server, grpctest.ServeBufconn(t, server)
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

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if got := status.Code(err); got != want {
		t.Fatalf("status code = %s, want %s (error %v)", got, want, err)
	}
}
