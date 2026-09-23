package telemetry

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
)

// Where an OTLP endpoint can come from — this service's configuration keys and
// the standard OpenTelemetry variables — and the path each signal appends to a
// root. A signal-specific setting is already a complete endpoint for that
// signal; a signal-agnostic one is a root that OTLP defines each signal's path
// relative to.
const (
	// SharedOTLPExporterConfigKey names the service-owned OTLP endpoint shared
	// by traces and metrics when it contains a collector root.
	SharedOTLPExporterConfigKey = "observability.otel.exporter.otlp_endpoint"
	// MetricExporterConfigKey names the service-owned metrics-only endpoint.
	MetricExporterConfigKey = "observability.otel.exporter.otlp_metrics_endpoint"

	otelExporterEndpointEnv        = "OTEL_EXPORTER_OTLP_ENDPOINT"
	otelExporterTracesEndpointEnv  = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	otelExporterMetricsEndpointEnv = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"

	otlpTracesPath  = "/v1/traces"
	otlpMetricsPath = "/v1/metrics"
)

// ExporterEndpoint is a resolved OTLP endpoint and the setting that supplied it.
type ExporterEndpoint struct {
	// URL is the full OTLP HTTP endpoint for one signal. Empty when nothing
	// named one.
	URL string
	// Source is the configuration key or environment variable URL came from, so
	// an operator can tell a platform-injected endpoint from this service's own.
	// Empty when URL is empty.
	Source string
	// ConfiguredByService distinguishes this service's configuration from an
	// endpoint supplied by the platform environment.
	ConfiguredByService bool
}

// Configured reports whether an exporter should be built.
func (e ExporterEndpoint) Configured() bool {
	return e.URL != ""
}

// fromConfig reports whether this service, rather than the platform, named the
// destination. It decides whether ambient credential and trust material is a
// conflict: material this service cannot verify must not travel to an endpoint
// this service chose.
func (e ExporterEndpoint) fromConfig() bool {
	return e.ConfiguredByService
}

// parseSignalOTLPEndpoint validates a complete endpoint for one signal. A missing
// path defaults to that signal's OTLP HTTP path; any other path is used exactly
// as given, because a signal-specific endpoint names its own route.
func parseSignalOTLPEndpoint(raw, signalPath string) (string, error) {
	parsedURL, err := parseOTLPURL(raw)
	if err != nil {
		return "", err
	}

	if hasNoOTLPPath(parsedURL) {
		parsedURL.Path = signalPath
		parsedURL.RawPath = ""
	}

	return parsedURL.String(), nil
}

// parseBaseOTLPEndpoint validates a signal-agnostic root and appends the signal's
// path, which is what OTLP defines OTEL_EXPORTER_OTLP_ENDPOINT to mean. A root
// carrying a path prefix keeps it, so a collector mounted under a sub-path still
// resolves.
func parseBaseOTLPEndpoint(raw, signalPath string) (string, error) {
	parsedURL, err := parseOTLPURL(raw)
	if err != nil {
		return "", err
	}

	parsedURL.Path = pathpkg.Join("/", parsedURL.Path, signalPath)
	parsedURL.RawPath = ""

	return parsedURL.String(), nil
}

// otlpCandidate is one step of the ordered search for a signal's exporter
// endpoint: a value and the setting that supplied it.
type otlpCandidate struct {
	// source names the configuration key or environment variable, and is what a
	// resolved [ExporterEndpoint] reports as its Source.
	source string
	// raw is what that setting holds. Blank skips this step.
	raw string
	// base marks a signal-agnostic collector root, which gets the signal's OTLP
	// path appended. The zero value is a complete endpoint for one signal.
	base bool
	// configuredByService marks values from the service's typed configuration.
	configuredByService bool
}

