package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"

	// profile:grpc:start
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/getkin/kin-openapi/openapi3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	// profile:grpc:end
)

func TestAuthnBootstrapOrder(t *testing.T) {
	t.Run("initial trust failure constructs no serving surface", func(t *testing.T) {
		resetShutdownConfigEnv(t)

		trustFailure := errors.New("test initial trust failure")
		var events []string
		wiring := testRuntimeWiring()
		wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
			return runtimeDependencies{}, nil
		}
		wiring.initAuthn = func(
			_ context.Context,
			cfg config.Config,
			_ *telemetry.Metrics,
			_ *slog.Logger,
		) (authnRuntime, error) {
			assertAuthnBootstrapConfig(t, cfg)
			events = append(events, "initial_trust")
			return nil, trustFailure
		}
		wiring.authnStage = func(stage authnBootstrapStage) {
			events = append(events, string(stage))
		}
		wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error {
			events = append(events, "listener", "admission")
			return nil
		}

		err := runWithRuntime(nil, wiring)
		if !errors.Is(err, trustFailure) {
			t.Fatalf("runWithRuntime() error = %v, want initial trust failure", err)
		}
		if !slices.Equal(events, []string{"initial_trust"}) {
			t.Fatalf("bootstrap events = %v, want no router, server, listener, or admission after trust failure", events)
		}
	})

	t.Run("successful trust precedes serving surfaces and admission", func(t *testing.T) {
		resetShutdownConfigEnv(t)

		stopServing := errors.New("test serving stop")
		var events []string
		wiring := testRuntimeWiring()
		wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
			return runtimeDependencies{}, nil
		}
		wiring.initAuthn = func(
			_ context.Context,
			cfg config.Config,
			_ *telemetry.Metrics,
			_ *slog.Logger,
		) (authnRuntime, error) {
			assertAuthnBootstrapConfig(t, cfg)
			events = append(events, "initial_trust")
			return fakeAuthnRuntime{}, nil
		}
		wiring.authnStage = func(stage authnBootstrapStage) {
			events = append(events, string(stage))
		}
		wiring.serve = func(_ context.Context, _ context.Context, args serveRuntimeArgs) error {
			events = append(events, "listener")
			args.admission.MarkReady()
			events = append(events, "admission")
			return stopServing
		}

		err := runWithRuntime(nil, wiring)
		if !errors.Is(err, stopServing) {
			t.Fatalf("runWithRuntime() error = %v, want test serving stop", err)
		}
		want := []string{
			"initial_trust",
			string(authnStageTrustEstablished),
			string(authnStageHTTPRouterBuilt),
			string(authnStageHTTPServerBuilt),
			"listener",
			"admission",
		}
		if !slices.Equal(events, want) {
			t.Fatalf("bootstrap events = %v, want %v", events, want)
		}
	})

	t.Run("server failure after transfer closes authn once", func(t *testing.T) {
		resetShutdownConfigEnv(t)
		runtime := &recordingAuthnRuntime{}
		var events []string
		wiring := testRuntimeWiring()
		wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
			return runtimeDependencies{}, nil
		}
		wiring.initAuthn = func(context.Context, config.Config, *telemetry.Metrics, *slog.Logger) (authnRuntime, error) {
			events = append(events, "initial_trust")
			return runtime, nil
		}
		wiring.authnStage = func(stage authnBootstrapStage) {
			events = append(events, string(stage))
		}
		listenFail := errors.New("test listen failure")
		wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error {
			events = append(events, "listener")
			return listenFail
		}
		err := runWithRuntime(nil, wiring)
		if !errors.Is(err, listenFail) {
			t.Fatalf("runWithRuntime() error = %v, want listen failure", err)
		}
		if runtime.closes.Load() != 1 {
			t.Fatalf("authn closes = %d, want 1", runtime.closes.Load())
		}
		if !slices.Contains(events, string(authnStageTrustEstablished)) {
			t.Fatalf("events = %v", events)
		}
	})

	t.Run("drain closes authn before dependencies and telemetry", func(t *testing.T) {
		resetShutdownConfigEnv(t)
		runtime := &recordingAuthnRuntime{}
		var events []string
		wiring := testRuntimeWiring()
		wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
			return runtimeDependencies{}, nil
		}
		wiring.initAuthn = func(context.Context, config.Config, *telemetry.Metrics, *slog.Logger) (authnRuntime, error) {
			return runtime, nil
		}
		wiring.lifecycle = func(stage runtimeLifecycleStage) {
			if runtime.closes.Load() > 0 {
				events = append(events, "authn_closed", string(stage))
				return
			}
			events = append(events, string(stage))
		}
		wiring.serve = func(_ context.Context, _ context.Context, args serveRuntimeArgs) error {
			args.admission.MarkReady()
			return errors.New("test drain")
		}
		_ = runWithRuntime(nil, wiring)
		if runtime.closes.Load() != 1 {
			t.Fatalf("authn closes = %d, want 1", runtime.closes.Load())
		}
		if !slices.Contains(events, string(runtimeLifecycleHTTPDrained)) {
			t.Fatalf("missing drain event: %v", events)
		}
		drained := slices.Index(events, string(runtimeLifecycleHTTPDrained))
		closed := slices.Index(events, "authn_closed")
		deps := slices.Index(events, string(runtimeLifecycleDependenciesClosed))
		if closed < 0 || drained < 0 || closed < drained {
			t.Fatalf("authn close was not after drain: %v", events)
		}
		if deps >= 0 && deps < closed {
			t.Fatalf("dependencies closed before authn: %v", events)
		}
		runtime.Close()
		if runtime.closes.Load() != 1 {
			t.Fatalf("repeated close was not idempotent: %d", runtime.closes.Load())
		}
	})

	t.Run("forced drain cancels in-flight provider work before authn close", func(t *testing.T) {
		resetShutdownConfigEnv(t)
		started := make(chan struct{})
		finished := make(chan error, 1)
		runtime := &recordingAuthnRuntime{block: started}
		wiring := testRuntimeWiring()
		wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
			return runtimeDependencies{}, nil
		}
		wiring.initAuthn = func(context.Context, config.Config, *telemetry.Metrics, *slog.Logger) (authnRuntime, error) {
			return runtime, nil
		}
		wiring.serve = func(ctx context.Context, _ context.Context, args serveRuntimeArgs) error {
			args.admission.MarkReady()
			go func() {
				_, err := runtime.ResolveHTTP(ctx, &openapi3filter.AuthenticationInput{})
				finished <- err
			}()
			<-started
			return ctx.Err()
		}
		_ = runWithRuntime(nil, wiring)
		err := <-finished
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("in-flight provider error = %v, want cancellation", err)
		}
		if runtime.closes.Load() != 1 {
			t.Fatalf("authn closes = %d, want 1", runtime.closes.Load())
		}
	})
}

