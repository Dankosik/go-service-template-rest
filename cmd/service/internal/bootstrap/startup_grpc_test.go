package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewGRPCRuntimeBuildsExplicitPlaintextServer(t *testing.T) {
	t.Parallel()

	cfg := grpcRuntimeTestConfig()
	cfg.GRPC.Server.TransportSecurity = "plaintext"
	cfg.GRPC.Server.AllowPlaintext = true

	registered := false
	server, err := newGRPCRuntime(
		cfg,
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		grpcRuntimeBindings{
			Services: []grpcx.RegisterService{
				func(grpc.ServiceRegistrar) { registered = true },
			},
		},
	)
	if err != nil {
		t.Fatalf("newGRPCRuntime() error = %v", err)
	}
	if server == nil {
		t.Fatal("newGRPCRuntime() server = nil, want configured server")
	}
	if !registered {
		t.Fatal("newGRPCRuntime() did not apply generated service registrations")
	}
	t.Cleanup(func() { _ = server.Close() })
}

//nolint:paralleltest // Installs a process-wide tracer provider to observe bootstrap wiring.
func TestNewGRPCRuntimeMapsObservabilityPolicy(t *testing.T) {
	spanRecorder := telemetrytest.InstallSpanRecorder(t)

	for _, testCase := range []struct {
		name           string
		configure      func(*config.GRPCServerConfig)
		wantAccessLog  bool
		wantHealthSpan bool
	}{
		{
			name: "health access log enabled",
			configure: func(cfg *config.GRPCServerConfig) {
				cfg.AccessLogHealthChecks = true
				cfg.AccessLogSuccessSampleRate = 1
				cfg.AccessLogSlowThreshold = time.Hour
			},
			wantAccessLog: true,
		},
		{
			name: "successful health call sampled out",
			configure: func(cfg *config.GRPCServerConfig) {
				cfg.AccessLogHealthChecks = true
				cfg.AccessLogSuccessSampleRate = 0
				cfg.AccessLogSlowThreshold = time.Hour
			},
		},
		{
			name: "slow threshold precedes sampling",
			configure: func(cfg *config.GRPCServerConfig) {
				cfg.AccessLogHealthChecks = true
				cfg.AccessLogSuccessSampleRate = 0
				cfg.AccessLogSlowThreshold = time.Nanosecond
			},
			wantAccessLog: true,
		},
		{
			name: "health telemetry enabled",
			configure: func(cfg *config.GRPCServerConfig) {
				cfg.AccessLogHealthChecks = false
				cfg.AccessLogSuccessSampleRate = 0
				cfg.AccessLogSlowThreshold = 0
				cfg.TelemetryHealthChecks = true
			},
			wantHealthSpan: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := grpcRuntimeTestConfig()
			cfg.GRPC.Server.TransportSecurity = "plaintext"
			cfg.GRPC.Server.AllowPlaintext = true
			testCase.configure(&cfg.GRPC.Server)

			var output bytes.Buffer
			log := slog.New(slog.NewJSONHandler(&output, nil))
			connection, server := startGRPCRuntimeConnection(t, cfg, log)
			healthClient := healthgrpc.NewHealthClient(connection)
			spansBefore := len(spanRecorder.Ended())

			if _, err := healthClient.Check(t.Context(), &healthgrpc.HealthCheckRequest{}); err != nil {
				t.Fatalf("Health.Check() error = %v", err)
			}
			// A client reply can arrive before the server stats handler publishes
			// its ended span. Graceful shutdown is the owned RPC-lifecycle barrier.
			if err := server.Shutdown(t.Context()); err != nil {
				t.Fatalf("Server.Shutdown() error = %v", err)
			}

			if got := strings.Contains(output.String(), `"msg":"grpc_request"`); got != testCase.wantAccessLog {
				t.Fatalf("health access-log emitted = %t, want %t; log = %q", got, testCase.wantAccessLog, output.String())
			}
			gotHealthSpan := false
			for _, span := range spanRecorder.Ended()[spansBefore:] {
				if span.Name() == "grpc.health.v1.Health/Check" && span.SpanKind() == trace.SpanKindServer {
					gotHealthSpan = true
					break
				}
			}
			if gotHealthSpan != testCase.wantHealthSpan {
				t.Fatalf(
					"server health span emitted = %t, want %t; spans = %v",
					gotHealthSpan,
					testCase.wantHealthSpan,
					spanRecorder.Ended()[spansBefore:],
				)
			}
		})
	}
}

