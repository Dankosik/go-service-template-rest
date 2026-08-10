package grpcx

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/grpclimits"
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

// validateConfig proves the bounds NewServer is about to hand grpc-go.
//
// The access-log and lifetime rules come from internal/grpclimits, which
// internal/config applies to the same values on the way in; this function owns
// only the capacity bounds, where the two owners are deliberately different.
// config_parity_test.go proves that what the loader accepts still builds a
// server.
func validateConfig(cfg Config) error {
	if cfg.MaxConcurrentRPCs <= 0 {
		return errors.New("build gRPC server: max concurrent RPCs must be positive")
	}
	if cfg.MaxConcurrentHealthRPCs <= 0 {
		return errors.New("build gRPC server: max concurrent health RPCs must be positive")
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
	if violation := grpclimits.ValidateAccessLog(grpclimits.AccessLog{
		SuccessSampleRate: cfg.AccessLogSuccessSampleRate,
		SlowThreshold:     cfg.AccessLogSlowThreshold,
	}); violation != nil {
		return refuse(violation)
	}
	if violation := grpclimits.ValidateLifetime(grpclimits.Lifetime{
		UnaryTimeout:          cfg.UnaryTimeout,
		StreamTimeout:         cfg.StreamTimeout,
		MaxConnectionIdle:     cfg.MaxConnectionIdle,
		MaxConnectionAge:      cfg.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.MaxConnectionAgeGrace,
		ServerPingInterval:    cfg.ServerPingInterval,
		ServerPingTimeout:     cfg.ServerPingTimeout,
		MinClientPingInterval: cfg.MinClientPingInterval,
	}); violation != nil {
		return refuse(violation)
	}
	return nil
}

// refuse renders a shared bound violation for a direct package user, who never
// saw a configuration file and so is told the bound in prose rather than by its
// configuration leaf.
func refuse(violation *grpclimits.Violation) error {
	return fmt.Errorf("build gRPC server: %s %s", violation.FieldWords(), violation.Rule)
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
