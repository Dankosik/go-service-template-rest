package grpcx

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/failure"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/types/known/durationpb"
)

// handlerErrorBoundary sanitizes what a generated handler returns. Standard
// health RPCs pass through untouched so their own status semantics survive.
func handlerErrorBoundary(log *slog.Logger, mappers []failure.Mapper) aroundRPC {
	mappers = slices.Clone(mappers)
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) error {
		err := call(ctx)
		if isHealthMethod(fullMethod) {
			return err
		}
		mapped, sanitized := mapHandlerError(err, mappers, methodDomain(fullMethod))
		if sanitized {
			recordUnhandledFailure(ctx, log, fullMethod, err)
		}
		return mapped
	}
}

// policyErrorBoundary sanitizes what a supplied policy interceptor returns.
func policyErrorBoundary(log *slog.Logger) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) error {
		err := call(ctx)
		mapped, sanitized := mapPolicyError(err)
		if sanitized {
			recordUnhandledFailure(ctx, log, fullMethod, err)
		}
		return mapped
	}
}

func mapHandlerError(err error, mappers []failure.Mapper, domain string) (error, bool) {
	if err == nil {
		return nil, false
	}
	if mapped, ok := mapContextError(err); ok {
		return mapped, false
	}
	if owned, ok := errors.AsType[*ownedStatusError](err); ok {
		return owned, false
	}
	// Generated Unimplemented<Service>Server methods return an ordinary status.
	// Preserve forward-compatible generated behavior without trusting its text.
	if grpcStatus, ok := directStatus(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return ownedStatus(codes.Unimplemented, "method not implemented"), false
	}
	if mapped, ok := failure.Classify(err, mappers); ok {
		return mappedStatus(mapped, domain), false
	}
	return ownedStatus(codes.Internal, failure.SanitizedDetail), true
}

func mapPolicyError(err error) (error, bool) {
	if err == nil {
		return nil, false
	}
	if mapped, ok := mapContextError(err); ok {
		return mapped, false
	}
	// Deliberately do not unwrap: only a status the policy returned directly is
	// service-owned output.
	if _, ok := directStatus(err); ok {
		return err, false
	}
	return ownedStatus(codes.Internal, failure.SanitizedDetail), true
}

func mapContextError(err error) (error, bool) {
	switch {
	case errors.Is(err, context.Canceled):
		return ownedStatus(codes.Canceled, "request canceled"), true
	case errors.Is(err, context.DeadlineExceeded):
		return ownedStatus(codes.DeadlineExceeded, "request deadline exceeded"), true
	default:
		return nil, false
	}
}

// directStatus inspects only err itself; errors.As would also trust a status a
// policy merely wrapped.
func directStatus(err error) (*status.Status, bool) {
	statusErr, ok := err.(interface{ GRPCStatus() *status.Status })
	if !ok {
		return nil, false
	}
	grpcStatus := statusErr.GRPCStatus()
	return grpcStatus, grpcStatus != nil
}

func methodDomain(fullMethod string) string {
	method := strings.TrimPrefix(fullMethod, "/")
	service, _, found := strings.Cut(method, "/")
	if !found {
		return ""
	}
	return service
}

// recordUnhandledFailure writes down the failure behind a generic INTERNAL.
//
// It is the only place the error a handler or policy actually returned is
// recorded. The status carries no detail on purpose — see the package doc on why
// a dependency's own text is not the caller's business — and the access log
// carries only the code, so without this the whole %w chain a service built is
// discarded at the boundary and an INTERNAL is undiagnosable without reproducing
// the call.
//
// Correlation is not assembled here; internal/observability/logctx publishes
// request_id, trace_id, and span_id from the context the record is logged with.
//
// What it records is the class chain, never the message. The error's text is
// exactly what this boundary refused to give the caller, and a log is not a
// safer place to put a credential than a status:
// TestUnhandledFailureLogOmitsHandlerText drives a handler returning a secret
// and asserts none of it reaches here. See failure.ClassChain for what that buys
// and what it costs.
//
// The span is deliberately untouched. RecordError would publish the same text as
// exception.message, and the bounded error.type this package could set instead is
// already derivable from the status otelgrpc reports.
func recordUnhandledFailure(ctx context.Context, log *slog.Logger, method string, err error) {
	log.LogAttrs(
		ctx,
		slog.LevelError,
		"grpc_unhandled_failure",
		slog.String("rpc.method", method),
		slog.String("error_chain", failure.ClassChain(err)),
	)
}

