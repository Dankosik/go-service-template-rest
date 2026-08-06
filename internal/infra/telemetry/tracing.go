package telemetry

import (
	"context"
	"fmt"
	"os"
	"slices"
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
	ServiceName    string
	ServiceVersion string
	// ServiceCommit is the source revision the binary was built from, published
	// as vcs.revision so a span names the build that produced it.
	ServiceCommit string
	// ServiceInstanceID identifies this replica. Resolve it once per process with
	// ResolveInstanceID and pass the same value to SetupMetrics; see
	// resourceIdentity for what an absent instance identity costs.
	ServiceInstanceID string
	DeploymentEnv     string
	TracesSampler     string
	TracesSamplerArg  float64
	Exporter          TraceExporterConfig
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

	res, err := newResource(ctx, resourceIdentity{
		serviceName:    cfg.ServiceName,
		serviceVersion: cfg.ServiceVersion,
		serviceCommit:  cfg.ServiceCommit,
		instanceID:     cfg.ServiceInstanceID,
		deploymentEnv:  cfg.DeploymentEnv,
	})
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

	provider := newTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return endpoint, provider.Shutdown, nil
}

// AmbientOTLPExporterEnv returns the sorted names of non-empty
// OTEL_EXPORTER_OTLP_* process variables. The endpoint variables among them are
// honored when this service names no endpoint of its own; the rest are not read.
// Callers subtract whatever supplied the endpoint and report the remainder, so
// an injected setting never looks effective when it is not.
func AmbientOTLPExporterEnv() []string {
	var names []string
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "OTEL_EXPORTER_OTLP_") && strings.TrimSpace(value) != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// traceExporterEnvConflicts are the standard OpenTelemetry exporter variables
// this service must not ignore when it configures its own exporter.
//
// otlptracehttp applies ambient environment first and explicit options second, so
// WithEndpointURL makes an injected ENDPOINT, TRACES_ENDPOINT, or INSECURE
// harmless. Credential and trust material is different: this service never sets
// client certificates or a root CA pool, and sets headers only when
// observability.otel.exporter.otlp_headers is non-empty, so these variables would
// silently travel to the collector unverified.
//
// This applies only when observability.otel.exporter.otlp_endpoint named the
// destination. When the endpoint came from the platform's own variables, the
// platform owns the whole exporter configuration; rejecting its credentials would
// refuse the ordinary injected-collector deployment for no gain.
//
// Kept sorted so reported output is stable.
var traceExporterEnvConflicts = []string{
	"OTEL_EXPORTER_OTLP_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_HEADERS",
	"OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_TRACES_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_TRACES_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_TRACES_HEADERS",
}

// ConflictingTraceExporterEnv returns the non-empty ambient exporter variables
// that a configured exporter cannot safely ignore. See
// traceExporterEnvConflicts for why the endpoint and transport-tuning variables
// are deliberately absent.
func ConflictingTraceExporterEnv() []string {
	names := make([]string, 0, len(traceExporterEnvConflicts))
	for _, name := range traceExporterEnvConflicts {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func rejectConflictingTraceExporterEnv() error {
	if names := ConflictingTraceExporterEnv(); len(names) > 0 {
		return fmt.Errorf(
			"unsupported ambient otel exporter environment (%s): injected credentials and trust material are not verifiable here; configure observability.otel.exporter.* instead",
			strings.Join(names, ", "),
		)
	}

	return nil
}

func newTracerProvider(options ...sdktrace.TracerProviderOption) *sdktrace.TracerProvider {
	// OTel SDK v1.40 merges resource.Environment() inside sdktrace.WithResource.
	// Clear only the resource env keys while the provider is built so config remains the sole resource source.
	return sdktrace.NewTracerProvider(options...)
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
// wins. When it names nothing, the standard OpenTelemetry endpoint variables are
// honored: they are what a platform collector injects, and a service that
// ignored them would report healthy, answer every request, and export no trace.
//
// Configured headers are a credential, so they pin the destination. When this
// service names headers but no endpoint there is no fallback, because sending
// the service's own credentials to an endpoint it never named is the one
// outcome this resolution must not create.
func ResolveTraceExporterEndpoint(cfg TraceExporterConfig) (TraceExporterEndpoint, error) {
	if raw := strings.TrimSpace(cfg.OTLPEndpoint); raw != "" {
		endpoint, err := parseSignalOTLPEndpoint(raw, otlpTracesPath)
		if err != nil {
			return TraceExporterEndpoint{}, err
		}
		return TraceExporterEndpoint{URL: endpoint.endpointURL, Source: TraceExporterConfigKey}, nil
	}
	if strings.TrimSpace(cfg.OTLPHeaders) != "" {
		return TraceExporterEndpoint{}, nil
	}

	if raw, ok := nonEmptyEnv(otelExporterTracesEndpointEnv); ok {
		endpoint, err := parseSignalOTLPEndpoint(raw, otlpTracesPath)
		if err != nil {
			return TraceExporterEndpoint{}, fmt.Errorf("%s: %w", otelExporterTracesEndpointEnv, err)
		}
		return TraceExporterEndpoint{URL: endpoint.endpointURL, Source: otelExporterTracesEndpointEnv}, nil
	}
	if raw, ok := nonEmptyEnv(otelExporterEndpointEnv); ok {
		endpoint, err := parseBaseOTLPEndpoint(raw, otlpTracesPath)
		if err != nil {
			return TraceExporterEndpoint{}, fmt.Errorf("%s: %w", otelExporterEndpointEnv, err)
		}
		return TraceExporterEndpoint{URL: endpoint.endpointURL, Source: otelExporterEndpointEnv}, nil
	}

	return TraceExporterEndpoint{}, nil
}
