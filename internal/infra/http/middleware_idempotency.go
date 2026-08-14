package httpx

import (
	"context"
	"net/http"
	"slices"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
)

type (
	idempotencyKeyContextKey     struct{}
	idempotencyAttemptContextKey struct{}
)

func captureIdempotencyKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := slices.Clone(r.Header.Values(httpidempotency.Header))
		r = r.WithContext(context.WithValue(r.Context(), idempotencyKeyContextKey{}, values))
		next.ServeHTTP(w, r)
	})
}

func capturedIdempotencyKey(ctx context.Context) []string {
	values, _ := ctx.Value(idempotencyKeyContextKey{}).([]string)
	return values
}

func contextWithIdempotencyAttempt(ctx context.Context, attempt httpidempotency.Attempt) context.Context {
	return context.WithValue(ctx, idempotencyAttemptContextKey{}, attempt)
}

// IdempotencyAttemptFromContext returns the authenticated scope and raw key an
// opted handler may canonicalize before it enters the Store.
func IdempotencyAttemptFromContext(ctx context.Context) (httpidempotency.Attempt, bool) {
	attempt, ok := ctx.Value(idempotencyAttemptContextKey{}).(httpidempotency.Attempt)
	return attempt, ok
}
