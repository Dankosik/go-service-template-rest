package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/prometheus/otlptranslator"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
)

const (
	metricCardinalityLimit = 2000

	// MetricExporterConfigKey is this service's own metrics exporter endpoint
	// setting.
	MetricExporterConfigKey = "observability.otel.exporter.otlp_metrics_endpoint"
)

// MetricsConfig defines the service resource attached to metric instruments and
// where those instruments are exported.
type MetricsConfig struct {
	// Resource is the identity every exported series is attributed to, and must
	// be the same value SetupTracing was given; see ResourceConfig.
	Resource ResourceConfig
	Exporter MetricExporterConfig
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

// MetricsResult reports what metrics setup achieved.
//
// A returned error from SetupMetrics means no meter provider exists at all;
// ExportErr means only the OTLP push path could not be built, and the provider
// is installed and serving the Prometheus registry.
type MetricsResult struct {
	// Shutdown flushes and stops the provider. Nil only when SetupMetrics
	// returned an error.
	Shutdown func(context.Context) error
	// Endpoint is the OTLP destination that was resolved, whether or not a
	// reader could be built for it. An operator debugging where metrics went
	// needs to see what was attempted.
	Endpoint ExporterEndpoint
	// ExportErr reports that OTLP push is unavailable. Scrape still works.
	ExportErr error
}

// PushInitialized reports whether the OTLP metrics reader was built during
// startup. Runtime delivery failures are reported by the OTel SDK error handler.
func (r MetricsResult) PushInitialized() bool {
	return r.ExportErr == nil && r.Endpoint.Configured()
}

// SetupMetrics bridges OpenTelemetry instruments into the service Prometheus
// registry, and pushes them over OTLP when an endpoint resolves.
//
// Both readers, not one. The Prometheus endpoint answers a scraper that can reach
// this pod's diagnostics listener; the OTLP reader answers the deployment shape
// where nothing can.
//
// An unusable OTLP destination degrades the push path and nothing else: failing
// the whole setup would take the meter provider with it, leaving no metrics at
// all and no way to record the gauge reporting that.
func SetupMetrics(ctx context.Context, metrics *Metrics, cfg MetricsConfig) (MetricsResult, error) {
	if metrics == nil || metrics.registry == nil {
		return MetricsResult{}, errors.New("setup metrics: registry is required")
	}

	res, err := newResource(ctx, cfg.Resource)
	if err != nil {
		return MetricsResult{}, err
	}
	promExporter, err := otelprometheus.New(
		otelprometheus.WithRegisterer(metrics.registry),
		otelprometheus.WithTranslationStrategy(otlptranslator.UnderscoreEscapingWithSuffixes),
	)
	if err != nil {
		return MetricsResult{}, fmt.Errorf("create prometheus exporter: %w", err)
	}

	options := []sdkmetric.Option{
		sdkmetric.WithCardinalityLimit(metricCardinalityLimit),
		sdkmetric.WithExemplarFilter(exemplar.TraceBasedFilter),
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	}

	endpoint, exportErr := resolveMetricExporterEndpoint(cfg.Exporter)
	if exportErr == nil && endpoint.Configured() {
		var reader sdkmetric.Reader
		if reader, exportErr = newOTLPMetricReader(ctx, endpoint, cfg.Exporter); exportErr == nil {
			// The export interval comes from OTEL_METRIC_EXPORT_INTERVAL or the
			// SDK's 60s default. This service adds no setting of its own for it:
			// whoever runs the collector owns that cadence, and the standard
			// variable is already how they express it.
			options = append(options, sdkmetric.WithReader(reader))
		}
	}

	otelSetupMu.Lock()
	defer otelSetupMu.Unlock()

	// Shutdown flushes every reader the provider owns, so the periodic reader's
	// last interval is exported rather than dropped on exit.
	provider, err := startMeterProvider(ctx, options, func(provider *sdkmetric.MeterProvider) error {
		// Registered on the provider rather than on the Prometheus registry, so
		// runtime instruments reach both readers.
		return otelruntime.Start(otelruntime.WithMeterProvider(provider))
	})
	if err != nil {
		return MetricsResult{}, fmt.Errorf("start go runtime metrics: %w", err)
	}

	metrics.meterProvider = provider
	otel.SetMeterProvider(provider)

	return MetricsResult{Shutdown: provider.Shutdown, Endpoint: endpoint, ExportErr: exportErr}, nil
}

func startMeterProvider(
	ctx context.Context,
	options []sdkmetric.Option,
	start func(*sdkmetric.MeterProvider) error,
) (*sdkmetric.MeterProvider, error) {
	provider := sdkmetric.NewMeterProvider(options...)
	if err := start(provider); err != nil {
		return nil, errors.Join(err, provider.Shutdown(ctx))
	}
	return provider, nil
}

func newOTLPMetricReader(
	ctx context.Context,
	endpoint ExporterEndpoint,
	cfg MetricExporterConfig,
) (sdkmetric.Reader, error) {
	// Only when this service named the destination. When the platform's own
	// variables named it, the platform owns the whole exporter configuration and
	// its credentials belong to the collector it also named.
	if endpoint.fromConfig() {
		if err := rejectConflictingAmbientEnv(metricExporterEnvConflicts); err != nil {
			return nil, err
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

// resolveMetricExporterEndpoint reports which OTLP metrics endpoint the exporter
// will use, and which setting supplied it.
//
// The order mirrors trace resolution — [resolveOTLPEndpoint] owns what it means —
// with one addition metrics alone have: a bare
// observability.otel.exporter.otlp_endpoint serves both signals, because naming a
// collector root is what an operator means by it.
func resolveMetricExporterEndpoint(cfg MetricExporterConfig) (ExporterEndpoint, error) {
	owned := []otlpCandidate{{source: MetricExporterConfigKey, raw: cfg.OTLPEndpoint}}
	// A root only. Once the shared value carries a path it is an endpoint for one
	// signal and says nothing about where the other one goes, so metrics fall
	// through to their own settings instead of borrowing the traces route.
	if shared := strings.TrimSpace(cfg.SharedOTLPEndpoint); namesOTLPRoot(shared) {
		owned = append(owned, otlpCandidate{source: TraceExporterConfigKey, raw: shared, base: true})
	}

	return resolveOTLPEndpoint(
		otlpMetricsPath,
		cfg.OTLPHeaders,
		owned,
		ambientOTLPCandidates(otelExporterMetricsEndpointEnv),
	)
}
