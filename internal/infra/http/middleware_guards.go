package httpx

import (
	"net/http"
	"strings"
)

const contentTypeOptionsHeader = "X-Content-Type-Options"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeOptionsHeader, "nosniff")
		next.ServeHTTP(w, r)
	})
}

func RequestFramingGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasTransferEncoding := len(r.TransferEncoding) > 0 || strings.TrimSpace(r.Header.Get("Transfer-Encoding")) != ""
		hasContentLength := strings.TrimSpace(r.Header.Get("Content-Length")) != ""
		if hasTransferEncoding && hasContentLength {
			w.Header().Set("Connection", "close")
			writeProblem(w, r, http.StatusBadRequest, "bad request", "invalid request framing")
			return
		}

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
			writeProblem(w, r, http.StatusRequestEntityTooLarge, "request entity too large", "request body exceeds limit")
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}
