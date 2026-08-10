package grpcx

import (
	"context"
	"fmt"
	"log/slog"

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
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/stats"
)

// What a [Server] is built from, and the one assembly that builds it. Config
// owns the bounds a deployment tunes; this file owns the collaborators a
// composition root supplies and the order they are wired in. server.go picks up
// where NewServer returns, and holds nothing about construction: adding a
// grpc.ServerOption and changing what draining means are separate edits, and
// keeping them apart is what stops them from colliding in one file's history.

// RegisterService attaches one generated service implementation to a
// registrar. It is the seam that keeps generated handlers out of this package:
// the composition root closes over its feature and passes only this function.
type RegisterService func(grpc.ServiceRegistrar)

// Options supplies process-owned collaborators without making this transport
// import feature packages or the bootstrap composition root.
type Options struct {
	// TransportCredentials is nil for plaintext, which is a deliberate choice
	// by the composition root rather than a default: NewServer does not invent
	// transport security. It sits here rather than on [Config] because the
	// composition root builds it — from certificate files, a secret manager, or
	// a mesh identity — instead of copying a validated bound; the client half in
	// internal/infra/grpcclient puts it on its Options for the same reason.
	TransportCredentials credentials.TransportCredentials

	// Optional observability collaborators. Logger falls back to
	// slog.Default(), the two providers to their no-op implementations, and
	// Propagators to W3C trace context, so a test can leave all four unset.
	Logger         *slog.Logger
	MeterProvider  metric.MeterProvider
	TracerProvider trace.TracerProvider
	Propagators    propagation.TextMapPropagator

	// DomainErrors classifies handler errors that this transport should render
	// as something other than INTERNAL. It is the same slice the HTTP router
	// receives, so one domain identity answers consistently on both. An error
	// no mapper claims is sanitized, text included.
	DomainErrors []failure.Mapper

	// ErrorDomain scopes the machine-readable reason a classified error carries,
	// so two services' reasons cannot collide. It is the service's own identity,
	// which is why the composition root supplies it rather than this package
	// inventing one; empty omits the detail entirely.
	//
	// A configured value on this struct reaches nothing on its own, so
	// .golangci.yml lists Options for exhaustruct: a composition that forgets
	// this field fails lint rather than silently publishing statuses without it.
	ErrorDomain string

	// Load observes the admission decision for process-wide capacity signals.
	// It defaults to a recorder that does nothing.
	Load LoadRecorder

	// Services are registered during NewServer, before Serve; registration
	// cannot happen later. The package doc owns why.
	Services []RegisterService

	// UnaryPolicy and StreamPolicy carry authentication, authorization, and any
	// other cross-cutting policy the contract requires; this package supplies
	// none of them. The package doc owns how to fill them, their position in the
	// chain, and what that position means for a policy author.
	UnaryPolicy  []grpc.UnaryServerInterceptor
	StreamPolicy []grpc.StreamServerInterceptor
}

// withOptionDefaults fills the collaborators [Options] documents as optional, so
// the composition in NewServer reads as one uninterrupted sequence and every
// later reader of options sees a non-nil value.
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
	if options.Propagators == nil {
		options.Propagators = propagation.TraceContext{}
	}
	if options.Load == nil {
		options.Load = noopLoadRecorder{}
	}
	return options
}

// LoadRecorder observes process-wide admission decisions. Admitted returns the
// release function for one admitted business RPC; Shed reports business work
// rejected by its full budget, and HealthShed reports a health RPC rejected by
// its separate budget.
type LoadRecorder interface {
	Admitted(ctx context.Context) func()
	Shed(ctx context.Context)
	HealthShed(ctx context.Context)
}

// noopLoadRecorder is the [Options.Load] default, so the admission policy always
// has a recorder to call and never guards the field.
type noopLoadRecorder struct{}

func (noopLoadRecorder) Admitted(context.Context) func() {
	return func() {}
}

func (noopLoadRecorder) Shed(context.Context) {}

func (noopLoadRecorder) HealthShed(context.Context) {}

