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

const (
	DefaultMaxHeaderListBytes     = uint32(16 << 10)
	DefaultMaxReceiveMessageBytes = 4 << 20
	DefaultMaxSendMessageBytes    = 4 << 20
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
		MaxHeaderListBytes:     DefaultMaxHeaderListBytes,
		MaxReceiveMessageBytes: DefaultMaxReceiveMessageBytes,
		MaxSendMessageBytes:    DefaultMaxSendMessageBytes,
	}
}

// Options contains explicit trust and telemetry dependencies.
type Options struct {
	TransportCredentials credentials.TransportCredentials
	MeterProvider        metric.MeterProvider
	TracerProvider       trace.TracerProvider
	Propagation          PropagationPolicy
}

// New constructs a shareable ClientConn without performing network I/O. The
// caller owns Close and each operation owns its context deadline and retry
// semantics.
func New(cfg Config, options Options) (*grpc.ClientConn, error) {
	target := strings.TrimSpace(cfg.Target)
	if err := validateConfig(target, cfg, options); err != nil {
		return nil, err
	}

	meterProvider := options.MeterProvider
	if meterProvider == nil {
		meterProvider = metricnoop.NewMeterProvider()
	}
	tracerProvider := options.TracerProvider
	if tracerProvider == nil {
		tracerProvider = tracenoop.NewTracerProvider()
	}
	propagators := policyPropagator{policy: options.Propagation}

	connection, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(options.TransportCredentials),
		grpc.WithDisableServiceConfig(),
		grpc.WithNoProxy(),
		grpc.WithResolvers(sanitizingResolverBuilders(target)...),
		grpc.WithChainUnaryInterceptor(propagationUnaryInterceptor),
		grpc.WithChainStreamInterceptor(propagationStreamInterceptor),
		grpc.WithMaxHeaderListSize(cfg.MaxHeaderListBytes),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxReceiveMessageBytes),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMessageBytes),
		),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(
			otelgrpc.WithMeterProvider(meterProvider),
			otelgrpc.WithTracerProvider(tracerProvider),
			otelgrpc.WithPropagators(propagators),
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("build gRPC client connection: %w", err)
	}
	return connection, nil
}

// validateConfig proves the trust and bounds New is about to hand grpc-go, so
// the construction below reads as one uninterrupted sequence. target arrives
// already trimmed because New dials that value rather than cfg.Target.
func validateConfig(target string, cfg Config, options Options) error {
	if target == "" {
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
