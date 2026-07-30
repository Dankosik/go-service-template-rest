package grpcx

import (
	"context"
	"errors"
	"math"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServerHealthFollowsAdmissionAndDrain(t *testing.T) {
	server, connection := startTestServer(t, testServerConfig(), nil)
	healthClient := healthgrpc.NewHealthClient(connection)

	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_NOT_SERVING)
	server.MarkServing()
	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_SERVING)
	server.StartDrain()
	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_NOT_SERVING)

	if err := server.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestServerHealthPreservesStandardUnknownServiceStatus(t *testing.T) {
	_, connection := startTestServer(t, testServerConfig(), nil)
	healthClient := healthgrpc.NewHealthClient(connection)

	_, err := healthClient.Check(t.Context(), &healthgrpc.HealthCheckRequest{
		Service: "service.not.registered",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestServerPolicyStatusBypassesHandlerErrorSanitization(t *testing.T) {
	handlerCalled := false
	register := func(registrar grpc.ServiceRegistrar) {
		registerTestUnaryService(registrar, testUnaryService{
			call: func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				handlerCalled = true
				return &emptypb.Empty{}, nil
			},
		})
	}
	_, connection := startTestServerWithOptions(t, testServerConfig(), register, Options{
		UnaryPolicy: []grpc.UnaryServerInterceptor{
			func(
				context.Context,
				any,
				*grpc.UnaryServerInfo,
				grpc.UnaryHandler,
			) (any, error) {
				return nil, status.Error(codes.Unauthenticated, "authentication required")
			},
		},
	})

	err := connection.Invoke(
		t.Context(),
		testUnaryFullMethod,
		&emptypb.Empty{},
		&emptypb.Empty{},
	)
	assertStatusCode(t, err, codes.Unauthenticated)
	if detail := status.Convert(err).Message(); detail != "authentication required" {
		t.Fatalf("policy status detail = %q, want authentication required", detail)
	}
	if handlerCalled {
		t.Fatal("authentication policy entered the business handler")
	}
}

func TestServerStreamingPolicyStatusBypassesHandlerErrorSanitization(t *testing.T) {
	_, connection := startTestServerWithOptions(t, testServerConfig(), nil, Options{
		StreamingPolicy: []grpc.StreamServerInterceptor{
			func(
				any,
				grpc.ServerStream,
				*grpc.StreamServerInfo,
				grpc.StreamHandler,
			) error {
				return status.Error(codes.PermissionDenied, "stream access denied")
			},
		},
	})
	healthClient := healthgrpc.NewHealthClient(connection)

	stream, err := healthClient.Watch(t.Context(), &healthgrpc.HealthCheckRequest{})
	if err == nil {
		_, err = stream.Recv()
	}
	assertStatusCode(t, err, codes.PermissionDenied)
	if detail := status.Convert(err).Message(); detail != "stream access denied" {
		t.Fatalf("streaming policy status detail = %q, want stream access denied", detail)
	}
}

func TestServerSanitizesRawPolicyErrors(t *testing.T) {
	register := func(registrar grpc.ServiceRegistrar) {
		registerTestUnaryService(registrar, testUnaryService{
			call: func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, nil
			},
		})
	}
	_, connection := startTestServerWithOptions(t, testServerConfig(), register, Options{
		UnaryPolicy: []grpc.UnaryServerInterceptor{
			func(
				context.Context,
				any,
				*grpc.UnaryServerInfo,
				grpc.UnaryHandler,
			) (any, error) {
				return nil, errors.New("authentication dependency credential=secret")
			},
		},
	})

	err := connection.Invoke(
		t.Context(),
		testUnaryFullMethod,
		&emptypb.Empty{},
		&emptypb.Empty{},
	)
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("policy error detail = %q, want sanitized detail", detail)
	}
}

func TestServerSanitizesRawStreamingPolicyErrors(t *testing.T) {
	_, connection := startTestServerWithOptions(t, testServerConfig(), nil, Options{
		StreamingPolicy: []grpc.StreamServerInterceptor{
			func(
				any,
				grpc.ServerStream,
				*grpc.StreamServerInfo,
				grpc.StreamHandler,
			) error {
				return errors.New("authorization dependency credential=secret")
			},
		},
	})
	healthClient := healthgrpc.NewHealthClient(connection)

	stream, err := healthClient.Watch(t.Context(), &healthgrpc.HealthCheckRequest{})
	if err == nil {
		_, err = stream.Recv()
	}
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("streaming policy error detail = %q, want sanitized detail", detail)
	}
}

func TestServerSanitizesUntrustedHandlerStatus(t *testing.T) {
	register := func(registrar grpc.ServiceRegistrar) {
		registerTestUnaryService(registrar, testUnaryService{
			call: func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return nil, status.Error(codes.PermissionDenied, "dependency credential=secret")
			},
		})
	}
	_, connection := startTestServer(t, testServerConfig(), register)

	err := connection.Invoke(
		t.Context(),
		testUnaryFullMethod,
		&emptypb.Empty{},
		&emptypb.Empty{},
	)
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("handler status detail = %q, want sanitized detail", detail)
	}
}

