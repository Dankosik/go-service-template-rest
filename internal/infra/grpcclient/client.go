package grpcclient

import (
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
}

// DefaultConfig returns the template's conservative transport constraints for
// target. A feature raises them only with matching workload and memory proof.
func DefaultConfig(target string) Config {
	return Config{
		Target:                 target,
		MaxHeaderListBytes:     16 << 10,
		MaxReceiveMessageBytes: 4 << 20,
		MaxSendMessageBytes:    4 << 20,
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
	return nil
}
