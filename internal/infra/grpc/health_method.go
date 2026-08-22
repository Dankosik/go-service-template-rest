package grpcx

import (
	"strings"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

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
