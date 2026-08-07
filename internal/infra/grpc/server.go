package grpcx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/stats"
)

// Server adapts grpc.Server to the process runtime's bounded lifecycle.
//
// Every Server comes from [NewServer]; a nil pointer and the zero value are
// programming errors rather than a disabled server, and the methods below do
// not defend against them. A composition root that builds this transport
// conditionally holds the absence in its own variable — cmd/service keeps a nil
// grpcRuntimeServer interface and checks it once per use — because a method
// that quietly did nothing on a nil receiver would turn an unchecked [NewServer]
// error into a process that starts and never serves.
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

	// One list value serves both chains, so a policy cannot reach unary RPCs and
	// miss streaming ones, and the order cannot drift between them.
	builtins := builtinPolicies(options.Logger, accessLogs, admission)
	handlerErrors := handlerErrorBoundary(options.DomainErrors)

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
		grpc.ChainUnaryInterceptor(unaryChain(builtins, options.UnaryPolicy, handlerErrors)...),
		grpc.ChainStreamInterceptor(streamChain(builtins, options.StreamPolicy, handlerErrors)...),
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
	s.forceStop()
	return nil
}

func (s *Server) forceStop() {
	s.stopOnce.Do(func() {
		go s.server.Stop()
		close(s.forceStarted)
	})
}
