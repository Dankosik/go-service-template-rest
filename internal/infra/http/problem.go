package httpx

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/example/go-service-template-rest/internal/reqctx"
)

const (
	problemJSONContentType        = "application/problem+json; charset=utf-8"
	malformedRequestProblemDetail = "request is malformed or invalid"
)

type problemResponse struct {
	code   problem.Code
	detail string
	// invalidParams is set only by the validator-rejection path. Every other
	// caller leaves it empty, because a failure this service chose to sanitize
	// has nothing field-shaped to point at.
	invalidParams []fieldViolation
}

// notFoundProblem is what an unrouted path answers.
//
// Two chi callbacks reach it: NotFound, and MethodNotAllowed for a path that
// matched no route at all, which chi still routes through the method callback.
// They answer identically on purpose, and a name says so where two identical
// composite literals six lines apart only let a reader hope so.
func notFoundProblem() problemResponse {
	return problemResponse{code: problem.CodeNotFound, detail: "resource not found"}
}

// timeBudgetExceededProblem is what a request that outlived its deadline
// answers.
//
// Two layers reach it: the generated strict-server wrapper usually commits the
// response first, and RequestTimeout is the backstop for a handler chain built
// on the exported Harden that never goes through one. Which fires depends on how
// a service is wired rather than on anything the caller did, so the caller must
// not be able to tell them apart.
func timeBudgetExceededProblem() problemResponse {
	return problemResponse{code: problem.CodeGatewayTimeout, detail: "request exceeded its time budget"}
}

// problemRecord carries the problem code this request was answered with back out
// to the access log.
//
// It is a pointer in the context rather than a value written onto the request,
// because the request struct the log sees is not the one a handler answers on:
// chi replaces it when it installs its route context, so an in-place mutation
// deeper in the chain is invisible from outside the generated router. One shared
// holder installed at the top is what survives that.
//
// The value is only read after the handler has returned, on the same goroutine
// that served the request, so it needs no synchronization.
type problemRecord struct {
	code problem.Code
}

type problemRecordContextKey struct{}

func contextWithProblemRecord(ctx context.Context) (context.Context, *problemRecord) {
	record := &problemRecord{}
	return context.WithValue(ctx, problemRecordContextKey{}, record), record
}

// recordProblemCode publishes the code answered for this request. The first one
// wins: an outer middleware that replaces an inner response — Recover turning a
// panic into a 500 — is reporting the same failure, and the innermost code is the
// one that says what actually went wrong.
func recordProblemCode(ctx context.Context, code problem.Code) {
	record, ok := ctx.Value(problemRecordContextKey{}).(*problemRecord)
	if !ok || record.code != "" {
		return
	}
	record.code = code
}

// problemCodeForRequest returns the problem code this request was answered with,
// or the empty string when it was not answered with one.
func problemCodeForRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	record, ok := r.Context().Value(problemRecordContextKey{}).(*problemRecord)
	if !ok {
		return ""
	}
	return string(record.code)
}

// retryAfterSeconds rounds up: a sub-second budget must not be advertised as
// "retry immediately".
func retryAfterSeconds(delay time.Duration) int {
	seconds := int(math.Ceil(delay.Seconds()))
	return max(seconds, 1)
}

func writeProblem(w http.ResponseWriter, r *http.Request, response problemResponse) {
	definition := problem.ForCodeOrInternal(response.code)
	if r != nil {
		recordProblemCode(r.Context(), definition.Code)
	}
	p := openapi.Problem{
		Code:          string(definition.Code),
		Detail:        optionalProblemString(response.detail),
		Instance:      nil,
		InvalidParams: optionalInvalidParams(response.invalidParams),
		RequestId:     nil,
		Status:        int32(definition.Status), // #nosec G115 -- catalog entries are fixed HTTP status constants.
		Title:         definition.Title,
		Type:          definition.TypeURI,
	}
	if r != nil {
		p.RequestId = optionalProblemString(reqctx.RequestID(r.Context()))
	}

	w.Header().Set("Content-Type", problemJSONContentType)
	w.WriteHeader(definition.Status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		// The status and headers are already committed; callers cannot recover here.
		return
	}
}

func writeMalformedRequestProblem(w http.ResponseWriter, r *http.Request, violations []fieldViolation) {
	writeProblem(w, r, problemResponse{
		code:          problem.CodeBadRequest,
		detail:        malformedRequestProblemDetail,
		invalidParams: violations,
	})
}

func optionalInvalidParams(violations []fieldViolation) *[]openapi.InvalidParam {
	if len(violations) == 0 {
		return nil
	}
	params := make([]openapi.InvalidParam, 0, len(violations))
	for _, violation := range violations {
		params = append(params, openapi.InvalidParam{Name: violation.Field, Reason: violation.Reason})
	}
	return &params
}

func optionalProblemString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
