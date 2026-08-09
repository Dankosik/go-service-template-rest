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

// The gRPC admission signals are exported because two other builds emit or
// assert on the same series and must not spell them again: the local benchmark
// server in examples/grpc-reference-service, which stands in for this package,
// and internal/infra/grpc's benchmark composition proof. A renamed series is a
// broken dashboard, and a copy renamed on one side alone is one nobody notices.
const (
	GRPCServerMeterName = "service.grpc.server"

	ActiveRPCsInstrument = "rpc.server.active_requests"
	ShedRPCsInstrument   = "rpc.server.shed_requests"

	ActiveRPCsDescription = "RPCs currently executing a handler, against the grpc.server.max_concurrent_rpcs limit."
	ShedRPCsDescription   = "RPCs rejected before running a handler because the process RPC limit was reached."
	RPCsUnit              = "{rpc}"
)

// ServerLoad is what the request path records about its own admission control.
//
// otelhttp supplies none of this: its instruments are all keyed on the request,
// so a shed request appears only as one more 503, indistinguishable from a
// saturated connection pool answering 503 on the same route — and none of them
// report how close the service runs to the limit http.max_in_flight sets.
type ServerLoad struct {
	active metric.Int64UpDownCounter
	shed   metric.Int64Counter
}

// ServerLoad builds the admission-control instruments. Errors are folded into
// no-op instruments rather than returned: a metric that cannot be created must
// not stop the service from serving, and the OpenTelemetry API guarantees a
// usable instrument alongside the error.
func (m *Metrics) ServerLoad() ServerLoad {
	return m.serverLoad(
		serverMeterName,
		activeRequestsInstrument,
		shedRequestsInstrument,
		"Requests currently executing a handler, against the http.max_in_flight limit.",
		"Requests rejected without running a handler because the in-flight limit was reached.",
		"{request}",
	)
}

// GRPCServerLoad builds the admission-control instruments for business RPCs.
func (m *Metrics) GRPCServerLoad() ServerLoad {
	return m.serverLoad(
		GRPCServerMeterName,
		ActiveRPCsInstrument,
		ShedRPCsInstrument,
		ActiveRPCsDescription,
		ShedRPCsDescription,
		RPCsUnit,
	)
}

func (m *Metrics) serverLoad(
	meterName string,
	activeName string,
	shedName string,
	activeDescription string,
	shedDescription string,
	unit string,
) ServerLoad {
	meter := m.MeterProvider().Meter(meterName)
	active, _ := meter.Int64UpDownCounter(
		activeName,
		metric.WithDescription(activeDescription),
		metric.WithUnit(unit),
	)
	shed, _ := meter.Int64Counter(
		shedName,
		metric.WithDescription(shedDescription),
		metric.WithUnit(unit),
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
