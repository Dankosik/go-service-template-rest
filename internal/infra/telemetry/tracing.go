package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
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
	OTLPEndpoint string
	OTLPHeaders  string
}

// TraceExporterTarget describes the explicit application-configured OTLP trace exporter target.
type TraceExporterTarget struct {
	Configured bool
	Target     string
	Scheme     string
}

type traceOTLPEndpoint struct {
	endpointURL string
	target      string
	scheme      string
}

var otelSetupMu sync.Mutex

var unsupportedOTLPProxyEnvVars = []string{
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"NO_PROXY",
	"http_proxy",
	"https_proxy",
	"no_proxy",
}

func SetupTracing(ctx context.Context, cfg TracingConfig) (func(context.Context) error, error) {
	sampler, err := buildTraceSampler(cfg.TracesSampler, cfg.TracesSamplerArg)
	if err != nil {
		return nil, err
	}

	res, err := newResource(ctx, cfg.ServiceName, cfg.ServiceVersion, cfg.DeploymentEnv)
	if err != nil {
		return nil, err
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
		if err := rejectUnsupportedAmbientTraceExporterEnv(); err != nil {
			return nil, err
		}
		exporter, err := otlptracehttp.New(ctx, exporterOptions...)
		if err != nil {
			return nil, fmt.Errorf("create otlp trace exporter: %w", err)
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	}

	otelSetupMu.Lock()
	defer otelSetupMu.Unlock()

	provider := newTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return provider.Shutdown, nil
}

func rejectUnsupportedAmbientTraceExporterEnv() error {
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "OTEL_EXPORTER_OTLP_") && strings.TrimSpace(value) != "" {
			return fmt.Errorf("unsupported ambient otel exporter environment: OTEL_EXPORTER_OTLP*")
		}
	}

	for _, name := range unsupportedOTLPProxyEnvVars {
		if os.Getenv(name) != "" {
			return fmt.Errorf("unsupported ambient otlp proxy environment: proxy variables are not supported for otlp exporter")
		}
	}
	return nil
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
	options := make([]otlptracehttp.Option, 0, 2)
	endpoint, configured, err := traceExporterOTLPEndpoint(cfg)
	if err != nil {
		return nil, false, err
	}
	if !configured {
		return options, false, nil
	}

	options = append(options, otlptracehttp.WithEndpointURL(endpoint.endpointURL))
	if headers := strings.TrimSpace(cfg.OTLPHeaders); headers != "" {
		parsedHeaders, err := parseOTLPHeaders(headers)
		if err != nil {
			return nil, false, err
		}
		options = append(options, otlptracehttp.WithHeaders(parsedHeaders))
	}

	return options, true, nil
}

// DescribeTraceExporterTarget returns the explicit OTLP trace exporter network target, if configured.
func DescribeTraceExporterTarget(cfg TraceExporterConfig) (TraceExporterTarget, error) {
	endpoint, configured, err := traceExporterOTLPEndpoint(cfg)
	if err != nil {
		return TraceExporterTarget{}, err
	}
	if !configured {
		return TraceExporterTarget{}, nil
	}

	return TraceExporterTarget{
		Configured: true,
		Target:     endpoint.target,
		Scheme:     endpoint.scheme,
	}, nil
}

func traceExporterOTLPEndpoint(cfg TraceExporterConfig) (traceOTLPEndpoint, bool, error) {
	raw := strings.TrimSpace(cfg.OTLPEndpoint)
	if raw == "" {
		return traceOTLPEndpoint{}, false, nil
	}

	endpoint, err := parseTraceOTLPEndpoint(raw)
	if err != nil {
		return traceOTLPEndpoint{}, false, err
	}
	return endpoint, true, nil
}

// parseTraceOTLPEndpoint validates the configured exporter URL fail-closed:
// explicit http/https scheme, non-empty host, no userinfo/query/fragment.
// A missing path defaults to the OTLP HTTP traces path /v1/traces.
func parseTraceOTLPEndpoint(raw string) (traceOTLPEndpoint, error) {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return traceOTLPEndpoint{}, fmt.Errorf("parse otlp endpoint: invalid endpoint")
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return traceOTLPEndpoint{}, fmt.Errorf("parse otlp endpoint: unsupported scheme")
	}
	if parsedURL.User != nil {
		return traceOTLPEndpoint{}, fmt.Errorf("parse otlp endpoint: userinfo is not supported")
	}
	if strings.TrimSpace(parsedURL.Hostname()) == "" {
		return traceOTLPEndpoint{}, fmt.Errorf("parse otlp endpoint: empty host")
	}
	if parsedURL.RawQuery != "" {
		return traceOTLPEndpoint{}, fmt.Errorf("parse otlp endpoint: query is not supported")
	}
	if parsedURL.Fragment != "" {
		return traceOTLPEndpoint{}, fmt.Errorf("parse otlp endpoint: fragment is not supported")
	}

	if path := strings.TrimSpace(parsedURL.EscapedPath()); path == "" || path == "/" {
		parsedURL.Path = "/v1/traces"
	}

	return traceOTLPEndpoint{
		endpointURL: parsedURL.String(),
		target:      parsedURL.Host,
		scheme:      scheme,
	}, nil
}

func parseOTLPHeaders(raw string) (map[string]string, error) {
	headers := make(map[string]string)

	pairs := strings.Split(raw, ",")
	for i, pair := range pairs {
		entry := strings.TrimSpace(pair)
		if entry == "" {
			continue
		}
		rawKey, rawValue, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d", i+1)
		}
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if key == "" {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: empty header key", i+1)
		}
		if !validOTLPHeaderKey(key) {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: invalid header key", i+1)
		}
		if value == "" {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: empty header value", i+1)
		}
		headers[key] = value
	}

	if len(headers) == 0 {
		return nil, fmt.Errorf("parse otlp headers: no valid header pairs")
	}
	return headers, nil
}

func validOTLPHeaderKey(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		b := key[i]
		if (b >= 'a' && b <= 'z') ||
			(b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') ||
			strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b)) {
			continue
		}
		return false
	}
	return true
}