func mappedStatus(mapped failure.Classification, domain string) error {
	code := codes.Internal
	switch mapped.Code {
	case failure.CodeBadRequest, failure.CodeUnprocessableContent:
		code = codes.InvalidArgument
	// profile:http-idempotency-postgres:start
	case failure.CodeIdempotencyKeyMismatch:
		code = codes.InvalidArgument
	// profile:http-idempotency-postgres:end
	case failure.CodeUnauthorized:
		code = codes.Unauthenticated
	case failure.CodeForbidden:
		code = codes.PermissionDenied
	case failure.CodeNotFound:
		code = codes.NotFound
	case failure.CodeMethodNotAllowed:
		code = codes.Unimplemented
	case failure.CodeAlreadyExists:
		code = codes.AlreadyExists
	case failure.CodeRequestEntityTooLarge, failure.CodeTooManyRequests:
		code = codes.ResourceExhausted
	// profile:authn-bearer:start
	case failure.CodeRequestHeaderFieldsTooLarge:
		code = codes.ResourceExhausted
	// profile:authn-bearer:end
	case failure.CodeServiceUnavailable:
		code = codes.Unavailable
	// profile:http-idempotency-postgres:start
	case failure.CodeIdempotencyUnavailable, failure.CodeIdempotencyOutcomeUnknown:
		code = codes.Unavailable
	// profile:http-idempotency-postgres:end
	case failure.CodeGatewayTimeout:
		code = codes.DeadlineExceeded
	case failure.CodeInternalError:
		code = codes.Internal
	}

	detail := cmp.Or(strings.TrimSpace(mapped.Detail), failure.SanitizedDetail)

	rendered := status.New(code, detail)
	if details := classifiedDetails(mapped, domain); len(details) > 0 {
		// A detail that cannot be attached must not cost the caller its status.
		// The only documented failure is an OK code, which this function cannot
		// produce, so the arm exists to keep that promise rather than to handle
		// a case anyone has seen.
		if withDetails, err := rendered.WithDetails(details...); err == nil {
			rendered = withDetails
		}
	}
	return &ownedStatusError{status: rendered}
}

// classifiedDetails renders the parts of a classified answer that a caller
// should read as data rather than prose.
//
// Both carry repository-owned values only: the mapper's own delay, and the
// failure code. No handler text, dependency identity, or peer-supplied value
// becomes a detail.
//
// The reason is the code upper-snake-cased because google.rpc.ErrorInfo
// documents Reason as at most 63 characters matching [A-Z][A-Z0-9_]+[A-Z0-9].
// Every catalog code is lower snake case ASCII, so the rendering is total and
// injective; TestEveryFailureCodeRendersAConformingReason holds the whole
// catalog to it. failure.Code remains the only stored identity.
func classifiedDetails(mapped failure.Classification, domain string) []protoadapt.MessageV1 {
	details := make([]protoadapt.MessageV1, 0, 2)
	if mapped.RetryAfter > 0 {
		details = append(details, &errdetails.RetryInfo{
			RetryDelay: durationpb.New(mapped.RetryAfter),
		})
	}
	if domain != "" {
		details = append(details, &errdetails.ErrorInfo{
			Reason: strings.ToUpper(string(mapped.Code)),
			Domain: domain,
		})
	}
	return details
}

// ownedStatusError marks repository-owned output. Stream validation returns its
// status from RecvMsg inside the handler boundary, so chain position alone cannot
// distinguish it from a handler's ordinary status.Error.
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
