package httpx

import (
	"encoding/json"
	"net/http"
)

const problemJSONContentType = "application/problem+json; charset=utf-8"

type problemDetails struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	p := problemDetails{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	}
	if r != nil {
		p.RequestID = requestIDFromContext(r.Context())
	}

	w.Header().Set("Content-Type", problemJSONContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
