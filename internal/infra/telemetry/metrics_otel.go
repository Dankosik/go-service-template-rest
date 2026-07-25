package telemetry

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/prometheus/otlptranslator"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	otlpmetrichttp "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
)

const (
	metricCardinalityLimit = 2000

	startupMeterName = "service.startup"

	traceExporterStateInstrument = "service.startup.trace_exporter.active"

	// MetricExporterConfigKey is this service's own metrics exporter endpoint
	// setting.
	MetricExporterConfigKey = "observability.otel.exporter.otlp_metrics_endpoint"
)

// MetricsConfig defines the service resource attached to metric instruments and
// where those instruments are exported.
type MetricsConfig struct {
	ServiceName    string
	ServiceVersion string
	// ServiceInstanceID identifies this replica, and must be the same value
	// SetupTracing was given; see resourceIdentity.
	ServiceInstanceID string
	DeploymentEnv     string
	Exporter          MetricExporterConfig
}

// MetricExporterConfig names where metrics are pushed, in addition to the
// Prometheus registry that is always served.
type MetricExporterConfig struct {
	// OTLPEndpoint is observability.otel.exporter.otlp_metrics_endpoint: a full
	// OTLP HTTP metrics endpoint. A missing path defaults to /v1/metrics.
	OTLPEndpoint string
	// SharedOTLPEndpoint is observability.otel.exporter.otlp_endpoint, and is
	// used only when it names a bare collector root — in which case metrics
	// resolve to <root>/v1/metrics. A value that already carries a path is an
	// endpoint for one signal and says nothing about where the other one goes.
	SharedOTLPEndpoint string
	// OTLPHeaders is observability.otel.exporter.otlp_headers. A collector
	// credential belongs to the collector rather than to one signal, so the same
	// value covers both.
	OTLPHeaders string
}

// metricExporterEnvConflicts are the ambient credential and trust variables a
// configured metrics exporter must not silently honor, for the same reason
// traceExporterEnvConflicts exists: this service sets no client certificate and
// no root CA pool, so injected material would travel to a collector this service
// named with nothing having verified it.
//
// Kept sorted so reported output is stable.
var metricExporterEnvConflicts = []string{
	"OTEL_EXPORTER_OTLP_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_HEADERS",
	"OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_METRICS_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_METRICS_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
}

// SetupMetrics bridges OpenTelemetry instruments into the service Prometheus
// registry, and pushes them over OTLP when an endpoint resolves.
//
// Both readers, not one. The Prometheus endpoint answers a scraper that can reach
// this pod's diagnostics listener; the OTLP reader answers the deployment shape
// where nothing can — a collector the platform names through
// OTEL_EXPORTER_OTLP_ENDPOINT and that this service pushes to. Traces already
// honored that variable. Metrics that did not meant a service could export every
// span while exporting no metric at all, on a port the container image does not
// publish, and look correctly instrumented while doing it.
func SetupMetrics(ctx context.Context, metrics *Metrics, cfg MetricsConfig) (func(context.Context) error, ExporterEndpoint, error) {
	if metrics == nil || metrics.registry == nil {
		return nil, ExporterEndpoint{}, fmt.Errorf("setup metrics: registry is required")
	}

	res, err := newResource(ctx, resourceIdentity{
		serviceName:    cfg.ServiceName,
		serviceVersion: cfg.ServiceVersion,
		instanceID:     cfg.ServiceInstanceID,
		deploymentEnv:  cfg.DeploymentEnv,
	})
	if err != nil {
		return nil, ExporterEndpoint{}, err
	}
	promExporter, err := otelprometheus.New(
		otelprometheus.WithRegisterer(metrics.registry),
		otelprometheus.WithTranslationStrategy(otlptranslator.UnderscoreEscapingWithSuffixes),
	)
	if err != nil {
		return nil, ExporterEndpoint{}, fmt.Errorf("create prometheus exporter: %w", err)
	}

	options := []sdkmetric.Option{
		sdkmetric.WithCardinalityLimit(metricCardinalityLimit),
		sdkmetric.WithExemplarFilter(exemplar.TraceBasedFilter),
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	}

	endpoint, err := ResolveMetricExporterEndpoint(cfg.Exporter)
	if err != nil {
		return nil, ExporterEndpoint{}, err
	}
	if endpoint.Configured() {
		reader, readerErr := newOTLPMetricReader(ctx, endpoint, cfg.Exporter)
		if readerErr != nil {
			return nil, endpoint, readerErr
		}
		// The export interval comes from OTEL_METRIC_EXPORT_INTERVAL or the SDK's
		// 60s default. This service adds no setting of its own for it: whoever
		// runs the collector owns that cadence, and the standard variable is
		// already how they express it.
		options = append(options, sdkmetric.WithReader(reader))
	}

	otelSetupMu.Lock()
	defer otelSetupMu.Unlock()

	// Shutdown flushes every reader the provider owns, so the periodic reader's
	// last interval is exported rather than dropped on exit.
	provider := newMeterProvider(options...)

	// Runtime metrics are registered on the provider rather than on the Prometheus
	// registry, so they reach both readers.
	//
	// The version this replaced used the Prometheus client's Go collector, which
	// only the /metrics handler reads. A service that pushes to a collector — the
	// deployment this file's OTLP reader exists for, on a port the container image
	// does not publish — therefore exported request and pool metrics and no
	// goroutine, heap, or GC series at all. The first symptom of a leaked goroutine
	// was the OOM kill, and pprof is off by default, so the evidence was gone by
	// the time anyone could look.
	//
	// This registers observable instruments and a callback; it starts no goroutine
	// and no timer, which is what keeps the package's goleak check meaningful.
	if err := otelruntime.Start(otelruntime.WithMeterProvider(provider)); err != nil {
		return nil, endpoint, fmt.Errorf("start go runtime metrics: %w", err)
	}

	metrics.meterProvider = provider
	otel.SetMeterProvider(provider)

	return provider.Shutdown, endpoint, nil
}

