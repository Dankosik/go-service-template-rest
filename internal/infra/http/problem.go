package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/example/go-service-template-rest/internal/api"
)

const problemJSONContentType = "application/problem+json; charset=utf-8"
const malformedRequestProblemDetail = "request is malformed or invalid"

func writeProblem(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	p := api.Problem{
		Type:   "about:blank",
		Title:  title,
		Status: int32(status),
		Detail: optionalProblemString(detail),
	}
	if r != nil {
		p.RequestId = optionalProblemString(requestIDFromContext(r.Context()))
	}

	w.Header().Set("Content-Type", problemJSONContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

func writeMalformedRequestProblem(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, http.StatusBadRequest, "bad request", malformedRequestProblemDetail)
}

func optionalProblemString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
