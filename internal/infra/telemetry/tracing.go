package telemetry

import (
	"context"
	"fmt"
	"sync"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type TracingConfig struct {
	// Resource is the identity every exported span is attributed to, and must be
	// the same value SetupMetrics was given; see ResourceConfig.
	Resource         ResourceConfig
	TracesSampler    string
	TracesSamplerArg float64
	Exporter         TraceExporterConfig
}

type TraceExporterConfig struct {
	OTLPEndpoint string
	OTLPHeaders  string
}

// TraceExporterEndpoint is the resolved OTLP traces endpoint and the setting that
// supplied it. Metrics resolve the same shape through the same primitives; see
// otlp_endpoint.go.
type TraceExporterEndpoint = ExporterEndpoint

var otelSetupMu sync.Mutex

// SetupTracing installs the tracer provider and reports which OTLP endpoint the
// exporter resolved to, so the caller can record and log that decision. The
// shutdown function is non-nil only when a provider was installed. If setup
// fails after endpoint resolution, the endpoint is still returned with the
// error, and shutdown is nil.
func SetupTracing(ctx context.Context, cfg TracingConfig) (endpoint TraceExporterEndpoint, shutdown func(context.Context) error, err error) {
	sampler, err := buildTraceSampler(cfg.TracesSampler, cfg.TracesSamplerArg)
	if err != nil {
		return TraceExporterEndpoint{}, nil, err
	}

	res, err := newResource(ctx, cfg.Resource)
	if err != nil {
		return TraceExporterEndpoint{}, nil, err
	}

	endpoint, err = resolveTraceExporterEndpoint(cfg.Exporter)
	if err != nil {
		return TraceExporterEndpoint{}, nil, err
	}

	options := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if endpoint.Configured() {
		exporter, err := newOTLPTraceExporter(ctx, endpoint, cfg.Exporter)
		if err != nil {
			return endpoint, nil, err
		}
		options = append(options, sdktrace.WithSampler(sampler), sdktrace.WithBatcher(exporter))
	} else {
		// Keep valid trace IDs for propagation and log correlation without recording spans that cannot be exported.
		options = append(options, sdktrace.WithSampler(sdktrace.NeverSample()))
	}

	otelSetupMu.Lock()
	defer otelSetupMu.Unlock()

	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return endpoint, provider.Shutdown, nil
}

func buildTraceSampler(name string, arg float64) (sdktrace.Sampler, error) {
	if err := otelconfig.ValidateTraceSampler(name, arg); err != nil {
		return nil, fmt.Errorf("build trace sampler: %w", err)
	}

	switch otelconfig.TraceSamplerOrDefault(name) {
	case otelconfig.SamplerAlwaysOn:
		return sdktrace.AlwaysSample(), nil
	case otelconfig.SamplerAlwaysOff:
		return sdktrace.NeverSample(), nil
	case otelconfig.SamplerTraceIDRatio:
		return sdktrace.TraceIDRatioBased(arg), nil
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(arg)), nil
	}
}

func newOTLPTraceExporter(
	ctx context.Context,
	endpoint TraceExporterEndpoint,
	cfg TraceExporterConfig,
) (sdktrace.SpanExporter, error) {
	headers, err := otlpExporterHeaders(endpoint, traceExporterEnvConflicts, cfg.OTLPHeaders)
	if err != nil {
		return nil, err
	}

	exporterOptions := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(endpoint.URL)}
	if len(headers) != 0 {
		exporterOptions = append(exporterOptions, otlptracehttp.WithHeaders(headers))
	}

	exporter, err := otlptracehttp.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}
	return exporter, nil
}

// resolveTraceExporterEndpoint reports which OTLP traces endpoint the exporter
// will use, and which setting supplied it.
//
// observability.otel.exporter.otlp_endpoint is this service's own setting and
// wins; the standard OpenTelemetry variables answer for the platform. Traces
// have one owned setting, so the whole order is the argument list below and
// [resolveOTLPEndpoint] owns what that order means — including why a configured
// header stops it.
func resolveTraceExporterEndpoint(cfg TraceExporterConfig) (TraceExporterEndpoint, error) {
	return resolveOTLPEndpoint(
		otlpTracesPath,
		cfg.OTLPHeaders,
		[]otlpCandidate{{source: SharedOTLPExporterConfigKey, raw: cfg.OTLPEndpoint, configuredByService: true}},
		ambientOTLPCandidates(otelExporterTracesEndpointEnv),
	)
}
