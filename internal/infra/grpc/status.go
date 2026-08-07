package grpcx

import (
	"context"
	"errors"
	"strings"

	"github.com/example/go-service-template-rest/internal/problem"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// handlerErrorBoundary and [policyErrorBoundary] are this transport's two error
// boundaries. They are the same mechanism — [mapError] — and differ only in the
// [trustedStatus] each passes it; the package doc owns why they sit where they
// do in the chain.
//
// This one is innermost and sanitizes what a generated handler returns. Standard
// health RPCs pass through untouched so their own status semantics survive.
func handlerErrorBoundary(mappers []problem.Mapper) aroundRPC {
	return func(_ context.Context, fullMethod string, call func() error) error {
		err := call()
		if isHealthMethod(fullMethod) {
			return err
		}
		return mapError(err, ownedStatusOnly, mappers)
	}
}

// policyErrorBoundary sanitizes what a supplied policy interceptor returns.
//
// It is an [aroundRPC] directly rather than a constructor returning one, because
// it captures nothing: its siblings in [builtinPolicies] each close over a
// collaborator, and this one has none to close over.
func policyErrorBoundary(_ context.Context, _ string, call func() error) error {
	return mapError(call(), anyServiceStatus, nil)
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
