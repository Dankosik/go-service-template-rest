package httpx

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/Dankosik/search-service/internal/api"
	"github.com/Dankosik/search-service/internal/infra/telemetry"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type RouterConfig struct {
	MaxBodyBytes int64
}

func NewRouter(log *slog.Logger, h Handlers, metrics *telemetry.Metrics, cfg RouterConfig) http.Handler {
	strict := newStrictHandlers(h, metrics)
	server := api.NewStrictHandlerWithOptions(strict, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeProblem(w, r, http.StatusBadRequest, "bad request", err.Error())
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

	apiSubrouter := api.HandlerWithOptions(server, api.ChiServerOptions{
		Middlewares: []api.MiddlewareFunc{
			captureRouteLabelMiddleware,
			otelMiddleware,
		},
	})

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

	return handler
}

func newRootRouter(apiSubrouter http.Handler, metricsHandler http.Handler) chi.Router {
	root := chi.NewRouter()
	root.Method(http.MethodGet, "/metrics", metricsHandler)
	root.Mount("/", apiSubrouter)
	applyHTTPPolicy(root)
	return root
}

func applyHTTPPolicy(root chi.Router) {
	root.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, r, http.StatusNotFound, "not found", "resource not found")
	})

	root.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		allowMethods := allowedMethodsForPath(root, r.URL.Path)

		if r.Method == http.MethodOptions {
			if len(allowMethods) == 0 {
				writeProblem(w, r, http.StatusNotFound, "not found", "resource not found")
				return
			}
			allowMethods = ensureMethodAllowed(allowMethods, http.MethodOptions)
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
