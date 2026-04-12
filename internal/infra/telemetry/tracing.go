package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type TracingConfig struct {
	ServiceName      string
	ServiceVersion   string
	DeploymentEnv    string
	TracesSampler    string
	TracesSamplerArg float64
	Exporter         TraceExporterConfig
}

type TraceExporterConfig struct {
	OTLPEndpoint       string
	OTLPTracesEndpoint string
	OTLPHeaders        string
	OTLPProtocol       string
}

func SetupTracing(ctx context.Context, cfg TracingConfig) (func(context.Context) error, error) {
	serviceName := strings.TrimSpace(cfg.ServiceName)
	serviceVersion := strings.TrimSpace(cfg.ServiceVersion)
	deploymentEnv := strings.TrimSpace(cfg.DeploymentEnv)

	sampler, err := buildTraceSampler(cfg.TracesSampler, cfg.TracesSamplerArg)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", serviceVersion),
			attribute.String("deployment.environment.name", deploymentEnv),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	}

	exporterOptions, exporterConfigured, err := buildTraceExporterOptions(cfg.Exporter)
	if err != nil {
		return nil, err
	}
	if exporterConfigured {
		exporter, err := otlptracehttp.New(ctx, exporterOptions...)
		if err != nil {
			return nil, fmt.Errorf("create otlp trace exporter: %w", err)
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	}

	provider := newTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider.Shutdown, nil
}

func newTracerProvider(options ...sdktrace.TracerProviderOption) *sdktrace.TracerProvider {
	// OTel SDK v1.40 merges resource.Environment() inside sdktrace.WithResource.
	// Clear only the resource env keys while the provider is built so config remains the sole resource source.
	restore := withoutOTELResourceEnv()
	defer restore()
	return sdktrace.NewTracerProvider(options...)
}

func withoutOTELResourceEnv() func() {
	const (
		otelResourceAttributesEnv = "OTEL_RESOURCE_ATTRIBUTES"
		otelServiceNameEnv        = "OTEL_SERVICE_NAME"
	)

	resourceAttrs, hadResourceAttrs := os.LookupEnv(otelResourceAttributesEnv)
	serviceName, hadServiceName := os.LookupEnv(otelServiceNameEnv)
	_ = os.Unsetenv(otelResourceAttributesEnv)
	_ = os.Unsetenv(otelServiceNameEnv)

	return func() {
		if hadResourceAttrs {
			_ = os.Setenv(otelResourceAttributesEnv, resourceAttrs)
		} else {
			_ = os.Unsetenv(otelResourceAttributesEnv)
		}
		if hadServiceName {
			_ = os.Setenv(otelServiceNameEnv, serviceName)
		} else {
			_ = os.Unsetenv(otelServiceNameEnv)
		}
	}
}

func buildTraceSampler(name string, arg float64) (sdktrace.Sampler, error) {
	if !otelconfig.TraceSamplerArgFinite(arg) {
		return nil, fmt.Errorf("trace sampler arg must be finite")
	}
	if !otelconfig.TraceSamplerArgInRange(arg) {
		return nil, fmt.Errorf("trace sampler arg must be in range [0,1]")
	}

	samplerName := otelconfig.TraceSamplerOrDefault(name)

	switch samplerName {
	case otelconfig.SamplerAlwaysOn:
		return sdktrace.AlwaysSample(), nil
	case otelconfig.SamplerAlwaysOff:
		return sdktrace.NeverSample(), nil
	case otelconfig.SamplerTraceIDRatio:
		return sdktrace.TraceIDRatioBased(arg), nil
	case otelconfig.SamplerParentBasedTraceIDRatio:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(arg)), nil
	default:
		return nil, fmt.Errorf("unsupported trace sampler %q", name)
	}
}

func buildTraceExporterOptions(cfg TraceExporterConfig) ([]otlptracehttp.Option, bool, error) {
	protocol := otelconfig.OTLPProtocolOrDefault(cfg.OTLPProtocol)
	if !otelconfig.OTLPProtocolSupported(protocol) {
		return nil, false, fmt.Errorf("unsupported otlp protocol %q", cfg.OTLPProtocol)
	}

	options := make([]otlptracehttp.Option, 0, 4)
	configured := false

	endpoint := strings.TrimSpace(cfg.OTLPTracesEndpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(cfg.OTLPEndpoint)
	}
	if endpoint != "" {
		parsedOptions, err := parseOTLPEndpointOptions(endpoint)
		if err != nil {
			return nil, false, err
		}
		options = append(options, parsedOptions...)
		configured = true
	}
	if headers := strings.TrimSpace(cfg.OTLPHeaders); headers != "" {
		parsedHeaders, err := parseOTLPHeaders(headers)
		if err != nil {
			return nil, false, err
		}
		options = append(options, otlptracehttp.WithHeaders(parsedHeaders))
		configured = true
	}

	return options, configured, nil
}

func parseOTLPEndpointOptions(raw string) ([]otlptracehttp.Option, error) {
	if !strings.Contains(raw, "://") {
		parsedURL, err := url.Parse("//" + raw)
		if err != nil {
			return nil, fmt.Errorf("parse otlp endpoint %q: %w", raw, err)
		}
		return otlpEndpointOptions(raw, parsedURL, true)
	}

	parsedURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse otlp endpoint %q: %w", raw, err)
	}

	insecure := false
	switch strings.ToLower(parsedURL.Scheme) {
	case "http":
		insecure = true
	case "https":
	default:
		return nil, fmt.Errorf("parse otlp endpoint %q: unsupported scheme %q", raw, parsedURL.Scheme)
	}

	return otlpEndpointOptions(raw, parsedURL, insecure)
}

func otlpEndpointOptions(raw string, parsedURL *url.URL, insecure bool) ([]otlptracehttp.Option, error) {
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("parse otlp endpoint %q: empty host", raw)
	}
	if parsedURL.RawQuery != "" {
		return nil, fmt.Errorf("parse otlp endpoint %q: query is not supported", raw)
	}

	options := make([]otlptracehttp.Option, 0, 3)
	if insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}
	options = append(options, otlptracehttp.WithEndpoint(parsedURL.Host))
	path := strings.TrimSpace(parsedURL.EscapedPath())
	if path != "" && path != "/" {
		options = append(options, otlptracehttp.WithURLPath(path))
	}

	return options, nil
}

func parseOTLPHeaders(raw string) (map[string]string, error) {
	headers := make(map[string]string)

	pairs := strings.Split(raw, ",")
	for i, pair := range pairs {
		entry := strings.TrimSpace(pair)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d", i+1)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: empty header key", i+1)
		}
		if value == "" {
			if !canEchoOTLPHeaderKeyInError(key) {
				return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: empty header value", i+1)
			}
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d for header %q: empty header value", i+1, key)
		}
		headers[key] = value
	}

	if len(headers) == 0 {
		return nil, fmt.Errorf("parse otlp headers: no valid header pairs")
	}
	return headers, nil
}

func canEchoOTLPHeaderKeyInError(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		b := key[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' {
			continue
		}
		return false
	}
	return true
}
