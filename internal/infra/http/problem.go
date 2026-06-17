package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/example/go-service-template-rest/internal/api"
)

const (
	problemJSONContentType        = "application/problem+json; charset=utf-8"
	malformedRequestProblemDetail = "request is malformed or invalid"
)

type problemResponse struct {
	status int
	title  string
	detail string
}

func writeProblem(w http.ResponseWriter, r *http.Request, problem problemResponse) {
	status := problemHTTPStatus(problem.status)
	p := api.Problem{
		Detail:    optionalProblemString(problem.detail),
		RequestId: nil,
		Status:    int32(status),
		Title:     problem.title,
		Type:      "about:blank",
	}
	if r != nil {
		p.RequestId = optionalProblemString(requestIDFromContext(r.Context()))
	}

	w.Header().Set("Content-Type", problemJSONContentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		// The status and headers are already committed; callers cannot recover here.
		return
	}
}

func writeMalformedRequestProblem(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, problemResponse{
		status: http.StatusBadRequest,
		title:  "bad request",
		detail: malformedRequestProblemDetail,
	})
}

func optionalProblemString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func problemHTTPStatus(status int) int {
	if status < 100 || status > 999 {
		return http.StatusInternalServerError
	}
	return status
}
