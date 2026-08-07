package grpcx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Config contains transport bounds already validated by the runtime config
// owner. NewServer validates them again so direct package users cannot
// accidentally recover an unlimited native default.
//
// Every bound is an int, including the two grpc-go takes as uint32, so a
// composition root passes its configuration straight through and NewServer owns
// the single range check that proves the conversion.
type Config struct {
	// MaxConcurrentRPCs bounds the RPCs executing a handler at once across the
	// whole process. Standard health RPCs are exempt, so probe load cannot
	// consume the budget; anything over the limit is shed as ResourceExhausted
	// rather than queued.
	MaxConcurrentRPCs int

	// The four native transport bounds, passed to grpc.MaxConcurrentStreams,
	// grpc.MaxHeaderListSize, grpc.MaxRecvMsgSize, and grpc.MaxSendMsgSize.
	MaxConcurrentStreams   int
	MaxHeaderListBytes     int
	MaxReceiveMessageBytes int
	MaxSendMessageBytes    int

	// AccessLogHealthChecks admits standard health RPCs to the access log, and
	// TelemetryHealthChecks is the same decision for spans and metrics. Both
	// are off by default because a liveness probe otherwise dominates each
	// signal; they are separate fields because operators usually want neither,
	// occasionally want telemetry alone, and rarely want the log.
	AccessLogHealthChecks bool
	TelemetryHealthChecks bool

	// AccessLogSuccessSampleRate is the fraction of successful RPCs that reach
	// the access log, in [0,1]. The decision is a hash of the request ID rather
	// than an RNG draw, so one request is logged consistently at every hop and
	// the sampling costs no lock or allocation. Two consequences follow: an RPC
	// arriving without a usable request ID is always logged, and sampling is
	// per-identifier rather than per-RPC. Failed RPCs ignore this and always
	// log.
	AccessLogSuccessSampleRate float64

	// AccessLogSlowThreshold always logs a successful RPC at or above this
	// duration, whatever the sample rate. Zero disables the exemption.
	AccessLogSlowThreshold time.Duration
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
	if err := validateUint32Bound("max concurrent streams", cfg.MaxConcurrentStreams); err != nil {
		return err
	}
	if err := validateUint32Bound("max header list bytes", cfg.MaxHeaderListBytes); err != nil {
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

// validateUint32Bound owns the one place a caller-supplied transport bound is
// proven to fit the uint32 grpc-go asks for; [Config] owns why those fields are
// int.
func validateUint32Bound(name string, value int) error {
	if value <= 0 || uint64(value) > math.MaxUint32 {
		return fmt.Errorf(
			"build gRPC server: %s must be in range [1,%d]",
			name,
			uint64(math.MaxUint32),
		)
	}
	return nil
}

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
	DomainErrors []problem.Mapper

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

// LoadRecorder observes the process-wide admission decision. Admitted returns
// the release function for one admitted RPC; Shed reports one rejected by a
// full admission budget.
type LoadRecorder interface {
	Admitted(ctx context.Context) func()
	Shed(ctx context.Context)
}

// noopLoadRecorder is the [Options.Load] default, so the admission policy always
// has a recorder to call and never guards the field.
type noopLoadRecorder struct{}

func (noopLoadRecorder) Admitted(context.Context) func() {
	return func() {}
}

func (noopLoadRecorder) Shed(context.Context) {}
