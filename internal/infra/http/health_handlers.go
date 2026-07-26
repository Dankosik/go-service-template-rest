package httpx

import (
	"context"

	"github.com/example/go-service-template-rest/internal/openapi"
)

var _ openapi.StrictServerInterface = (*strictHandlers)(nil)

func (h strictHandlers) HealthLive(_ context.Context, _ openapi.HealthLiveRequestObject) (openapi.HealthLiveResponseObject, error) {
	return openapi.HealthLive200TextResponse("ok"), nil
}

// HealthReady answers from cached readiness state and performs no dependency
// I/O. A probe route that checked a pooled dependency per request would need
// that dependency's own capacity in order to report on it, so a saturated pool
// would answer 503 and get the instance evicted while it was still serving.
//
// Both gates are in-memory reads: the startup gate is an atomic flag, and
// health.Cached returns whatever the background refresher last observed.
func (h strictHandlers) HealthReady(ctx context.Context, _ openapi.HealthReadyRequestObject) (openapi.HealthReadyResponseObject, error) {
	if err := h.readinessGate(ctx); err != nil {
		//nolint:nilerr // The strict contract represents not-ready as a typed 503; returning err would emit a 500 instead.
		return openapi.HealthReady503TextResponse("not ready"), nil
	}
	if err := h.health.Cached(); err != nil {
		//nolint:nilerr // Same: the readiness verdict is the response, not a transport fault.
		return openapi.HealthReady503TextResponse("not ready"), nil
	}

	return openapi.HealthReady200TextResponse("ok"), nil
}
