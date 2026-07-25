package telemetry

import (
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

// New builds the service metric registry.
//
// The Prometheus Go collector is deliberately absent: SetupMetrics registers the
// OpenTelemetry go.* runtime instruments on the meter provider instead, and those
// reach the OTLP reader as well as this registry. Registering both would publish
// the same runtime facts twice under two naming schemes.
//
// The process collector stays, and is the one signal that remains scrape-only:
// open file descriptors, resident memory, and process CPU seconds come from the
// operating system rather than the Go runtime, and no OpenTelemetry instrument
// here supplies them. A deployment that reaches this service only through a
// collector does not get them — named here rather than left to be discovered.
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
