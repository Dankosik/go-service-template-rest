package httpx

import (
	"net/http"
)

const contentTypeOptionsHeader = "X-Content-Type-Options"

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
			writeProblem(w, r, problemResponse{code: problemCodeRequestEntityTooLarge, detail: "request body exceeds limit"})
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}