func TestNewGRPCRuntimeRejectsUnreadableTLSCredentials(t *testing.T) {
	t.Parallel()

	cfg := grpcRuntimeTestConfig()
	cfg.GRPC.Server.TransportSecurity = "tls"
	cfg.GRPC.Server.TLS.CertFile = t.TempDir() + "/missing.crt"
	cfg.GRPC.Server.TLS.KeyFile = t.TempDir() + "/missing.key"

	server, err := newGRPCRuntime(
		cfg,
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		grpcRuntimeBindings{},
	)
	if server != nil {
		t.Fatal("newGRPCRuntime() returned a server with unreadable credentials")
	}
	if err == nil || !strings.Contains(err.Error(), "load gRPC server TLS credentials") {
		t.Fatalf("newGRPCRuntime() error = %v, want safe TLS load failure", err)
	}
}

func TestNewGRPCRuntimeLoadsValidTLSCredentials(t *testing.T) {
	t.Parallel()

	certificate, key := testServerTLSMaterial(t)
	cfg := grpcRuntimeTestConfig()
	cfg.GRPC.Server.TransportSecurity = "tls"
	cfg.GRPC.Server.TLS.CertFile = writeTestTLSFile(t, "service.crt", certificate)
	cfg.GRPC.Server.TLS.KeyFile = writeTestTLSFile(t, "service.key", key)

	server, err := newGRPCRuntime(
		cfg,
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		grpcRuntimeBindings{},
	)
	if err != nil {
		t.Fatalf("newGRPCRuntime() error = %v", err)
	}
	if server == nil {
		t.Fatal("newGRPCRuntime() server = nil, want valid TLS server")
	}
	t.Cleanup(func() { _ = server.Close() })
}

func TestNewGRPCRuntimeRejectsMismatchedTLSCredentials(t *testing.T) {
	t.Parallel()

	certificate, _ := testServerTLSMaterial(t)
	_, mismatchedKey := testServerTLSMaterial(t)
	cfg := grpcRuntimeTestConfig()
	cfg.GRPC.Server.TransportSecurity = "tls"
	cfg.GRPC.Server.TLS.CertFile = writeTestTLSFile(t, "service.crt", certificate)
	cfg.GRPC.Server.TLS.KeyFile = writeTestTLSFile(t, "mismatched.key", mismatchedKey)

	server, err := newGRPCRuntime(
		cfg,
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		grpcRuntimeBindings{},
	)
	if server != nil {
		t.Fatal("newGRPCRuntime() returned a server with mismatched TLS credentials")
	}
	if err == nil || !strings.Contains(err.Error(), "load gRPC server TLS credentials") {
		t.Fatalf("newGRPCRuntime() error = %v, want safe TLS mismatch failure", err)
	}
}

