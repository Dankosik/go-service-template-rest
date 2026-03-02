package httpx

import (
	"log/slog"
	"net/http"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func NewRouter(log *slog.Logger, h Handlers, metrics *telemetry.Metrics) http.Handler {
	strict := newStrictHandlers(h, metrics)
	server := api.NewStrictHandler(strict, nil)
	var handler http.Handler = api.Handler(server)
	handler = Recover(log, handler)
	handler = AccessLog(log, strict.metrics, handler)
	handler = RequestCorrelation(handler)

	return handler
}
