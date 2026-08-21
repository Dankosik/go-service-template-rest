package config

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

// profile:object-storage:start
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

// profile:object-storage:end

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
	// profile:authn-oidc-jwt:end
	// profile:outbound-auth-oauth2-client-credentials:start
	setOutboundAuthTestEnv(t)
	// profile:outbound-auth-oauth2-client-credentials:end
	// profile:object-storage:start
	setObjectStorageTestEnv(t)
	// profile:object-storage:end
}

// profile:outbound-auth-oauth2-client-credentials:start
//
//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func setOutboundAuthTestEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"APP__OUTBOUND_AUTH__TOKEN_URL":     "https://auth.example.com/oauth/token",
		"APP__OUTBOUND_AUTH__CLIENT_ID":     "test-client",
		"APP__OUTBOUND_AUTH__CLIENT_SECRET": "test-client-secret",
		"APP__OUTBOUND_AUTH__SCOPES":        "payments.read payments.write",
		"APP__OUTBOUND_AUTH__RESOURCE":      "https://payments.example.com",
	} {
		t.Setenv(key, value)
	}
}

// profile:outbound-auth-oauth2-client-credentials:end
// profile:object-storage:start
//
//nolint:paralleltest // This test mutates process-global environment or working directory.
func setObjectStorageTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP__OBJECT_STORAGE__PROVIDER", "amazon_s3")
	t.Setenv("APP__OBJECT_STORAGE__REGION", "us-east-1")
	t.Setenv("APP__OBJECT_STORAGE__BUCKET", "examplebucket")
	t.Setenv("APP__OBJECT_STORAGE__EXPECTED_BUCKET_OWNER", "123456789012")
	t.Setenv("APP__OBJECT_STORAGE__CREDENTIAL_SOURCE", "aws_default")
	t.Setenv("APP__OBJECT_STORAGE__MAX_OBJECT_BYTES", "10485760")
}

// profile:object-storage:end

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

	return slices.Sorted(maps.Keys(keySet))
}
