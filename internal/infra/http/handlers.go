package httpx

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/example/go-service-template-rest/internal/health"
	// profile:inbound-webhooks-standard:start
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	// profile:inbound-webhooks-standard:end
	"github.com/example/go-service-template-rest/internal/openapi"
)

// probeOperationCount is the always-present generated surface this package owns.
const probeOperationCount = 2

type Handlers struct {
	Health        *health.Service
	ReadinessGate func(context.Context) error
	// API implements every generated operation not owned by this transport. It is
	// the seam a service composes through: adding a business operation means
	// implementing it in the feature package, not editing this package.
	API openapi.StrictServerInterface
	// profile:inbound-webhooks-standard:start
	InboundWebhook inboundwebhook.Receiver
	// profile:inbound-webhooks-standard:end
}

type strictHandlers struct {
	// The embedded interface promotes the service's own operations. Explicit
	// transport-owned methods shadow it.
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
	ownedOperations := probeOperationCount
	// profile:inbound-webhooks-standard:start
	ownedOperations++
	// profile:inbound-webhooks-standard:end
	if declared := reflect.TypeFor[openapi.StrictServerInterface]().NumMethod(); h.API == nil && declared > ownedOperations {
		return strictHandlers{}, fmt.Errorf(
			"http router: handlers API is required: the OpenAPI contract declares %d operations and this package implements %d",
			declared,
			ownedOperations,
		)
	}

	return strictHandlers{
		StrictServerInterface: h.API,
		health:                h.Health,
		readinessGate:         h.ReadinessGate,
	}, nil
}
