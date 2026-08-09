package grpcx

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/example/go-service-template-rest/internal/failure"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/types/known/durationpb"
)

// handlerErrorBoundary and [policyErrorBoundary] are this transport's two error
// boundaries. They are the same mechanism — [mapError] — and differ only in the
// [trustedStatus] each passes it; the package doc owns why they sit where they
// do in the chain.
//
// This one is innermost and sanitizes what a generated handler returns. Standard
// health RPCs pass through untouched so their own status semantics survive.
func handlerErrorBoundary(log *slog.Logger, rendering errorRendering) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) error {
		err := call(ctx)
		if isHealthMethod(fullMethod) {
			return err
		}
		mapped, sanitized := mapError(err, ownedStatusOnly, rendering)
		if sanitized {
			recordUnhandledFailure(ctx, log, fullMethod, err)
		}
		return mapped
	}
}

// errorRendering is everything a classified answer needs beyond the error
// itself: which mappers may claim it, and the domain that scopes the reason a
// caller matches on. The two travel together because every signature between the
// boundary and the status would otherwise carry an unnamed pair.
type errorRendering struct {
	mappers []failure.Mapper
	// domain scopes ErrorInfo.Reason, so two services' reasons cannot collide.
	// Empty omits the detail rather than publishing an unscoped reason; the
	// composition root always supplies one, and exhaustruct on [Options] is what
	// keeps that true.
	domain string
}

// policyErrorBoundary sanitizes what a supplied policy interceptor returns.
//
// It closes over the logger for the same reason its siblings in
// [builtinPolicies] close over their collaborators: a policy failure this
// boundary refuses to hand the caller is discarded unless something writes it
// down, and this is the only position that still holds it.
func policyErrorBoundary(log *slog.Logger) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) error {
		err := call(ctx)
		mapped, sanitized := mapError(err, anyServiceStatus, errorRendering{})
		if sanitized {
			recordUnhandledFailure(ctx, log, fullMethod, err)
		}
		return mapped
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
//
// The second result reports that err reached the caller as a generic INTERNAL
// carrying none of what actually failed. That is the one outcome worth a record,
// and it is returned rather than logged here so this function stays a pure
// mapping: the context and method a record needs live at the boundary, not in
// the mapping.
func mapError(err error, trusted trustedStatus, rendering errorRendering) (error, bool) {
	if err == nil {
		return nil, false
	}
	if errors.Is(err, context.Canceled) {
		return ownedStatus(codes.Canceled, "request canceled"), false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ownedStatus(codes.DeadlineExceeded, "request deadline exceeded"), false
	}
	if owned, ok := trusted(err); ok {
		return owned, false
	}
	if mapped, ok := failure.Classify(err, rendering.mappers); ok {
		return mappedStatus(mapped, rendering.domain), false
	}
	return ownedStatus(codes.Internal, failure.SanitizedDetail), true
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
// TestServerObservabilitySanitizesValuesAndFiltersUnknownMethods drives a
// handler returning a secret and asserts none of it reaches here. See
// failure.ClassChain for what that buys and what it costs.
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
	// profile:authn-oidc-jwt:start
	case failure.CodeRequestHeaderFieldsTooLarge:
		code = codes.ResourceExhausted
	// profile:authn-oidc-jwt:end
	case failure.CodeServiceUnavailable:
		code = codes.Unavailable
	case failure.CodeGatewayTimeout:
		code = codes.DeadlineExceeded
	case failure.CodeInternalError:
		code = codes.Internal
	}

	detail := strings.TrimSpace(mapped.Detail)
	if detail == "" {
		detail = failure.SanitizedDetail
	}

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
