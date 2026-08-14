package httpx

import (
	"context"
	"net/http"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

type idempotencyEnvelope struct {
	matcher          routers.Router
	registry         idempotencyRegistry
	terminalObserver func(context.Context, httpidempotency.Decision, error)
}

func newIdempotencyEnvelope(
	spec *openapi3.T,
	operations []IdempotencyOperation,
	terminalObserver func(context.Context, httpidempotency.Decision, error),
) (idempotencyEnvelope, error) {
	registry, err := newIdempotencyRegistry(spec, operations)
	if err != nil {
		return idempotencyEnvelope{}, err
	}
	matcher, err := gorillamux.NewRouter(spec)
	if err != nil {
		//nolint:wrapcheck // Preserve the router constructor's accepted registration diagnostic.
		return idempotencyEnvelope{}, err
	}
	return idempotencyEnvelope{matcher: matcher, registry: registry, terminalObserver: terminalObserver}, nil
}

func (e idempotencyEnvelope) enforce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, _, err := e.matcher.FindRoute(r)
		if err != nil || route == nil || route.Operation == nil {
			next.ServeHTTP(w, r)
			return
		}
		operation, opted := e.registry.operations[route.Operation.OperationID]
		if !opted {
			next.ServeHTTP(w, r)
			return
		}
		scope, authorized := operation.authorize(r.Context(), r)
		if !authorized || scope.Validate() != nil ||
			scope.OperationID != operation.contract.OperationID || scope.APIVersion != operation.contract.APIVersion {
			writeIdempotencyForbidden(w, r)
			return
		}
		key, err := httpidempotency.ParseKey(capturedIdempotencyKey(r.Context()), operation.contract.KeyMaxBytes)
		if err != nil {
			writeIdempotencyBadKey(w, r)
			return
		}
		decision := operation.admit(r.Context(), scope)
		if decision.Outcome != httpidempotency.OutcomeExecute && e.terminalObserver != nil {
			e.terminalObserver(r.Context(), decision, nil)
		}
		if writeIdempotencyDecision(w, r, operation.contract, decision) {
			return
		}
		attempt := httpidempotency.Attempt{Scope: scope, Key: key}
		next.ServeHTTP(w, r.WithContext(contextWithIdempotencyAttempt(r.Context(), attempt)))
	})
}

// prepareValidation masks every invalid captured key from the generated schema
// validator. The envelope later parses the captured value after current
// authorization, preserving the required authorization-before-key order.
func (e idempotencyEnvelope) prepareValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, _, err := e.matcher.FindRoute(r)
		if err != nil || route == nil || route.Operation == nil {
			next.ServeHTTP(w, r)
			return
		}
		operation, opted := e.registry.operations[route.Operation.OperationID]
		key := capturedIdempotencyKey(r.Context())
		if !opted {
			next.ServeHTTP(w, r)
			return
		}
		if _, err := httpidempotency.ParseKey(key, operation.contract.KeyMaxBytes); err == nil {
			next.ServeHTTP(w, r)
			return
		}
		request := r.Clone(r.Context())
		request.Header.Set(httpidempotency.Header, "")
		next.ServeHTTP(w, request)
	})
}
