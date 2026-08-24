package grpcx

import (
	"context"
	"strings"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type healthDrain struct {
	ctx    context.Context //nolint:containedctx // One server-lifetime cancellation signal.
	cancel context.CancelFunc
}

func newHealthDrain() healthDrain {
	ctx, cancel := context.WithCancel(context.Background())
	return healthDrain{ctx: ctx, cancel: cancel}
}

func (d healthDrain) stop() { d.cancel() }

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
// method grpc-go adds to it later is exempted from routine access logs, the
// business RPC budget, and protocol telemetry without an edit here.
// Over-matching costs a metric series or the wrong finite budget rather than
// publishing work or leaving it unbounded.
//
// A policy deciding which RPCs are public must not share this definition: that
// is a trust decision, and over-matching there publishes an RPC nobody meant to
// publish.
// profile:authn-bearer:start
// internal/infra/oidcjwt/grpc.go names Check exactly for that reason.
// profile:authn-bearer:end
func isHealthMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, healthMethodPrefix)
}