func TestServerCorrelationReturnsAcceptedRequestID(t *testing.T) {
	observed := make(chan string, 2)
	register := func(registrar grpc.ServiceRegistrar) {
		registerTestUnaryService(registrar, testUnaryService{
			call: func(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
				observed <- reqctx.RequestID(ctx)
				return &emptypb.Empty{}, nil
			},
		})
	}
	_, connection := startTestServer(t, testServerConfig(), register)

	for _, testCase := range []struct {
		name      string
		candidate string
		want      string
	}{
		{name: "accepted", candidate: "request_123", want: "request_123"},
		{name: "replaced", candidate: "user@example.com"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := metadata.NewOutgoingContext(
				t.Context(),
				metadata.Pairs(requestIDMetadataKey, testCase.candidate),
			)
			var header metadata.MD
			if err := connection.Invoke(
				ctx,
				testUnaryFullMethod,
				&emptypb.Empty{},
				&emptypb.Empty{},
				grpc.Header(&header),
			); err != nil {
				t.Fatalf("Invoke() error = %v", err)
			}
			got := header.Get(requestIDMetadataKey)
			if len(got) != 1 {
				t.Fatalf("response request IDs = %v, want one", got)
			}
			if !reqctx.ValidRequestID(got[0]) {
				t.Fatalf("response request ID %q is invalid", got[0])
			}
			if testCase.want != "" && got[0] != testCase.want {
				t.Fatalf("response request ID = %q, want %q", got[0], testCase.want)
			}
			if testCase.want == "" && got[0] == testCase.candidate {
				t.Fatalf("invalid request ID %q was echoed", got[0])
			}
			if handlerRequestID := <-observed; handlerRequestID != got[0] {
				t.Fatalf("handler request ID = %q, want %q", handlerRequestID, got[0])
			}
		})
	}
}

func TestServerShutdownForcesBlockedRPCOnCanceledBudget(t *testing.T) {
	entered := make(chan struct{})
	handlerDone := make(chan struct{})
	var enteredOnce sync.Once
	register := func(registrar grpc.ServiceRegistrar) {
		registerTestUnaryService(registrar, testUnaryService{
			call: func(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
				enteredOnce.Do(func() { close(entered) })
				<-ctx.Done()
				close(handlerDone)
				return nil, ctx.Err()
			},
		})
	}
	server, connection := startTestServer(t, testServerConfig(), register)

	callDone := make(chan error, 1)
	go func() {
		callDone <- connection.Invoke(
			t.Context(),
			testUnaryFullMethod,
			&emptypb.Empty{},
			&emptypb.Empty{},
		)
	}()
	<-entered

	shutdownCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := server.Shutdown(shutdownCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Shutdown() error = %v, want context.Canceled", err)
	}
	<-handlerDone
	if err := <-callDone; err == nil {
		t.Fatal("blocked RPC succeeded after forced shutdown")
	}
	if err := server.Close(); err != nil {
		t.Fatalf("repeated Close() error = %v", err)
	}
}

func TestServerShutdownReturnsWhenHandlerIgnoresCancellation(t *testing.T) {
	entered := make(chan struct{})
	handlerDone := make(chan struct{})
	release := make(chan struct{})
	releaseHandler := sync.OnceFunc(func() { close(release) })
	defer releaseHandler()

	register := func(registrar grpc.ServiceRegistrar) {
		registerTestUnaryService(registrar, testUnaryService{
			call: func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				close(entered)
				<-release
				close(handlerDone)
				return &emptypb.Empty{}, nil
			},
		})
	}
	server, connection := startTestServer(t, testServerConfig(), register)

	callDone := make(chan error, 1)
	go func() {
		callDone <- connection.Invoke(
			context.Background(),
			testUnaryFullMethod,
			&emptypb.Empty{},
			&emptypb.Empty{},
		)
	}()
	<-entered

	shutdownCtx, cancel := context.WithCancel(context.Background())
	cancel()
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- server.Shutdown(shutdownCtx) }()

	select {
	case err := <-shutdownDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Shutdown() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Shutdown() did not honor the canceled shutdown budget")
	}
	select {
	case <-handlerDone:
		t.Fatal("forced shutdown waited for or completed a handler that ignored cancellation")
	default:
	}

	if err := server.Shutdown(t.Context()); err != nil {
		t.Fatalf("repeated Shutdown() after forced stop = %v, want nil", err)
	}

	releaseHandler()
	<-handlerDone
	// The handler and transport stop race after release: the client may observe
	// either its response or the forced connection close. The lifecycle
	// contract is the bounded shutdown return while the handler is still
	// parked, not a stronger claim about that concurrent terminal outcome.
	<-callDone
}

