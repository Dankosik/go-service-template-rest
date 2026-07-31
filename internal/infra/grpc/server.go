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

	healthMu sync.Mutex
	draining bool
	// profile:authn-oidc-jwt:start
	admitted   bool
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

	log := options.Logger
	if log == nil {
		log = slog.Default()
	}
	meterProvider := options.MeterProvider
	if meterProvider == nil {
		meterProvider = metricnoop.NewMeterProvider()
	}
	tracerProvider := options.TracerProvider
	if tracerProvider == nil {
		tracerProvider = tracenoop.NewTracerProvider()
	}
	propagators := options.Propagators
	if propagators == nil {
		propagators = propagation.TraceContext{}
	}
	load := options.Load
	if load == nil {
		load = noopLoadRecorder{}
	}
	admission := newAdmissionLimiter(cfg.MaxConcurrentRPCs, load)
	accessLogs := accessLogPolicy{
		logHealthChecks:   cfg.LogHealthChecks,
		successSampleRate: cfg.AccessLogSuccessSampleRate,
		slowThreshold:     cfg.AccessLogSlowThreshold,
	}
	// otelgrpc derives rpc.method from the peer-supplied HTTP/2 path before
	// dispatch. Keep its server signals descriptor-backed so unknown paths cannot
	// create unbounded metric series or span names.
	registeredMethods := make(map[string]struct{})

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		correlationUnaryInterceptor(),
		accessLogUnaryInterceptor(log, accessLogs),
		recoveryUnaryInterceptor(log),
		admissionUnaryInterceptor(admission),
		policyErrorBoundaryUnaryInterceptor(),
	}
	unaryInterceptors = append(unaryInterceptors, options.UnaryPolicy...)
	unaryInterceptors = append(unaryInterceptors, errorMappingUnaryInterceptor(options.DomainErrors))

	streamInterceptors := []grpc.StreamServerInterceptor{
		correlationStreamInterceptor(),
		accessLogStreamInterceptor(log, accessLogs),
		recoveryStreamInterceptor(log),
		admissionStreamInterceptor(admission),
		policyErrorBoundaryStreamInterceptor(),
	}
	streamInterceptors = append(streamInterceptors, options.StreamingPolicy...)
	streamInterceptors = append(streamInterceptors, errorMappingStreamInterceptor(options.DomainErrors))

	serverOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(cfg.MaxConcurrentStreams),
		grpc.MaxHeaderListSize(cfg.MaxHeaderListBytes),
		grpc.MaxRecvMsgSize(cfg.MaxReceiveMessageBytes),
		grpc.MaxSendMsgSize(cfg.MaxSendMessageBytes),
		grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithMeterProvider(meterProvider),
			otelgrpc.WithTracerProvider(tracerProvider),
			otelgrpc.WithPropagators(propagators),
			otelgrpc.WithFilter(func(info *stats.RPCTagInfo) bool {
				_, ok := registeredMethods[info.FullMethodName]
				return ok && (cfg.TelemetryHealthChecks || !isHealthMethod(info.FullMethodName))
			}),
		)),
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	}
	if cfg.TransportCredentials != nil {
		serverOptions = append(serverOptions, grpc.Creds(cfg.TransportCredentials))
	}

	nativeServer := grpc.NewServer(serverOptions...)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
	healthgrpc.RegisterHealthServer(nativeServer, healthServer)
	for _, register := range options.Services {
		if register != nil {
			register(nativeServer)
		}
	}
	for serviceName, service := range nativeServer.GetServiceInfo() {
		for _, method := range service.Methods {
			registeredMethods["/"+serviceName+"/"+method.Name] = struct{}{}
		}
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

func validateConfig(cfg Config) error {
	if cfg.MaxConcurrentRPCs <= 0 {
		return errors.New("build gRPC server: max concurrent RPCs must be positive")
	}
	if cfg.MaxConcurrentStreams == 0 {
		return errors.New("build gRPC server: max concurrent streams must be positive")
	}
	if cfg.MaxHeaderListBytes == 0 {
		return errors.New("build gRPC server: max header list bytes must be positive")
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
	// profile:authn-oidc-jwt:start
	s.admitted = true
	if !s.authnReady {
		s.health.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
		return
	}
	// profile:authn-oidc-jwt:end
	s.health.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
}

// profile:authn-oidc-jwt:start

// SetAuthnReady composes current authentication trust into standard health.
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
	status := healthgrpc.HealthCheckResponse_NOT_SERVING
	if s.admitted && ready {
		status = healthgrpc.HealthCheckResponse_SERVING
	}
	s.health.SetServingStatus("", status)
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

var _ interface {
	Serve(listener net.Listener) error
	Shutdown(ctx context.Context) error
	Close() error
} = (*Server)(nil)

var _ interface {
	MarkServing()
	StartDrain()
} = (*Server)(nil)
