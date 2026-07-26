package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
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
	// Idempotency makes a repeated POST or PATCH carrying an Idempotency-Key
	// answer with the first attempt's result instead of doing the work twice.
	// Nil leaves the middleware out of the chain entirely; see IdempotencyStore
	// for what a store has to guarantee.
	Idempotency IdempotencyStore
	// IdempotencyOutcomeTimeout bounds recording or releasing a reservation after
	// the handler has answered. It is required when Idempotency is set: that call
	// is detached from the request budget on purpose, so without a bound of its own
	// a stalled store holds the handler goroutine and its in-flight slot forever.
	// See idempotencyOutcomeContext.
	IdempotencyOutcomeTimeout time.Duration
	// DomainErrors classify the errors a generated operation returns instead of a
	// typed response. See problem.Mapper for why the seam exists and why the type
	// lives in a leaf package rather than here.
	DomainErrors []problem.Mapper
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

// defaultAuthenticateChallenge is the HTTP authentication scheme advertised when
// a service declares security requirements without naming its own challenge.
const defaultAuthenticateChallenge = "Bearer"

// NewRouter builds the service router for this repository's own OpenAPI contract:
// the generated strict server behind the request validator, wrapped in the
// hardened chain Harden owns.
func NewRouter(log *slog.Logger, h Handlers, metrics *telemetry.Metrics, cfg RouterConfig) (http.Handler, error) {
	if log == nil {
		return nil, fmt.Errorf("http router: logger is required")
	}
	strict, err := newStrictHandlers(h)
	if err != nil {
		return nil, err
	}

	rejectRequest := RejectRequest(log, cfg.AuthenticateChallenge)

	server := openapi.NewStrictHandlerWithOptions(strict, nil, generatedStrictServerOptions(rejectRequest, cfg.DomainErrors))
	requestValidator, err := openAPIRequestValidator(cfg.Authenticate, rejectRequest)
	if err != nil {
		return nil, err
	}

	apiSubrouter := openapi.HandlerWithOptions(server, generatedChiServerOptions(rejectRequest, requestValidator))

	return Harden(log, metrics, cfg, apiSubrouter)
}

// Harden wraps an API handler in this repository's middleware chain and its
// 404/405/Allow fallback policy, and is what NewRouter is built on.
//
// It is exported because the chain is the valuable part and it is entirely
// independent of any generated type: a service whose OpenAPI contract lives in
// its own package — the reference example is one — can inherit correlation,
// tracing, access logging, body limits, the request budget, load shedding, and
// panic recovery without reimplementing them. Building a second router by hand is
// how a service ends up with none of them while believing it inherited all of
// them.
//
// The order is the contract, outermost first:
//
//	RequestCorrelation → OTel → SecurityHeaders → AccessLog → RequestBodyLimit
//	→ RequestTimeout → MaxInFlight → Recover → Idempotent → apiSubrouter
func Harden(log *slog.Logger, metrics *telemetry.Metrics, cfg RouterConfig, apiSubrouter http.Handler) (http.Handler, error) {
	if log == nil {
		return nil, fmt.Errorf("http router: logger is required")
	}
	if metrics == nil {
		return nil, fmt.Errorf("http router: metrics is required")
	}
	if apiSubrouter == nil {
		return nil, fmt.Errorf("http router: api subrouter is required")
	}
	if cfg.MaxBodyBytes <= 0 {
		return nil, fmt.Errorf("http router: max body bytes must be > 0")
	}
	if cfg.RequestTimeout <= 0 {
		return nil, fmt.Errorf("http router: request timeout must be > 0")
	}
	if cfg.MaxInFlight < 0 {
		return nil, fmt.Errorf("http router: max in flight must be >= 0")
	}
	// Rejected at construction rather than defaulted, because the default that
	// would look harmless here is the unbounded one this replaced.
	if cfg.Idempotency != nil && cfg.IdempotencyOutcomeTimeout <= 0 {
		return nil, fmt.Errorf("http router: idempotency outcome timeout must be > 0 when an idempotency store is configured")
	}
	// Same reason: a limiter with no key silently limits nothing, which looks
	// exactly like a limiter that is working.
	if cfg.RateLimit != nil && cfg.RateLimitKey == nil {
		return nil, fmt.Errorf("http router: rate limit key is required when a rate limiter is configured")
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
		func(next http.Handler) http.Handler { return Recover(log, next) },
		// Innermost, so it wraps the handler and nothing else: it must see the
		// response the operation actually produced, and a panic that Recover
		// turns into a 500 must unwind through it so the reservation is released
		// rather than recorded as a completed attempt.
		func(next http.Handler) http.Handler {
			return Idempotent(cfg.Idempotency, cfg.IdempotencyOutcomeTimeout, log, next)
		},
	)

	return RequestCorrelation(rootRouter), nil
}

