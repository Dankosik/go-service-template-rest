package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/openapi"
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

	server := openapi.NewStrictHandlerWithOptions(strict, nil, generatedStrictServerOptions(rejectRequest))
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
//	→ RequestTimeout → MaxInFlight → Recover → apiSubrouter
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
		func(next http.Handler) http.Handler { return MaxInFlight(cfg.MaxInFlight, next) },
		func(next http.Handler) http.Handler { return Recover(log, next) },
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

func generatedStrictServerOptions(rejectRequest func(http.ResponseWriter, *http.Request, error)) openapi.StrictHTTPServerOptions {
	return openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: rejectRequest,
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			// A handler that returns its expired context is reporting a spent
			// request budget, not an internal fault, and this is the path most
			// timeouts actually take: the generated wrapper commits a response
			// here, so RequestTimeout never sees an uncommitted one. Reporting
			// it as 500 would hide every slow dependency inside the error rate.
			if errors.Is(err, context.DeadlineExceeded) {
				writeProblem(w, r, problemResponse{
					code:   problemCodeGatewayTimeout,
					detail: "request exceeded its time budget",
				})
				return
			}
			writeProblem(w, r, problemResponse{code: problemCodeInternalError, detail: "request failed"})
		},
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
		writeProblem(w, r, problemResponse{code: problemCodeRequestEntityTooLarge, detail: "request body exceeds limit"})
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
			writeProblem(w, r, problemResponse{code: problemCodeUnauthorized, detail: "credentials are missing or invalid"})
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

	attrs := []any{slog.String("error_class", strictRequestErrorClass(err))}
	if r != nil {
		if requestID := requestIDFromContext(r.Context()); requestID != "" {
			attrs = append(attrs, slog.String("request_id", requestID))
		}
		traceID, spanID := traceIDsFromContext(r.Context())
		if traceID != "" {
			attrs = append(attrs, slog.String("trace_id", traceID), slog.String("span_id", spanID))
		}
	}
	log.Warn("rejected malformed HTTP request", attrs...)
}

func strictRequestErrorClass(err error) string {
	if err == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", err)
}

func applyHTTPPolicy(root chi.Router) {
	root.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, r, problemResponse{code: problemCodeNotFound, detail: "resource not found"})
	})

	root.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		allowMethods := allowedMethodsForPath(root, r.URL.Path)
		if len(allowMethods) == 0 {
			writeProblem(w, r, problemResponse{code: problemCodeNotFound, detail: "resource not found"})
			return
		}
		allowMethods = ensureMethodAllowed(allowMethods, http.MethodOptions)

		if r.Method == http.MethodOptions {
			setAllowHeader(w, allowMethods)

			if isCORSPreflightRequest(r) {
				writeProblem(w, r, problemResponse{code: problemCodeMethodNotAllowed, detail: "cors preflight is not enabled"})
				return
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		setAllowHeader(w, allowMethods)
		writeProblem(w, r, problemResponse{code: problemCodeMethodNotAllowed, detail: "method is not allowed for this resource"})
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
// strict-server response failures: a spent request budget becomes 504, anything
// else becomes 500.
//
// A service wiring its own generated strict server needs this, or every slow
// dependency hides inside its 5xx error rate.
func RejectResponse() func(http.ResponseWriter, *http.Request, error) {
	return generatedStrictServerOptions(nil).ResponseErrorHandlerFunc
}

// ProblemTypeURI maps an HTTP status to the stable problem type this repository
// publishes for it, so a service filling its own generated Problem values cannot
// drift from the envelope the fallback paths emit.
func ProblemTypeURI(status int) string {
	for _, code := range []problemCode{
		problemCodeBadRequest,
		problemCodeUnauthorized,
		problemCodeForbidden,
		problemCodeNotFound,
		problemCodeMethodNotAllowed,
		problemCodeRequestEntityTooLarge,
		problemCodeInternalError,
		problemCodeServiceUnavailable,
		problemCodeGatewayTimeout,
	} {
		if _, definition := problemDefinitionFor(code); definition.status == status {
			return definition.typeURI
		}
	}
	_, internal := problemDefinitionFor(problemCodeInternalError)
	return internal.typeURI
}
