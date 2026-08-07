package grpcx

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	requestIDMetadataKey = reqctx.RequestIDMetadataKey
	healthMethodPrefix   = "/grpc.health.v1.Health/"
)

func correlationUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		enriched, requestID := contextWithRequestID(ctx)
		if err := grpc.SetHeader(enriched, metadata.Pairs(requestIDMetadataKey, requestID)); err != nil {
			return nil, ownedStatus(codes.Internal, "failed to prepare response metadata")
		}
		return handler(enriched, request)
	}
}

func correlationStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		enriched, requestID := contextWithRequestID(stream.Context())
		if err := stream.SetHeader(metadata.Pairs(requestIDMetadataKey, requestID)); err != nil {
			return ownedStatus(codes.Internal, "failed to prepare response metadata")
		}
		return handler(server, serverStreamWithContext{ServerStream: stream, ctx: enriched})
	}
}

func contextWithRequestID(ctx context.Context) (context.Context, string) {
	incoming, _ := metadata.FromIncomingContext(ctx)
	values := incoming.Get(requestIDMetadataKey)
	// Exactly one value is a candidate. With two or more, which identifier the
	// logs, the response metadata, and the next hop agree on would depend on an
	// ordering the caller controls, so none of them is accepted and a fresh ID
	// is minted instead. This is deliberately stricter than the HTTP transport,
	// where Header.Get takes the first of several.
	candidate := ""
	if len(values) == 1 {
		candidate = values[0]
	}
	return reqctx.ContextWithAcceptedRequestID(ctx, candidate)
}

// serverStreamWithContext replaces the context a streaming handler observes.
// The package doc owns why a policy package declares its own copy rather than
// sharing this one.
// profile:authn-oidc-jwt:start
// internal/infra/oidcjwt/grpc.go is that copy.
// profile:authn-oidc-jwt:end
type serverStreamWithContext struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx // grpc.ServerStream requires Context to return the enriched RPC context.
}

func (s serverStreamWithContext) Context() context.Context {
	return s.ctx
}

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

// deadlineAround caps how long the work below it may occupy a handler.
//
// It derives the RPC context's deadline rather than replacing it, so a caller
// deadline that is already earlier still wins: context.WithTimeout never extends
// a parent's. That makes this a cap by construction rather than by a comparison
// anyone has to write.
//
// A non-positive timeout disables the cap, which is how the stream bound ships.
// Health RPCs are exempt for the reason isHealthMethod records: a probe must not
// be cut by a business budget, just as it does not consume the admission one.
func deadlineAround(timeout time.Duration) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) error {
		if timeout <= 0 || isHealthMethod(fullMethod) {
			return call(ctx)
		}
		bounded, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return call(bounded)
	}
}

func recoveryAround(log *slog.Logger) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logRecoveredPanic(ctx, log, fullMethod, recovered)
				err = ownedStatus(codes.Internal, "request failed")
			}
		}()
		return call(ctx)
	}
}

func logRecoveredPanic(ctx context.Context, log *slog.Logger, method string, recovered any) {
	log.ErrorContext(
		ctx,
		"grpc_panic_recovered",
		"rpc.method", method,
		"panic.type", fmt.Sprintf("%T", recovered),
		"stack", string(debug.Stack()),
	)
}

type admissionLimiter struct {
	sem  *semaphore.Weighted
	load LoadRecorder
}

func newAdmissionLimiter(limit int, load LoadRecorder) *admissionLimiter {
	return &admissionLimiter{
		sem:  semaphore.NewWeighted(int64(limit)),
		load: load,
	}
}

// around holds one admission slot for the work below it. One limiter value backs
// both chains, which is what makes the budget process-wide rather than per RPC
// kind. Since NewServer builds the two policy lists separately, that sharing is
// its composition rather than a structural guarantee:
// TestOneLimiterServesBothInterceptorTypes proves this method is shareable, and
// TestAdmissionBudgetIsProcessWide proves the server actually shares it.
func (l *admissionLimiter) around(ctx context.Context, fullMethod string, call func(context.Context) error) error {
	if isHealthMethod(fullMethod) {
		return call(ctx)
	}
	if !l.sem.TryAcquire(1) {
		l.load.Shed(ctx)
		return ownedStatus(codes.ResourceExhausted, "server is at capacity")
	}
	defer l.sem.Release(1)
	release := l.load.Admitted(ctx)
	defer release()
	return call(ctx)
}

// isHealthMethod matches the whole standard health service by prefix, so a
// method grpc-go adds to it later is exempted from admission, routine access
// logs, and protocol telemetry without an edit here. Exempting one probe too
// many costs an operator a metric series; failing to exempt one costs the
// service its admission budget under probe load.
//
// A policy deciding which RPCs are public must not share this definition: that
// is a trust decision, and over-matching there publishes an RPC nobody meant to
// publish.
// profile:authn-oidc-jwt:start
// internal/infra/oidcjwt/grpc.go names Check and Watch exactly for that reason.
// profile:authn-oidc-jwt:end
func isHealthMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, healthMethodPrefix)
}
