package oidcjwt

// Proof that grpc.go's interceptors hold the same boundary over a real TLS
// connection that grpc_test.go proves against them directly: a served RPC, a
// real credential, and the statuses a caller actually receives.

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
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	tlsAuthnUnaryMethod  = "/oidcjwt.test.Authn/Unary"
	tlsAuthnStreamMethod = "/oidcjwt.test.Authn/Watch"
)

func TestGRPCAuthnBoundaryOverTLS(t *testing.T) {
	now := testNow
	signingKey := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, signingKey)
	serverTLS, clientTLS := testGRPCTLSConfigs(t, signingKey)
	var unaryCalls atomic.Int64
	var streamCalls atomic.Int64

	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(serverTLS)),
		grpc.ChainUnaryInterceptor(verifier.UnaryInterceptor()),
		grpc.ChainStreamInterceptor(verifier.StreamInterceptor()),
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

	if _, err := healthpb.NewHealthClient(connection).Check(t.Context(), &healthpb.HealthCheckRequest{}); err != nil {
		t.Fatalf("public TLS health check error = %v", err)
	}

	token := signToken(t, signingKey, "key-1", "at+jwt", validClaims(now))
	credentialCtx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer "+token))
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

	if err := connection.Invoke(t.Context(), tlsAuthnUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{}); err == nil {
		t.Fatal("unauthenticated TLS unary call succeeded")
	}
	if unaryCalls.Load() != 1 || streamCalls.Load() != 1 {
		t.Fatalf("application calls = (unary %d, stream %d), want one authenticated call each", unaryCalls.Load(), streamCalls.Load())
	}
}

func assertAuthenticatedRPCContext(ctx context.Context, t *testing.T) {
	t.Helper()
	principal, ok := reqctx.PrincipalFromContext(ctx)
	if !ok || principal.Subject != "opaque-subject" {
		t.Fatalf("RPC principal = (%+v, %v), want opaque subject", principal, ok)
	}
	incoming, _ := metadata.FromIncomingContext(ctx)
	if len(incoming.Get("authorization")) != 0 {
		t.Fatal("handler-visible authorization metadata was retained")
	}
}

// registerTLSAuthnService registers the one unary and one streaming method this
// file serves over a real connection, handing each the RPC context its
// interceptor produced.
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

func testGRPCTLSConfigs(t *testing.T, key *rsa.PrivateKey) (*tls.Config, *tls.Config) {
	t.Helper()
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
	return &tls.Config{ //nolint:gosec // Test-only TLS uses the repository's minimum through crypto/tls defaults.
			MinVersion:   tls.VersionTLS13,
			Certificates: []tls.Certificate{certificate},
		}, &tls.Config{ //nolint:gosec // The test verifies a fixed self-signed server authority.
			MinVersion: tls.VersionTLS13,
			RootCAs:    roots,
			ServerName: "localhost",
		}
}
