package httpx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
)

const (
	problemJSONContentType        = "application/problem+json; charset=utf-8"
	malformedRequestProblemDetail = "request is malformed or invalid"
	// sanitizedFailureDetail is the client-visible text for every failure this
	// transport refused to describe: an unclassified handler error and a
	// recovered panic. One constant because the two must stay indistinguishable
	// to a caller — a detail that separated them would report which internal
	// path broke, which is the whole thing the sanitization withholds.
	sanitizedFailureDetail = "request failed"
)

type problemResponse struct {
	code   problem.Code
	detail string
	// invalidParams is set only by the validator-rejection path. Every other
	// caller leaves it empty, because a failure this service chose to sanitize
	// has nothing field-shaped to point at.
	invalidParams []fieldViolation
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

func writeProblem(w http.ResponseWriter, r *http.Request, response problemResponse) {
	definition := problemDefinitionFor(response.code)
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
		p.RequestId = optionalProblemString(requestIDFromContext(r.Context()))
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

// problemDefinitionFor resolves a code against the shared catalog, substituting
// the internal-error entry for one it does not publish.
//
// The substitution is safe because the callers are this package's own fallback
// paths, which pass constants from that same catalog. A caller that can pass an
// arbitrary status uses problem.For, which refuses instead — see its
// documentation for why a plausible wrong answer is the worse failure.
func problemDefinitionFor(code problem.Code) problem.Definition {
	if definition, ok := problem.ForCode(code); ok {
		return definition
	}
	internalError, _ := problem.ForCode(problem.CodeInternalError)
	return internalError
}

func optionalProblemString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
