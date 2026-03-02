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

	maxBodyBytes := cfg.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1 << 20
	}

	handler := api.Handler(server)
	handler = Recover(log, handler)
	handler = RequestFramingGuard(handler)
	handler = RequestBodyLimit(maxBodyBytes, handler)
	handler = AccessLog(log, strict.metrics, handler)
	handler = SecurityHeaders(handler)
	handler = RequestCorrelation(handler)
	handler = otelhttp.NewHandler(handler, "http.server")

	return handler
}
