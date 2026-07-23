package telemetry

import (
	"context"
	"fmt"

	"github.com/prometheus/otlptranslator"
	"go.opentelemetry.io/otel"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
)

const metricCardinalityLimit = 2000

// MetricsConfig defines the service resource attached to metric instruments.
type MetricsConfig struct {
	ServiceName    string
	ServiceVersion string
	DeploymentEnv  string
}

// SetupMetrics bridges OpenTelemetry instruments into the service Prometheus registry.
func SetupMetrics(ctx context.Context, metrics *Metrics, cfg MetricsConfig) (func(context.Context) error, error) {
	if metrics == nil || metrics.registry == nil {
		return nil, fmt.Errorf("setup metrics: registry is required")
	}

	res, err := newResource(ctx, cfg.ServiceName, cfg.ServiceVersion, cfg.DeploymentEnv)
	if err != nil {
		return nil, err
	}
	exporter, err := otelprometheus.New(
		otelprometheus.WithRegisterer(metrics.registry),
		otelprometheus.WithTranslationStrategy(otlptranslator.UnderscoreEscapingWithSuffixes),
	)
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	otelSetupMu.Lock()
	defer otelSetupMu.Unlock()

	provider := newMeterProvider(
		sdkmetric.WithCardinalityLimit(metricCardinalityLimit),
		sdkmetric.WithExemplarFilter(exemplar.TraceBasedFilter),
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(res),
	)
	metrics.meterProvider = provider
	otel.SetMeterProvider(provider)

	return provider.Shutdown, nil
}

func newMeterProvider(options ...sdkmetric.Option) *sdkmetric.MeterProvider {
	restore := withoutOTELResourceEnv()
	defer restore()
	return sdkmetric.NewMeterProvider(options...)
}
