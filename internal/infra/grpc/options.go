package grpcx

import (
	"fmt"
	"log/slog"

	"buf.build/go/protovalidate"
	"github.com/example/go-service-template-rest/internal/failure"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/stats"
)

// RegisterService attaches one generated implementation before serving starts.
type RegisterService func(grpc.ServiceRegistrar)

// Options contains only collaborators the transport cannot safely invent.
type Options struct {
	TransportCredentials credentials.TransportCredentials
	Logger               *slog.Logger
	MeterProvider        metric.MeterProvider
	TracerProvider       trace.TracerProvider
	DomainErrors         []failure.Mapper
	Services             []RegisterService
	UnaryPolicy          []grpc.UnaryServerInterceptor
	StreamPolicy         []grpc.StreamServerInterceptor
}

func withOptionDefaults(options Options) Options {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.MeterProvider == nil {
		options.MeterProvider = metricnoop.NewMeterProvider()
	}
	if options.TracerProvider == nil {
		options.TracerProvider = tracenoop.NewTracerProvider()
	}
	return options
}

// NewServer builds the complete native server product with fixed safe defaults.
func NewServer(options Options) (*Server, error) {
	return newServer(defaultServerConfig(), options)
}

// newServer exists only so focused tests can lower a bound without exporting it
// as service configuration.
func newServer(cfg serverConfig, options Options) (*Server, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	validator, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("build protobuf validator: %w", err)
	}
	options = withOptionDefaults(options)
	admission := newAdmissionPolicy(
		cfg.maxConcurrentRPCs,
		cfg.maxConcurrentHealthRPCs,
		newServerLoad(options.MeterProvider),
	)
	healthDrain := newHealthDrain()
	registeredMethods := make(methodSet)
	handlerErrors := handlerErrorBoundary(options.Logger, options.DomainErrors)

	serverOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(cfg.maxConcurrentStreams),
		grpc.MaxHeaderListSize(cfg.maxHeaderListBytes),
		grpc.MaxRecvMsgSize(cfg.maxReceiveMessageBytes),
		grpc.MaxSendMsgSize(cfg.maxSendMessageBytes),
		grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithMeterProvider(options.MeterProvider),
			otelgrpc.WithTracerProvider(options.TracerProvider),
			otelgrpc.WithPropagators(propagation.TraceContext{}),
			otelgrpc.WithFilter(func(info *stats.RPCTagInfo) bool {
				_, known := registeredMethods[info.FullMethodName]
				return known && !isHealthMethod(info.FullMethodName)
			}),
		)),
		grpc.StatsHandler(admission.statsHandler()),
		grpc.ChainUnaryInterceptor(unaryChain(
			options.Logger,
			admission,
			cfg.unaryTimeout,
			options.UnaryPolicy,
			validator,
			handlerErrors,
		)...),
		grpc.ChainStreamInterceptor(streamChain(
			options.Logger,
			healthDrain,
			admission,
			options.StreamPolicy,
			validator,
			handlerErrors,
		)...),
	}
	if options.TransportCredentials != nil {
		serverOptions = append(serverOptions, grpc.Creds(options.TransportCredentials))
	}

	nativeServer := grpc.NewServer(serverOptions...)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
	healthgrpc.RegisterHealthServer(nativeServer, healthServer)
	if err := registerServices(nativeServer, options.Services, registeredMethods); err != nil {
		healthDrain.stop()
		nativeServer.Stop()
		return nil, err
	}
	return &Server{
		server:      nativeServer,
		health:      healthServer,
		healthDrain: healthDrain,
		drain:       admission.drain,
		stopDone:    make(chan struct{}),
	}, nil
}

func registerServices(server *grpc.Server, services []RegisterService, methods methodSet) error {
	for index, register := range services {
		if register == nil {
			return fmt.Errorf("build gRPC server: service registration at index %d is nil", index)
		}
		register(server)
	}
	for serviceName, service := range server.GetServiceInfo() {
		for _, method := range service.Methods {
			methods["/"+serviceName+"/"+method.Name] = struct{}{}
		}
	}
	return nil
}

type methodSet map[string]struct{}
