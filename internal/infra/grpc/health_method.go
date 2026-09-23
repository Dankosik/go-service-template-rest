package grpcx

import (
	"context"
	"strings"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// healthDrain ends standard health RPCs when the server drains. A Watch stream
// lives as long as its peer, and GracefulStop waits for every open stream, so
// without this cancellation a watching client would hold Shutdown until its
// deadline. health.Server.Shutdown only publishes NOT_SERVING; it does not end
// the streams.
type healthDrain struct {
	ctx    context.Context //nolint:containedctx // One server-lifetime cancellation signal.
	cancel context.CancelFunc
}

func newHealthDrain() healthDrain {
	ctx, cancel := context.WithCancel(context.Background())
	return healthDrain{ctx: ctx, cancel: cancel}
}

func (d healthDrain) cancelAll() { d.cancel() }

func (d healthDrain) around(ctx context.Context, fullMethod string, call func(context.Context) error) error {
	if !isHealthMethod(fullMethod) {
		return call(ctx)
	}
	bounded, cancel := context.WithCancel(ctx)
	if d.ctx.Err() != nil {
		cancel()
	}
	stop := context.AfterFunc(d.ctx, cancel) //nolint:contextcheck // Server drain cancels this RPC-derived context.
	defer stop()
	defer cancel()
	return call(bounded)
}

const healthMethodPrefix = "/grpc.health.v1.Health/"

func isHealthCheck(fullMethod string) bool {
	return fullMethod == healthpb.Health_Check_FullMethodName
}

// isHealthMethod matches the whole standard health service by prefix, so a
// method grpc-go adds to it later is exempted from the business RPC budget,
// deadline, drain accounting, error sanitizing, and protocol telemetry without an
// edit here.
// Over-matching costs a metric series or the wrong finite budget rather than
// publishing work or leaving it unbounded.
//
// A policy deciding which RPCs are public must not share this definition: that
// is a trust decision, and over-matching there publishes an RPC nobody meant to
// publish.
// profile:authn-bearer:start
// internal/infra/bearerauthn/grpc.go names Check exactly for that reason.
// profile:authn-bearer:end
func isHealthMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, healthMethodPrefix)
}
