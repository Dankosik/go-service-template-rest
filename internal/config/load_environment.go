package config

import (
	"os"
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

func removeSectionScalarOverridesInPlace(values map[string]any) []string {
	return removeSectionScalarOverridesRecursive(values, "", knownConfigSections())
}

func removeSectionScalarOverridesRecursive(values map[string]any, prefix string, knownSections map[string]struct{}) []string {
	sectionScalarOverrideKeys := make([]string, 0)
	for key, value := range values {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + keyDelimiter + key
		}

		if _, ok := knownSections[fullKey]; ok && !configSectionValueIsMap(value) {
			sectionScalarOverrideKeys = append(sectionScalarOverrideKeys, fullKey)
			delete(values, key)
			continue
		}
		nested, ok := value.(map[string]any)
		if !ok {
			continue
		}
		sectionScalarOverrideKeys = append(sectionScalarOverrideKeys, removeSectionScalarOverridesRecursive(nested, fullKey, knownSections)...)
	}
	return sectionScalarOverrideKeys
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

func lookupNonEmptyEnv(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}
