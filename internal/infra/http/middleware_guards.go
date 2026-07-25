package httpx

import (
	"net/http"

	"github.com/example/go-service-template-rest/internal/problem"
)

const contentTypeOptionsHeader = "X-Content-Type-Options"

// SecurityHeaders sets the response headers that belong to a JSON API served
// behind a platform edge. That is one header: nosniff. HSTS belongs to whatever
// terminates TLS, and the framing and content-security headers protect documents
// a browser renders, which this service does not serve. A service that starts
// serving HTML adds them here with its own policy.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeOptionsHeader, "nosniff")
		next.ServeHTTP(w, r)
	})
}

func RequestBodyLimit(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if maxBytes <= 0 {
			next.ServeHTTP(w, r)
			return
		}
		if r.ContentLength > maxBytes {
			writeProblem(w, r, problemResponse{code: problem.CodeRequestEntityTooLarge, detail: "request body exceeds limit"})
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}
