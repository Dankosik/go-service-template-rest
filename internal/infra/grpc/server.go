package grpcx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"sync"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/stats"
)

// Server adapts grpc.Server to the process runtime's bounded lifecycle.
type Server struct {
	server *grpc.Server
	health *health.Server

	// Health inputs, all guarded by healthMu and combined by publishHealthLocked.
	healthMu sync.Mutex
	draining bool
	admitted bool
	// profile:authn-oidc-jwt:start
	authnReady bool
	// profile:authn-oidc-jwt:end

	gracefulOnce sync.Once
	stopOnce     sync.Once
	gracefulDone chan struct{}
	forceStarted chan struct{}
}

// NewServer builds a native gRPC server with standard health, OpenTelemetry,
// finite bounds, and the repository interceptor policies.
func NewServer(cfg Config, options Options) (*Server, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	options = withOptionDefaults(options)
	admission := newAdmissionLimiter(cfg.MaxConcurrentRPCs, options.Load)
	accessLogs := accessLogPolicy{
		logHealthChecks:   cfg.LogHealthChecks,
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

	// One list value serves both chains, so a policy cannot reach unary RPCs and
	// miss streaming ones, and the order cannot drift between them.
	builtins := builtinPolicies(options.Logger, accessLogs, admission)
	errorMapping := errorMappingAround(options.DomainErrors)

	// #nosec G115 -- validateConfig bounds both to [1,math.MaxUint32] above.
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
		grpc.ChainUnaryInterceptor(unaryChain(builtins, options.UnaryPolicy, errorMapping)...),
		grpc.ChainStreamInterceptor(streamChain(builtins, options.StreamPolicy, errorMapping)...),
	}
	if options.TransportCredentials != nil {
		serverOptions = append(serverOptions, grpc.Creds(options.TransportCredentials))
	}

	nativeServer := grpc.NewServer(serverOptions...)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
	healthgrpc.RegisterHealthServer(nativeServer, healthServer)
	registerServices(nativeServer, options.Services, registeredMethods)

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

// Names for the built-in policies, used by [builtinPolicies] and by the
// benchmark variants in performance_test.go that measure a deliberate subset of
// the chain.
const (
	builtinAccessLog           = "access_log"
	builtinRecovery            = "recovery"
	builtinAdmission           = "admission"
	builtinPolicyErrorBoundary = "policy_error_boundary"
)

// builtinPolicy is one entry in this package's own policy chain, carrying the
// name the benchmark refers to it by.
type builtinPolicy struct {
	name   string
	around aroundRPC
}

// builtinPolicies returns this package's own policies, outermost first. Every
// position is load-bearing, and the package doc owns the order, what each one
// buys, and why this is a function rather than a literal — read it before
// reordering this list.
//
// Correlation and error mapping are not in the list because they do not sit next
// to each other: correlation is outermost and is the one policy still written
// per RPC kind, while error mapping must land inside the supplied policy.
func builtinPolicies(
	log *slog.Logger,
	accessLogs accessLogPolicy,
	admission *admissionLimiter,
) []builtinPolicy {
	return []builtinPolicy{
		{name: builtinAccessLog, around: accessLogAround(log, accessLogs)},
		{name: builtinRecovery, around: recoveryAround(log)},
		{name: builtinAdmission, around: admission.around},
		{name: builtinPolicyErrorBoundary, around: policyErrorBoundaryAround()},
	}
}

// unaryChain and streamChain assemble the one interceptor order this transport
// serves, outermost first: correlation, the builtins, the supplied policy, then
// error mapping. Both RPC kinds and the benchmark variants in
// performance_test.go build their chains here, so a position cannot drift
// between them; the package doc owns what each position buys.
func unaryChain(
	builtins []builtinPolicy,
	supplied []grpc.UnaryServerInterceptor,
	errorMapping aroundRPC,
) []grpc.UnaryServerInterceptor {
	chain := []grpc.UnaryServerInterceptor{correlationUnaryInterceptor()}
	for _, policy := range builtins {
		chain = append(chain, unaryPolicy(policy.around))
	}
	chain = append(chain, supplied...)
	return append(chain, unaryPolicy(errorMapping))
}

func streamChain(
	builtins []builtinPolicy,
	supplied []grpc.StreamServerInterceptor,
	errorMapping aroundRPC,
) []grpc.StreamServerInterceptor {
	chain := []grpc.StreamServerInterceptor{correlationStreamInterceptor()}
	for _, policy := range builtins {
		chain = append(chain, streamPolicy(policy.around))
	}
	chain = append(chain, supplied...)
	return append(chain, streamPolicy(errorMapping))
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

// registerServices attaches every supplied service to server and records the
// full method name of everything now registered, health included, into methods.
//
// Filling the set here keeps the write adjacent to the registration it reads,
// which is the ordering NewServer's telemetry filter depends on: the set is
// written once, before Serve, and only read afterwards.
func registerServices(server *grpc.Server, services []RegisterService, methods methodSet) {
	for _, register := range services {
		if register != nil {
			register(server)
		}
	}
	for serviceName, service := range server.GetServiceInfo() {
		for _, method := range service.Methods {
			methods["/"+serviceName+"/"+method.Name] = struct{}{}
		}
	}
}

// validateConfig proves the bounds NewServer is about to hand grpc-go.
//
// internal/config.validateGRPCConfig restates the same access-log rules for the
// service's own configuration file, so a new access-log bound needs a rule in
// both places. config_parity_test.go owns why there are two owners and holds
// them to one answer.
func validateConfig(cfg Config) error {
	if cfg.MaxConcurrentRPCs <= 0 {
		return errors.New("build gRPC server: max concurrent RPCs must be positive")
	}
	if err := uint32Bound("max concurrent streams", cfg.MaxConcurrentStreams); err != nil {
		return err
	}
	if err := uint32Bound("max header list bytes", cfg.MaxHeaderListBytes); err != nil {
		return err
	}
	if cfg.MaxReceiveMessageBytes <= 0 {
		return errors.New("build gRPC server: max receive message bytes must be positive")
	}
	if cfg.MaxSendMessageBytes <= 0 {
		return errors.New("build gRPC server: max send message bytes must be positive")
	}
	if math.IsNaN(cfg.AccessLogSuccessSampleRate) ||
		math.IsInf(cfg.AccessLogSuccessSampleRate, 0) ||
		cfg.AccessLogSuccessSampleRate < 0 ||
		cfg.AccessLogSuccessSampleRate > 1 {
		return errors.New("build gRPC server: access-log success sample rate must be finite and in range [0,1]")
	}
	if cfg.AccessLogSlowThreshold < 0 {
		return errors.New("build gRPC server: access-log slow threshold must be non-negative")
	}
	return nil
}

// uint32Bound owns the one place a caller-supplied transport bound is proven to
// fit the uint32 grpc-go asks for; [Config] owns why those fields are int.
func uint32Bound(name string, value int) error {
	if value <= 0 || uint64(value) > math.MaxUint32 {
		return fmt.Errorf(
			"build gRPC server: %s must be in range [1,%d]",
			name,
			uint64(math.MaxUint32),
		)
	}
	return nil
}

// methodSet holds the full RPC method names of every registered service.
type methodSet map[string]struct{}

// publishHealthLocked republishes standard health from the current inputs. It
// owns the whole rule so each caller below only records its own input:
//
//	SERVING iff startup admitted the process and authentication trust is
//	current. Drain is terminal and is published by StartDrain instead, so a
//	later input can never make a draining server serving again.
//
// Adding a readiness input means adding a field and one term here, not another
// copy of the composite in a third method.
func (s *Server) publishHealthLocked() {
	serving := s.admitted
	// profile:authn-oidc-jwt:start
	serving = serving && s.authnReady
	// profile:authn-oidc-jwt:end
	status := healthgrpc.HealthCheckResponse_NOT_SERVING
	if serving {
		status = healthgrpc.HealthCheckResponse_SERVING
	}
	s.health.SetServingStatus("", status)
}

// MarkServing publishes the shared startup admission result to gRPC health.
func (s *Server) MarkServing() {
	if s == nil {
		return
	}
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.draining {
		return
	}
	s.admitted = true
	s.publishHealthLocked()
}

// profile:authn-oidc-jwt:start

// SetAuthnReady composes current authentication trust into standard health.
//
// Its proof lives in authn_health_test.go rather than server_test.go, because
// this method and that file are both removed when a service is generated with
// AUTHN=none.
func (s *Server) SetAuthnReady(ready bool) {
	if s == nil {
		return
	}
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.draining {
		return
	}
	s.authnReady = ready
	s.publishHealthLocked()
}

// profile:authn-oidc-jwt:end

// StartDrain makes every registered health service NOT_SERVING and prevents a
// later startup result from making it serving again.
func (s *Server) StartDrain() {
	if s == nil {
		return
	}
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.draining {
		return
	}
	s.draining = true
	s.health.Shutdown()
}

// Serve runs the native server and normalizes the expected stop result.
func (s *Server) Serve(listener net.Listener) error {
	if s == nil || s.server == nil {
		return errors.New("serve gRPC: server is nil")
	}
	err := s.server.Serve(listener)
	if err == nil || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return fmt.Errorf("serve native gRPC server: %w", err)
}

// Shutdown waits for active RPCs until ctx expires, then forces every remaining
// transport stream to stop. grpc-go's stop calls may remain parked while a
// handler ignores its canceled RPC context, so this adapter deliberately does
// not join those process-lifetime goroutines past the caller's shutdown budget.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	s.gracefulOnce.Do(func() {
		go func() {
			s.server.GracefulStop()
			close(s.gracefulDone)
		}()
	})

	select {
	case <-s.gracefulDone:
		return nil
	case <-s.forceStarted:
		return nil
	case <-ctx.Done():
		s.forceStop()
		return fmt.Errorf("shutdown gRPC server: %w", ctx.Err())
	}
}

// Close immediately abandons active RPCs. Repeated calls are safe.
func (s *Server) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	s.forceStop()
	return nil
}

func (s *Server) forceStop() {
	s.stopOnce.Do(func() {
		go s.server.Stop()
		close(s.forceStarted)
	})
}

type noopLoadRecorder struct{}

func (noopLoadRecorder) Admitted(context.Context) func() {
	return func() {}
}

func (noopLoadRecorder) Shed(context.Context) {}
