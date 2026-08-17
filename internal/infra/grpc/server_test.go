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
	"math"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/example/go-service-template-rest/internal/waittest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServerHealthFollowsAdmissionAndDrain(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
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

func TestServerCorrelationReturnsAcceptedRequestID(t *testing.T) {
	t.Parallel()
	observed := make(chan string, 2)
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
				observed <- reqctx.RequestID(ctx)
				return &emptypb.Empty{}, nil
			})
	}
	_, connection := startTestServer(t, testServerConfig(), register)

	for _, testCase := range []struct {
		name       string
		candidates []string
		want       string
	}{
		{name: "accepted", candidates: []string{"request_123"}, want: "request_123"},
		{name: "replaced", candidates: []string{"user@example.com"}},
		{name: "absent"},
		{
			// Two values would make the accepted identifier depend on an
			// ordering the caller controls, so neither is a candidate.
			name:       "duplicated",
			candidates: []string{"request_123", "request_456"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			pairs := make([]string, 0, 2*len(testCase.candidates))
			for _, candidate := range testCase.candidates {
				pairs = append(pairs, requestIDMetadataKey, candidate)
			}
			ctx := metadata.NewOutgoingContext(
				t.Context(),
				metadata.Pairs(pairs...),
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
			if testCase.want == "" && slices.Contains(testCase.candidates, got[0]) {
				t.Fatalf("unaccepted request ID %q was echoed", got[0])
			}
			if handlerRequestID := <-observed; handlerRequestID != got[0] {
				t.Fatalf("handler request ID = %q, want %q", handlerRequestID, got[0])
			}
		})
	}
}

func TestServerShutdownForcesBlockedRPCOnCanceledBudget(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
			t.Parallel()
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

// A nil service registration is a composition defect, and skipping it would
// produce a server that starts and serves without the method it was meant to
// publish — which no probe and no test of the other services can see.
func TestNewServerRejectsNilServiceRegistration(t *testing.T) {
	t.Parallel()
	server, err := NewServer(testServerConfig(), Options{
		Services: []RegisterService{nil},
	})
	if err == nil {
		t.Fatal("NewServer() accepted a nil service registration")
	}
	if server != nil {
		t.Fatal("NewServer() returned a server alongside its error")
	}
}