// NewServer builds a native gRPC server with standard health, OpenTelemetry,
// finite bounds, and the repository interceptor policies.
func NewServer(cfg Config, options Options) (*Server, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	options = withOptionDefaults(options)
	admission := newAdmissionPolicy(cfg.MaxConcurrentRPCs, cfg.MaxConcurrentHealthRPCs, options.Load)
	accessLogs := accessLogPolicy{
		logHealthChecks:   cfg.AccessLogHealthChecks,
		successSampleRate: cfg.AccessLogSuccessSampleRate,
		slowThreshold:     cfg.AccessLogSlowThreshold,
	}
	// otelgrpc derives rpc.method from the peer-supplied HTTP/2 path before
	// dispatch. Keep its server signals descriptor-backed so unknown paths cannot
	// create unbounded metric series or span names.
	//
	// The filter closure below captures this set while it is still empty;
	// registerServices fills it after registration and before Serve, and it is
	// never written again. That ordering is what makes the unsynchronized reads
	// from every RPC goroutine safe.
	registeredMethods := make(methodSet)

	// One builder produces both lists, so a policy cannot reach unary RPCs and
	// miss streaming ones, and the order cannot drift between them. Only the
	// timeout differs, which is why this is two calls rather than one shared
	// value.
	//
	// The admission policy is built once above and handed to both, because that
	// is what makes each concurrency budget process-wide rather than per RPC
	// kind. Nothing structural enforces it now that the list is built twice, so
	// TestAdmissionBudgetIsProcessWide drives a real server to prove it.
	unaryBuiltins := builtinPolicies(options.Logger, accessLogs, admission, cfg.UnaryTimeout)
	streamBuiltins := builtinPolicies(options.Logger, accessLogs, admission, cfg.StreamTimeout)
	handlerErrors := handlerErrorBoundary(options.Logger, errorRendering{
		mappers: options.DomainErrors,
		domain:  options.ErrorDomain,
	})

	// #nosec G115 -- config.go's validateConfig, called at the top of this
	// function, bounds both to [1,math.MaxUint32].
	maxStreams, maxHeaderBytes := uint32(cfg.MaxConcurrentStreams), uint32(cfg.MaxHeaderListBytes)
	serverOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(maxStreams),
		grpc.MaxHeaderListSize(maxHeaderBytes),
		grpc.MaxRecvMsgSize(cfg.MaxReceiveMessageBytes),
		grpc.MaxSendMsgSize(cfg.MaxSendMessageBytes),
		grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithMeterProvider(options.MeterProvider),
			otelgrpc.WithTracerProvider(options.TracerProvider),
			otelgrpc.WithPropagators(options.Propagators),
			otelgrpc.WithFilter(func(info *stats.RPCTagInfo) bool {
				_, ok := registeredMethods[info.FullMethodName]
				return ok && (cfg.TelemetryHealthChecks || !isHealthMethod(info.FullMethodName))
			}),
		)),
		grpc.ChainUnaryInterceptor(
			unaryChain(options.Logger, unaryBuiltins, options.UnaryPolicy, handlerErrors)...,
		),
		grpc.ChainStreamInterceptor(
			streamChain(options.Logger, streamBuiltins, options.StreamPolicy, handlerErrors)...,
		),
		// A zero MaxConnectionAge is how rotation stays off: grpc-go reads it as
		// infinity. Every other value here is validated positive, so this is the
		// one field whose zero is a decision rather than a defect.
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     cfg.MaxConnectionIdle,
			MaxConnectionAge:      cfg.MaxConnectionAge,
			MaxConnectionAgeGrace: cfg.MaxConnectionAgeGrace,
			Time:                  cfg.ServerPingInterval,
			Timeout:               cfg.ServerPingTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             cfg.MinClientPingInterval,
			PermitWithoutStream: cfg.PermitPingWithoutStream,
		}),
	}
	if options.TransportCredentials != nil {
		serverOptions = append(serverOptions, grpc.Creds(options.TransportCredentials))
	}

	nativeServer := grpc.NewServer(serverOptions...)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
	healthgrpc.RegisterHealthServer(nativeServer, healthServer)
	if err := registerServices(nativeServer, options.Services, registeredMethods); err != nil {
		nativeServer.Stop()
		return nil, err
	}

	return &Server{
		server: nativeServer,
		health: healthServer,
		// profile:authn-oidc-jwt:start
		authnReady: true,
		// profile:authn-oidc-jwt:end
		gracefulDone: make(chan struct{}),
		forceStarted: make(chan struct{}),
	}, nil
}

// registerServices attaches every supplied service to server and records the
// full method name of everything now registered, health included, into methods.
//
// Filling the set here keeps the write adjacent to the registration it reads,
// which is the ordering NewServer's telemetry filter depends on: the set is
// written once, before Serve, and only read afterwards.
//
// A nil entry is refused rather than skipped. Skipping it produces a server that
// starts and serves without a method its composition meant to publish — a
// failure no probe and no test of the remaining services can see, and the same
// class of unchecked programming error [Server] declines to absorb elsewhere.
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

// methodSet holds the full RPC method names of every registered service.
type methodSet map[string]struct{}
