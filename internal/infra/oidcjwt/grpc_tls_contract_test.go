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
	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
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
	unauthenticatedWatch := make(chan struct{}, 1)
	observeUnauthenticatedWatch := func(
		service any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		incoming, _ := metadata.FromIncomingContext(stream.Context())
		if info.FullMethod == healthpb.Health_Watch_FullMethodName && len(incoming.Get("authorization")) == 0 {
			select {
			case unauthenticatedWatch <- struct{}{}:
			default:
			}
		}
		return handler(service, stream)
	}

	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(serverTLS)),
		grpc.ChainUnaryInterceptor(verifier.UnaryInterceptor()),
		grpc.ChainStreamInterceptor(observeUnauthenticatedWatch, verifier.StreamInterceptor()),
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
	watchObservations := make(chan tlsWatchObservation, 1)
	healthpb.RegisterHealthServer(server, &observedTLSHealthServer{
		HealthServer: healthServer,
		observations: watchObservations,
	})

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

	clientConfig := grpcclient.DefaultConfig("dns:///" + listener.Addr().String())
	unauthenticatedConnection, err := grpcclient.New(
		clientConfig,
		grpcclient.Options{TransportCredentials: credentials.NewTLS(clientTLS)},
	)
	if err != nil {
		t.Fatalf("build unauthenticated health-aware client: %v", err)
	}
	t.Cleanup(func() { _ = unauthenticatedConnection.Close() })
	unauthenticatedConnection.Connect()
	waitForTLSContractEvent(t, unauthenticatedWatch, "unauthenticated standard health Watch")

	callCtx, cancelCall := context.WithTimeout(t.Context(), time.Second)
	err = unauthenticatedConnection.Invoke(
		callCtx,
		tlsAuthnUnaryMethod,
		&emptypb.Empty{},
		&emptypb.Empty{},
		grpc.PerRPCCredentials(staticBearerCredential(token)),
	)
	cancelCall()
	if err == nil {
		t.Fatal("call-scoped credential made an unauthenticated health backend eligible")
	}
	if got := unaryCalls.Load(); got != 1 {
		t.Fatalf("application unary calls after ineligible client = %d, want 1", got)
	}

	authenticatedConnection, err := grpcclient.New(
		clientConfig,
		grpcclient.Options{
			TransportCredentials: credentials.NewTLS(clientTLS),
			PerRPCCredentials:    staticBearerCredential(token),
		},
	)
	if err != nil {
		t.Fatalf("build authenticated health-aware client: %v", err)
	}
	t.Cleanup(func() { _ = authenticatedConnection.Close() })
	authenticatedConnection.Connect()
	observation := waitForTLSContractEvent(t, watchObservations, "authenticated standard health Watch")
	if observation.subject != "opaque-subject" {
		t.Fatalf("health Watch principal subject = %q, want opaque-subject", observation.subject)
	}
	if observation.hasReservedMetadata {
		t.Fatal("health Watch received credential-supplied reserved metadata")
	}

	callCtx, cancelCall = context.WithTimeout(t.Context(), time.Second)
	err = authenticatedConnection.Invoke(callCtx, tlsAuthnUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{})
	cancelCall()
	if err != nil {
		t.Fatalf("connection-credential application call error = %v", err)
	}
	if got := unaryCalls.Load(); got != 2 {
		t.Fatalf("application unary calls after authenticated client = %d, want 2", got)
	}
}

type staticBearerCredential string

func (credential staticBearerCredential) GetRequestMetadata(
	context.Context,
	...string,
) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + string(credential),
		"baggage":       "forged=value",
		"traceparent":   "forged",
		"tracestate":    "forged",
		"x-request-id":  "forged",
	}, nil
}

func (staticBearerCredential) RequireTransportSecurity() bool { return true }

type tlsWatchObservation struct {
	subject             string
	hasReservedMetadata bool
}

type observedTLSHealthServer struct {
	healthpb.HealthServer

	observations chan<- tlsWatchObservation
}

func (s *observedTLSHealthServer) Watch(
	request *healthpb.HealthCheckRequest,
	stream grpc.ServerStreamingServer[healthpb.HealthCheckResponse],
) error {
	observation := tlsWatchObservation{}
	if principal, ok := reqctx.PrincipalFromContext(stream.Context()); ok {
		observation.subject = principal.Subject
	}
	incoming, _ := metadata.FromIncomingContext(stream.Context())
	for _, key := range []string{"baggage", "traceparent", "tracestate", "x-request-id"} {
		observation.hasReservedMetadata = observation.hasReservedMetadata || len(incoming.Get(key)) > 0
	}
	select {
	case s.observations <- observation:
	default:
	}
	//nolint:wrapcheck // The test adapter preserves grpc-go's health result.
	return s.HealthServer.Watch(request, stream)
}

func waitForTLSContractEvent[T any](t *testing.T, events <-chan T, name string) T {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	select {
	case event := <-events:
		return event
	case <-ctx.Done():
		t.Fatalf("%s did not occur: %v", name, ctx.Err())
		var zero T
		return zero
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
