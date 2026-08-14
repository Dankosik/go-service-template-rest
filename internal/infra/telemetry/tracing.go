package telemetry

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	// TraceExporterConfigKey is this service's own exporter endpoint setting.
	TraceExporterConfigKey = "observability.otel.exporter.otlp_endpoint"
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

// fromConfig reports whether this service, rather than the platform, named the
// destination. It decides whether ambient credential and trust material is a
// conflict: material this service cannot verify must not travel to an endpoint
// this service chose.
func (e TraceExporterEndpoint) fromConfig() bool {
	return e.Source == TraceExporterConfigKey
}

var otelSetupMu sync.Mutex

// SetupTracing installs the tracer provider and reports which OTLP endpoint the
// exporter resolved to, so the caller can record and log that decision.
func SetupTracing(ctx context.Context, cfg TracingConfig) (TraceExporterEndpoint, func(context.Context) error, error) {
	sampler, err := buildTraceSampler(cfg.TracesSampler, cfg.TracesSamplerArg)
	if err != nil {
		return TraceExporterEndpoint{}, nil, err
	}

	res, err := newResource(ctx, cfg.Resource)
	if err != nil {
		return TraceExporterEndpoint{}, nil, err
	}

	exporterOptions, endpoint, err := buildTraceExporterOptions(cfg.Exporter)
	if err != nil {
		return TraceExporterEndpoint{}, nil, err
	}
	if !endpoint.Configured() {
		// Keep valid trace IDs for propagation and log correlation without recording spans that cannot be exported.
		sampler = sdktrace.NeverSample()
	}

	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	}
	if endpoint.Configured() {
		if endpoint.fromConfig() {
			if err := rejectConflictingTraceExporterEnv(); err != nil {
				return endpoint, nil, err
			}
		}
		exporter, err := otlptracehttp.New(ctx, exporterOptions...)
		if err != nil {
			return endpoint, nil, fmt.Errorf("create otlp trace exporter: %w", err)
		}
		options = append(options, sdktrace.WithBatcher(exporter))
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

func buildTraceExporterOptions(cfg TraceExporterConfig) ([]otlptracehttp.Option, TraceExporterEndpoint, error) {
	options := make([]otlptracehttp.Option, 0, 2)
	endpoint, err := ResolveTraceExporterEndpoint(cfg)
	if err != nil {
		return nil, TraceExporterEndpoint{}, err
	}
	if !endpoint.Configured() {
		return options, endpoint, nil
	}

	options = append(options, otlptracehttp.WithEndpointURL(endpoint.URL))
	if headers := strings.TrimSpace(cfg.OTLPHeaders); headers != "" {
		parsedHeaders, err := parseOTLPHeaders(headers)
		if err != nil {
			return nil, TraceExporterEndpoint{}, err
		}
		options = append(options, otlptracehttp.WithHeaders(parsedHeaders))
	}

	return options, endpoint, nil
}

// ResolveTraceExporterEndpoint reports which OTLP traces endpoint the exporter
// will use, and which setting supplied it.
//
// observability.otel.exporter.otlp_endpoint is this service's own setting and
// wins; the standard OpenTelemetry variables answer for the platform. Traces
// have one owned setting, so the whole order is the argument list below and
// [resolveOTLPEndpoint] owns what that order means — including why a configured
// header stops it.
func ResolveTraceExporterEndpoint(cfg TraceExporterConfig) (TraceExporterEndpoint, error) {
	return resolveOTLPEndpoint(
		otlpTracesPath,
		cfg.OTLPHeaders,
		[]otlpCandidate{{source: TraceExporterConfigKey, raw: cfg.OTLPEndpoint}},
		ambientOTLPCandidates(otelExporterTracesEndpointEnv),
	)
}