type recordingAuthnRuntime struct {
	fakeAuthnRuntime

	closes atomic.Int64
	block  chan struct{}
}

func (r *recordingAuthnRuntime) Close() {
	r.closes.CompareAndSwap(0, 1)
}

func (r *recordingAuthnRuntime) ResolveHTTP(ctx context.Context, input *openapi3filter.AuthenticationInput) (reqctx.Principal, error) {
	if r.block != nil {
		close(r.block)
		<-ctx.Done()
		return reqctx.Principal{}, fmt.Errorf("provider canceled: %w", ctx.Err())
	}
	return r.fakeAuthnRuntime.ResolveHTTP(ctx, input)
}

// profile:grpc:start

func TestAuthnUsesIndependentTransportAdmission(t *testing.T) {
	entered := make(chan struct{}, 2)
	block := make(chan struct{})
	verifier := &independentAdmissionVerifier{entered: entered, block: block}
	runtime, err := bearerauthn.New(verifier, nil)
	if err != nil {
		t.Fatalf("bearerauthn.New() error = %v", err)
	}
	httpHandler := httpx.MaxInFlight(1, telemetry.ServerLoad{}, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		_, resolveErr := runtime.ResolveHTTP(request.Context(), &openapi3filter.AuthenticationInput{
			SecurityScheme:         &openapi3.SecurityScheme{Type: "http", Scheme: "bearer"},
			RequestValidationInput: &openapi3filter.RequestValidationInput{Request: request},
		})
		if resolveErr != nil {
			http.Error(w, resolveErr.Error(), http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	const method = "/bootstrap.test.Admission/Call"
	server, err := grpcx.NewServer(grpcx.Options{
		UnaryPolicy: []grpc.UnaryServerInterceptor{runtime.UnaryInterceptor()},
		Services: []grpcx.RegisterService{
			func(registrar grpc.ServiceRegistrar) {
				grpctest.Register(registrar, grpctest.Unary(
					method,
					func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
						return &emptypb.Empty{}, nil
					},
				))
			},
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	connection := grpctest.ServeBufconn(t, server)
	credential := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer token"))

	httpDone := make(chan struct{})
	go func() {
		defer close(httpDone)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/secure", nil)
		request.Header.Set("Authorization", "Bearer token")
		httpHandler.ServeHTTP(httptest.NewRecorder(), request)
	}()
	grpcDone := make(chan error, 1)
	go func() {
		grpcDone <- connection.Invoke(credential, method, &emptypb.Empty{}, &emptypb.Empty{})
	}()
	waittest.ReceiveSignal(t, entered, 2*time.Second, "first transport to enter verifier")
	waittest.ReceiveSignal(t, entered, 2*time.Second, "second transport to enter verifier")
	if verifier.calls.Load() != 2 || verifier.peak.Load() != 2 {
		t.Fatalf("verifier calls/peak = %d/%d, want 2/2", verifier.calls.Load(), verifier.peak.Load())
	}
	close(block)
	<-httpDone
	if err := waittest.Receive(t, grpcDone, 2*time.Second, "gRPC call to finish"); err != nil {
		t.Fatalf("gRPC invoke error = %v", err)
	}
}

type independentAdmissionVerifier struct {
	entered  chan struct{}
	block    <-chan struct{}
	calls    atomic.Int64
	inFlight atomic.Int64
	peak     atomic.Int64
}

func (v *independentAdmissionVerifier) Verify(ctx context.Context, _ string) (bearerauthn.Result, error) {
	v.calls.Add(1)
	current := v.inFlight.Add(1)
	defer v.inFlight.Add(-1)
	for {
		previous := v.peak.Load()
		if current <= previous || v.peak.CompareAndSwap(previous, current) {
			break
		}
	}
	if v.entered != nil {
		select {
		case v.entered <- struct{}{}:
		default:
		}
	}
	if v.block != nil {
		select {
		case <-v.block:
		case <-ctx.Done():
			return bearerauthn.Result{}, fmt.Errorf("wait for test barrier: %w", ctx.Err())
		}
	}
	return bearerauthn.Result{Principal: reqctx.Principal{Subject: "bootstrap-test-subject"}}, nil
}

func (v *independentAdmissionVerifier) Close() {}

// profile:grpc:end

func assertAuthnBootstrapConfig(t *testing.T, cfg config.Config) {
	t.Helper()
	if cfg.Authn.Issuer != "https://issuer.example.com" ||
		cfg.Authn.Audience != "service-api" {
		t.Fatalf("validated authn config = %+v, want exact test policy", cfg.Authn)
	}
	// profile:authn-oidc-jwt:start
	if cfg.Authn.TokenProfile != "resource-server" {
		t.Fatalf("validated authn token profile = %q, want resource-server", cfg.Authn.TokenProfile)
	}
	// profile:authn-oidc-jwt:end
}

type fakeAuthnRuntime struct{}

func (fakeAuthnRuntime) Close() {}

func (fakeAuthnRuntime) ResolveHTTP(
	_ context.Context,
	input *openapi3filter.AuthenticationInput,
) (reqctx.Principal, error) {
	if input != nil &&
		input.RequestValidationInput != nil &&
		input.RequestValidationInput.Request != nil {
		input.RequestValidationInput.Request.Header.Del("Authorization")
	}
	return reqctx.Principal{Subject: "bootstrap-test-subject"}, nil
}

// profile:grpc:start

func (fakeAuthnRuntime) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		return handler(ctx, request)
	}
}

func (fakeAuthnRuntime) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return handler(server, stream)
	}
}

// profile:grpc:end
