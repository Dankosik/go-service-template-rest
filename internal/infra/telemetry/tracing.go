package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
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

	// The standard OpenTelemetry endpoint variables. The signal-specific one is
	// already a complete traces endpoint; the signal-agnostic one is a root that
	// OTLP defines the traces path relative to.
	otelExporterTracesEndpointEnv = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	otelExporterEndpointEnv       = "OTEL_EXPORTER_OTLP_ENDPOINT"

	otlpTracesPath = "/v1/traces"
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

type traceOTLPEndpoint struct {
	endpointURL string
}

// TraceExporterEndpoint is the resolved OTLP traces endpoint and the setting
// that supplied it.
type TraceExporterEndpoint struct {
	// URL is the full OTLP HTTP traces endpoint. Empty when nothing named one.
	URL string
	// Source is the configuration key or environment variable URL came from, so
	// an operator can tell a platform-injected endpoint from this service's own.
	// Empty when URL is empty.
	Source string
}

// Configured reports whether a trace exporter should be built.
func (e TraceExporterEndpoint) Configured() bool {
	return e.URL != ""
}

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

	res, err := newResource(ctx, cfg.ServiceName, cfg.ServiceVersion, cfg.DeploymentEnv)
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
// otlptracehttp applies ambient environment first and explicit options second,
// so an explicit option wins for everything it covers. WithEndpointURL covers
// the endpoint, the URL path, and the TLS scheme, which makes an injected
// ENDPOINT, TRACES_ENDPOINT, or INSECURE harmless. Credential and trust material
// is different: this service never sets client certificates or a root CA pool,
// and it sets headers only when observability.otel.exporter.otlp_headers is
// non-empty, so these variables would silently travel to the collector
// unverified.
//
// This applies only when observability.otel.exporter.otlp_endpoint named the
// destination. When the endpoint itself came from the platform's own variables,
// the platform owns the whole exporter configuration and its credentials belong
// to the collector it also named; rejecting them there would refuse the ordinary
// injected-collector deployment for no gain.
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
		endpoint, err := parseTraceOTLPEndpoint(raw)
		if err != nil {
			return TraceExporterEndpoint{}, err
		}
		return TraceExporterEndpoint{URL: endpoint.endpointURL, Source: TraceExporterConfigKey}, nil
	}
	if strings.TrimSpace(cfg.OTLPHeaders) != "" {
		return TraceExporterEndpoint{}, nil
	}

	if raw, ok := nonEmptyEnv(otelExporterTracesEndpointEnv); ok {
		endpoint, err := parseTraceOTLPEndpoint(raw)
		if err != nil {
			return TraceExporterEndpoint{}, fmt.Errorf("%s: %w", otelExporterTracesEndpointEnv, err)
		}
		return TraceExporterEndpoint{URL: endpoint.endpointURL, Source: otelExporterTracesEndpointEnv}, nil
	}
	if raw, ok := nonEmptyEnv(otelExporterEndpointEnv); ok {
		endpoint, err := parseTraceOTLPBaseEndpoint(raw)
		if err != nil {
			return TraceExporterEndpoint{}, fmt.Errorf("%s: %w", otelExporterEndpointEnv, err)
		}
		return TraceExporterEndpoint{URL: endpoint.endpointURL, Source: otelExporterEndpointEnv}, nil
	}

	return TraceExporterEndpoint{}, nil
}

func nonEmptyEnv(name string) (string, bool) {
	value := strings.TrimSpace(os.Getenv(name))
	return value, value != ""
}

// parseTraceOTLPEndpoint validates a complete OTLP traces endpoint. A missing
// path defaults to the OTLP HTTP traces path; any other path is used exactly as
// given, because a signal-specific endpoint names its own route.
func parseTraceOTLPEndpoint(raw string) (traceOTLPEndpoint, error) {
	parsedURL, err := parseTraceOTLPURL(raw)
	if err != nil {
		return traceOTLPEndpoint{}, err
	}

	if path := strings.TrimSpace(parsedURL.EscapedPath()); path == "" || path == "/" {
		parsedURL.Path = otlpTracesPath
		parsedURL.RawPath = ""
	}

	return traceOTLPEndpoint{
		endpointURL: parsedURL.String(),
	}, nil
}

// parseTraceOTLPBaseEndpoint validates a signal-agnostic OTLP root and appends
// the traces path, which is what OTLP defines OTEL_EXPORTER_OTLP_ENDPOINT to
// mean. A root carrying a path prefix keeps it, so a collector mounted under a
// sub-path still resolves.
func parseTraceOTLPBaseEndpoint(raw string) (traceOTLPEndpoint, error) {
	parsedURL, err := parseTraceOTLPURL(raw)
	if err != nil {
		return traceOTLPEndpoint{}, err
	}

	parsedURL.Path = pathpkg.Join("/", parsedURL.Path, otlpTracesPath)
	parsedURL.RawPath = ""

	return traceOTLPEndpoint{
		endpointURL: parsedURL.String(),
	}, nil
}

// parseTraceOTLPURL validates an exporter URL fail-closed: explicit http/https
// scheme, non-empty host, no userinfo/query/fragment. Errors never echo the raw
// value, which can carry a credential.
func parseTraceOTLPURL(raw string) (*url.URL, error) {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse otlp endpoint: invalid endpoint")
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("parse otlp endpoint: unsupported scheme")
	}
	if parsedURL.User != nil {
		return nil, fmt.Errorf("parse otlp endpoint: userinfo is not supported")
	}
	if strings.TrimSpace(parsedURL.Hostname()) == "" {
		return nil, fmt.Errorf("parse otlp endpoint: empty host")
	}
	if parsedURL.RawQuery != "" {
		return nil, fmt.Errorf("parse otlp endpoint: query is not supported")
	}
	if parsedURL.Fragment != "" {
		return nil, fmt.Errorf("parse otlp endpoint: fragment is not supported")
	}

	return parsedURL, nil
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
