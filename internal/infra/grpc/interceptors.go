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
	requestIDMetadataKey = "x-request-id"
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
	candidate := ""
	if len(values) == 1 {
		candidate = values[0]
	}
	return reqctx.ContextWithAcceptedRequestID(ctx, candidate)
}

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

func accessLogUnaryInterceptor(log *slog.Logger, policy accessLogPolicy) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (response any, err error) {
		if !policy.enabled(ctx, log, info.FullMethod) {
			return handler(ctx, request)
		}
		started := time.Now()
		response, err = handler(ctx, request)
		logRPCCompletion(ctx, log, info.FullMethod, err, time.Since(started), policy)
		return response, err
	}
}

func accessLogStreamInterceptor(log *slog.Logger, policy accessLogPolicy) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if !policy.enabled(stream.Context(), log, info.FullMethod) {
			return handler(server, stream)
		}
		started := time.Now()
		err := handler(server, stream)
		logRPCCompletion(stream.Context(), log, info.FullMethod, err, time.Since(started), policy)
		return err
	}
}

func (p accessLogPolicy) enabled(ctx context.Context, log *slog.Logger, fullMethod string) bool {
	return (p.logHealthChecks || !isHealthMethod(fullMethod)) &&
		log.Enabled(ctx, slog.LevelInfo)
}

func logRPCCompletion(
	ctx context.Context,
	log *slog.Logger,
	fullMethod string,
	err error,
	elapsed time.Duration,
	policy accessLogPolicy,
) {
	code := status.Code(err)
	if !policy.shouldLog(ctx, code, elapsed) {
		return
	}
	log.LogAttrs(
		ctx,
		slog.LevelInfo,
		"grpc_request",
		slog.String("rpc.method", fullMethod),
		slog.String("rpc.status", code.String()),
		slog.Int64("duration_ms", elapsed.Milliseconds()),
	)
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
	// without a shared RNG, lock, or allocation on the RPC path.
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

func errorMappingUnaryInterceptor(mappers []problem.Mapper) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		response, err := handler(ctx, request)
		if isHealthMethod(info.FullMethod) {
			return response, err
		}
		return response, mapError(err, mappers)
	}
}

func errorMappingStreamInterceptor(mappers []problem.Mapper) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(server, stream)
		if isHealthMethod(info.FullMethod) {
			return err
		}
		return mapError(err, mappers)
	}
}

func policyErrorBoundaryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		response, err := handler(ctx, request)
		return response, mapPolicyError(err)
	}
}

func policyErrorBoundaryStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return mapPolicyError(handler(server, stream))
	}
}

func mapPolicyError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return ownedStatus(codes.Canceled, "request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ownedStatus(codes.DeadlineExceeded, "request deadline exceeded")
	}
	if statusErr, ok := err.(interface{ GRPCStatus() *status.Status }); ok && statusErr.GRPCStatus() != nil {
		return err
	}
	return ownedStatus(codes.Internal, "request failed")
}

func mapError(err error, mappers []problem.Mapper) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return ownedStatus(codes.Canceled, "request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ownedStatus(codes.DeadlineExceeded, "request deadline exceeded")
	}
	if owned, ok := errors.AsType[*ownedStatusError](err); ok {
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

func recoveryUnaryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (response any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logRecoveredPanic(ctx, log, info.FullMethod, recovered)
				response = nil
				err = ownedStatus(codes.Internal, "request failed")
			}
		}()
		return handler(ctx, request)
	}
}

func recoveryStreamInterceptor(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logRecoveredPanic(stream.Context(), log, info.FullMethod, recovered)
				err = ownedStatus(codes.Internal, "request failed")
			}
		}()
		return handler(server, stream)
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

func admissionUnaryInterceptor(limiter *admissionLimiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if isHealthMethod(info.FullMethod) {
			return handler(ctx, request)
		}
		if !limiter.sem.TryAcquire(1) {
			limiter.load.Shed(ctx)
			return nil, ownedStatus(codes.ResourceExhausted, "server is at capacity")
		}
		defer limiter.sem.Release(1)
		release := limiter.load.Admitted(ctx)
		defer release()
		return handler(ctx, request)
	}
}

func admissionStreamInterceptor(limiter *admissionLimiter) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if isHealthMethod(info.FullMethod) {
			return handler(server, stream)
		}
		if !limiter.sem.TryAcquire(1) {
			limiter.load.Shed(stream.Context())
			return ownedStatus(codes.ResourceExhausted, "server is at capacity")
		}
		defer limiter.sem.Release(1)
		release := limiter.load.Admitted(stream.Context())
		defer release()
		return handler(server, stream)
	}
}

func isHealthMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, healthMethodPrefix)
}

// ownedStatusError is the provenance marker for a status this adapter created
// from repository-owned policy. A handler's ordinary status.Error does not
// carry this marker and is therefore sanitized by mapError.
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
