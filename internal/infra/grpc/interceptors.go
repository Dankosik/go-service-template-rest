package grpcx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
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

// aroundRPC is one server policy, written once for both RPC kinds. It observes
// or replaces the result of the work below it and may decline to run that work
// at all. [unaryPolicy] and [streamPolicy] adapt it to grpc-go's two separate
// interceptor types; the package doc owns why a policy is written this way
// rather than against those types directly.
type aroundRPC func(ctx context.Context, fullMethod string, call func() error) error

// unaryPolicy adapts one policy to the unary interceptor type.
//
// The response is returned even alongside a non-nil error, as the hand-written
// interceptors this replaced did: grpc-go discards it, and an outer interceptor
// that inspects it sees what it saw before. When the handler panics, the
// assignment below never runs, so recovery still answers with a nil response.
func unaryPolicy(around aroundRPC) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		var response any
		err := around(ctx, info.FullMethod, func() error {
			var callErr error
			response, callErr = handler(ctx, request)
			return callErr
		})
		return response, err
	}
}

// streamPolicy adapts one policy to the streaming interceptor type.
func streamPolicy(around aroundRPC) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return around(stream.Context(), info.FullMethod, func() error {
			return handler(server, stream)
		})
	}
}

type accessLogPolicy struct {
	logHealthChecks   bool
	successSampleRate float64
	slowThreshold     time.Duration
}

func accessLogAround(log *slog.Logger, policy accessLogPolicy) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func() error) error {
		if !policy.enabled(ctx, log, fullMethod) {
			return call()
		}
		started := time.Now()
		err := call()

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

func (p accessLogPolicy) enabled(ctx context.Context, log *slog.Logger, fullMethod string) bool {
	return (p.logHealthChecks || !isHealthMethod(fullMethod)) &&
		log.Enabled(ctx, slog.LevelInfo)
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

func errorMappingAround(mappers []problem.Mapper) aroundRPC {
	return func(_ context.Context, fullMethod string, call func() error) error {
		err := call()
		if isHealthMethod(fullMethod) {
			return err
		}
		return mapError(err, ownedStatusOnly, mappers)
	}
}

func policyErrorBoundaryAround() aroundRPC {
	return func(_ context.Context, _ string, call func() error) error {
		return mapError(call(), anyServiceStatus, nil)
	}
}

// trustedStatus reports whether one error boundary may hand err's gRPC status
// to the caller unchanged, and yields the status error to return. Reporting
// false sanitizes the error into a generic INTERNAL. The two implementations
// below are the only difference between this transport's two error boundaries.
type trustedStatus func(err error) (error, bool)

// ownedStatusOnly trusts only a status this package built from repository
// policy, so a generated handler's own status.Error cannot make a dependency's
// code and detail the client's answer by accident.
func ownedStatusOnly(err error) (error, bool) {
	if owned, ok := errors.AsType[*ownedStatusError](err); ok {
		return owned, true
	}
	return nil, false
}

// anyServiceStatus trusts any status the error itself carries. The boundary
// using it wraps the policy interceptors supplied through Options.UnaryPolicy
// and Options.StreamPolicy, which live in other packages and therefore cannot
// construct an ownedStatusError; an ordinary status.Error already carries a
// status, which is the whole of what this boundary asks for, and
// TestServerPolicyErrorBoundary returns exactly that from a policy. It
// deliberately does not unwrap: only a status a policy chose to return directly
// is its own output.
func anyServiceStatus(err error) (error, bool) {
	if statusErr, ok := err.(interface{ GRPCStatus() *status.Status }); ok && statusErr.GRPCStatus() != nil {
		return err, true
	}
	return nil, false
}

// mapError converts a handler or policy failure into the status this transport
// returns. Cancellation and deadlines answer first at every boundary because
// they are the caller's own signal rather than a service outcome; trusted then
// decides how much of the remaining error is already deliberate output.
func mapError(err error, trusted trustedStatus, mappers []problem.Mapper) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return ownedStatus(codes.Canceled, "request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ownedStatus(codes.DeadlineExceeded, "request deadline exceeded")
	}
	if owned, ok := trusted(err); ok {
		return owned
	}
	if mapped, ok := problem.Classify(err, mappers); ok {
		return mappedStatus(mapped)
	}
	return ownedStatus(codes.Internal, "request failed")
}

func mappedStatus(mapped problem.Mapped) error {
	code := codes.Internal
	switch mapped.Code {
	case problem.CodeBadRequest, problem.CodeUnprocessableContent:
		code = codes.InvalidArgument
	case problem.CodeUnauthorized:
		code = codes.Unauthenticated
	case problem.CodeForbidden:
		code = codes.PermissionDenied
	case problem.CodeNotFound:
		code = codes.NotFound
	case problem.CodeMethodNotAllowed:
		code = codes.Unimplemented
	case problem.CodeConflict:
		code = codes.Aborted
	case problem.CodeRequestEntityTooLarge, problem.CodeTooManyRequests:
		code = codes.ResourceExhausted
	// profile:authn-oidc-jwt:start
	case problem.CodeRequestHeaderFieldsTooLarge:
		code = codes.ResourceExhausted
	// profile:authn-oidc-jwt:end
	case problem.CodeServiceUnavailable:
		code = codes.Unavailable
	case problem.CodeGatewayTimeout:
		code = codes.DeadlineExceeded
	case problem.CodeInternalError:
		code = codes.Internal
	}

	detail := strings.TrimSpace(mapped.Detail)
	if detail == "" {
		if definition, ok := problem.ForCode(mapped.Code); ok {
			detail = definition.Title
		} else {
			detail = "request failed"
		}
	}
	return ownedStatus(code, detail)
}

func recoveryAround(log *slog.Logger) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func() error) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logRecoveredPanic(ctx, log, fullMethod, recovered)
				err = ownedStatus(codes.Internal, "request failed")
			}
		}()
		return call()
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
// kind; TestAdmissionLimitIsSharedAcrossUnaryAndStreamingRPCs proves it end to
// end.
func (l *admissionLimiter) around(ctx context.Context, fullMethod string, call func() error) error {
	if isHealthMethod(fullMethod) {
		return call()
	}
	if !l.sem.TryAcquire(1) {
		l.load.Shed(ctx)
		return ownedStatus(codes.ResourceExhausted, "server is at capacity")
	}
	defer l.sem.Release(1)
	release := l.load.Admitted(ctx)
	defer release()
	return call()
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

// ownedStatusError is the provenance marker for a status this adapter created
// from repository-owned policy. A handler's ordinary status.Error does not
// carry this marker and is therefore sanitized by ownedStatusOnly.
type ownedStatusError struct {
	status *status.Status
}

func ownedStatus(code codes.Code, detail string) error {
	return &ownedStatusError{status: status.New(code, detail)}
}

func (e *ownedStatusError) Error() string {
	return e.status.Err().Error()
}

func (e *ownedStatusError) GRPCStatus() *status.Status {
	return e.status
}
