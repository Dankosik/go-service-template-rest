package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/example/go-service-template-rest/internal/openapi"
)

const (
	problemJSONContentType        = "application/problem+json; charset=utf-8"
	malformedRequestProblemDetail = "request is malformed or invalid"
)

type problemResponse struct {
	code   problemCode
	detail string
}

type problemCode string

const (
	problemCodeBadRequest            problemCode = "bad_request"
	problemCodeUnauthorized          problemCode = "unauthorized"
	problemCodeForbidden             problemCode = "forbidden"
	problemCodeNotFound              problemCode = "not_found"
	problemCodeMethodNotAllowed      problemCode = "method_not_allowed"
	problemCodeConflict              problemCode = "conflict"
	problemCodeRequestEntityTooLarge problemCode = "request_entity_too_large"
	problemCodeUnprocessableContent  problemCode = "unprocessable_content"
	problemCodeTooManyRequests       problemCode = "too_many_requests"
	problemCodeInternalError         problemCode = "internal_error"
	problemCodeServiceUnavailable    problemCode = "service_unavailable"
	problemCodeGatewayTimeout        problemCode = "gateway_timeout"
)

type problemDefinition struct {
	status  int
	title   string
	typeURI string
}

// problemCatalog is the single source of the problem envelope this repository
// publishes, and ProblemTypeURI resolves against this same map.
//
// The version this replaced kept a second, hand-maintained list of codes for the
// status lookup. Adding a code here without adding it there made the lookup fall
// through to the internal-error type, so a service filling a generated 409 got a
// response advertising itself as a server fault while carrying status 409 —
// which is exactly what a client keying its retry policy off `type` acts on.
//
// 409, 422, and 429 are here because a domain layer produces them and the
// runtime cannot: no fallback path in this package emits them, so they exist for
// an operation that declares them. A code with no matching `components/responses`
// entry in the contract is unreachable, not wrong.
var problemCatalog = map[problemCode]problemDefinition{
	problemCodeBadRequest: {
		status:  http.StatusBadRequest,
		title:   "bad request",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.1",
	},
	problemCodeUnauthorized: {
		status:  http.StatusUnauthorized,
		title:   "unauthorized",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.2",
	},
	problemCodeForbidden: {
		status:  http.StatusForbidden,
		title:   "forbidden",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.4",
	},
	problemCodeNotFound: {
		status:  http.StatusNotFound,
		title:   "not found",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5",
	},
	problemCodeMethodNotAllowed: {
		status:  http.StatusMethodNotAllowed,
		title:   "method not allowed",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.6",
	},
	problemCodeConflict: {
		status:  http.StatusConflict,
		title:   "conflict",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10",
	},
	problemCodeRequestEntityTooLarge: {
		status:  http.StatusRequestEntityTooLarge,
		title:   "request entity too large",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.14",
	},
	problemCodeUnprocessableContent: {
		status:  http.StatusUnprocessableEntity,
		title:   "unprocessable content",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.21",
	},
	problemCodeTooManyRequests: {
		status: http.StatusTooManyRequests,
		title:  "too many requests",
		// RFC 9110 stops at 426; 429 is still defined by RFC 6585.
		typeURI: "https://www.rfc-editor.org/rfc/rfc6585#section-4",
	},
	problemCodeInternalError: {
		status:  http.StatusInternalServerError,
		title:   "internal server error",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1",
	},
	problemCodeServiceUnavailable: {
		status:  http.StatusServiceUnavailable,
		title:   "service unavailable",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.4",
	},
	problemCodeGatewayTimeout: {
		status:  http.StatusGatewayTimeout,
		title:   "gateway timeout",
		typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.5",
	},
}

func writeProblem(w http.ResponseWriter, r *http.Request, problem problemResponse) {
	code, definition := problemDefinitionFor(problem.code)
	p := openapi.Problem{
		Code:      string(code),
		Detail:    optionalProblemString(problem.detail),
		Instance:  nil,
		RequestId: nil,
		Status:    int32(definition.status), // #nosec G115 -- catalog entries are fixed HTTP status constants.
		Title:     definition.title,
		Type:      definition.typeURI,
	}
	if r != nil {
		p.RequestId = optionalProblemString(requestIDFromContext(r.Context()))
	}

	w.Header().Set("Content-Type", problemJSONContentType)
	w.WriteHeader(definition.status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		// The status and headers are already committed; callers cannot recover here.
		return
	}
}

func writeMalformedRequestProblem(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, problemResponse{
		code:   problemCodeBadRequest,
		detail: malformedRequestProblemDetail,
	})
}

// problemDefinitionFor resolves a code, substituting the internal-error entry for
// one this package does not publish. The substitution is safe because the caller
// is this package's own fallback paths, which pass constants; a caller that can
// pass an arbitrary status uses ProblemTypeURI, which refuses instead.
func problemDefinitionFor(code problemCode) (problemCode, problemDefinition) {
	if definition, ok := problemCatalog[code]; ok {
		return code, definition
	}
	return problemCodeInternalError, problemCatalog[problemCodeInternalError]
}

func optionalProblemString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
