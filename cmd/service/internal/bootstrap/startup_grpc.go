package bootstrap

import (
	"fmt"
	"log/slog"
	"math"

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
	Services        []grpcx.RegisterService
	UnaryPolicy     []grpc.UnaryServerInterceptor
	StreamingPolicy []grpc.StreamServerInterceptor
}

func newGRPCRuntime(
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	domainErrors []problem.Mapper,
	bindings grpcRuntimeBindings,
) (*grpcx.Server, error) {
	maxConcurrentStreams, err := grpcUint32Bound(
		"grpc.server.max_concurrent_streams",
		cfg.GRPC.Server.MaxConcurrentStreams,
	)
	if err != nil {
		return nil, err
	}
	maxHeaderListBytes, err := grpcUint32Bound(
		"grpc.server.max_header_list_bytes",
		cfg.GRPC.Server.MaxHeaderListBytes,
	)
	if err != nil {
		return nil, err
	}

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
		grpcx.Config{
			MaxConcurrentRPCs:          cfg.GRPC.Server.MaxConcurrentRPCs,
			MaxConcurrentStreams:       maxConcurrentStreams,
			MaxHeaderListBytes:         maxHeaderListBytes,
			MaxReceiveMessageBytes:     cfg.GRPC.Server.MaxReceiveMessageBytes,
			MaxSendMessageBytes:        cfg.GRPC.Server.MaxSendMessageBytes,
			LogHealthChecks:            cfg.GRPC.Server.AccessLogHealthChecks,
			AccessLogSuccessSampleRate: cfg.GRPC.Server.AccessLogSuccessSampleRate,
			AccessLogSlowThreshold:     cfg.GRPC.Server.AccessLogSlowThreshold,
			TelemetryHealthChecks:      cfg.GRPC.Server.TelemetryHealthChecks,
			TransportCredentials:       transportCredentials,
		},
		grpcx.Options{
			Logger:          log,
			MeterProvider:   metrics.MeterProvider(),
			TracerProvider:  otel.GetTracerProvider(),
			Propagators:     propagation.TraceContext{},
			DomainErrors:    domainErrors,
			Load:            metrics.GRPCServerLoad(),
			Services:        bindings.Services,
			UnaryPolicy:     bindings.UnaryPolicy,
			StreamingPolicy: bindings.StreamingPolicy,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("build gRPC server: %w", err)
	}
	return server, nil
}

func grpcUint32Bound(name string, value int) (uint32, error) {
	if value <= 0 || uint64(value) > math.MaxUint32 {
		return 0, fmt.Errorf(
			"build gRPC server: %s must be in range [1,%d]",
			name,
			uint64(math.MaxUint32),
		)
	}
	return uint32(value), nil // #nosec G115 -- range checked immediately above.
}
