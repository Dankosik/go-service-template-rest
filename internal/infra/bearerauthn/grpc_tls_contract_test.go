package bearerauthn

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	tlsAuthnUnaryMethod  = "/bearerauthn.test.Authn/Unary"
	tlsAuthnStreamMethod = "/bearerauthn.test.Authn/Watch"
)

func TestGRPCAuthnBoundaryOverTLS(t *testing.T) {
	serverTLS, clientTLS := testGRPCTLSConfigs(t)
	verifier := &fakeVerifier{
		result: Result{
			Principal: reqctx.Principal{Subject: "subject-1"},
			ExpiresAt: time.Unix(1_900_003_600, 0),
		},
	}
	runtime := newTestRuntime(t, verifier)
	var unaryCalls atomic.Int64
	var streamCalls atomic.Int64
	connection := startTLSAuthnServer(t, serverTLS, clientTLS, runtime, &unaryCalls, &streamCalls)
	healthClient := healthpb.NewHealthClient(connection)

	if _, err := healthClient.Check(t.Context(), &healthpb.HealthCheckRequest{}); err != nil {
		t.Fatalf("public TLS health check error = %v", err)
	}
	watch, err := healthClient.Watch(t.Context(), &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("start unauthenticated TLS health Watch: %v", err)
	}
	if err := watch.RecvMsg(&healthpb.HealthCheckResponse{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unauthenticated TLS health Watch status = %v, want Unauthenticated", status.Code(err))
	}

	credentialCtx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer token"))
	if err := connection.Invoke(credentialCtx, tlsAuthnUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatalf("authenticated TLS unary call error = %v", err)
	}
	stream, err := connection.NewStream(
		credentialCtx,
		&grpc.StreamDesc{ServerStreams: true},
		tlsAuthnStreamMethod,
	)
	if err != nil {
		t.Fatalf("new authenticated TLS stream: %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("send TLS stream request: %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("close TLS stream send: %v", err)
	}
	if err := stream.RecvMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("receive TLS stream response: %v", err)
	}
	if unaryCalls.Load() != 1 || streamCalls.Load() != 1 {
		t.Fatalf("application calls = (unary %d, stream %d), want one authenticated call each", unaryCalls.Load(), streamCalls.Load())
	}

	for _, testCase := range []struct {
		name   string
		md     metadata.MD
		err    error
		code   codes.Code
		detail string
	}{
		{name: "missing", code: codes.Unauthenticated, detail: "authentication credential is missing or invalid"},
		{
			name:   "duplicate",
			md:     metadata.Pairs("authorization", "Bearer one", "authorization", "Bearer two"),
			code:   codes.Unauthenticated,
			detail: "authentication credential is missing or invalid",
		},
		{
			name:   "malformed",
			md:     metadata.Pairs("authorization", "Basic token"),
			code:   codes.Unauthenticated,
			detail: "authentication credential is missing or invalid",
		},
		{
			name:   "oversize",
			md:     metadata.Pairs("authorization", "Bearer "+strings.Repeat("x", MaxTokenBytes+1)),
			code:   codes.ResourceExhausted,
			detail: "authentication credential is too large",
		},
		{
			name:   "invalid",
			md:     metadata.Pairs("authorization", "Bearer token"),
			err:    NewError(KindInvalid),
			code:   codes.Unauthenticated,
			detail: "authentication credential is missing or invalid",
		},
		{
			name:   "unavailable",
			md:     metadata.Pairs("authorization", "Bearer token"),
			err:    NewError(KindUnavailable),
			code:   codes.Unavailable,
			detail: "authentication trust is unavailable",
		},
		{
			name:   "canceled",
			md:     metadata.Pairs("authorization", "Bearer token"),
			err:    context.Canceled,
			code:   codes.Canceled,
			detail: "authentication canceled",
		},
		{
			name:   "deadline",
			md:     metadata.Pairs("authorization", "Bearer token"),
			err:    context.DeadlineExceeded,
			code:   codes.DeadlineExceeded,
			detail: "authentication deadline exceeded",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			caseVerifier := &fakeVerifier{err: testCase.err}
			caseRuntime := newTestRuntime(t, caseVerifier)
			caseConn := startTLSAuthnServer(t, serverTLS, clientTLS, caseRuntime, new(atomic.Int64), new(atomic.Int64))
			ctx := t.Context()
			if testCase.md != nil {
				ctx = metadata.NewOutgoingContext(ctx, testCase.md)
			}
			err := caseConn.Invoke(ctx, tlsAuthnUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{})
			if status.Code(err) != testCase.code {
				t.Fatalf("status = %v, want %v (%v)", status.Code(err), testCase.code, err)
			}
			if status.Convert(err).Message() != testCase.detail {
				t.Fatalf("detail = %q, want %q", status.Convert(err).Message(), testCase.detail)
			}
			if strings.Contains(err.Error(), "poison") || strings.Contains(err.Error(), "parser") {
				t.Fatal("status leaked authentication error detail")
			}
		})
	}
}

