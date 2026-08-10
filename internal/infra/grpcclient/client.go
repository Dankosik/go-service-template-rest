package grpcclient

import (
	"errors"
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/observability/correlationpolicy"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/health" // Install grpc-go's standard client health checker.
	"google.golang.org/grpc/keepalive"
)

// What the composition root supplies, and the construction that turns it plus a
// [Config] into a connection. config.go owns the Config shape itself.

// Options supplies the trust decision and the observability collaborators the
// composition root owns.
type Options struct {
	// TransportCredentials is required, and [New] refuses a nil value. Plaintext
	// is spelled insecure.NewCredentials() so that dialing without transport
	// security is a visible decision, as grpc.NewClient itself requires. The
	// server adapter's field of the same name is optional because it binds a
	// listener the deployment owns; this one chooses how far to trust a peer it
	// is about to send credentials to.
	TransportCredentials credentials.TransportCredentials
	// PerRPCCredentials optionally supplies one connection credential for both
	// application RPCs and grpc-go control streams such as standard health Watch.
	// The dependency adapter creates and refreshes it; New only removes reserved
	// correlation metadata before grpc-go sends what remains.
	PerRPCCredentials credentials.PerRPCCredentials

	// Optional observability collaborators. Both fall back to their no-op
	// implementations, so a test can leave them unset.
	MeterProvider  metric.MeterProvider
	TracerProvider trace.TracerProvider

	// Propagation selects which locally owned correlation values cross this
	// client's trust boundary, once per dependency rather than per call. The
	// zero value emits none; the package doc owns how to choose among the three.
	Propagation PropagationPolicy
}

// New constructs a shareable ClientConn without performing network I/O. The
// caller owns Close and each operation owns its context deadline and retry
// semantics.
func New(cfg Config, options Options) (*grpc.ClientConn, error) {
	cfg.Target = strings.TrimSpace(cfg.Target)
	if err := validateConfig(cfg, options); err != nil {
		return nil, err
	}
	options = withOptionDefaults(options)

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(options.TransportCredentials),
		grpc.WithDisableServiceConfig(),
		// This client's own default service config, which the option above does
		// not refuse: it rejects only what a resolver supplies. The policy's own
		// doc owns why that leaves the trust boundary unchanged.
		grpc.WithDefaultServiceConfig(cfg.LoadBalancing.serviceConfig(cfg.HealthCheck)),
		grpc.WithNoProxy(),
		grpc.WithResolvers(sanitizingResolverBuilders(cfg.Target)...),
		grpc.WithChainUnaryInterceptor(propagationUnaryInterceptor),
		grpc.WithChainStreamInterceptor(propagationStreamInterceptor),
		grpc.WithMaxHeaderListSize(cfg.MaxHeaderListBytes),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxReceiveMessageBytes),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMessageBytes),
		),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(
			otelgrpc.WithMeterProvider(options.MeterProvider),
			otelgrpc.WithTracerProvider(options.TracerProvider),
			otelgrpc.WithPropagators(correlationpolicy.NewPropagator(options.Propagation, requestIDMetadataKey)),
		)),
	}
	if options.PerRPCCredentials != nil {
		dialOptions = append(
			dialOptions,
			grpc.WithPerRPCCredentials(wrapPerRPCCredentials(options.PerRPCCredentials)),
		)
	}
	if cfg.KeepalivePingInterval > 0 {
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.KeepalivePingInterval,
			Timeout:             cfg.KeepalivePingTimeout,
			PermitWithoutStream: true,
		}))
	}

	connection, err := grpc.NewClient(cfg.Target, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("build gRPC client connection: %w", err)
	}
	return connection, nil
}

// withOptionDefaults fills the collaborators [Options] documents as optional, so
// the construction in [New] reads as one uninterrupted sequence and every later
// reader of options sees a non-nil value. The server adapter in
// internal/infra/grpc has the same helper for the same reason.
func withOptionDefaults(options Options) Options {
	if options.MeterProvider == nil {
		options.MeterProvider = metricnoop.NewMeterProvider()
	}
	if options.TracerProvider == nil {
		options.TracerProvider = tracenoop.NewTracerProvider()
	}
	return options
}

// validateConfig proves the trust and bounds New is about to hand grpc-go.
// cfg.Target is already trimmed, because New dials exactly the value checked
// here.
func validateConfig(cfg Config, options Options) error {
	if cfg.Target == "" {
		return errors.New("build gRPC client: target is required")
	}
	if options.TransportCredentials == nil {
		return errors.New("build gRPC client: transport credentials are required")
	}
	if cfg.MaxHeaderListBytes == 0 {
		return errors.New("build gRPC client: max header list bytes must be positive")
	}
	if cfg.MaxReceiveMessageBytes <= 0 {
		return errors.New("build gRPC client: max receive message bytes must be positive")
	}
	if cfg.MaxSendMessageBytes <= 0 {
		return errors.New("build gRPC client: max send message bytes must be positive")
	}
	if !options.Propagation.Valid() {
		return errors.New("build gRPC client: propagation policy is invalid")
	}
	if !cfg.LoadBalancing.valid() {
		return errors.New("build gRPC client: load-balancing policy is invalid")
	}
	if cfg.KeepalivePingInterval < 0 ||
		cfg.KeepalivePingTimeout < 0 ||
		(cfg.KeepalivePingInterval == 0) != (cfg.KeepalivePingTimeout == 0) {
		return errors.New("build gRPC client: keepalive ping interval and timeout must both be positive or both be zero")
	}
	return nil
}
