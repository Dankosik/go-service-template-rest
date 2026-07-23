package httpx

import (
	"net/http"

	"github.com/example/go-service-template-rest/internal/requestmeta"
)

const requestIDHeader = "X-Request-ID"

func RequestCorrelation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, requestID := requestmeta.ContextWithRequestID(r.Context(), r.Header.Get(requestIDHeader))

		w.Header().Set(requestIDHeader, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
