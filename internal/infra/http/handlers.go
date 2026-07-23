package httpx

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/health"
)

type Handlers struct {
	Health        *health.Service
	ReadinessGate func(context.Context) error
}

type strictHandlers struct {
	health           *health.Service
	readinessGate    func(context.Context) error
	readinessTimeout time.Duration
}

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
