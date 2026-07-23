package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/requestmeta"
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
	problemCodeNotFound              problemCode = "not_found"
	problemCodeMethodNotAllowed      problemCode = "method_not_allowed"
	problemCodeRequestEntityTooLarge problemCode = "request_entity_too_large"
	problemCodeInternalError         problemCode = "internal_error"
)

type problemDefinition struct {
	status  int
	title   string
	typeURI string
}

func writeProblem(w http.ResponseWriter, r *http.Request, problem problemResponse) {
	code, definition := problemDefinitionFor(problem.code)
	p := api.Problem{
		Code:      string(code),
		Detail:    optionalProblemString(problem.detail),
		Instance:  nil,
		RequestId: nil,
		Status:    int32(definition.status), // #nosec G115 -- catalog entries are fixed HTTP status constants.
		Title:     definition.title,
		Type:      definition.typeURI,
	}
	if r != nil {
		p.RequestId = optionalProblemString(requestmeta.RequestIDFromContext(r.Context()))
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

func problemDefinitionFor(code problemCode) (problemCode, problemDefinition) {
	switch code {
	case problemCodeBadRequest:
		return code, problemDefinition{
			status:  http.StatusBadRequest,
			title:   "bad request",
			typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.1",
		}
	case problemCodeNotFound:
		return code, problemDefinition{
			status:  http.StatusNotFound,
			title:   "not found",
			typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5",
		}
	case problemCodeMethodNotAllowed:
		return code, problemDefinition{
			status:  http.StatusMethodNotAllowed,
			title:   "method not allowed",
			typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.6",
		}
	case problemCodeRequestEntityTooLarge:
		return code, problemDefinition{
			status:  http.StatusRequestEntityTooLarge,
			title:   "request entity too large",
			typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.14",
		}
	case problemCodeInternalError:
		return code, problemDefinition{
			status:  http.StatusInternalServerError,
			title:   "internal server error",
			typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1",
		}
	default:
		return problemDefinitionFor(problemCodeInternalError)
	}
}

func optionalProblemString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
