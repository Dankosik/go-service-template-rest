// Shared test harness for package grpcclient's external tests: the two ways to
// stand a peer up, the TLS pair three proofs verify against, the one way to
// re-run this binary as a child process, and the resolver stub both resolver
// proofs need.
//
// A test needing a server to dial composes these rather than hand-rolling another
// listen/serve/teardown block. Most reach for startMetadataCaptureServer, since
// what this package guards is which metadata crosses the boundary;
// startTestServer is the plain half. The peers that are not gRPC servers — the
// raw HTTP/2 peer in transparent_retry_test.go and the proxy in
// resolver_live_test.go — share only listenLoopback, because what they serve is
// the thing under proof.
//
// The white-box tests in propagation_internal_test.go and
// resolver_internal_test.go are package grpcclient and cannot reach this file.

package grpcclient_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

// listenLoopback opens a loopback listener on an unused port.
func listenLoopback(t *testing.T) net.Listener {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	return listener
}

// startTestServer serves a gRPC server on loopback until the test ends and
// returns the passthrough target grpcclient.New should dial.
//
// register runs before Serve; serverOptions reach grpc.NewServer unchanged, so a
// test supplying credentials or its own interceptors passes them here.
func startTestServer(
	t *testing.T,
	register func(*grpc.Server),
	serverOptions ...grpc.ServerOption,
) string {
	t.Helper()

	return "passthrough:///" + serveTestServer(t, listenLoopback(t), register, serverOptions...)
}

// serveTestServer is startTestServer's half for a caller that owns its listener,
// and registers this package's standard teardown: stop the server, then drain
// Serve's result. It returns the listener address.
//
// Only the live-resolver proof needs this directly: it must bind a non-loopback
// address and dial the bare one.
func serveTestServer(
	t *testing.T,
	listener net.Listener,
	register func(*grpc.Server),
	serverOptions ...grpc.ServerOption,
) string {
	t.Helper()

	server := grpc.NewServer(serverOptions...)
	register(server)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		if err := <-serveDone; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})
	return listener.Addr().String()
}

// registerServingHealth registers the standard health service reporting SERVING,
// which is what a test that only needs some RPC to succeed should dial.
func registerServingHealth(server *grpc.Server) {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(server, healthServer)
}

// startMetadataCaptureServer serves a peer that records the metadata each RPC
// arrived with, and returns the unary channel, the streaming channel, and the
// target to dial. Each channel is buffered for one RPC, so a test drives one
// call per kind and reads what crossed the boundary.
//
// This is the peer most tests here want: what this package guards is which
// metadata reaches the wire, and that is only observable from the receiving
// end. serverOptions reach grpc.NewServer ahead of the capture interceptors, so
// a test supplying credentials passes them here.
func startMetadataCaptureServer(
	t *testing.T,
	serverOptions ...grpc.ServerOption,
) (<-chan metadata.MD, <-chan metadata.MD, string) {
	t.Helper()

	unaryMetadata := make(chan metadata.MD, 1)
	streamMetadata := make(chan metadata.MD, 1)
	serverOptions = append(serverOptions,
		grpc.UnaryInterceptor(func(
			ctx context.Context,
			request any,
			_ *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (any, error) {
			incoming, _ := metadata.FromIncomingContext(ctx)
			unaryMetadata <- incoming.Copy()
			return handler(ctx, request)
		}),
		grpc.StreamInterceptor(func(
			service any,
			stream grpc.ServerStream,
			_ *grpc.StreamServerInfo,
			handler grpc.StreamHandler,
		) error {
			incoming, _ := metadata.FromIncomingContext(stream.Context())
			streamMetadata <- incoming.Copy()
			return handler(service, stream)
		}),
	)
	target := startTestServer(t, registerServingHealth, serverOptions...)

	return unaryMetadata, streamMetadata, target
}

// runTestBinaryChild re-runs this test binary for one child test and returns its
// combined output, failing the parent if the child does.
//
// Two proofs need a child process, both because their answer depends on
// process-global state a test cannot mutate without changing every other test in
// the binary: the resolver registry and resolver.GetDefaultScheme() in
// resolver_selection_test.go, and the proxy environment in resolver_live_test.go.
// Each entry in env is a "KEY=value" addition to the child's environment.
func runTestBinaryChild(t *testing.T, testName string, env ...string) []byte {
	t.Helper()

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	command := exec.CommandContext(
		t.Context(),
		executable,
		"-test.run=^"+testName+"$",
		"-test.count=1",
	)
	command.Env = append(os.Environ(), env...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("child test %s failed: %v\n%s", testName, err, output)
	}
	return output
}

// testTLSMaterial returns a server certificate and the leaf a client must trust
// to verify it.
//
// It borrows httptest's self-signed pair rather than generating one, which is
// what keeps three TLS proofs here free of certificate construction. That pair
// is issued for example.com and 127.0.0.1, so a client verifying it passes
// ServerName "example.com"; the server is closed immediately because only its
// certificate is wanted.
func testTLSMaterial(t *testing.T) (tls.Certificate, *x509.Certificate) {
	t.Helper()

	source := httptest.NewTLSServer(http.NotFoundHandler())
	defer source.Close()

	return source.TLS.Certificates[0], source.Certificate()
}

// testTLSCredentials wraps [testTLSMaterial] as the server and client halves of
// one trusted pair, for a proof that only needs a connection to succeed.
func testTLSCredentials( //nolint:ireturn // grpc-go exposes transport credentials as an interface.
	t *testing.T,
) (credentials.TransportCredentials, credentials.TransportCredentials) {
	t.Helper()

	serverCertificate, leafCertificate := testTLSMaterial(t)
	roots := x509.NewCertPool()
	roots.AddCert(leafCertificate)
	return credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{serverCertificate},
			MinVersion:   tls.VersionTLS12,
		}), credentials.NewTLS(&tls.Config{
			RootCAs:    roots,
			ServerName: "example.com",
			MinVersion: tls.VersionTLS12,
		})
}

// nopResolver satisfies resolver.Resolver for a builder that publishes its state
// once during Build and has nothing to do afterwards.
type nopResolver struct{}

func (nopResolver) ResolveNow(resolver.ResolveNowOptions) {}
func (nopResolver) Close()                                {}