func TestNewServerRejectsUnboundedConfig(t *testing.T) {
	valid := testServerConfig()
	for _, testCase := range []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "RPCs", mutate: func(cfg *Config) { cfg.MaxConcurrentRPCs = 0 }},
		{name: "streams", mutate: func(cfg *Config) { cfg.MaxConcurrentStreams = 0 }},
		{name: "headers", mutate: func(cfg *Config) { cfg.MaxHeaderListBytes = 0 }},
		{name: "receive message", mutate: func(cfg *Config) { cfg.MaxReceiveMessageBytes = 0 }},
		{name: "send message", mutate: func(cfg *Config) { cfg.MaxSendMessageBytes = 0 }},
		{name: "negative sample rate", mutate: func(cfg *Config) { cfg.AccessLogSuccessSampleRate = -0.1 }},
		{name: "large sample rate", mutate: func(cfg *Config) { cfg.AccessLogSuccessSampleRate = 1.1 }},
		{name: "NaN sample rate", mutate: func(cfg *Config) { cfg.AccessLogSuccessSampleRate = math.NaN() }},
		{name: "infinite sample rate", mutate: func(cfg *Config) { cfg.AccessLogSuccessSampleRate = math.Inf(1) }},
		{name: "negative slow threshold", mutate: func(cfg *Config) { cfg.AccessLogSlowThreshold = -time.Nanosecond }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := valid
			testCase.mutate(&cfg)
			if _, err := NewServer(cfg, Options{}); err == nil {
				t.Fatal("NewServer() error = nil, want invalid bound rejected")
			}
		})
	}
}

func assertHealthStatus(
	t *testing.T,
	client healthgrpc.HealthClient,
	want healthgrpc.HealthCheckResponse_ServingStatus,
) {
	t.Helper()
	response, err := client.Check(t.Context(), &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}
	if got := response.GetStatus(); got != want {
		t.Fatalf("Health.Check() status = %s, want %s", got, want)
	}
}

func testServerConfig() Config {
	return Config{
		MaxConcurrentRPCs:          4,
		MaxConcurrentStreams:       4,
		MaxHeaderListBytes:         16 << 10,
		MaxReceiveMessageBytes:     4 << 20,
		MaxSendMessageBytes:        4 << 20,
		AccessLogSuccessSampleRate: 1,
	}
}

func startTestServer(
	t *testing.T,
	cfg Config,
	register RegisterService,
) (*Server, *grpc.ClientConn) {
	t.Helper()

	return startTestServerWithOptions(t, cfg, register, Options{})
}

func startTestServerWithOptions(
	t *testing.T,
	cfg Config,
	register RegisterService,
	options Options,
) (*Server, *grpc.ClientConn) {
	t.Helper()

	services := append([]RegisterService(nil), options.Services...)
	if register != nil {
		services = append(services, register)
	}
	options.Services = services
	server, err := NewServer(cfg, options)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	listener := bufconn.Listen(1 << 20)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	connection, err := grpc.NewClient(
		"passthrough:///grpcx-test",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		if closeErr := server.Close(); closeErr != nil {
			t.Errorf("Server.Close() after client setup failure = %v", closeErr)
		}
		_ = listener.Close()
		t.Fatalf("grpc.NewClient() error = %v", err)
	}

	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
		}
		if err := server.Close(); err != nil {
			t.Errorf("Server.Close() error = %v", err)
		}
		if err := <-serveDone; err != nil {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})

	return server, connection
}

const testUnaryFullMethod = "/grpcx.test.Service/Call"

type testUnaryServiceServer interface {
	Call(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error)
}

type testUnaryService struct {
	call func(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
}

func (s testUnaryService) Call(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error) {
	return s.call(ctx, request)
}

func registerTestUnaryService( //nolint:dupl // Manual descriptors intentionally mirror generated unary handlers.
	registrar grpc.ServiceRegistrar,
	service testUnaryServiceServer,
) {
	registrar.RegisterService(&grpc.ServiceDesc{
		ServiceName: "grpcx.test.Service",
		HandlerType: (*testUnaryServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Call",
				Handler: func(
					implementation any,
					ctx context.Context,
					decode func(any) error,
					interceptor grpc.UnaryServerInterceptor,
				) (any, error) {
					request := new(emptypb.Empty)
					if err := decode(request); err != nil {
						return nil, err
					}
					service, ok := implementation.(testUnaryServiceServer)
					if !ok {
						return nil, errors.New("test service implementation has unexpected type")
					}
					handler := func(ctx context.Context, decoded any) (any, error) {
						typedRequest, ok := decoded.(*emptypb.Empty)
						if !ok {
							return nil, errors.New("test request has unexpected type")
						}
						return service.Call(ctx, typedRequest)
					}
					if interceptor == nil {
						return handler(ctx, request)
					}
					return interceptor(ctx, request, &grpc.UnaryServerInfo{
						Server:     implementation,
						FullMethod: testUnaryFullMethod,
					}, handler)
				},
			},
		},
	}, service)
}

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if got := status.Code(err); got != want {
		t.Fatalf("status code = %s, want %s (error %v)", got, want, err)
	}
}
