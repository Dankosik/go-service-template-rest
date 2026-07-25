package telemetry

import (
	"context"
	"fmt"

	"github.com/prometheus/otlptranslator"
	"go.opentelemetry.io/otel"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
)

const (
	metricCardinalityLimit = 2000

	startupMeterName = "service.startup"

	traceExporterStateInstrument = "service.startup.trace_exporter.active"
)

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

// RecordTraceExporterState publishes whether the trace exporter is exporting.
// Metrics setup succeeds independently of tracing setup, so this value stays
// observable when tracing is degraded — which is the case an operator must be
// able to alert on, because a service with no trace export still reports
// healthy and answers every request.
func (m *Metrics) RecordTraceExporterState(ctx context.Context, active bool) error {
	// No unit: the Prometheus translation turns unit "1" into a `_ratio` suffix,
	// which would misname a boolean state.
	gauge, err := m.MeterProvider().Meter(startupMeterName).Int64Gauge(
		traceExporterStateInstrument,
		metric.WithDescription("1 when the OTLP trace exporter is configured and initialized, 0 otherwise."),
	)
	if err != nil {
		return fmt.Errorf("create trace exporter state gauge: %w", err)
	}

	state := int64(0)
	if active {
		state = 1
	}
	gauge.Record(ctx, state)
	return nil
}
