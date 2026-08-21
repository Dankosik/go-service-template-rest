package bootstrap

import (
	"context"
	"log/slog"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const bootstrapTestMethod = "/bootstrap.test.Service/Call"

func TestNewGRPCRuntimeThreadsServiceAndPolicyBindings(t *testing.T) {
	called := false
	server, err := newGRPCRuntime(
		grpcRuntimeTestConfig(),
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		grpcRuntimeBindings{
			Services: []grpcx.RegisterService{func(registrar grpc.ServiceRegistrar) {
				grpctest.Register(registrar, grpctest.Unary(
					bootstrapTestMethod,
					func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
						called = true
						return &emptypb.Empty{}, nil
					},
				))
			}},
			UnaryPolicy: []grpc.UnaryServerInterceptor{func(
				context.Context,
				any,
				*grpc.UnaryServerInfo,
				grpc.UnaryHandler,
			) (any, error) {
				return nil, status.Error(codes.PermissionDenied, "policy denied")
			}},
		},
	)
	if err != nil {
		t.Fatalf("newGRPCRuntime() error = %v", err)
	}
	connection := grpctest.ServeBufconn(t, server)
	err = connection.Invoke(t.Context(), bootstrapTestMethod, &emptypb.Empty{}, &emptypb.Empty{})
	if code := status.Code(err); code != codes.PermissionDenied {
		t.Fatalf("Invoke() code = %s, want %s", code, codes.PermissionDenied)
	}
	if called {
		t.Fatal("denied RPC reached handler")
	}
}

func TestNewGRPCRuntimeRejectsUnreadableTLSCredentials(t *testing.T) {
	cfg := grpcRuntimeTestConfig()
	cfg.GRPC.Server.TransportSecurity = "tls"
	cfg.GRPC.Server.TLS = config.GRPCTLSConfig{
		CertFile: "/missing/service.crt",
		KeyFile:  "/missing/service.key",
	}
	if server, err := newGRPCRuntime(
		cfg,
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		grpcRuntimeBindings{},
	); err == nil || server != nil {
		t.Fatalf("newGRPCRuntime() = (%v, %v), want startup failure", server, err)
	}
}

func grpcRuntimeTestConfig() config.Config {
	return config.Config{GRPC: config.GRPCConfig{Server: config.GRPCServerConfig{
		Enabled:           true,
		Addr:              "127.0.0.1:0",
		TransportSecurity: "plaintext",
	}}}
}
