package grpcclient_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestNewIsLazyAndRequiresTargetAndCredentials(t *testing.T) {
	t.Parallel()
	connection, err := grpcclient.New(
		"dns:///unreachable.invalid:443",
		grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	for _, testCase := range []struct {
		name    string
		target  string
		options grpcclient.Options
	}{
		{name: "missing target", options: grpcclient.Options{TransportCredentials: insecure.NewCredentials()}},
		{name: "missing credentials", target: "dns:///service.internal:9000"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if connection, err := grpcclient.New(testCase.target, testCase.options); err == nil {
				_ = connection.Close()
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestNewDisablesResolverServiceConfig(t *testing.T) {
	var calls atomic.Int64
	target := startTestServer(t, func(server *grpc.Server) {
		grpctest.Register(server, grpctest.Unary(
			"/grpcclient.test.Service/Call",
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				if calls.Add(1) == 1 {
					return nil, status.Error(codes.Unavailable, "retry would hide this")
				}
				return &emptypb.Empty{}, nil
			},
		))
	})

	address := target[len("passthrough:///"):]
	builder := serviceConfigResolver{address: address}
	resolver.Register(builder)
	connection, err := grpcclient.New(
		builder.Scheme()+":///service",
		grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	err = connection.Invoke(
		t.Context(),
		"/grpcclient.test.Service/Call",
		&emptypb.Empty{},
		&emptypb.Empty{},
	)
	if code := status.Code(err); code != codes.Unavailable {
		t.Fatalf("Invoke() code = %s, want %s", code, codes.Unavailable)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("handler calls = %d, resolver retry policy was applied", got)
	}
}

func TestNewUsesExplicitTLSCredentials(t *testing.T) {
	serverCredentials, clientCredentials := testTLSCredentials(t)
	target := startTestServer(t, registerServingHealth, grpc.Creds(serverCredentials))
	connection, err := grpcclient.New(
		target,
		grpcclient.Options{TransportCredentials: clientCredentials},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if _, err := healthgrpc.NewHealthClient(connection).Check(ctx, &healthgrpc.HealthCheckRequest{}); err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}
}

type serviceConfigResolver struct {
	address string
}

func (serviceConfigResolver) Scheme() string { return "grpcclient-service-config" }

//nolint:ireturn // Implements resolver.Builder.
func (b serviceConfigResolver) Build(
	_ resolver.Target,
	connection resolver.ClientConn,
	_ resolver.BuildOptions,
) (resolver.Resolver, error) {
	connection.UpdateState(resolver.State{ //nolint:errcheck // Test resolver has no recovery path.
		Addresses: []resolver.Address{{Addr: b.address}},
		ServiceConfig: connection.ParseServiceConfig(`{
			"methodConfig":[{
				"name":[{"service":"grpcclient.test.Service","method":"Call"}],
				"retryPolicy":{
					"maxAttempts":2,
					"initialBackoff":"0.001s",
					"maxBackoff":"0.001s",
					"backoffMultiplier":1,
					"retryableStatusCodes":["UNAVAILABLE"]
				}
			}]
		}`),
	})
	return nopResolver{}, nil
}

type nopResolver struct{}

func (nopResolver) ResolveNow(resolver.ResolveNowOptions) {}
func (nopResolver) Close()                                {}

var _ resolver.Builder = serviceConfigResolver{}
