package telemetry

import (
	"context"
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
	// serverMeterName owns the instruments the HTTP chain records itself, as
	// opposed to the ones otelhttp derives from the request.
	serverMeterName = "service.http.server"

	activeRequestsInstrument = "http.server.active_requests"
	shedRequestsInstrument   = "http.server.shed_requests"
)

// ServerLoad is what the request path records about its own admission control.
//
// otelhttp supplies none of this. Its instruments are http.server.request.duration
// and the two body-size histograms, all keyed on the request — so a shed request
// appears only as one more 503, indistinguishable from a saturated connection pool
// answering 503 on the same route. Neither tells an operator how close the service
// is running to its concurrency limit, which is the number http.max_in_flight has
// to be tuned against. Shipping a load shedder without them means the limit is set
// once, from a guess, and never revisited.
type ServerLoad struct {
	active metric.Int64UpDownCounter
	shed   metric.Int64Counter
}

// ServerLoad builds the admission-control instruments. Errors are folded into
// no-op instruments rather than returned: a metric that cannot be created must
// not stop the service from serving, and the OpenTelemetry API guarantees a
// usable instrument alongside the error.
func (m *Metrics) ServerLoad() ServerLoad {
	meter := m.MeterProvider().Meter(serverMeterName)

	active, _ := meter.Int64UpDownCounter(
		activeRequestsInstrument,
		metric.WithDescription("Requests currently executing a handler, against the http.max_in_flight limit."),
		metric.WithUnit("{request}"),
	)
	shed, _ := meter.Int64Counter(
		shedRequestsInstrument,
		metric.WithDescription("Requests rejected without running a handler because the in-flight limit was reached."),
		metric.WithUnit("{request}"),
	)
	return ServerLoad{active: active, shed: shed}
}

// Admitted reports a request entering a handler, and returns the release that
// reports it leaving. A nil-safe zero value keeps callers that were built without
// a registry working.
func (l ServerLoad) Admitted(ctx context.Context) func() {
	if l.active == nil {
		return func() {}
	}
	l.active.Add(ctx, 1)
	return func() { l.active.Add(ctx, -1) }
}

// Shed reports a request rejected at the door.
func (l ServerLoad) Shed(ctx context.Context) {
	if l.shed == nil {
		return
	}
	l.shed.Add(ctx, 1)
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
