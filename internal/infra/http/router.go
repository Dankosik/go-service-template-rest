package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

type RouterConfig struct {
	MaxBodyBytes int64
	// RequestTimeout bounds every handler. See RequestTimeout in
	// middleware_timeout.go for why the net/http server timeouts do not.
	RequestTimeout time.Duration
	// MaxInFlight bounds concurrent handler execution. Zero disables shedding.
	// See MaxInFlight in middleware_inflight.go for why a time budget alone is
	// not backpressure.
	MaxInFlight    int
	OTelServerName string
	// LogHealthProbes re-enables access logging for platform probe routes,
	// which are excluded by default.
	LogHealthProbes bool
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
	// RateLimit rejects a caller that is over its budget with 429. Nil leaves the
	// middleware out of the chain, which is the shipped default; see RateLimiter
	// for why the limit and the identity it charges are the service's decision,
	// and NewKeyedRateLimiter for the one implementation this repository ships.
	RateLimit RateLimiter
	// RateLimitKey reports which bucket a request is charged against, and is
	// required when RateLimit is set. There is no default: the identity worth
	// limiting is the whole decision, and the two candidates a template could
	// guess — the client address and a forwarded-for header — are respectively
	// useless and spoofable without knowing the edge. See HeaderRateLimitKey.
	RateLimitKey RateLimitKeyFunc
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
	return Harden(log, metrics, cfg, apiSubrouter)
}

// Harden wraps an API handler in this repository's middleware chain and its
// 404/405/Allow fallback policy, and is what NewRouter is built on.
//
// It is exported and independent of any generated type, so a service whose
// OpenAPI contract lives in its own package inherits the chain instead of
// rebuilding it by hand and silently getting none of it.
//
// The order is the contract, outermost first:
//
//	RequestCorrelation → OTel → SecurityHeaders → AccessLog → RequestBodyLimit
//	→ RequestTimeout → MaxInFlight → RateLimit → Recover → apiSubrouter
func Harden(log *slog.Logger, metrics *telemetry.Metrics, cfg RouterConfig, apiSubrouter http.Handler) (http.Handler, error) {
	if log == nil {
		return nil, errors.New("http router: logger is required")
	}
	if metrics == nil {
		return nil, errors.New("http router: metrics is required")
	}
	if apiSubrouter == nil {
		return nil, errors.New("http router: api subrouter is required")
	}
	if cfg.MaxBodyBytes <= 0 {
		return nil, errors.New("http router: max body bytes must be > 0")
	}
	if cfg.RequestTimeout <= 0 {
		return nil, errors.New("http router: request timeout must be > 0")
	}
	if cfg.MaxInFlight < 0 {
		return nil, errors.New("http router: max in flight must be >= 0")
	}
	// Same reason: a limiter with no key silently limits nothing, which looks
	// exactly like a limiter that is working.
	if cfg.RateLimit != nil && cfg.RateLimitKey == nil {
		return nil, errors.New("http router: rate limit key is required when a rate limiter is configured")
	}

	// Built once here rather than per request. The meter provider is already
	// installed by this point in startup, so these instruments reach both the
	// Prometheus registry and the OTLP reader.
	serverLoad := metrics.ServerLoad()

	otelMiddleware := otelhttp.NewMiddleware(
		"http.server",
		otelhttp.WithMeterProvider(metrics.MeterProvider()),
		otelhttp.WithPropagators(propagation.TraceContext{}),
		otelhttp.WithServerName(otelServerName(cfg.OTelServerName)),
	)

	rootRouter := newRootRouter(
		apiSubrouter,
		otelMiddleware,
		SecurityHeaders,
		func(next http.Handler) http.Handler { return AccessLog(log, cfg.LogHealthProbes, next) },
		func(next http.Handler) http.Handler { return RequestBodyLimit(cfg.MaxBodyBytes, next) },
		// The budget sits inside the access log, so a timed-out request is still
		// recorded with its duration, and outside Recover, so a recovered panic
		// has already committed its response before the budget is inspected.
		func(next http.Handler) http.Handler { return RequestTimeout(cfg.RequestTimeout, next) },
		// Shedding sits inside the budget so a shed request is still traced and
		// access-logged, and outside Recover because it never runs a handler.
		func(next http.Handler) http.Handler { return MaxInFlight(cfg.MaxInFlight, serverLoad, next) },
		// Rate limiting sits inside shedding, so a service past capacity as a whole
		// answers 503 before it starts attributing the overload to one caller, and
		// well outside the generated validator, so an over-budget caller does not
		// get their body schema-validated before being told no.
		func(next http.Handler) http.Handler { return RateLimit(cfg.RateLimit, cfg.RateLimitKey, next) },
		// Innermost here, and still outside the generated router, so a panic
		// raised inside an operation becomes a sanitized 500.
		func(next http.Handler) http.Handler { return Recover(log, next) },
	)

	return RequestCorrelation(rootRouter), nil
}

func newRootRouter(
	apiSubrouter http.Handler,
	middlewares ...func(http.Handler) http.Handler,
) *chi.Mux {
	root := chi.NewRouter()
	root.Use(middlewares...)
	root.Mount("/", apiSubrouter)
	applyHTTPPolicy(root)
	return root
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

func otelServerName(configured string) string {
	serverName := strings.TrimSpace(configured)
	if serverName == "" {
		serverName = "service"
	}
	// Port zero keeps server.address bounded without inventing a listening port.
	return net.JoinHostPort(serverName, "0")
}