func startTLSAuthnServer(
	t *testing.T,
	serverTLS, clientTLS *tls.Config,
	runtime *Runtime,
	unaryCalls, streamCalls *atomic.Int64,
) *grpc.ClientConn {
	t.Helper()
	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(serverTLS)),
		grpc.ChainUnaryInterceptor(runtime.UnaryInterceptor()),
		grpc.ChainStreamInterceptor(runtime.StreamInterceptor()),
	)
	registerTLSAuthnService(
		server,
		func(ctx context.Context) {
			unaryCalls.Add(1)
			assertAuthenticatedRPCContext(ctx, t)
		},
		func(ctx context.Context) {
			streamCalls.Add(1)
			assertAuthenticatedRPCContext(ctx, t)
		},
	)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for TLS gRPC test: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.GracefulStop()
		if serveErr := <-serveDone; serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			t.Errorf("TLS gRPC Serve() error = %v", serveErr)
		}
	})

	connection, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := connection.Close(); closeErr != nil {
			t.Errorf("close TLS gRPC connection: %v", closeErr)
		}
	})
	return connection
}

func assertAuthenticatedRPCContext(ctx context.Context, t *testing.T) {
	t.Helper()
	principal, ok := reqctx.PrincipalFromContext(ctx)
	if !ok || principal.Subject != "subject-1" {
		t.Fatalf("RPC principal = (%+v, %v), want subject-1", principal, ok)
	}
	incoming, _ := metadata.FromIncomingContext(ctx)
	if len(incoming.Get("authorization")) != 0 {
		t.Fatal("handler-visible authorization metadata was retained")
	}
}

func registerTLSAuthnService(
	registrar grpc.ServiceRegistrar,
	unary func(context.Context),
	stream func(context.Context),
) {
	grpctest.Register(
		registrar,
		grpctest.Unary(
			tlsAuthnUnaryMethod,
			func(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
				unary(ctx)
				return &emptypb.Empty{}, nil
			},
		),
		grpctest.ServerStream(tlsAuthnStreamMethod, func(serverStream grpc.ServerStream) error {
			var request emptypb.Empty
			if err := serverStream.RecvMsg(&request); err != nil {
				return fmt.Errorf("receive TLS authn test request: %w", err)
			}
			stream(serverStream.Context())
			if err := serverStream.SendMsg(&emptypb.Empty{}); err != nil {
				return fmt.Errorf("send TLS authn test response: %w", err)
			}
			return nil
		}),
	)
}

func testGRPCTLSConfigs(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate TLS test key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Unix(1_700_000_000, 0),
		NotAfter:     time.Unix(2_200_000_000, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create fixed TLS test certificate: %v", err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal TLS test private key: %v", err)
	}
	certificate, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}),
	)
	if err != nil {
		t.Fatalf("load TLS test key pair: %v", err)
	}
	parsed, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		t.Fatalf("parse TLS test certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(parsed)
	serverConfig := &tls.Config{ //nolint:gosec // Test-only TLS uses the repository's minimum through crypto/tls defaults.
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{certificate},
	}
	clientConfig := &tls.Config{ //nolint:gosec // The test verifies a fixed self-signed server authority.
		MinVersion: tls.VersionTLS13,
		RootCAs:    roots,
		ServerName: "localhost",
	}
	return serverConfig, clientConfig
}
