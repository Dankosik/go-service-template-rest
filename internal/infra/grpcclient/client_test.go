package grpcclient_test

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	builder := manual.NewBuilderWithScheme("grpcclient-service-config")
	builder.BuildCallback = func(
		_ resolver.Target,
		connection resolver.ClientConn,
		_ resolver.BuildOptions,
	) {
		if err := connection.UpdateState(resolver.State{
			Addresses: []resolver.Address{{Addr: address}},
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
		}); err != nil {
			t.Errorf("resolver UpdateState() error = %v", err)
		}
	}
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

func TestNewAppliesFixedTransportBounds(t *testing.T) {
	const method = "/grpcclient.test.PayloadService/Call"

	var handlerCalls atomic.Int64
	oversizedPayload := make([]byte, 4<<20)
	oversizedHeader := strings.Repeat("x", 16<<10)
	target := startTestServer(t, func(server *grpc.Server) {
		grpctest.Register(server, grpctest.Unary(
			method,
			func(ctx context.Context, request *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error) {
				handlerCalls.Add(1)
				if len(request.GetValue()) == 0 {
					return &wrapperspb.BytesValue{Value: oversizedPayload}, nil
				}
				if err := grpc.SetHeader(ctx, metadata.Pairs("oversized", oversizedHeader)); err != nil {
					return nil, fmt.Errorf("set response header: %w", err)
				}
				return &wrapperspb.BytesValue{}, nil
			},
		))
	})
	connection, err := grpcclient.New(
		target,
		grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	invoke := func(request, response *wrapperspb.BytesValue) error {
		return connection.Invoke(t.Context(), method, request, response)
	}

	if err := invoke(&wrapperspb.BytesValue{Value: oversizedPayload}, &wrapperspb.BytesValue{}); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("oversized send error = %v, want %s", err, codes.ResourceExhausted)
	}
	if got := handlerCalls.Load(); got != 0 {
		t.Fatalf("handler calls after oversized send = %d, want 0", got)
	}
	if err := invoke(&wrapperspb.BytesValue{}, &wrapperspb.BytesValue{}); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("oversized response error = %v, want %s", err, codes.ResourceExhausted)
	}
	if got := handlerCalls.Load(); got != 1 {
		t.Fatalf("handler calls after oversized response = %d, want 1", got)
	}
	if err := invoke(&wrapperspb.BytesValue{Value: []byte{1}}, &wrapperspb.BytesValue{}); err == nil {
		t.Fatal("oversized response metadata succeeded")
	}
	if got := handlerCalls.Load(); got != 2 {
		t.Fatalf("handler calls after oversized metadata = %d, want 2", got)
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