func TestNewGRPCRuntimeRejectsUint32OverflowingTransportBounds(t *testing.T) {
	t.Parallel()

	if strconv.IntSize < 64 {
		t.Skip("int cannot represent a value above uint32 on this architecture")
	}
	overflow := int(uint64(math.MaxUint32) + 1)
	for _, testCase := range []struct {
		name   string
		field  string
		mutate func(*config.Config)
	}{
		{
			name:  "concurrent streams",
			field: "grpc.server.max_concurrent_streams",
			mutate: func(cfg *config.Config) {
				cfg.GRPC.Server.MaxConcurrentStreams = overflow
			},
		},
		{
			name:  "header list bytes",
			field: "grpc.server.max_header_list_bytes",
			mutate: func(cfg *config.Config) {
				cfg.GRPC.Server.MaxHeaderListBytes = overflow
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := grpcRuntimeTestConfig()
			testCase.mutate(&cfg)
			server, err := newGRPCRuntime(
				cfg,
				slog.New(slog.DiscardHandler),
				telemetry.New(),
				nil,
				grpcRuntimeBindings{},
			)
			if server != nil {
				t.Fatalf("newGRPCRuntime() returned a server with overflowing %s", testCase.field)
			}
			if err == nil || !strings.Contains(err.Error(), testCase.field) {
				t.Fatalf(
					"newGRPCRuntime() error = %v, want %s range failure",
					err,
					testCase.field,
				)
			}
		})
	}
}

func TestNewGRPCRuntimeThreadsServicePolicyBindings(t *testing.T) {
	t.Parallel()

	cfg := grpcRuntimeTestConfig()
	cfg.GRPC.Server.TransportSecurity = "plaintext"
	cfg.GRPC.Server.AllowPlaintext = true
	var policyCalled atomic.Bool
	server, err := newGRPCRuntime(
		cfg,
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		grpcRuntimeBindings{
			UnaryPolicy: []grpc.UnaryServerInterceptor{
				func(
					context.Context,
					any,
					*grpc.UnaryServerInfo,
					grpc.UnaryHandler,
				) (any, error) {
					policyCalled.Store(true)
					return nil, status.Error(codes.Unauthenticated, "authentication required")
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("newGRPCRuntime() error = %v", err)
	}

	listener := bufconn.Listen(1 << 20)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	connection, err := grpc.NewClient(
		"passthrough:///bootstrap-policy-test",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		_ = server.Close()
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = connection.Close()
		_ = server.Close()
		if err := <-serveDone; err != nil {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})

	_, err = healthgrpc.NewHealthClient(connection).Check(
		t.Context(),
		&healthgrpc.HealthCheckRequest{},
	)
	if code := status.Code(err); code != codes.Unauthenticated {
		t.Fatalf("Health.Check() policy code = %s, want %s", code, codes.Unauthenticated)
	}
	if !policyCalled.Load() {
		t.Fatal("bootstrap did not thread the unary service policy")
	}
}

func testServerTLSMaterial(t *testing.T) ([]byte, []byte) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"example.com"},
	}
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		publicKey,
		privateKey,
	)
	if err != nil {
		t.Fatalf("x509.CreateCertificate() error = %v", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("x509.MarshalPKCS8PrivateKey() error = %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certificateDER,
		}), pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyDER,
		})
}

func writeTestTLSFile(t *testing.T, name string, contents []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	return path
}

func startGRPCRuntimeConnection(
	t *testing.T,
	cfg config.Config,
	log *slog.Logger,
) (*grpc.ClientConn, *grpcx.Server) {
	t.Helper()

	server, err := newGRPCRuntime(cfg, log, telemetry.New(), nil, grpcRuntimeBindings{})
	if err != nil {
		t.Fatalf("newGRPCRuntime() error = %v", err)
	}
	server.MarkServing()

	listener := bufconn.Listen(1 << 20)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	connection, err := grpc.NewClient(
		"passthrough:///bootstrap-observability-test",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		_ = server.Close()
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = connection.Close()
		_ = server.Close()
		if err := <-serveDone; err != nil {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})
	return connection, server
}

func grpcRuntimeTestConfig() config.Config {
	return config.Config{
		GRPC: config.GRPCConfig{
			Server: config.GRPCServerConfig{
				Enabled:                    true,
				Addr:                       "127.0.0.1:9091",
				MaxConnections:             16,
				MaxConcurrentRPCs:          8,
				MaxConcurrentStreams:       4,
				MaxHeaderListBytes:         16 << 10,
				MaxReceiveMessageBytes:     4 << 20,
				MaxSendMessageBytes:        4 << 20,
				AccessLogSuccessSampleRate: 1,
			},
		},
	}
}
