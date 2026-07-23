package httpx

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/app/health"
)

const pingResponseBody = "pong"

type Handlers struct {
	Health        *health.Service
	ReadinessGate func(context.Context) error
}

type strictHandlers struct {
	health           *health.Service
	readinessGate    func(context.Context) error
	readinessTimeout time.Duration
}

var _ api.StrictServerInterface = (*strictHandlers)(nil)

func newStrictHandlers(h Handlers, readinessTimeout time.Duration) (strictHandlers, error) {
	if h.Health == nil {
		return strictHandlers{}, fmt.Errorf("http router: health service is required")
	}
	if h.ReadinessGate == nil {
		return strictHandlers{}, fmt.Errorf("http router: readiness gate is required")
	}
	if readinessTimeout <= 0 {
		return strictHandlers{}, fmt.Errorf("http router: readiness timeout must be > 0")
	}

	return strictHandlers{
		health:           h.Health,
		readinessGate:    h.ReadinessGate,
		readinessTimeout: readinessTimeout,
	}, nil
}

func (h strictHandlers) Ping(_ context.Context, _ api.PingRequestObject) (api.PingResponseObject, error) {
	return api.Ping200TextResponse(pingResponseBody), nil
}

// TEMPLATE EXAMPLE: delete this handler with its OpenAPI operation if unused,
// or replace it with real app-layer behavior and remove the marker.
func (h strictHandlers) TemplateExample(
	_ context.Context,
	request api.TemplateExampleRequestObject,
) (api.TemplateExampleResponseObject, error) {
	if request.Body == nil {
		return nil, fmt.Errorf("template example body is required")
	}

	messages := make([]string, request.Params.Copies)
	for i := range messages {
		messages[i] = request.Body.Message
	}

	return api.TemplateExample200JSONResponse{
		Slug:     request.Slug,
		Messages: messages,
	}, nil
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
