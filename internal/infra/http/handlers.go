package httpx

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

type Handlers struct {
	Health        *health.Service
	Ping          *ping.Service
	ReadinessGate func(context.Context) error
}

type strictHandlers struct {
	health           *health.Service
	ping             *ping.Service
	metrics          *telemetry.Metrics
	readinessGate    func(context.Context) error
	readinessTimeout time.Duration
}

var _ api.StrictServerInterface = (*strictHandlers)(nil)

func newStrictHandlers(h Handlers, metrics *telemetry.Metrics, readinessTimeout time.Duration) (strictHandlers, error) {
	if h.Health == nil {
		return strictHandlers{}, fmt.Errorf("http router: health service is required")
	}
	if h.Ping == nil {
		return strictHandlers{}, fmt.Errorf("http router: ping service is required")
	}
	if h.ReadinessGate == nil {
		return strictHandlers{}, fmt.Errorf("http router: readiness gate is required")
	}
	if metrics == nil {
		return strictHandlers{}, fmt.Errorf("http router: metrics is required")
	}
	if readinessTimeout <= 0 {
		return strictHandlers{}, fmt.Errorf("http router: readiness timeout must be > 0")
	}

	return strictHandlers{
		health:           h.Health,
		ping:             h.Ping,
		metrics:          metrics,
		readinessGate:    h.ReadinessGate,
		readinessTimeout: readinessTimeout,
	}, nil
}

func (h strictHandlers) Ping(_ context.Context, _ api.PingRequestObject) (api.PingResponseObject, error) {
	return api.Ping200TextResponse(h.ping.Pong()), nil
}

func (h strictHandlers) HealthLive(_ context.Context, _ api.HealthLiveRequestObject) (api.HealthLiveResponseObject, error) {
	return api.HealthLive200TextResponse("ok"), nil
}

func (h strictHandlers) HealthReady(ctx context.Context, _ api.HealthReadyRequestObject) (api.HealthReadyResponseObject, error) {
	readyCtx, cancel := context.WithTimeout(ctx, h.readinessTimeout)
	defer cancel()

	if err := h.readinessGate(readyCtx); err != nil {
		return api.HealthReady503TextResponse("not ready"), nil
	}
	if err := h.health.Ready(readyCtx); err != nil {
		return api.HealthReady503TextResponse("not ready"), nil
	}

	return api.HealthReady200TextResponse("ok"), nil
}

func (h strictHandlers) Metrics(_ context.Context, _ api.MetricsRequestObject) (api.MetricsResponseObject, error) {
	// /metrics is served by the documented root-router exception, not the generated strict path.
	return nil, fmt.Errorf("metrics strict handler is not runtime-owned")
}
