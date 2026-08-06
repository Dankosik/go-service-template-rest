package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/problem"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type grpcRuntimeBindings struct {
	Services     []grpcx.RegisterService
	UnaryPolicy  []grpc.UnaryServerInterceptor
	StreamPolicy []grpc.StreamServerInterceptor
}

func newGRPCRuntime(
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	domainErrors []problem.Mapper,
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
		loaded, err := credentials.NewServerTLSFromFile(
			cfg.GRPC.Server.TLS.CertFile,
			cfg.GRPC.Server.TLS.KeyFile,
		)
		if err != nil {
			return nil, fmt.Errorf("load gRPC server TLS credentials: %w", err)
		}
		transportCredentials = loaded
	}

	server, err := grpcx.NewServer(
		grpcServerConfig(cfg.GRPC.Server),
		grpcx.Options{
			TransportCredentials: transportCredentials,

			Logger:         log,
			MeterProvider:  metrics.MeterProvider(),
			TracerProvider: otel.GetTracerProvider(),
			Propagators:    propagation.TraceContext{},
			// The same slice the HTTP router receives. A service appends its
			// mappers at runtimeDependencies.DomainErrors, never here.
			DomainErrors: domainErrors,
			Load:         metrics.GRPCServerLoad(),
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

// grpcServerConfig is this service's one crossing from loaded configuration to
// the transport adapter's bounds. It is a named function rather than a literal
// inside newGRPCRuntime so that the crossing has an owner a test can drive
// directly: TestGRPCServerConfigFillsEveryTransportBound proves a populated
// configuration leaves no bound behind, which a test asserting individual fields
// goes on reporting as fine once a tenth one is added.
//
// The mapping lives here, in the composition root, rather than in
// internal/infra/grpc, because that adapter depends on internal/problem and
// internal/reqctx only — docs/repo-architecture.md owns that direction, and a
// transport adapter that knew this repository's configuration shape could not be
// reused by a service that restructured it. internal/config cannot own it
// either: the depguard rule config_no_runtime_owners stops it importing runtime
// adapters. Two other composition roots perform the same crossing for their own
// bounds, each with its own literal — examples/grpc-reference-service's
// benchmark server and its service test.
func grpcServerConfig(server config.GRPCServerConfig) grpcx.Config {
	return grpcx.Config{
		MaxConcurrentRPCs:          server.MaxConcurrentRPCs,
		MaxConcurrentStreams:       server.MaxConcurrentStreams,
		MaxHeaderListBytes:         server.MaxHeaderListBytes,
		MaxReceiveMessageBytes:     server.MaxReceiveMessageBytes,
		MaxSendMessageBytes:        server.MaxSendMessageBytes,
		LogHealthChecks:            server.AccessLogHealthChecks,
		AccessLogSuccessSampleRate: server.AccessLogSuccessSampleRate,
		AccessLogSlowThreshold:     server.AccessLogSlowThreshold,
		TelemetryHealthChecks:      server.TelemetryHealthChecks,
	}
}
