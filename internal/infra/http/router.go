package httpx

import (
	"context"
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
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
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
	apiMiddlewares := []openapi.MiddlewareFunc{requestValidator(spec, cfg.Authenticate, rejectRequest)}
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

func requestValidator(
	spec *openapi3.T,
	authenticate openapi3filter.AuthenticationFunc,
	rejectRequest func(http.ResponseWriter, *http.Request, error),
) openapi.MiddlewareFunc {
	return oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		DoNotValidateServers: true,
		Options: openapi3filter.Options{
			AuthenticationFunc: authenticate,
		},
		ErrorHandlerWithOpts: func(
			_ context.Context,
			err error,
			w http.ResponseWriter,
			r *http.Request,
			_ oapimiddleware.ErrorHandlerOpts,
		) {
			rejectRequest(w, r, err)
		},
	})
}

func generatedStrictServerOptions(
	log *slog.Logger,
	rejectRequest func(http.ResponseWriter, *http.Request, error),
	domainErrors []failure.Mapper,
) openapi.StrictHTTPServerOptions {
	return openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  rejectRequest,
		ResponseErrorHandlerFunc: handleGeneratedResponseError(log, domainErrors),
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
