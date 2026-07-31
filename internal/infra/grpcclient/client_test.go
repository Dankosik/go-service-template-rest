package grpcclient_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestDefaultConfigIsFinite(t *testing.T) {
	t.Parallel()

	cfg := grpcclient.DefaultConfig("dns:///unreachable.invalid:443")
	if cfg.Target != "dns:///unreachable.invalid:443" {
		t.Fatalf("Target = %q", cfg.Target)
	}
	if cfg.MaxHeaderListBytes != 16<<10 {
		t.Fatalf("MaxHeaderListBytes = %d, want %d", cfg.MaxHeaderListBytes, 16<<10)
	}
	if cfg.MaxReceiveMessageBytes != 4<<20 {
		t.Fatalf("MaxReceiveMessageBytes = %d, want %d", cfg.MaxReceiveMessageBytes, 4<<20)
	}
	if cfg.MaxSendMessageBytes != 4<<20 {
		t.Fatalf("MaxSendMessageBytes = %d, want %d", cfg.MaxSendMessageBytes, 4<<20)
	}
}

func TestNewPerformsNoNetworkIOAndReturnsOwnedConnection(t *testing.T) {
	t.Parallel()

	connection, err := grpcclient.New(
		grpcclient.DefaultConfig("dns:///unreachable.invalid:443"),
		grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("ClientConn.Close() error = %v", err)
	}
}

func TestNewRejectsMissingTrustAndBounds(t *testing.T) {
	t.Parallel()

	valid := grpcclient.DefaultConfig("dns:///service.internal:9000")
	for _, testCase := range []struct {
		name        string
		mutate      func(*grpcclient.Config)
		credentials bool
	}{
		{name: "target", mutate: func(cfg *grpcclient.Config) { cfg.Target = "" }, credentials: true},
		{name: "credentials"},
		{
			name:        "headers",
			mutate:      func(cfg *grpcclient.Config) { cfg.MaxHeaderListBytes = 0 },
			credentials: true,
		},
		{
			name:        "zero receive message",
			mutate:      func(cfg *grpcclient.Config) { cfg.MaxReceiveMessageBytes = 0 },
			credentials: true,
		},
		{
			name:        "negative receive message",
			mutate:      func(cfg *grpcclient.Config) { cfg.MaxReceiveMessageBytes = -1 },
			credentials: true,
		},
		{
			name:        "zero send message",
			mutate:      func(cfg *grpcclient.Config) { cfg.MaxSendMessageBytes = 0 },
			credentials: true,
		},
		{
			name:        "negative send message",
			mutate:      func(cfg *grpcclient.Config) { cfg.MaxSendMessageBytes = -1 },
			credentials: true,
		},
		{
			name:        "unknown propagation policy",
			credentials: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			if testCase.mutate != nil {
				testCase.mutate(&cfg)
			}
			options := grpcclient.Options{}
			if testCase.credentials {
				options.TransportCredentials = insecure.NewCredentials()
			}
			if testCase.name == "unknown propagation policy" {
				options.Propagation = grpcclient.PropagationPolicy(255)
			}
			if connection, err := grpcclient.New(cfg, options); err == nil {
				_ = connection.Close()
				t.Fatal("New() error = nil, want invalid configuration rejected")
			}
		})
	}
}

func TestClientEnforcesSendAndReceiveMessageLimits(t *testing.T) {
	var handlerCalls atomic.Int64
	target := startPayloadServer(t, func(
		_ context.Context,
		request *wrapperspb.BytesValue,
	) (*wrapperspb.BytesValue, error) {
		handlerCalls.Add(1)
		if len(request.GetValue()) == 0 {
			return &wrapperspb.BytesValue{Value: make([]byte, 1024)}, nil
		}
		return &wrapperspb.BytesValue{}, nil
	})

	t.Run("send", func(t *testing.T) {
		cfg := grpcclient.DefaultConfig(target)
		cfg.MaxSendMessageBytes = 64
		connection, err := grpcclient.New(
			cfg,
			grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
		)
		if err != nil {
			t.Fatalf("grpcclient.New() error = %v", err)
		}
		t.Cleanup(func() { _ = connection.Close() })

		err = connection.Invoke(
			t.Context(),
			testPayloadFullMethod,
			&wrapperspb.BytesValue{Value: make([]byte, 1024)},
			&wrapperspb.BytesValue{},
		)
		if code := status.Code(err); code != codes.ResourceExhausted {
			t.Fatalf("oversized send code = %s, want %s", code, codes.ResourceExhausted)
		}
		if calls := handlerCalls.Load(); calls != 0 {
			t.Fatalf("handler calls after oversized send = %d, want 0", calls)
		}
	})

	t.Run("receive", func(t *testing.T) {
		cfg := grpcclient.DefaultConfig(target)
		cfg.MaxReceiveMessageBytes = 64
		connection, err := grpcclient.New(
			cfg,
			grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
		)
		if err != nil {
			t.Fatalf("grpcclient.New() error = %v", err)
		}
		t.Cleanup(func() { _ = connection.Close() })

		err = connection.Invoke(
			t.Context(),
			testPayloadFullMethod,
			&wrapperspb.BytesValue{},
			&wrapperspb.BytesValue{},
		)
		if code := status.Code(err); code != codes.ResourceExhausted {
			t.Fatalf("oversized receive code = %s, want %s", code, codes.ResourceExhausted)
		}
		if calls := handlerCalls.Load(); calls != 1 {
			t.Fatalf("handler calls after oversized receive = %d, want 1", calls)
		}
	})
}

