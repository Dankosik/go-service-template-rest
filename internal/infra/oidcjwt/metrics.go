package oidcjwt

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

const meterName = "service.authn"

type authnMetrics struct {
	verifications   metric.Int64Counter
	refreshFailures metric.Int64Counter
}

func newAuthnMetrics(provider metric.MeterProvider) authnMetrics {
	if provider == nil {
		provider = metricnoop.NewMeterProvider()
	}
	meter := provider.Meter(meterName)
	verifications, _ := meter.Int64Counter(
		"authn.verifications",
		metric.WithUnit("{verification}"),
		metric.WithDescription("Authentication verification outcomes."),
	)
	refreshFailures, _ := meter.Int64Counter(
		"authn.jwks.refresh_failures",
		metric.WithUnit("{failure}"),
		metric.WithDescription("Failed attempts to refresh OIDC signing keys."),
	)
	return authnMetrics{verifications: verifications, refreshFailures: refreshFailures}
}

func (m authnMetrics) recordVerification(ctx context.Context, carrier transport, err error) {
	if m.verifications == nil {
		return
	}
	result := "success"
	reason := ""
	if err != nil {
		result = "failure"
		reason = "invalid"
		if kind, ok := KindOf(err); ok {
			if detail, found := detailFor(kind); found {
				reason = detail.reason
			}
		}
	}
	attributes := []attribute.KeyValue{
		attribute.String("authn.transport", string(carrier)),
		attribute.String("authn.result", result),
	}
	if reason != "" {
		attributes = append(attributes, attribute.String("authn.reason", reason))
	}
	m.verifications.Add(ctx, 1, metric.WithAttributes(attributes...))
}

func (m authnMetrics) recordRefreshFailure(ctx context.Context) {
	if m.refreshFailures != nil {
		m.refreshFailures.Add(ctx, 1)
	}
}
