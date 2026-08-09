package grpcx

import (
	"context"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/internal/failure"
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
	// MaxConcurrentRPCs bounds the business RPCs executing a handler at once
	// across the whole process. Anything over the limit is shed as
	// ResourceExhausted rather than queued. The standard health service holds
	// MaxConcurrentHealthRPCs instead, so no amount of probe or watch traffic can
	// consume this budget.
	MaxConcurrentRPCs int

	// MaxConcurrentHealthRPCs bounds the standard health service's concurrent
	// RPCs, excluding Check, which holds no slot at all so a saturated instance
	// stays probeable.
	//
	// It exists because grpc-go's client-side health checker keeps one
	// Health/Watch stream open per subchannel for the connection's whole life,
	// and this repository's own client enables it by default. Counted against
	// MaxConcurrentRPCs, every connected peer would permanently hold a business
	// slot, and shedding a watch would make each round-robin caller drop the
	// backend entirely rather than lose only the excess.
	//
	// One watch per connection is the legitimate shape, so a composition root
	// fills this from its connection limit. That admits every well-behaved peer
	// while still bounding a hostile one, which could otherwise open
	// MaxConcurrentStreams watches on every connection it holds.
	MaxConcurrentHealthRPCs int

	// The four native transport bounds, passed to grpc.MaxConcurrentStreams,
	// grpc.MaxHeaderListSize, grpc.MaxRecvMsgSize, and grpc.MaxSendMsgSize.
	MaxConcurrentStreams   int
	MaxHeaderListBytes     int
	MaxReceiveMessageBytes int
	MaxSendMessageBytes    int

	// UnaryTimeout and StreamTimeout cap how long one RPC may occupy a handler.
	// The cap derives the RPC context's deadline, so a caller deadline that is
	// already earlier wins and neither value can extend one. Non-positive
	// disables the cap for that RPC kind.
	//
	// Cancellation is the protection, not the response: a handler that ignores
	// its context keeps its goroutine and its admission slot, exactly as
	// internal/infra/http records for its own request budget. StreamTimeout is
	// disabled by default because a long-lived stream's duration policy belongs
	// to the feature that owns the stream.
	UnaryTimeout  time.Duration
	StreamTimeout time.Duration

	// The liveness bounds. None of them can end an RPC in progress: the idle
	// clock only runs while nothing is outstanding, and the ping bound closes
	// only when a ping goes unanswered, which means the peer is gone. That is
	// what makes them safe to leave on while rotation stays off.
	//
	// MinClientPingInterval and PermitPingWithoutStream are the enforcement
	// policy rather than this server's own behavior: they bound what a client
	// may do. grpc-go's defaults reject a ping more often than every five
	// minutes and any ping with no active stream, which would disconnect this
	// repository's own client half.
	MaxConnectionIdle       time.Duration
	ServerPingInterval      time.Duration
	ServerPingTimeout       time.Duration
	MinClientPingInterval   time.Duration
	PermitPingWithoutStream bool

	// MaxConnectionAge rotates connections, and is the only bound here that ends
	// work in progress: at this age the connection is drained with GOAWAY and
	// force-closed once MaxConnectionAgeGrace expires, cutting every RPC and
	// stream still running. grpc-go adds ±10% jitter to spread connection
	// storms, and reads zero as infinity, which is how rotation is disabled.
	//
	// Enable it behind an L4 balancer or any hop that pins a caller to one
	// replica for the life of a connection. The grace must then be positive and
	// at least UnaryTimeout, so a unary RPC still inside its own budget is not
	// cut by the force-close.
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration

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
