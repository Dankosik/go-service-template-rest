package oidcjwt

import (
	"context"

	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

type jwksMetrics struct {
	refreshFailures metric.Int64Counter
}

func newJWKSMetrics(provider metric.MeterProvider) jwksMetrics {
	if provider == nil {
		provider = metricnoop.NewMeterProvider()
	}
	meter := provider.Meter(bearerauthn.MeterName)
	refreshFailures, _ := meter.Int64Counter(
		"authn.jwks.refresh_failures",
		metric.WithUnit("{failure}"),
		metric.WithDescription("Failed attempts to refresh OIDC signing keys."),
	)
	return jwksMetrics{refreshFailures: refreshFailures}
}

func (m jwksMetrics) recordRefreshFailure(ctx context.Context) {
	if m.refreshFailures != nil {
		m.refreshFailures.Add(ctx, 1)
	}
}
