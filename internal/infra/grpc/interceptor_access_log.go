package grpcx

import (
	"context"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type accessLogPolicy struct {
	logHealthChecks   bool
	successSampleRate float64
	slowThreshold     time.Duration
}

func accessLogAround(log *slog.Logger, policy accessLogPolicy) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) error {
		if !policy.logsMethod(fullMethod) || !log.Enabled(ctx, slog.LevelInfo) {
			return call(ctx)
		}
		started := time.Now()
		err := call(ctx)

		elapsed := time.Since(started)
		if code := status.Code(err); policy.shouldLog(ctx, code, elapsed) {
			log.LogAttrs(
				ctx,
				slog.LevelInfo,
				"grpc_request",
				slog.String("rpc.method", fullMethod),
				slog.String("rpc.status", code.String()),
				slog.Int64("duration_ms", elapsed.Milliseconds()),
			)
		}
		return err
	}
}

// logsMethod reports whether this policy admits fullMethod to the access log at
// all; [accessLogPolicy.shouldLog] then decides on the RPC's outcome. Whether
// the logger would emit the record is the caller's question rather than the
// policy's, which is why both predicates read from policy state alone.
func (p accessLogPolicy) logsMethod(fullMethod string) bool {
	return p.logHealthChecks || !isHealthMethod(fullMethod)
}

func (p accessLogPolicy) shouldLog(ctx context.Context, code codes.Code, elapsed time.Duration) bool {
	if code != codes.OK {
		return true
	}
	if p.slowThreshold > 0 && elapsed >= p.slowThreshold {
		return true
	}
	return sampleRequestID(reqctx.RequestID(ctx), p.successSampleRate)
}

func sampleRequestID(requestID string, rate float64) bool {
	if requestID == "" || rate >= 1 {
		return true
	}
	if rate <= 0 {
		return false
	}

	// FNV-1a is sufficient here because the input is already a validated,
	// non-secret correlation identifier. It keeps the decision deterministic
	// without a shared RNG, lock, or allocation on the RPC path. It is written
	// out rather than taken from hash/fnv because that package hands back a
	// hash.Hash64 interface, which allocates on every RPC; maphash is seeded per
	// process and would sample one request differently at each hop.
	const (
		fnvOffset64 = uint64(14695981039346656037)
		fnvPrime64  = uint64(1099511628211)
	)
	hash := fnvOffset64
	for index := range len(requestID) {
		hash ^= uint64(requestID[index])
		hash *= fnvPrime64
	}

	const sampleBuckets = uint64(1_000_000_000)
	bucket := hash % sampleBuckets
	return float64(bucket)/float64(sampleBuckets) < rate
}
