package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/example/go-service-template-rest/internal/failure"
	// profile:http-idempotency-postgres:start
	"github.com/example/go-service-template-rest/internal/httpidempotency"
	// profile:http-idempotency-postgres:end
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

type RouterConfig struct {
	HardenConfig

	// Authenticate validates one security requirement declared by the OpenAPI
	// contract. Nil rejects every secured operation with 401, which is the
	// correct default for a contract that declares a scheme the service has not
	// implemented yet. Operations with no security requirement never reach it.
	Authenticate openapi3filter.AuthenticationFunc
	// AuthenticateChallenge is the WWW-Authenticate value sent with a 401. It
	// must name an HTTP authentication scheme, not a contract securityScheme
	// key. Defaults to Bearer.
	AuthenticateChallenge string
	// DomainErrors classify the errors a generated operation returns instead of a
	// typed response. The neutral type lets the same classification feed gRPC.
	DomainErrors []failure.Mapper
}

// NewRouter builds the service router for this repository's own OpenAPI contract:
// the generated strict server behind the request validator, wrapped in the
// hardened chain Harden owns.
func NewRouter(log *slog.Logger, h Handlers, metrics *telemetry.Metrics, cfg RouterConfig) (http.Handler, error) {
	if log == nil {
		return nil, errors.New("http router: logger is required")
	}
	strict, err := newStrictHandlers(h)
	if err != nil {
		return nil, err
	}

	rejectRequest := RejectRequest(log, cfg.AuthenticateChallenge)
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("http router: load embedded OpenAPI spec: %w", err)
	}
	// profile:http-idempotency-postgres:start
	idempotencyEnabled, err := validateIdempotentOperations(spec)
	if err != nil {
		return nil, err
	}
	// profile:http-idempotency-postgres:end
	validator, err := requestValidator(spec, cfg.Authenticate, rejectRequest)
	if err != nil {
		return nil, err
	}
	apiMiddlewares := []openapi.MiddlewareFunc{validator}
	// profile:http-idempotency-postgres:start
	// oapi-codegen wraps first to last, so capture runs before validation while
	// parsing remains in the authenticated handler through NewRequestFromContext.
	if idempotencyEnabled {
		apiMiddlewares = append(apiMiddlewares, httpidempotency.CaptureKey)
	}
	// profile:http-idempotency-postgres:end

	server := openapi.NewStrictHandlerWithOptions(strict, nil, generatedStrictServerOptions(log, rejectRequest, cfg.DomainErrors))
	// profile:inbound-webhooks-standard:start
	server = inboundRawServer{ServerInterface: server, receiver: h.InboundWebhook}
	// profile:inbound-webhooks-standard:end

	apiSubrouter := openapi.HandlerWithOptions(
		server,
		generatedChiServerOptions(rejectRequest, apiMiddlewares...),
	)
	return Harden(log, metrics, cfg.HardenConfig, apiSubrouter)
}

// requestValidator validates every request against the contract, as
// openapi3filter.ValidateRequest does: security, then parameters, then the body,
// stopping at the first failure. Each step runs once, against the operation
// kin-openapi's own router matched.
//
// A body step whose schema is structural is decided first by one streaming pass
// (see request_body_shape.go). A body that pass does not admit goes through
// openapi3filter.ValidateRequestBody, so every rejection is the validator's.
func requestValidator(
	spec *openapi3.T,
	authenticate openapi3filter.AuthenticationFunc,
	rejectRequest func(http.ResponseWriter, *http.Request, error),
) (openapi.MiddlewareFunc, error) {
	// The service is reached through whatever host its edge presents, so the
	// contract's servers take no part in matching a route.
	spec.Servers = nil
	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		return nil, fmt.Errorf("http router: build OpenAPI request router: %w", err)
	}
	bodies := structuralRequestBodies(spec)
	options := &openapi3filter.Options{AuthenticationFunc: authenticate}
	withoutBody := &openapi3filter.Options{AuthenticationFunc: authenticate, ExcludeRequestBody: true}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route, pathParams, err := router.FindRoute(r)
			if err != nil {
				rejectRequest(w, r, err)
				return
			}
			input := &openapi3filter.RequestValidationInput{
				Request:    r,
				PathParams: pathParams,
				Route:      route,
				Options:    options,
			}
			body, structural := bodies[route.Operation]
			if structural {
				input.Options = withoutBody
			}
			err = openapi3filter.ValidateRequest(r.Context(), input)
			if err == nil && structural && !body.admits(r) {
				input.Options = options
				err = openapi3filter.ValidateRequestBody(r.Context(), input, route.Operation.RequestBody.Value)
			}
			if err != nil {
				rejectRequest(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

func generatedStrictServerOptions(
	log *slog.Logger,
	rejectRequest func(http.ResponseWriter, *http.Request, error),
	domainErrors []failure.Mapper,
) openapi.StrictHTTPServerOptions {
	return openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  rejectRequest,
		ResponseErrorHandlerFunc: RejectResponse(log, domainErrors...),
	}
}

func generatedChiServerOptions(
	rejectRequest func(http.ResponseWriter, *http.Request, error),
	middlewares ...openapi.MiddlewareFunc,
) openapi.ChiServerOptions {
	return openapi.ChiServerOptions{
		Middlewares:      middlewares,
		ErrorHandlerFunc: rejectRequest,
	}
}
