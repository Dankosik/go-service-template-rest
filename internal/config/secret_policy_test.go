package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNonLocalRejectsSecretLikeValuesInConfigFile(t *testing.T) {

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	allowedRoot := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", allowedRoot)

	path := filepath.Join(allowedRoot, "config.yaml")
	content := `
app:
  env: prod
postgres:
  enabled: true
  dsn: "postgres://app:secret@localhost:5432/app?sslmode=disable"
`
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatal("LoadDetailed() expected secret source policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestLocalRejectsSecretLikeValuesInConfigFile(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	path := writeTempConfig(t, `
postgres:
  enabled: true
  dsn: "postgres://app:secret@localhost:5432/app?sslmode=disable"
`)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatal("LoadDetailed() expected secret source policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestConfigFileAllowsEmptySecretLikePlaceholders(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	path := writeTempConfig(t, `
# profile:database-postgres:start
postgres:
  dsn: ""
# profile:database-postgres:end
observability:
  otel:
    exporter:
      otlp_headers: ""
`)

	if _, _, err := LoadDetailed(LoadOptions{ConfigPath: path}); err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil for empty secret-like placeholders", err)
	}
}

//nolint:paralleltest // Subtests reset process-wide configuration environment.
//nolint:paralleltest // This test mutates process-global environment or working directory.

// profile:outbound-auth-oauth2-client-credentials:start
//
//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestConfigFileRejectsCommonFutureSecretLikeKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantKey string
	}{
		{name: "client secret", content: "oauth:\n  client_secret: \"secret\"\n", wantKey: "oauth.client_secret"},
		{name: "jwt secret", content: "security:\n  jwt_secret: \"secret\"\n", wantKey: "security.jwt_secret"},
		{name: "api key", content: "provider:\n  api_key: \"secret\"\n", wantKey: "provider.api_key"},
		{name: "private key", content: "tls:\n  private_key: \"secret\"\n", wantKey: "tls.private_key"},
		{name: "top level token", content: "token: \"secret\"\n", wantKey: "token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resetConfigEnv(t)

			path := writeTempConfig(t, tt.content)
			_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
			if err == nil {
				t.Fatalf("LoadDetailed() expected secret policy rejection for %s", tt.wantKey)
			}
			if !errors.Is(err, ErrSecretPolicy) {
				t.Fatalf("error = %v, want ErrSecretPolicy", err)
			}
			if !strings.Contains(err.Error(), tt.wantKey) {
				t.Fatalf("error = %v, want rejected key %q", err, tt.wantKey)
			}
		})
	}
}

func TestSecretLikeConfigKeyPolicyAllowsNonSecretShapes(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"http.addr", "metadata.public_key"} {
		if isSecretLikeConfigKey(key) {
			t.Fatalf("isSecretLikeConfigKey(%q) = true, want false", key)
		}
	}
}

func TestOutboundAuthSecretSourcePolicy(t *testing.T) {

	resetConfigEnv(t)
	const canary = "outbound-client-secret-canary"
	path := writeTempConfig(t, "outbound_auth:\n  client_secret: "+canary+"\n")
	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("LoadDetailed() error = %v, want ErrSecretPolicy", err)
	}
	if strings.Contains(err.Error(), canary) {
		t.Fatalf("LoadDetailed() error disclosed client secret: %v", err)
	}

	resetConfigEnv(t)
	path = writeTempConfig(t, "outbound_auth:\n  client_secret: \"\"\n")
	if _, _, err := LoadDetailed(LoadOptions{ConfigPath: path}); err != nil {
		t.Fatalf("LoadDetailed() with empty placeholder error = %v", err)
	}

	resetConfigEnv(t)
	t.Setenv("OAUTH_CLIENT_SECRET", canary)
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() with hostile ambient variable error = %v", err)
	}
	if cfg.OutboundAuth.ClientSecret == canary {
		t.Fatal("LoadDetailed() accepted an ambient OAuth secret convention")
	}
	if got := defaultValues()["outbound_auth.client_secret"]; got != "" {
		t.Fatalf("outbound_auth.client_secret default = %q, want empty", got)
	}
}

// profile:outbound-auth-oauth2-client-credentials:end
// profile:webhooks-durable:start
//
//nolint:paralleltest // This test mutates process-global environment or working directory.
//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestWebhookSecretSourcePolicy(t *testing.T) {

	resetConfigEnv(t)
	const canary = "webhook-secret-canary"
	path := writeTempConfig(t, "webhooks:\n  static_secrets: "+canary+"\n")
	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("LoadDetailed() error = %v, want ErrSecretPolicy", err)
	}
	if strings.Contains(err.Error(), canary) {
		t.Fatalf("LoadDetailed() error disclosed webhook secret: %v", err)
	}

	resetConfigEnv(t)
	path = writeTempConfig(t, "webhooks:\n  static_secrets: \"\"\n")
	if _, _, err := LoadDetailed(LoadOptions{ConfigPath: path}); err != nil {
		t.Fatalf("LoadDetailed() with empty placeholder error = %v", err)
	}

	resetConfigEnv(t)
	t.Setenv("APP__WEBHOOKS__STATIC_SECRETS", canary)
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() with environment webhook secret error = %v", err)
	}
	if cfg.Webhooks.StaticSecrets != canary {
		t.Fatal("LoadDetailed() did not accept environment webhook secret")
	}
}

// profile:webhooks-durable:end
