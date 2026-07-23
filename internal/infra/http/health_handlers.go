package httpx

import (
	"context"

	"github.com/example/go-service-template-rest/internal/openapi"
)

var _ openapi.StrictServerInterface = (*strictHandlers)(nil)

func (h strictHandlers) HealthLive(_ context.Context, _ openapi.HealthLiveRequestObject) (openapi.HealthLiveResponseObject, error) {
	return openapi.HealthLive200TextResponse("ok"), nil
}

func (h strictHandlers) HealthReady(ctx context.Context, _ openapi.HealthReadyRequestObject) (openapi.HealthReadyResponseObject, error) {
	readyCtx, cancel := context.WithTimeout(ctx, h.readinessTimeout)
	defer cancel()

	if err := h.readinessGate(readyCtx); err != nil {
		return openapi.HealthReady503TextResponse("not ready"), nil
	}
	if err := h.health.Ready(readyCtx); err != nil {
		return openapi.HealthReady503TextResponse("not ready"), nil
	}

	return openapi.HealthReady200TextResponse("ok"), nil
}
