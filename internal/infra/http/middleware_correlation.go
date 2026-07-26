package httpx

import (
	"net/http"
)

const requestIDHeader = "X-Request-ID"

func RequestCorrelation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, requestID := contextWithRequestID(r.Context(), r.Header.Get(requestIDHeader))
		// Installed outermost so every layer below can report the problem it
		// answered with, including the ones inside the generated router that serve
		// a request struct this one never sees. AccessLog reads it back.
		ctx, _ = contextWithProblemRecord(ctx)

		w.Header().Set(requestIDHeader, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
