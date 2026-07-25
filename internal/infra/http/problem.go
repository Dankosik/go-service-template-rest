package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
)

const (
	problemJSONContentType        = "application/problem+json; charset=utf-8"
	malformedRequestProblemDetail = "request is malformed or invalid"
)

type problemResponse struct {
	code   problem.Code
	detail string
}

func writeProblem(w http.ResponseWriter, r *http.Request, response problemResponse) {
	definition := problemDefinitionFor(response.code)
	p := openapi.Problem{
		Code:      string(definition.Code),
		Detail:    optionalProblemString(response.detail),
		Instance:  nil,
		RequestId: nil,
		Status:    int32(definition.Status), // #nosec G115 -- catalog entries are fixed HTTP status constants.
		Title:     definition.Title,
		Type:      definition.TypeURI,
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

func writeMalformedRequestProblem(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, problemResponse{
		code:   problem.CodeBadRequest,
		detail: malformedRequestProblemDetail,
	})
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
