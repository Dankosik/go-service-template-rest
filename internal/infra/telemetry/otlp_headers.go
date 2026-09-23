package telemetry

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/http/httpguts"
)

// otlpExporterHeaders applies the checks both signals make before building an
// exporter for a resolved endpoint: ambient credential and trust material is
// refused first, then the configured headers are parsed. The result is empty
// when no headers are configured.
//
// The refusal applies only when this service named the destination. When the
// platform's own variables named it, the platform owns the whole exporter
// configuration and its credentials belong to the collector it also named.
func otlpExporterHeaders(endpoint ExporterEndpoint, envConflicts []string, raw string) (map[string]string, error) {
	if endpoint.ConfiguredByService {
		if err := rejectConflictingAmbientEnv(envConflicts); err != nil {
			return nil, err
		}
	}
	headers := strings.TrimSpace(raw)
	if headers == "" {
		return map[string]string{}, nil
	}
	return parseOTLPHeaders(headers)
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
		if !httpguts.ValidHeaderFieldName(key) {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: invalid header key", i+1)
		}
		if value == "" {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: empty header value", i+1)
		}
		if !httpguts.ValidHeaderFieldValue(value) {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: invalid header value", i+1)
		}
		key = http.CanonicalHeaderKey(key)
		if _, duplicate := headers[key]; duplicate {
			return nil, fmt.Errorf("parse otlp headers: malformed entry at position %d: duplicate header key", i+1)
		}
		headers[key] = value
	}

	if len(headers) == 0 {
		return nil, errors.New("parse otlp headers: no valid header pairs")
	}
	return headers, nil
}