// RejectRequest returns the validator error mapper this repository installs:
// oversized bodies become 413, failed security requirements become 401 with a
// WWW-Authenticate challenge, and everything else becomes a sanitized 400.
//
// A service wiring its own generated validator needs this, or it reproduces the
// defect where a missing credential is reported as a malformed request.
func RejectRequest(log *slog.Logger, challenge string) func(http.ResponseWriter, *http.Request, error) {
	if strings.TrimSpace(challenge) == "" {
		challenge = defaultAuthenticateChallenge
	}
	return handleGeneratedRequestError(log, challenge)
}

// openAPIRequestValidator builds the contract validator and installs the
// authentication seam.
//
// authenticate must be forwarded even when nil. openapi3filter returns
// ErrAuthenticationServiceMissing when no AuthenticationFunc is set, so a
// contract that declares a security requirement fails closed either way — but it
// fails as an unmapped error, and handleAuthenticatedRequestError is what turns
// that into a 401 instead of a 400 that no client will retry with credentials.
func openAPIRequestValidator(
	authenticate openapi3filter.AuthenticationFunc,
	rejectRequest func(http.ResponseWriter, *http.Request, error),
) (openapi.MiddlewareFunc, error) {
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("http router: load embedded OpenAPI spec: %w", err)
	}

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
	}), nil
}

func otelServerName(configured string) string {
	serverName := strings.TrimSpace(configured)
	if serverName == "" {
		serverName = "service"
	}
	// Port zero keeps server.address bounded without inventing a listening port.
	return net.JoinHostPort(serverName, "0")
}

func generatedStrictServerOptions(
	rejectRequest func(http.ResponseWriter, *http.Request, error),
	domainErrors []problem.Mapper,
) openapi.StrictHTTPServerOptions {
	return openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  rejectRequest,
		ResponseErrorHandlerFunc: handleGeneratedResponseError(domainErrors),
	}
}

// handleGeneratedResponseError turns an error a generated operation returned into
// a problem response.
//
// The expired-context case is checked before any service mapper, because it is a
// transport fact rather than a domain one: a mapper that classified it would have
// to repeat this rule, and one that forgot would hide every slow dependency
// inside the 5xx rate.
func handleGeneratedResponseError(domainErrors []problem.Mapper) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		// A handler that returns its expired context is reporting a spent
		// request budget, not an internal fault, and this is the path most
		// timeouts actually take: the generated wrapper commits a response
		// here, so RequestTimeout never sees an uncommitted one. Reporting
		// it as 500 would hide every slow dependency inside the error rate.
		if errors.Is(err, context.DeadlineExceeded) {
			writeProblem(w, r, problemResponse{
				code:   problem.CodeGatewayTimeout,
				detail: "request exceeded its time budget",
			})
			return
		}

		if mapped, ok := problem.Classify(err, domainErrors); ok {
			if mapped.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(int(mapped.RetryAfter.Round(time.Second).Seconds())))
			}
			writeProblem(w, r, problemResponse{code: mapped.Code, detail: mapped.Detail})
			return
		}

		// An unclassified error is a fault this service did not anticipate, so it
		// stays a 500 with no detail: guessing a friendlier status for it is how a
		// bug starts reading as a client mistake.
		writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "request failed"})
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

func handleMalformedGeneratedRequest(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	logStrictRequestError(log, r, err)
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		writeProblem(w, r, problemResponse{code: problem.CodeRequestEntityTooLarge, detail: "request body exceeds limit"})
		return
	}
	writeMalformedRequestProblem(w, r)
}

