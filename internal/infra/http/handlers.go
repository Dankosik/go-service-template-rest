package httpx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/Dankosik/privacy-sanitization-service/internal/api"
	"github.com/Dankosik/privacy-sanitization-service/internal/app/health"
	"github.com/Dankosik/privacy-sanitization-service/internal/app/ping"
	"github.com/Dankosik/privacy-sanitization-service/internal/infra/telemetry"
)

type Handlers struct {
	Health         *health.Service
	Ping           *ping.Service
	BeforeReady    func(context.Context) error
	OnReadySuccess func(context.Context) error
}

type strictHandlers struct {
	health         *health.Service
	ping           *ping.Service
	metrics        *telemetry.Metrics
	beforeReady    func(context.Context) error
	onReadySuccess func(context.Context) error
}

const readinessTimeout = 2 * time.Second

var _ api.StrictServerInterface = (*strictHandlers)(nil)

func newStrictHandlers(h Handlers, metrics *telemetry.Metrics) strictHandlers {
	if h.Health == nil {
		h.Health = health.New()
	}
	if h.Ping == nil {
		h.Ping = ping.New()
	}
	if metrics == nil {
		metrics = telemetry.New()
	}

	return strictHandlers{
		health:         h.Health,
		ping:           h.Ping,
		metrics:        metrics,
		beforeReady:    h.BeforeReady,
		onReadySuccess: h.OnReadySuccess,
	}
}

func (h strictHandlers) Ping(_ context.Context, _ api.PingRequestObject) (api.PingResponseObject, error) {
	return api.Ping200TextResponse(h.ping.Pong()), nil
}

func (h strictHandlers) HealthLive(_ context.Context, _ api.HealthLiveRequestObject) (api.HealthLiveResponseObject, error) {
	return api.HealthLive200TextResponse("ok"), nil
}

func (h strictHandlers) HealthReady(ctx context.Context, _ api.HealthReadyRequestObject) (api.HealthReadyResponseObject, error) {
	readyCtx, cancel := context.WithTimeout(ctx, readinessTimeout)
	defer cancel()

	if h.beforeReady != nil {
		if err := h.beforeReady(readyCtx); err != nil {
			return api.HealthReady503TextResponse("not ready"), nil
		}
	}
	if err := h.health.Ready(readyCtx); err != nil {
		return api.HealthReady503TextResponse("not ready"), nil
	}
	if h.onReadySuccess != nil {
		if err := h.onReadySuccess(readyCtx); err != nil {
			return api.HealthReady503TextResponse("not ready"), nil
		}
	}

	return api.HealthReady200TextResponse("ok"), nil
}

func (h strictHandlers) Metrics(ctx context.Context, _ api.MetricsRequestObject) (api.MetricsResponseObject, error) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil).WithContext(ctx)
	resp := httptest.NewRecorder()
	h.metrics.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		return nil, fmt.Errorf("metrics handler returned status %d", resp.Code)
	}

	return api.Metrics200TextResponse(resp.Body.String()), nil
}
