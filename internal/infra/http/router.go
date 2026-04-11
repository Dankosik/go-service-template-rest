package httpx

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type RouterConfig struct {
	MaxBodyBytes     int64
	ReadinessTimeout time.Duration
}

func NewRouter(log *slog.Logger, h Handlers, metrics *telemetry.Metrics, cfg RouterConfig) (http.Handler, error) {
	if log == nil {
		return nil, fmt.Errorf("http router: logger is required")
	}

	strict, err := newStrictHandlers(h, metrics, cfg.ReadinessTimeout)
	if err != nil {
		return nil, err
	}

	server := api.NewStrictHandlerWithOptions(strict, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			logStrictRequestError(log, r, err)
			writeMalformedRequestProblem(w, r)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			writeProblem(w, r, http.StatusInternalServerError, "internal server error", "request failed")
		},
	})

	otelMiddleware := otelhttp.NewMiddleware(
		"http.server",
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			if route := routeLabelForRequest(r); route != "" {
				return route
			}
			if r != nil && r.Method != "" {
				return r.Method + " <unmatched>"
			}
			return operation
		}),
	)

	apiSubrouter := api.HandlerWithOptions(server, generatedChiServerOptions(log, captureRouteLabelMiddleware, otelMiddleware))

	// Serve /metrics directly on the root router to avoid full payload buffering in strict handler path.
	metricsHandler := otelMiddleware(captureRouteLabelMiddleware(strict.metrics.Handler()))
	rootRouter := newRootRouter(apiSubrouter, metricsHandler)

	maxBodyBytes := cfg.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1 << 20
	}

	var handler http.Handler = rootRouter
	handler = Recover(log, handler)
	handler = RequestFramingGuard(handler)
	handler = RequestBodyLimit(maxBodyBytes, handler)
	handler = AccessLog(log, strict.metrics, handler)
	handler = SecurityHeaders(handler)
	handler = RequestCorrelation(handler)

	return handler, nil
}

func generatedChiServerOptions(log *slog.Logger, middlewares ...api.MiddlewareFunc) api.ChiServerOptions {
	return api.ChiServerOptions{
		Middlewares: middlewares,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			logStrictRequestError(log, r, err)
			writeMalformedRequestProblem(w, r)
		},
	}
}

type manualRootRouteKey struct {
	method string
	path   string
}

type manualRootRoute struct {
	key     manualRootRouteKey
	handler http.Handler
}

const metricsRootRouteReason = "operational metrics is streamed from the root router while remaining visible in the OpenAPI contract"

var documentedManualRootRouteExceptions = map[manualRootRouteKey]string{
	{method: http.MethodGet, path: "/metrics"}: metricsRootRouteReason,
}

func newRootRouter(apiSubrouter http.Handler, metricsHandler http.Handler) chi.Router {
	root := chi.NewRouter()
	for _, route := range manualRootRoutes(metricsHandler) {
		root.Method(route.key.method, route.key.path, route.handler)
	}
	root.Mount("/", apiSubrouter)
	applyHTTPPolicy(root)
	return root
}

func manualRootRoutes(metricsHandler http.Handler) []manualRootRoute {
	return []manualRootRoute{
		{
			key: manualRootRouteKey{
				method: http.MethodGet,
				path:   "/metrics",
			},
			handler: metricsHandler,
		},
	}
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
		writeProblem(w, r, http.StatusNotFound, "not found", "resource not found")
	})

	root.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		allowMethods := allowedMethodsForPath(root, r.URL.Path)
		if len(allowMethods) > 0 {
			allowMethods = ensureMethodAllowed(allowMethods, http.MethodOptions)
		}

		if r.Method == http.MethodOptions {
			if len(allowMethods) == 0 {
				writeProblem(w, r, http.StatusNotFound, "not found", "resource not found")
				return
			}
			setAllowHeader(w, allowMethods)

			if isCORSPreflightRequest(r) {
				writeProblem(w, r, http.StatusMethodNotAllowed, "method not allowed", "cors preflight is not enabled")
				return
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		setAllowHeader(w, allowMethods)
		writeProblem(w, r, http.StatusMethodNotAllowed, "method not allowed", "method is not allowed for this resource")
	})
}

func allowedMethodsForPath(root chi.Router, path string) []string {
	if path == "" {
		path = "/"
	}

	candidates := []string{
		http.MethodConnect,
		http.MethodGet,
		http.MethodHead,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}

	allowMethods := make([]string, 0, len(candidates))
	for _, method := range candidates {
		routeContext := chi.NewRouteContext()
		if root.Match(routeContext, method, path) {
			allowMethods = append(allowMethods, method)
		}
	}
	return allowMethods
}

func setAllowHeader(w http.ResponseWriter, methods []string) {
	w.Header().Del("Allow")
	for _, method := range methods {
		w.Header().Add("Allow", method)
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
