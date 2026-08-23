package telemetry

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/http/httpguts"
)

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