// handleGeneratedRequestError maps a validator rejection, adding the one case
// handleMalformedGeneratedRequest cannot classify on its own.
//
// A failed security requirement is 401, not 400: the request framing was fine
// and the credential was the problem. Reporting it as 400 tells a client to stop
// rather than to authenticate, and no client library retries with credentials on
// a 400.
func handleGeneratedRequestError(log *slog.Logger, challenge string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		var securityErr *openapi3filter.SecurityRequirementsError
		if errors.As(err, &securityErr) {
			logStrictRequestError(log, r, err)
			// The challenge names an HTTP authentication scheme, which is not the
			// same vocabulary as the contract's securityScheme names: a contract
			// key like "bearerAuth" is not a legal challenge under RFC 9110. Only
			// the service knows which scheme it implements, so it supplies this.
			w.Header().Set("WWW-Authenticate", challenge)
			writeProblem(w, r, problemResponse{code: problem.CodeUnauthorized, detail: "credentials are missing or invalid"})
			return
		}
		handleMalformedGeneratedRequest(log, w, r, err)
	}
}

func newRootRouter(
	apiSubrouter http.Handler,
	middlewares ...func(http.Handler) http.Handler,
) chi.Router {
	root := chi.NewRouter()
	root.Use(middlewares...)
	root.Mount("/", apiSubrouter)
	applyHTTPPolicy(root)
	return root
}

func logStrictRequestError(log *slog.Logger, r *http.Request, err error) {
	if log == nil {
		return
	}

	// Correlation comes from the context rather than from attributes assembled
	// here; see internal/observability/logctx. A nil request still has to be
	// handled: the generated error handlers are declared with one, and this must
	// not panic on the path whose whole job is reporting a malformed request.
	//nolint:contextcheck // There is no parent context when the request is nil, which is the only case this branch exists for.
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	log.WarnContext(ctx, "rejected malformed HTTP request", slog.String("error_class", strictRequestErrorClass(err)))
}

func strictRequestErrorClass(err error) string {
	if err == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", err)
}

func applyHTTPPolicy(root chi.Router) {
	root.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, r, problemResponse{code: problem.CodeNotFound, detail: "resource not found"})
	})

	root.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		allowMethods := allowedMethodsForPath(root, r.URL.Path)
		if len(allowMethods) == 0 {
			writeProblem(w, r, problemResponse{code: problem.CodeNotFound, detail: "resource not found"})
			return
		}
		allowMethods = ensureMethodAllowed(allowMethods, http.MethodOptions)

		if r.Method == http.MethodOptions {
			setAllowHeader(w, allowMethods)

			if isCORSPreflightRequest(r) {
				writeProblem(w, r, problemResponse{code: problem.CodeMethodNotAllowed, detail: "cors preflight is not enabled"})
				return
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		setAllowHeader(w, allowMethods)
		writeProblem(w, r, problemResponse{code: problem.CodeMethodNotAllowed, detail: "method is not allowed for this resource"})
	})
}

func allowedMethodsForPath(root chi.Router, path string) []string {
	if path == "" {
		path = "/"
	}

	allowMethods := make([]string, 0, len(boundedHTTPMethods))
	for _, method := range boundedHTTPMethods {
		routeContext := chi.NewRouteContext()
		if root.Match(routeContext, method, path) {
			allowMethods = append(allowMethods, method)
		}
	}
	return allowMethods
}

func setAllowHeader(w http.ResponseWriter, methods []string) {
	w.Header().Del("Allow")
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
}

func ensureMethodAllowed(methods []string, method string) []string {
	if slices.Contains(methods, method) {
		return methods
	}
	return append(methods, method)
}

func isCORSPreflightRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	return r.Header.Get("Origin") != "" && r.Header.Get("Access-Control-Request-Method") != ""
}

// RejectResponse returns the mapper this repository installs for generated
// strict-server response failures: a spent request budget becomes 504, whatever
// the supplied mappers classify becomes their problem, and anything left becomes
// 500.
//
// A service wiring its own generated strict server needs this, or every slow
// dependency hides inside its 5xx error rate. Passing mappers is what lets its
// handlers return a domain error instead of hand-building a typed problem
// response per operation; see problem.Mapper.
func RejectResponse(domainErrors ...problem.Mapper) func(http.ResponseWriter, *http.Request, error) {
	return handleGeneratedResponseError(domainErrors)
}

// The status-to-problem lookup a service needs for its own generated Problem
// values is deliberately not here. It lives in internal/problem, because
// depguard forbids a feature package from importing this adapter — so a helper
// exported from here was reachable only by another adapter, and every service's
// httpapi package hand-copied the table instead. The copy in the reference
// example drifted exactly once, and answered a 409 with the internal-error type.
