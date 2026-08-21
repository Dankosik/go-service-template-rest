package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"

	// profile:grpc:start
	"google.golang.org/grpc"
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
}

func assertAuthnBootstrapConfig(t *testing.T, cfg config.Config) {
	t.Helper()
	if cfg.Authn.Issuer != "https://issuer.example.com" ||
		cfg.Authn.Audience != "service-api" ||
		cfg.Authn.TokenProfile != "resource-server" {
		t.Fatalf("validated authn config = %+v, want exact test policy", cfg.Authn)
	}
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
