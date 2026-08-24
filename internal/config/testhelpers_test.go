package config

import (
	"os"
	"path/filepath"
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

	clearConfigEnv(t)
	t.Setenv("APP__APP__ENV", "local")
	// profile:authn-bearer:start
	t.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	t.Setenv("APP__AUTHN__AUDIENCE", "https://api.example.com")
	// profile:authn-oidc-introspection:start
	t.Setenv("APP__AUTHN__INTROSPECTION_ENDPOINT", "https://idp.example.com/oauth/introspect")
	t.Setenv("APP__AUTHN__INTROSPECTION_TARGET_CLASS", "external-https")
	t.Setenv("APP__AUTHN__INTROSPECTION_CLIENT_ID", "rs-client")
	t.Setenv("APP__AUTHN__INTROSPECTION_CLIENT_SECRET", "rs-secret")
	// profile:authn-oidc-introspection:end
	// profile:authn-bearer:end
	// profile:object-storage:start
	setObjectStorageTestEnv(t)
	// profile:object-storage:end
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, namespacePrefix) {
			continue
		}
		t.Setenv(key, value)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
}

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
