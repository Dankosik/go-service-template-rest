package config

import (
	"strings"
)

const namespacePrefix = "APP__"

func collectNamespaceValues(environ []string) map[string]any {
	values := make(map[string]any)

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
			continue
		}
		values[targetKey] = envValue
	}

	return values
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

func namespaceEnvForConfigKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	return namespacePrefix + strings.ToUpper(strings.ReplaceAll(trimmed, keyDelimiter, "__"))
}
