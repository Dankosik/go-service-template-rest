// Server lifecycle and the error boundaries around it: does health follow
// admission and drain, does Shutdown give up on a handler that ignores
// cancellation, and does each boundary trust exactly the errors it should?
//
// The trust rules are proven twice on purpose — as units in interceptors_test.go
// and here through a real server, because only the second shows what a caller
// actually receives.

package grpcx

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/waittest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServerHealthFollowsAdmissionAndDrain(t *testing.T) {
	server, connection := startTestServer(t, testServerConfig(), nil)
	healthClient := healthgrpc.NewHealthClient(connection)

	assertHealthStatus(t, healthClient, healthgrpc.HealthCheckResponse_NOT_SERVING)
	server.SetServing(true)
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

// policyBoundaryCase is the contrast the policy error boundary exists to draw:
// a status the policy chose reaches the caller intact, and anything else becomes
// a sanitized INTERNAL. Both RPC kinds run the same two rows, because the
// boundary is one rule and the kinds differ only in how the caller drives them.
type policyBoundaryCase struct {
	name       string
	policyErr  error
	wantCode   codes.Code
	wantDetail string
}

func TestServerPolicyErrorBoundary(t *testing.T) {
	for _, testCase := range []policyBoundaryCase{
		{
			name:       "chosen status is service-owned output",
			policyErr:  status.Error(codes.Unauthenticated, "authentication required"),
			wantCode:   codes.Unauthenticated,
			wantDetail: "authentication required",
		},
		{
			name:       "raw error is sanitized",
			policyErr:  errors.New("authentication dependency credential=secret"),
			wantCode:   codes.Internal,
			wantDetail: "request failed",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handlerCalled := false
			register := func(registrar grpc.ServiceRegistrar) {
				registerUnaryTestService(registrar, testUnaryFullMethod,
					func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
						handlerCalled = true
						return &emptypb.Empty{}, nil
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
						return nil, testCase.policyErr
					},
				},
			})

			err := connection.Invoke(
				t.Context(),
				testUnaryFullMethod,
				&emptypb.Empty{},
				&emptypb.Empty{},
			)
			assertStatusCode(t, err, testCase.wantCode)
			if detail := status.Convert(err).Message(); detail != testCase.wantDetail {
				t.Fatalf("policy error detail = %q, want %q", detail, testCase.wantDetail)
			}
			if handlerCalled {
				t.Fatal("rejecting policy entered the business handler")
			}
		})
	}
}

func TestServerStreamingPolicyErrorBoundary(t *testing.T) {
	for _, testCase := range []policyBoundaryCase{
		{
			name:       "chosen status is service-owned output",
			policyErr:  status.Error(codes.PermissionDenied, "stream access denied"),
			wantCode:   codes.PermissionDenied,
			wantDetail: "stream access denied",
		},
		{
			name:       "raw error is sanitized",
			policyErr:  errors.New("authorization dependency credential=secret"),
			wantCode:   codes.Internal,
			wantDetail: "request failed",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, connection := startTestServerWithOptions(t, testServerConfig(), nil, Options{
				StreamPolicy: []grpc.StreamServerInterceptor{
					func(
						any,
						grpc.ServerStream,
						*grpc.StreamServerInfo,
						grpc.StreamHandler,
					) error {
						return testCase.policyErr
					},
				},
			})
			healthClient := healthgrpc.NewHealthClient(connection)

			stream, err := healthClient.Watch(t.Context(), &healthgrpc.HealthCheckRequest{})
			if err == nil {
				_, err = stream.Recv()
			}
			assertStatusCode(t, err, testCase.wantCode)
			if detail := status.Convert(err).Message(); detail != testCase.wantDetail {
				t.Fatalf("streaming policy error detail = %q, want %q", detail, testCase.wantDetail)
			}
		})
	}
}

func TestServerSanitizesUntrustedHandlerStatus(t *testing.T) {
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return nil, status.Error(codes.PermissionDenied, "dependency credential=secret")
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

func TestServerPreservesGeneratedUnimplementedStatusWithoutItsText(t *testing.T) {
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return nil, status.Error(codes.Unimplemented, "generated implementation detail")
			})
	}
	_, connection := startTestServer(t, testServerConfig(), register)
	err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	assertStatusCode(t, err, codes.Unimplemented)
	if got := status.Convert(err).Message(); got != "method not implemented" {
		t.Fatalf("status detail = %q", got)
	}
}

func TestServerShutdownForcesBlockedRPCOnCanceledBudget(t *testing.T) {
	entered := make(chan struct{})
	handlerDone := make(chan struct{})
	var enteredOnce sync.Once
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
				enteredOnce.Do(func() { close(entered) })
				<-ctx.Done()
				close(handlerDone)
				return nil, ctx.Err()
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
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				close(entered)
				<-release
				close(handlerDone)
				return &emptypb.Empty{}, nil
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

	if err := waittest.Receive(t, shutdownDone, time.Second, "Shutdown to honor the canceled shutdown budget"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Shutdown() error = %v, want context.Canceled", err)
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
		mutate func(*serverConfig)
	}{
		{name: "RPCs", mutate: func(cfg *serverConfig) { cfg.maxConcurrentRPCs = 0 }},
		{name: "health RPCs", mutate: func(cfg *serverConfig) { cfg.maxConcurrentHealthRPCs = 0 }},
		{name: "streams", mutate: func(cfg *serverConfig) { cfg.maxConcurrentStreams = 0 }},
		{name: "headers", mutate: func(cfg *serverConfig) { cfg.maxHeaderListBytes = 0 }},
		{name: "receive message", mutate: func(cfg *serverConfig) { cfg.maxReceiveMessageBytes = 0 }},
		{name: "send message", mutate: func(cfg *serverConfig) { cfg.maxSendMessageBytes = 0 }},
		{name: "unary timeout", mutate: func(cfg *serverConfig) { cfg.unaryTimeout = 0 }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := valid
			testCase.mutate(&cfg)
			if _, err := newServer(cfg, Options{}); err == nil {
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

// A nil service registration is a composition defect, and skipping it would
// produce a server that starts and serves without the method it was meant to
// publish — which no probe and no test of the other services can see.
func TestNewServerRejectsNilServiceRegistration(t *testing.T) {
	server, err := NewServer(Options{
		Services: []RegisterService{nil},
	})
	if err == nil {
		t.Fatal("NewServer() accepted a nil service registration")
	}
	if server != nil {
		t.Fatal("NewServer() returned a server alongside its error")
	}
}
