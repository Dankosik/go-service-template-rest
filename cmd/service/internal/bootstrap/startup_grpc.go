package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/failure"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
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

// serviceGRPCBindings is where this service composes its own gRPC surface;
// generated handlers stay outside grpcx.
//
// Assigning Services is correct — nothing else fills it. A policy is appended
// instead: a build profile may have filled the policy slices already, and an
// assignment compiles, passes every check, and silently drops what it replaced.
func serviceGRPCBindings(
	// profile:authn-oidc-jwt:start
	authn authnRuntime,
	// profile:authn-oidc-jwt:end
) grpcRuntimeBindings {
	bindings := grpcRuntimeBindings{
		// Register an owned service here, as
		// func(registrar grpc.ServiceRegistrar) { foov1.RegisterFooServer(registrar, impl) }.
		// See docs/grpc/runtime-and-streaming.md, "Register it in bootstrap".
		Services: nil,
	}
	// profile:authn-oidc-jwt:start
	bindings.UnaryPolicy = append(bindings.UnaryPolicy, authn.UnaryInterceptor())
	bindings.StreamPolicy = append(bindings.StreamPolicy, authn.StreamInterceptor())
	// profile:authn-oidc-jwt:end
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
		// startup_grpc_tls.go owns the settings themselves: the version floor,
		// per-handshake certificate reload, and optional client-certificate
		// verification.
		settings, err := grpcServerTLS(cfg.GRPC.Server.TLS, log)
		if err != nil {
			return nil, fmt.Errorf("load gRPC server TLS credentials: %w", err)
		}
		transportCredentials = credentials.NewTLS(settings)
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
			// The service's own identity, scoping the machine-readable reason a
			// classified error carries. It comes from the telemetry service name
			// because that is this repository's only configured service identity,
			// and sharing it means an ErrorInfo domain and a trace's service
			// attribute name the same thing. Renaming it for telemetry reasons
			// therefore changes a value remote callers match on.
			ErrorDomain:  cfg.Observability.OTel.ServiceName,
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
// The mapping lives here rather than in internal/infra/grpc because that adapter
// depends on internal/failure and internal/reqctx only, and a transport adapter
// that knew this repository's configuration shape could not be reused by a
// service that restructured it. internal/config cannot own it either: the
// depguard rule config_no_runtime_owners stops it importing runtime adapters.
//
// That is also why the crossing exists twice more, each unreachable from the
// others: serverConfigFromRuntime in internal/infra/grpc/config_parity_test.go
// is that package's oracle, and settingsFromDefaults in the benchmark server
// under examples/grpc-reference-service builds the same bounds for a measured
// run. A bound added to grpcx.Config belongs in all three; each has its own
// target-side test so a missed one fails there rather than here.
func grpcServerConfig(server config.GRPCServerConfig) grpcx.Config {
	return grpcx.Config{
		MaxConcurrentRPCs: server.MaxConcurrentRPCs,
		// The standard health service's budget is the connection limit, because
		// one Health/Watch per connection is what grpc-go's client-side health
		// checker opens and what this repository's own client enables by default.
		// Sizing it from anything smaller would shed a well-behaved peer's watch,
		// which costs that caller the whole backend rather than one RPC.
		MaxConcurrentHealthRPCs:    server.MaxConnections,
		MaxConcurrentStreams:       server.MaxConcurrentStreams,
		MaxHeaderListBytes:         server.MaxHeaderListBytes,
		MaxReceiveMessageBytes:     server.MaxReceiveMessageBytes,
		MaxSendMessageBytes:        server.MaxSendMessageBytes,
		AccessLogHealthChecks:      server.AccessLogHealthChecks,
		AccessLogSuccessSampleRate: server.AccessLogSuccessSampleRate,
		AccessLogSlowThreshold:     server.AccessLogSlowThreshold,
		TelemetryHealthChecks:      server.TelemetryHealthChecks,
		UnaryTimeout:               server.UnaryTimeout,
		StreamTimeout:              server.StreamTimeout,
		MaxConnectionIdle:          server.MaxConnectionIdle,
		ServerPingInterval:         server.ServerPingInterval,
		ServerPingTimeout:          server.ServerPingTimeout,
		MinClientPingInterval:      server.MinClientPingInterval,
		PermitPingWithoutStream:    server.PermitPingWithoutStream,
		MaxConnectionAge:           server.MaxConnectionAge,
		MaxConnectionAgeGrace:      server.MaxConnectionAgeGrace,
	}
}
