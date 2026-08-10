package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/example/go-service-template-rest/internal/failure"
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

	server := openapi.NewStrictHandlerWithOptions(strict, nil, generatedStrictServerOptions(log, rejectRequest, cfg.DomainErrors))
	requestValidator, err := openAPIRequestValidator(cfg.Authenticate, rejectRequest)
	if err != nil {
		return nil, err
	}

	apiSubrouter := openapi.HandlerWithOptions(
		server,
		generatedChiServerOptions(rejectRequest, requestValidator),
	)
	return Harden(log, metrics, cfg.HardenConfig, apiSubrouter)
}

// openAPIRequestValidator builds the contract validator and installs the
// authentication seam.
//
// authenticate must be forwarded even when nil: openapi3filter then returns
// ErrAuthenticationServiceMissing, which fails closed but as an unmapped error,
// and handleAuthenticatedRequestError is what turns that into a 401 rather than
// a 400 no client will retry with credentials.
func openAPIRequestValidator(
	authenticate openapi3filter.AuthenticationFunc,
	rejectRequest func(http.ResponseWriter, *http.Request, error),
) (openapi.MiddlewareFunc, error) {
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("http router: load embedded OpenAPI spec: %w", err)
	}
	return requestValidator(spec, authenticate, rejectRequest), nil
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
