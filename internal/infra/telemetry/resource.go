package telemetry

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func newResource(ctx context.Context, serviceName, serviceVersion, deploymentEnv string) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", strings.TrimSpace(serviceName)),
			attribute.String("service.version", strings.TrimSpace(serviceVersion)),
			attribute.String("deployment.environment.name", strings.TrimSpace(deploymentEnv)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}
	return res, nil
}