// resolveOTLPEndpoint walks the settings this service owns and then the ambient
// ones, returning the first that names an endpoint. Both signals resolve through
// it, so neither can answer this differently from the other.
//
// The two lists are separate rather than one because headers are a collector
// credential, and a configured one pins the destination to a setting this
// service owns: once the owned settings are exhausted the endpoint would come
// from ambient environment, and sending the service's own credentials somewhere
// it never named is what this must not create. Without configured headers the
// ambient variables are honored, because they are what a platform collector
// injects and ignoring them would leave a service reporting healthy while
// exporting nothing.
//
// Only an ambient parse failure names its source. An operator did not write that
// value in this service's configuration and has to be told which injected
// variable to fix, while a rejected owned setting is already reported by the
// configuration key that carries it.
func resolveOTLPEndpoint(signalPath, headers string, owned, ambient []otlpCandidate) (ExporterEndpoint, error) {
	for _, candidate := range owned {
		endpoint, ok, err := candidate.resolve(signalPath)
		if err != nil {
			return ExporterEndpoint{}, err
		}
		if ok {
			return endpoint, nil
		}
	}
	if strings.TrimSpace(headers) != "" {
		return ExporterEndpoint{}, nil
	}

	for _, candidate := range ambient {
		endpoint, ok, err := candidate.resolve(signalPath)
		if err != nil {
			return ExporterEndpoint{}, fmt.Errorf("%s: %w", candidate.source, err)
		}
		if ok {
			return endpoint, nil
		}
	}

	return ExporterEndpoint{}, nil
}

// resolve reports this step's endpoint, or ok=false when its setting holds
// nothing.
func (c otlpCandidate) resolve(signalPath string) (ExporterEndpoint, bool, error) {
	raw := strings.TrimSpace(c.raw)
	if raw == "" {
		return ExporterEndpoint{}, false, nil
	}
	parse := parseSignalOTLPEndpoint
	if c.base {
		parse = parseBaseOTLPEndpoint
	}
	endpointURL, err := parse(raw, signalPath)
	if err != nil {
		return ExporterEndpoint{}, false, err
	}
	return ExporterEndpoint{
		URL: endpointURL, Source: c.source,
		ConfiguredByService: c.configuredByService,
	}, true, nil
}

// ambientOTLPCandidates are the standard OpenTelemetry endpoint variables in the
// order OTLP resolves them: the signal's own variable, then the root each signal
// derives its path from. Only signalEnv differs per signal.
func ambientOTLPCandidates(signalEnv string) []otlpCandidate {
	return []otlpCandidate{
		{source: signalEnv, raw: os.Getenv(signalEnv)},
		{source: otelExporterEndpointEnv, raw: os.Getenv(otelExporterEndpointEnv), base: true},
	}
}

// namesOTLPRoot reports whether raw is a bare collector root rather than an
// endpoint for one signal.
//
// It is what lets one configured observability.otel.exporter.otlp_endpoint serve
// both signals: a root has a path OTLP would append to, while a value that
// already names /v1/traces is a traces endpoint and says nothing about where
// metrics go.
func namesOTLPRoot(raw string) bool {
	parsedURL, err := parseOTLPURL(raw)
	if err != nil {
		return false
	}
	return hasNoOTLPPath(parsedURL)
}

// hasNoOTLPPath reports whether u carries no path beyond the root.
func hasNoOTLPPath(u *url.URL) bool {
	path := strings.TrimSpace(u.EscapedPath())
	return path == "" || path == "/"
}

// parseOTLPURL validates an exporter URL fail-closed: explicit http/https scheme,
// non-empty host, no userinfo/query/fragment. Errors never echo the raw value,
// which can carry a credential.
func parseOTLPURL(raw string) (*url.URL, error) {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("parse otlp endpoint: invalid endpoint")
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, errors.New("parse otlp endpoint: unsupported scheme")
	}
	if parsedURL.User != nil {
		return nil, errors.New("parse otlp endpoint: userinfo is not supported")
	}
	if strings.TrimSpace(parsedURL.Hostname()) == "" {
		return nil, errors.New("parse otlp endpoint: empty host")
	}
	if parsedURL.RawQuery != "" {
		return nil, errors.New("parse otlp endpoint: query is not supported")
	}
	if parsedURL.Fragment != "" {
		return nil, errors.New("parse otlp endpoint: fragment is not supported")
	}

	return parsedURL, nil
}
