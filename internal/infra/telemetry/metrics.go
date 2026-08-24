package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

type Metrics struct {
	registry      *prometheus.Registry
	meterProvider metric.MeterProvider
}

const (
	startupMeterName = "service.startup"

	// Kept for dashboard compatibility: within the service.startup scope,
	// active means initialized at startup, not continuous collector delivery.
	traceExporterStateInstrument = "service.startup.trace_exporter.active"
)

// RecordTraceExporterInitialization publishes whether the trace exporter was
// configured and initialized during startup. Runtime export failures are
// reported by the OpenTelemetry SDK error handler.
func (m *Metrics) RecordTraceExporterInitialization(ctx context.Context, initialized bool) error {
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
	if initialized {
		state = 1
	}
	gauge.Record(ctx, state)
	return nil
}

// New builds the service metric registry.
//
// The Prometheus Go collector is deliberately absent: SetupMetrics registers the
// OpenTelemetry go.* runtime instruments on the meter provider instead, and those
// reach the OTLP reader as well as this registry. Registering both would publish
// the same runtime facts twice under two naming schemes.
//
// The process collector stays and is the one scrape-only signal: open file
// descriptors, resident memory, and process CPU seconds come from the operating
// system, and no OpenTelemetry instrument here supplies them, so a
// collector-only deployment does not get them.
func New() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return &Metrics{registry: registry}
}

// MeterProvider returns the configured provider or a no-op provider before telemetry setup.
func (m *Metrics) MeterProvider() metric.MeterProvider {
	if m == nil || m.meterProvider == nil {
		return metricnoop.NewMeterProvider()
	}
	return m.meterProvider
}

func (m *Metrics) Handler() http.Handler {
	if m == nil || m.registry == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
}
