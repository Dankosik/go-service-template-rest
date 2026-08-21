package grpcclient_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

func startTestServer(t *testing.T, register func(*grpc.Server), options ...grpc.ServerOption) string {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer(options...)
	register(server)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		if err := <-done; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve: %v", err)
		}
	})
	return "passthrough:///" + listener.Addr().String()
}

func registerServingHealth(server *grpc.Server) {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(server, healthServer)
}

func startMetadataCaptureServer(t *testing.T) (<-chan metadata.MD, string) {
	t.Helper()
	received := make(chan metadata.MD, 1)
	target := startTestServer(t, registerServingHealth, grpc.UnaryInterceptor(func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		incoming, _ := metadata.FromIncomingContext(ctx)
		received <- incoming.Copy()
		return handler(ctx, request)
	}))
	return received, target
}

func testTLSCredentials( //nolint:ireturn // grpc-go exposes credentials as interfaces.
	t *testing.T,
) (credentials.TransportCredentials, credentials.TransportCredentials) {
	t.Helper()
	source := httptest.NewTLSServer(http.NotFoundHandler())
	defer source.Close()
	certificate := source.TLS.Certificates[0]
	roots := x509.NewCertPool()
	roots.AddCert(source.Certificate())
	return credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{certificate},
			MinVersion:   tls.VersionTLS12,
		}), credentials.NewTLS(&tls.Config{
			RootCAs:    roots,
			ServerName: "example.com",
			MinVersion: tls.VersionTLS12,
		})
}
