package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/failure"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type grpcRuntimeBindings struct {
	Services     []grpcx.RegisterService
	UnaryPolicy  []grpc.UnaryServerInterceptor
	StreamPolicy []grpc.StreamServerInterceptor
}

// serviceGRPCBindings is where this service composes its own gRPC surface;
// generated handlers stay outside grpcx.
//
// Assigning Services is correct — nothing else fills it. A policy is appended
// instead: a build profile may have filled the policy slices already, and an
// assignment compiles, passes every check, and silently drops what it replaced.
func serviceGRPCBindings(
	// profile:authn-bearer:start
	authn authnRuntime,
	// profile:authn-bearer:end
) grpcRuntimeBindings {
	bindings := grpcRuntimeBindings{
		// Register an owned service here, as
		// func(registrar grpc.ServiceRegistrar) { foov1.RegisterFooServer(registrar, impl) }.
		// See docs/grpc/runtime-and-streaming.md, "Register it in bootstrap".
		Services: nil,
	}
	// profile:authn-bearer:start
	bindings.UnaryPolicy = append(bindings.UnaryPolicy, authn.UnaryInterceptor())
	bindings.StreamPolicy = append(bindings.StreamPolicy, authn.StreamInterceptor())
	// profile:authn-bearer:end
	return bindings
}

func newGRPCRuntime(
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	domainErrors []failure.Mapper,
	bindings grpcRuntimeBindings,
) (*grpcx.Server, error) {
	// internal/config owns which values this field may hold and refuses every
	// other one before startup reaches here, so the else branch below is
	// plaintext by proof rather than by fallback. A third mode is a change there
	// first: the accepted set, the rules that come with it, and the logged value
	// in startup_logging.go all belong to that owner, and this switch only turns
	// an already-proven value into credentials.
	var transportCredentials credentials.TransportCredentials
	if cfg.GRPC.Server.TransportSecurity == "tls" {
		settings, err := grpcServerTLS(cfg.GRPC.Server.TLS)
		if err != nil {
			return nil, fmt.Errorf("load gRPC server TLS credentials: %w", err)
		}
		transportCredentials = credentials.NewTLS(settings)
	}

	server, err := grpcx.NewServer(
		grpcx.Options{
			TransportCredentials: transportCredentials,

			Logger:         log,
			MeterProvider:  metrics.MeterProvider(),
			TracerProvider: otel.GetTracerProvider(),
			// The same slice the HTTP router receives. A service appends its
			// mappers at runtimeDependencies.DomainErrors, never here.
			DomainErrors: domainErrors,
			Services:     bindings.Services,
			UnaryPolicy:  bindings.UnaryPolicy,
			StreamPolicy: bindings.StreamPolicy,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("build gRPC runtime: %w", err)
	}
	return server, nil
}
