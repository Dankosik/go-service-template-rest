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
	"github.com/go-chi/chi/v5"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

type RouterConfig struct {
	MaxBodyBytes     int64
	ReadinessTimeout time.Duration
	OTelServerName   string
	// LogHealthProbes re-enables access logging for platform probe routes,
	// which are excluded by default.
	LogHealthProbes bool
}

func NewRouter(log *slog.Logger, h Handlers, metrics *telemetry.Metrics, cfg RouterConfig) (http.Handler, error) {
	if log == nil {
		return nil, fmt.Errorf("http router: logger is required")
	}

	if metrics == nil {
		return nil, fmt.Errorf("http router: metrics is required")
	}
	if cfg.MaxBodyBytes <= 0 {
		return nil, fmt.Errorf("http router: max body bytes must be > 0")
	}
	strict, err := newStrictHandlers(h, cfg.ReadinessTimeout)
	if err != nil {
		return nil, err
	}

	server := openapi.NewStrictHandlerWithOptions(strict, nil, generatedStrictServerOptions(log))
	requestValidator, err := openAPIRequestValidator(log)
	if err != nil {
		return nil, err
	}

	otelMiddleware := otelhttp.NewMiddleware(
		"http.server",
		otelhttp.WithMeterProvider(metrics.MeterProvider()),
		otelhttp.WithPropagators(propagation.TraceContext{}),
		otelhttp.WithServerName(otelServerName(cfg.OTelServerName)),
	)

	apiSubrouter := openapi.HandlerWithOptions(server, generatedChiServerOptions(log, requestValidator))

	rootRouter := newRootRouter(
		apiSubrouter,
		otelMiddleware,
		SecurityHeaders,
		func(next http.Handler) http.Handler { return AccessLog(log, cfg.LogHealthProbes, next) },
		func(next http.Handler) http.Handler { return RequestBodyLimit(cfg.MaxBodyBytes, next) },
		func(next http.Handler) http.Handler { return Recover(log, next) },
	)

	return RequestCorrelation(rootRouter), nil
}

func openAPIRequestValidator(log *slog.Logger) (openapi.MiddlewareFunc, error) {
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("http router: load embedded OpenAPI spec: %w", err)
	}

	return oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		DoNotValidateServers: true,
		ErrorHandlerWithOpts: func(
			_ context.Context,
			err error,
			w http.ResponseWriter,
			r *http.Request,
			_ oapimiddleware.ErrorHandlerOpts,
		) {
			handleMalformedGeneratedRequest(log, w, r, err)
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

func generatedStrictServerOptions(log *slog.Logger) openapi.StrictHTTPServerOptions {
	return openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			handleMalformedGeneratedRequest(log, w, r, err)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			writeProblem(w, r, problemResponse{code: problemCodeInternalError, detail: "request failed"})
		},
	}
}

func generatedChiServerOptions(log *slog.Logger, middlewares ...openapi.MiddlewareFunc) openapi.ChiServerOptions {
	return openapi.ChiServerOptions{
		Middlewares: middlewares,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			handleMalformedGeneratedRequest(log, w, r, err)
		},
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
