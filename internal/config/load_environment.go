package config

import (
	"strings"
)

const namespacePrefix = "APP__"

func collectNamespaceValues(environ []string) (map[string]any, []string) {
	values := make(map[string]any)
	malformedKeys := make([]string, 0)

	for _, entry := range environ {
		envKey, envValue, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if !strings.HasPrefix(envKey, namespacePrefix) {
			continue
		}
		targetKey := namespaceEnvToKey(envKey)
		if targetKey == "" {
			malformedKeys = append(malformedKeys, envKey)
			continue
		}
		values[targetKey] = envValue
	}

	return values, malformedKeys
}

func namespaceEnvToKey(envKey string) string {
	trimmed := strings.TrimPrefix(envKey, namespacePrefix)
	if trimmed == "" {
		return ""
	}

	parts := strings.Split(trimmed, "__")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			return ""
		}
		segments = append(segments, strings.ToLower(p))
	}
	return strings.Join(segments, keyDelimiter)
}
