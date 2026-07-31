package config

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}

func readEnvExample(t *testing.T, path string) map[string]string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	values := make(map[string]string)
	for lineNumber, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("%s:%d is not KEY=VALUE", path, lineNumber+1)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			t.Fatalf("%s:%d has an empty env key", path, lineNumber+1)
		}
		if !strings.HasPrefix(key, namespacePrefix) {
			continue
		}
		values[key] = strings.TrimSpace(value)
	}

	return values
}

func resetConfigEnv(t *testing.T) {
	t.Helper()

	previousValues := make(map[string]string)
	for _, key := range configEnvResetKeys(t) {
		if value, ok := os.LookupEnv(key); ok {
			previousValues[key] = value
			t.Setenv(key, value)
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
	t.Cleanup(func() {
		for _, key := range configEnvResetKeys(t) {
			if _, ok := previousValues[key]; !ok {
				_ = os.Unsetenv(key)
			}
		}
	})
	t.Setenv("APP__APP__ENV", "local")
	// profile:authn-oidc-jwt:start
	t.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	t.Setenv("APP__AUTHN__AUDIENCE", "https://api.example.com")
	t.Setenv("APP__AUTHN__TRUSTED_PROXY_CIDRS", "127.0.0.0/8,::1/128")
	// profile:authn-oidc-jwt:end
}

func configEnvResetKeys(t *testing.T) []string {
	t.Helper()

	knownKeys := configLeafKeysFromType(t, reflect.TypeFor[Config](), "")
	knownSections := knownConfigSections()
	keySet := make(map[string]struct{}, len(knownKeys)+len(knownSections))
	for _, key := range knownKeys {
		keySet[namespaceEnvForConfigKey(key)] = struct{}{}
	}
	for key := range knownSections {
		keySet[namespaceEnvForConfigKey(key)] = struct{}{}
	}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
