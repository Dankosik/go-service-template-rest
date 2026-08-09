package telemetry

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
)

// The standard OpenTelemetry endpoint variables. A signal-specific one is
// already a complete endpoint for that signal; the signal-agnostic one is a root
// that OTLP defines each signal's path relative to.
const (
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
}

// Configured reports whether an exporter should be built.
func (e ExporterEndpoint) Configured() bool {
	return e.URL != ""
}

type otlpEndpoint struct {
	endpointURL string
}

// parseSignalOTLPEndpoint validates a complete endpoint for one signal. A missing
// path defaults to that signal's OTLP HTTP path; any other path is used exactly
// as given, because a signal-specific endpoint names its own route.
func parseSignalOTLPEndpoint(raw, signalPath string) (otlpEndpoint, error) {
	parsedURL, err := parseOTLPURL(raw)
	if err != nil {
		return otlpEndpoint{}, err
	}

	if path := strings.TrimSpace(parsedURL.EscapedPath()); path == "" || path == "/" {
		parsedURL.Path = signalPath
		parsedURL.RawPath = ""
	}

	return otlpEndpoint{endpointURL: parsedURL.String()}, nil
}

// parseBaseOTLPEndpoint validates a signal-agnostic root and appends the signal's
// path, which is what OTLP defines OTEL_EXPORTER_OTLP_ENDPOINT to mean. A root
// carrying a path prefix keeps it, so a collector mounted under a sub-path still
// resolves.
func parseBaseOTLPEndpoint(raw, signalPath string) (otlpEndpoint, error) {
	parsedURL, err := parseOTLPURL(raw)
	if err != nil {
		return otlpEndpoint{}, err
	}

	parsedURL.Path = pathpkg.Join("/", parsedURL.Path, signalPath)
	parsedURL.RawPath = ""

	return otlpEndpoint{endpointURL: parsedURL.String()}, nil
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
	path := strings.TrimSpace(parsedURL.EscapedPath())
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
		return nil, errors.New("parse otlp headers: no valid header pairs")
	}
	return headers, nil
}

func validOTLPHeaderKey(key string) bool {
	if key == "" {
		return false
	}
	for i := range len(key) {
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

func nonEmptyEnv(name string) (string, bool) {
	value := strings.TrimSpace(os.Getenv(name))
	return value, value != ""
}