func TestClientEnforcesReceivedMetadataLimit(t *testing.T) {
	var handlerCalls atomic.Int64
	target := startPayloadServer(t, func(
		ctx context.Context,
		request *wrapperspb.BytesValue,
	) (*wrapperspb.BytesValue, error) {
		handlerCalls.Add(1)
		if len(request.GetValue()) > 0 {
			if err := grpc.SetHeader(
				ctx,
				metadata.Pairs("oversized", strings.Repeat("x", 8<<10)),
			); err != nil {
				return nil, fmt.Errorf("set oversized response metadata: %w", err)
			}
		}
		return &wrapperspb.BytesValue{}, nil
	})

	cfg := grpcclient.DefaultConfig(target)
	cfg.MaxHeaderListBytes = 1024
	connection, err := grpcclient.New(
		cfg,
		grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
	)
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	if err := connection.Invoke(
		t.Context(),
		testPayloadFullMethod,
		&wrapperspb.BytesValue{},
		&wrapperspb.BytesValue{},
	); err != nil {
		t.Fatalf("baseline Invoke() error = %v", err)
	}
	err = connection.Invoke(
		t.Context(),
		testPayloadFullMethod,
		&wrapperspb.BytesValue{Value: []byte("return oversized metadata")},
		&wrapperspb.BytesValue{},
	)
	if err == nil {
		t.Fatal("Invoke() with oversized response metadata succeeded")
	}
	if calls := handlerCalls.Load(); calls != 2 {
		t.Fatalf("handler calls = %d, want baseline and oversized-metadata calls", calls)
	}
}

func TestClientTLSVerifiesServerHostname(t *testing.T) {
	certificateSource := httptest.NewTLSServer(http.NotFoundHandler())
	serverCertificate := certificateSource.TLS.Certificates[0]
	leafCertificate := certificateSource.Certificate()
	certificateSource.Close()

	serverCredentials := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCertificate},
		MinVersion:   tls.VersionTLS12,
	})
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	server := grpc.NewServer(grpc.Creds(serverCredentials))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(server, healthServer)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		if err := <-serveDone; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})

	roots := x509.NewCertPool()
	roots.AddCert(leafCertificate)
	target := "passthrough:///" + listener.Addr().String()

	validConnection, err := grpcclient.New(
		grpcclient.DefaultConfig(target),
		grpcclient.Options{TransportCredentials: credentials.NewTLS(&tls.Config{
			RootCAs:    roots,
			ServerName: "example.com",
			MinVersion: tls.VersionTLS12,
		})},
	)
	if err != nil {
		t.Fatalf("grpcclient.New() valid TLS error = %v", err)
	}
	t.Cleanup(func() { _ = validConnection.Close() })
	if _, err := healthgrpc.NewHealthClient(validConnection).Check(
		t.Context(),
		&healthgrpc.HealthCheckRequest{},
	); err != nil {
		t.Fatalf("Health.Check() with matching hostname = %v", err)
	}

	invalidConnection, err := grpcclient.New(
		grpcclient.DefaultConfig(target),
		grpcclient.Options{TransportCredentials: credentials.NewTLS(&tls.Config{
			RootCAs:    roots,
			ServerName: "wrong.invalid",
			MinVersion: tls.VersionTLS12,
		})},
	)
	if err != nil {
		t.Fatalf("grpcclient.New() mismatched TLS construction error = %v", err)
	}
	t.Cleanup(func() { _ = invalidConnection.Close() })
	callCtx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_, err = healthgrpc.NewHealthClient(invalidConnection).Check(
		callCtx,
		&healthgrpc.HealthCheckRequest{},
	)
	if err == nil {
		t.Fatal("Health.Check() with mismatched hostname succeeded")
	}
}

const testPayloadFullMethod = "/grpcclient.test.PayloadService/Call"

type payloadServiceServer interface {
	Call(ctx context.Context, request *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error)
}

type payloadServiceFunc func(
	context.Context,
	*wrapperspb.BytesValue,
) (*wrapperspb.BytesValue, error)

func (f payloadServiceFunc) Call(
	ctx context.Context,
	request *wrapperspb.BytesValue,
) (*wrapperspb.BytesValue, error) {
	return f(ctx, request)
}

func startPayloadServer(
	t *testing.T,
	call payloadServiceFunc,
	serverOptions ...grpc.ServerOption,
) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	server := grpc.NewServer(serverOptions...)
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "grpcclient.test.PayloadService",
		HandlerType: (*payloadServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Call",
				Handler: func(
					implementation any,
					ctx context.Context,
					decode func(any) error,
					interceptor grpc.UnaryServerInterceptor,
				) (any, error) {
					request := new(wrapperspb.BytesValue)
					if err := decode(request); err != nil {
						return nil, err
					}
					service, ok := implementation.(payloadServiceServer)
					if !ok {
						return nil, errors.New("test payload service implementation has unexpected type")
					}
					handler := func(ctx context.Context, decoded any) (any, error) {
						typedRequest, ok := decoded.(*wrapperspb.BytesValue)
						if !ok {
							return nil, errors.New("test payload request has unexpected type")
						}
						return service.Call(ctx, typedRequest)
					}
					if interceptor == nil {
						return handler(ctx, request)
					}
					return interceptor(ctx, request, &grpc.UnaryServerInfo{
						Server:     implementation,
						FullMethod: testPayloadFullMethod,
					}, handler)
				},
			},
		},
	}, call)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		if err := <-serveDone; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})
	return "passthrough:///" + listener.Addr().String()
}
