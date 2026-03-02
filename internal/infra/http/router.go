package httpx

import (
	"log/slog"
	"net/http"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
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
			if r != nil && r.Pattern != "" {
				return r.Pattern
			}
			if r != nil && r.Method != "" {
				return r.Method + " <unmatched>"
			}
			return operation
		}),
	)

	apiHandler := api.HandlerWithOptions(server, api.StdHTTPServerOptions{
		Middlewares: []api.MiddlewareFunc{otelMiddleware},
	})

	// Serve /metrics directly to avoid buffering the full metrics payload in memory.
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", otelMiddleware(strict.metrics.Handler()))
	mux.Handle("/", apiHandler)

	maxBodyBytes := cfg.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1 << 20
	}

	var handler http.Handler = mux
	handler = Recover(log, handler)
	handler = RequestFramingGuard(handler)
	handler = RequestBodyLimit(maxBodyBytes, handler)
	handler = AccessLog(log, strict.metrics, handler)
	handler = SecurityHeaders(handler)
	handler = RequestCorrelation(handler)

	return handler
}