func newOTLPMetricReader(
	ctx context.Context,
	endpoint ExporterEndpoint,
	cfg MetricExporterConfig,
) (sdkmetric.Reader, error) {
	// Only when this service named the destination. When the platform's own
	// variables named it, the platform owns the whole exporter configuration and
	// its credentials belong to the collector it also named.
	if endpoint.Source == MetricExporterConfigKey {
		if names := ConflictingMetricExporterEnv(); len(names) > 0 {
			return nil, fmt.Errorf(
				"unsupported ambient otel exporter environment (%s): injected credentials and trust material are not verifiable here; configure observability.otel.exporter.* instead",
				strings.Join(names, ", "),
			)
		}
	}

	exporterOptions := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpointURL(endpoint.URL)}
	if headers := strings.TrimSpace(cfg.OTLPHeaders); headers != "" {
		parsedHeaders, err := parseOTLPHeaders(headers)
		if err != nil {
			return nil, err
		}
		exporterOptions = append(exporterOptions, otlpmetrichttp.WithHeaders(parsedHeaders))
	}

	exporter, err := otlpmetrichttp.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}
	return sdkmetric.NewPeriodicReader(exporter), nil
}

// ResolveMetricExporterEndpoint reports which OTLP metrics endpoint the exporter
// will use, and which setting supplied it.
//
// The order mirrors trace resolution, with one addition: a bare
// observability.otel.exporter.otlp_endpoint serves both signals, because naming a
// collector root is what an operator means by it. Once that value carries a path
// it is an endpoint for one signal, and metrics need their own.
//
// Configured headers are a credential, so they pin the destination. Past the
// settings this service owns, the endpoint would come from ambient environment
// this service never named, and sending its credentials there is the one outcome
// this resolution must not create.
func ResolveMetricExporterEndpoint(cfg MetricExporterConfig) (ExporterEndpoint, error) {
	if raw := strings.TrimSpace(cfg.OTLPEndpoint); raw != "" {
		endpoint, err := parseSignalOTLPEndpoint(raw, otlpMetricsPath)
		if err != nil {
			return ExporterEndpoint{}, err
		}
		return ExporterEndpoint{URL: endpoint.endpointURL, Source: MetricExporterConfigKey}, nil
	}
	if raw := strings.TrimSpace(cfg.SharedOTLPEndpoint); raw != "" && namesOTLPRoot(raw) {
		endpoint, err := parseBaseOTLPEndpoint(raw, otlpMetricsPath)
		if err != nil {
			return ExporterEndpoint{}, err
		}
		return ExporterEndpoint{URL: endpoint.endpointURL, Source: TraceExporterConfigKey}, nil
	}
	if strings.TrimSpace(cfg.OTLPHeaders) != "" {
		return ExporterEndpoint{}, nil
	}

	if raw, ok := nonEmptyEnv(otelExporterMetricsEndpointEnv); ok {
		endpoint, err := parseSignalOTLPEndpoint(raw, otlpMetricsPath)
		if err != nil {
			return ExporterEndpoint{}, fmt.Errorf("%s: %w", otelExporterMetricsEndpointEnv, err)
		}
		return ExporterEndpoint{URL: endpoint.endpointURL, Source: otelExporterMetricsEndpointEnv}, nil
	}
	if raw, ok := nonEmptyEnv(otelExporterEndpointEnv); ok {
		endpoint, err := parseBaseOTLPEndpoint(raw, otlpMetricsPath)
		if err != nil {
			return ExporterEndpoint{}, fmt.Errorf("%s: %w", otelExporterEndpointEnv, err)
		}
		return ExporterEndpoint{URL: endpoint.endpointURL, Source: otelExporterEndpointEnv}, nil
	}

	return ExporterEndpoint{}, nil
}

// ConflictingMetricExporterEnv returns the non-empty ambient exporter variables a
// configured metrics exporter cannot safely ignore.
func ConflictingMetricExporterEnv() []string {
	names := make([]string, 0, len(metricExporterEnvConflicts))
	for _, name := range metricExporterEnvConflicts {
		if _, ok := nonEmptyEnv(name); ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func newMeterProvider(options ...sdkmetric.Option) *sdkmetric.MeterProvider {
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
