package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/problem"
)

// RejectResponse returns the mapper this repository installs for generated
// strict-server response failures: a spent request budget becomes 504, whatever
// the supplied mappers classify becomes their problem, and anything left becomes
// 500.
//
// A service wiring its own generated strict server needs this, or every slow
// dependency hides inside its 5xx error rate. Passing mappers is what lets its
// handlers return a domain error instead of hand-building a typed problem
// response per operation; see failure.Mapper.
//
// log receives the failures no mapper claimed, which are the ones answered with
// a detail-free 500. A nil logger keeps that answer and loses the only record of
// what caused it, so pass the service's own.
func RejectResponse(log *slog.Logger, domainErrors ...failure.Mapper) func(http.ResponseWriter, *http.Request, error) {
	return handleGeneratedResponseError(log, domainErrors)
}

// handleGeneratedResponseError turns an error a generated operation returned into
// a problem response.
//
// The expired-context case is checked before any service mapper: it is a
// transport fact, and a mapper that forgot it would hide every slow dependency
// inside the 5xx rate.
//
// log is only ever used for the unclassified case. A classified failure is an
// answer this service chose, and the access log already carries its problem code;
// recording it again at ERROR would put every 404 in the error stream.
func handleGeneratedResponseError(log *slog.Logger, domainErrors []failure.Mapper) func(http.ResponseWriter, *http.Request, error) {
	domainErrors = slices.Clone(domainErrors)
	return func(w http.ResponseWriter, r *http.Request, err error) {
		// A handler that returns its expired context is reporting a spent
		// request budget, not an internal fault, and this is the path most
		// timeouts actually take: the generated wrapper commits a response
		// here, so RequestTimeout never sees an uncommitted one. Reporting
		// it as 500 would hide every slow dependency inside the error rate.
		if errors.Is(err, context.DeadlineExceeded) {
			writeProblem(w, r, timeBudgetExceededProblem())
			return
		}

		// Cancellation answers here rather than falling through to the
		// unclassified arm below. A caller that hung up is not a fault this
		// service can act on, and treating one as unclassified spends an ERROR
		// record and a 500 on every abandoned request — the two signals an
		// operator watches to decide whether the service is broken.
		//
		// It shares the 504 class with the budget above because HTTP has no
		// portable client-canceled status, which is the resolution
		// handleGeneratedRequestError already applies to a canceled trust check.
		// The two stay separable without a second code: an abandoned request
		// ends well inside the budget, and the access log carries its duration.
		if errors.Is(err, context.Canceled) {
			writeProblem(w, r, problemResponse{
				code:   problem.CodeGatewayTimeout,
				detail: "request was canceled by the caller",
			})
			return
		}

		if mapped, ok := failure.Classify(err, domainErrors); ok {
			if mapped.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(mapped.RetryAfter)))
			}
			writeProblem(w, r, problemResponse{code: problem.Code(mapped.Code), detail: mapped.Detail})
			return
		}

		// An unclassified error is a fault this service did not anticipate, so it
		// stays a 500 with no detail: guessing a friendlier status for it is how a
		// bug starts reading as a client mistake. Recording it here is what keeps
		// that sanitized answer diagnosable.
		recordUnhandledResponseError(log, r, err)
		writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: failure.SanitizedDetail})
	}
}

// recordUnhandledResponseError writes down the failure behind a sanitized 500.
//
// It is the only place the error a generated operation actually returned is
// recorded. The problem body carries no detail on purpose, and the access log
// carries only the status and problem code, so without this the whole %w chain a
// handler built is discarded at the boundary: an operator holding a 500 and a
// request id has no way to tell a dead connection pool from a nil map without
// reproducing the request.
//
// Correlation is not assembled here; internal/observability/logctx publishes
// request_id, trace_id, and span_id from the context the record is logged with.
//
// What it records is the class chain, never the message. This package already
// refuses to put a parser's text in a rejection record — see logStrictRequestError
// — and a handler's error is no safer: it can carry a credential, a DSN, or a
// row. See failure.ClassChain for what the chain buys and what it costs. The
// route is the matched template rather than the path, for the reason
// TestRecoverLogsPanicClassWithoutRawValue pins: a path carries identifiers.
//
// The span is deliberately untouched: RecordError would publish the same text as
// exception.message.
func recordUnhandledResponseError(log *slog.Logger, r *http.Request, err error) {
	// The generated strict server calls its response handler only with a non-nil
	// error, but RejectResponse is exported and reaches this from callers this
	// package does not own. Nothing below is worth a record without one.
	if err == nil {
		return
	}
	// A nil request is handled for the same reason logStrictRequestError handles
	// one: the generated error handlers are declared with a request, and the path
	// whose job is reporting a failure must not become a second one.
	//nolint:contextcheck // There is no parent context when the request is nil, which is the only case this branch exists for.
	ctx := context.Background()
	route := "<unmatched>"
	if r != nil {
		ctx = r.Context()
		if matched := joinMethodAndPattern(r.Method, routePathTemplateForRequest(r)); matched != "" {
			route = matched
		}
	}

	if log == nil {
		return
	}
	log.LogAttrs(
		ctx,
		slog.LevelError,
		"http_unhandled_failure",
		slog.String("route", route),
		slog.String("error_chain", failure.ClassChain(err)),
	)
}

// The status-to-problem lookup lives in internal/problem, not here: depguard
// forbids a feature package from importing this adapter, so a helper exported
// from here would be reachable only by another adapter.
