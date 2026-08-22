package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/example/go-service-template-rest/internal/failure"
	// profile:authn-bearer:start
	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	// profile:authn-bearer:end
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// defaultAuthenticateChallenge is the HTTP authentication scheme advertised when
// a service declares security requirements without naming its own challenge.
const defaultAuthenticateChallenge = "Bearer"

// RejectRequest returns the validator error mapper this repository installs:
// oversized bodies become 413, failed security requirements become 401 with a
// WWW-Authenticate challenge, and everything else becomes a sanitized 400.
//
// A service wiring its own generated validator needs this, or it reproduces the
// defect where a missing credential is reported as a malformed request.
func RejectRequest(log *slog.Logger, challenge string) func(http.ResponseWriter, *http.Request, error) {
	if strings.TrimSpace(challenge) == "" {
		challenge = defaultAuthenticateChallenge
	}
	return handleGeneratedRequestError(log, challenge)
}

func handleMalformedGeneratedRequest(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	logStrictRequestError(log, r, err)
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		writeProblem(w, r, problemResponse{code: problem.CodeRequestEntityTooLarge, detail: "request body exceeds limit"})
		return
	}
	// The one place a rejection tells the caller more than "invalid". The detail
	// stays generic; requestViolations owns what may go beside it.
	writeMalformedRequestProblem(w, r, requestViolations(err))
}

// handleGeneratedRequestError maps a validator rejection, adding the one case
// handleMalformedGeneratedRequest cannot classify on its own.
//
// A failed security requirement is 401, not 400: the framing was fine and the
// credential was the problem, and no client library retries with credentials on
// a 400.
func handleGeneratedRequestError(log *slog.Logger, challenge string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		// profile:authn-bearer:start
		if kind, ok := bearerauthn.KindOf(err); ok {
			logStrictRequestError(log, r, err)
			// In bearerauthn.Kind declaration order, as that package's errors.go is, so
			// a category added there lands in one obvious place here.
			switch kind {
			case bearerauthn.KindMissing:
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeProblem(w, r, problemResponse{code: problem.CodeUnauthorized, detail: "credentials are missing"})
			case bearerauthn.KindMalformed:
				w.Header().Set("WWW-Authenticate", `Bearer error="invalid_request"`)
				writeProblem(w, r, problemResponse{code: problem.CodeBadRequest, detail: "authentication credential is malformed"})
			case bearerauthn.KindOversize:
				writeProblem(w, r, problemResponse{
					code:   problem.CodeRequestHeaderFieldsTooLarge,
					detail: "authentication credential is too large",
				})
			case bearerauthn.KindInvalid:
				w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
				writeProblem(w, r, problemResponse{code: problem.CodeUnauthorized, detail: "credentials are invalid"})
			case bearerauthn.KindUnavailable:
				w.Header().Set("Retry-After", "30")
				writeProblem(w, r, problemResponse{code: problem.CodeServiceUnavailable, detail: "authentication trust is unavailable"})
			default:
				w.Header().Set("WWW-Authenticate", challenge)
				writeProblem(w, r, problemResponse{code: problem.CodeUnauthorized, detail: "credentials are invalid"})
			}
			return
		}
		// profile:authn-bearer:end
		if _, ok := errors.AsType[*openapi3filter.SecurityRequirementsError](err); ok {
			logStrictRequestError(log, r, err)
			// SecurityRequirementsError wraps resolver failures. Preserve a
			// canceled trust check as a transport failure instead of misreporting
			// it as a bad credential. HTTP has no portable client-canceled status,
			// so both caller cancellation and an expired request budget use the
			// existing 504 class.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				writeProblem(w, r, problemResponse{
					code:   problem.CodeGatewayTimeout,
					detail: "authentication verification did not complete",
				})
				return
			}
			// The challenge names an HTTP authentication scheme, which is not the
			// same vocabulary as the contract's securityScheme names: a contract
			// key like "bearerAuth" is not a legal challenge under RFC 9110. Only
			// the service knows which scheme it implements, so it supplies this.
			w.Header().Set("WWW-Authenticate", challenge)
			writeProblem(w, r, problemResponse{code: problem.CodeUnauthorized, detail: "credentials are missing or invalid"})
			return
		}
		handleMalformedGeneratedRequest(log, w, r, err)
	}
}

func logStrictRequestError(log *slog.Logger, r *http.Request, err error) {
	if log == nil {
		return
	}

	// Correlation comes from the context rather than from attributes assembled
	// here; see internal/observability/logctx. A nil request still has to be
	// handled: the generated error handlers are declared with one, and this must
	// not panic on the path whose whole job is reporting a malformed request.
	//nolint:contextcheck // There is no parent context when the request is nil, which is the only case this branch exists for.
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	// The chain rather than the outer %T this used to publish: a validator
	// rejection arrives wrapped, so that type was *openapi3filter.RequestError for
	// a missing credential, a body that failed its schema, and an unparseable path
	// parameter alike.
	//
	// The fields are what make a 400 actionable at all. Without them an operator
	// holding a spike of them knows only that some client is sending something
	// wrong; with them they know it is `/slug` and can go look at the caller.
	// Only the names are recorded — requestViolations owns why the reasons stay
	// out of the record and go to the caller instead.
	attrs := []slog.Attr{slog.String("error_chain", failure.ClassChain(err))}
	if fields := violationFields(requestViolations(err)); len(fields) > 0 {
		attrs = append(attrs, slog.Any("invalid_fields", fields))
	}
	log.LogAttrs(ctx, slog.LevelWarn, "http_request_rejected", attrs...)
}
