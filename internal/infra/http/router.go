package httpx

import (
	"log/slog"
	"net/http"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func NewRouter(log *slog.Logger, h Handlers, metrics *telemetry.Metrics) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", h.Live)
	mux.HandleFunc("GET /health/ready", h.Ready)
	mux.HandleFunc("GET /api/v1/ping", h.Pong)

	if metrics != nil {
		mux.Handle("GET /metrics", metrics.Handler())
	}

	var handler http.Handler = mux
	handler = Recover(log, handler)
	handler = AccessLog(log, metrics, handler)

	return handler
}
