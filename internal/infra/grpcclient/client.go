package grpcclient

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// Config contains the fixed target and finite per-call transport bounds.
//
// MaxHeaderListBytes is the uint32 grpc-go asks for, where the server adapter in
// internal/infra/grpc takes the same bound as an int. The difference follows the
// caller: that Config is filled from a configuration file, so it takes the int
// the file parses to and proves the conversion once on the way in, while these
// bounds come from [DefaultConfig] in Go source, where the narrower type is the
// check.
type Config struct {
	Target                 string
	MaxHeaderListBytes     uint32
	MaxReceiveMessageBytes int
	MaxSendMessageBytes    int

	// LoadBalancing selects how RPCs spread across the addresses Target
	// resolves to. Its own doc owns why it is a Config field.
	LoadBalancing LoadBalancingPolicy

	// KeepalivePingInterval and KeepalivePingTimeout keep a long-lived
	// connection usable through an idle intermediary. The documented shape here
	// is one connection per dependency built at startup, so without pings a NAT
	// or load balancer idle timeout discards it silently and the failure
	// surfaces as the next RPC's error rather than as a reconnect.
	//
	// Pings are sent with no active RPC, which is what makes them reach an idle
	// connection at all. That requires the peer to permit it: this repository's
	// server half does, and keepalive_parity_test.go holds the two defaults to
	// an interval the server accepts. grpc-go raises an interval below ten
	// seconds to ten.
	KeepalivePingInterval time.Duration
	KeepalivePingTimeout  time.Duration
}

// DefaultConfig returns the template's conservative transport constraints for
// target. A feature raises them only with matching workload and memory proof.
func DefaultConfig(target string) Config {
	return Config{
		Target:                 target,
		MaxHeaderListBytes:     16 << 10,
		MaxReceiveMessageBytes: 4 << 20,
		MaxSendMessageBytes:    4 << 20,
		LoadBalancing:          LoadBalancingRoundRobin,
		// Above grpc-go's ten-second floor, and above the ten seconds this
		// repository's server half accepts as a minimum.
		KeepalivePingInterval: 30 * time.Second,
		KeepalivePingTimeout:  10 * time.Second,
	}
}

// Options supplies the trust decision and the observability collaborators the
// composition root owns.
type Options struct {
	// TransportCredentials is required, and [New] refuses a nil value. Plaintext
	// is spelled insecure.NewCredentials() so that dialing without transport
	// security is a visible decision, which is also grpc-go's own rule for
	// grpc.NewClient. The server adapter in internal/infra/grpc takes a field of
	// the same name as optional, where nil means plaintext; that half binds a
	// listener whose exposure the deployment owns, while this one chooses how
	// much to trust a peer it is about to send credentials to.
	TransportCredentials credentials.TransportCredentials

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

	connection, err := grpc.NewClient(
		cfg.Target,
		grpc.WithTransportCredentials(options.TransportCredentials),
		grpc.WithDisableServiceConfig(),
		// This client's own default service config, which the option above does
		// not refuse: it rejects only what a resolver supplies. The policy's own
		// doc owns why that leaves the trust boundary unchanged.
		grpc.WithDefaultServiceConfig(cfg.LoadBalancing.serviceConfig()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    cfg.KeepalivePingInterval,
			Timeout: cfg.KeepalivePingTimeout,
			// Fixed rather than configured: reaching an idle connection is the
			// whole point of pinging one, and a caller that turned this off
			// would keep the fields while losing the behavior they exist for.
			PermitWithoutStream: true,
		}),
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
			otelgrpc.WithPropagators(policyPropagator{policy: options.Propagation}),
		)),
	)
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

// validateConfig proves the trust and bounds New is about to hand grpc-go, so
// the construction above reads as one uninterrupted sequence. cfg.Target is
// already trimmed, because New dials exactly the value checked here.
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
	if !options.Propagation.valid() {
		return errors.New("build gRPC client: propagation policy is invalid")
	}
	if !cfg.LoadBalancing.valid() {
		return errors.New("build gRPC client: load-balancing policy is invalid")
	}
	if cfg.KeepalivePingInterval <= 0 {
		return errors.New("build gRPC client: keepalive ping interval must be positive")
	}
	if cfg.KeepalivePingTimeout <= 0 {
		return errors.New("build gRPC client: keepalive ping timeout must be positive")
	}
	return nil
}
