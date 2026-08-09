package httpx

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/openapi"
)

// probeOperationCount is how many generated operations this package implements
// itself: HealthLive and HealthReady. It exists so a contract that grows past
// the platform probes without a Handlers.API is rejected at startup instead of
// panicking on a nil embedded interface once per request.
const probeOperationCount = 2

type Handlers struct {
	Health        *health.Service
	ReadinessGate func(context.Context) error
	// API implements every generated operation other than the platform probes.
	// It is the seam a service composes through: adding an operation means
	// implementing it here, not editing this package.
	API openapi.StrictServerInterface
}

type strictHandlers struct {
	// The embedded interface promotes the service's own operations. The explicit
	// HealthLive and HealthReady methods in health_handlers.go shadow whatever it
	// supplies for those two, so platform probe behavior stays owned here.
	openapi.StrictServerInterface

	health        *health.Service
	readinessGate func(context.Context) error
}

func newStrictHandlers(h Handlers) (strictHandlers, error) {
	if h.Health == nil {
		return strictHandlers{}, errors.New("http router: health service is required")
	}
	if h.ReadinessGate == nil {
		return strictHandlers{}, errors.New("http router: readiness gate is required")
	}
	if declared := reflect.TypeFor[openapi.StrictServerInterface]().NumMethod(); h.API == nil && declared > probeOperationCount {
		return strictHandlers{}, fmt.Errorf(
			"http router: handlers API is required: the OpenAPI contract declares %d operations and this package implements %d",
			declared,
			probeOperationCount,
		)
	}

	return strictHandlers{
		StrictServerInterface: h.API,
		health:                h.Health,
		readinessGate:         h.ReadinessGate,
	}, nil
}
