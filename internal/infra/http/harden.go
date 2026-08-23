package httpx

import (
	"cmp"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

// HardenConfig configures the reusable HTTP middleware chain.
type HardenConfig struct {
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
func Harden(log *slog.Logger, metrics *telemetry.Metrics, cfg HardenConfig, apiSubrouter http.Handler) (http.Handler, error) {
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
	serverLoad := newServerLoad(metrics.MeterProvider())

	otelOptions := []otelhttp.Option{
		otelhttp.WithMeterProvider(metrics.MeterProvider()),
		otelhttp.WithPropagators(propagation.TraceContext{}),
		otelhttp.WithServerName(otelServerName(cfg.OTelServerName)),
	}
	// profile:inbound-webhooks-standard:start
	// otelhttp records url.path before routing. The endpoint id is caller supplied
	// and belongs in neither trace attributes nor span names, so this public route
	// stays correlated through its access log rather than exporting the raw path.
	otelOptions = append(otelOptions, otelhttp.WithFilter(traceInboundWebhookRequest))
	// profile:inbound-webhooks-standard:end
	otelMiddleware := otelhttp.NewMiddleware("http.server", otelOptions...)

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

// profile:inbound-webhooks-standard:start
func traceInboundWebhookRequest(request *http.Request) bool {
	return request == nil || request.URL == nil || !strings.HasPrefix(request.URL.Path, "/webhooks/")
}

// profile:inbound-webhooks-standard:end

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

func otelServerName(configured string) string {
	serverName := cmp.Or(strings.TrimSpace(configured), "service")
	// Port zero keeps server.address bounded without inventing a listening port.
	return net.JoinHostPort(serverName, "0")
}
