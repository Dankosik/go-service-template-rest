package config

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	resetConfigEnv(t)

	cfg, report, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}

	if cfg.App.Env != "local" {
		t.Fatalf("App.Env = %q, want local", cfg.App.Env)
	}
	if cfg.App.Version != "dev" {
		t.Fatalf("App.Version = %q, want dev", cfg.App.Version)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ShutdownTimeout != 30*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 30s", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.HTTP.ReadinessTimeout != 3*time.Second {
		t.Fatalf("HTTP.ReadinessTimeout = %s, want 3s", cfg.HTTP.ReadinessTimeout)
	}
	if cfg.HTTP.ReadinessPropagationDelay != 15*time.Second {
		t.Fatalf("HTTP.ReadinessPropagationDelay = %s, want 15s", cfg.HTTP.ReadinessPropagationDelay)
	}
	if cfg.Redis.Mode != "cache" {
		t.Fatalf("Redis.Mode = %q, want cache", cfg.Redis.Mode)
	}
	if cfg.Redis.AllowStoreMode {
		t.Fatalf("Redis.AllowStoreMode = true, want false")
	}
	if cfg.Postgres.Enabled {
		t.Fatalf("Postgres.Enabled = true, want false")
	}
	if cfg.Postgres.DSN != "" {
		t.Fatalf("Postgres.DSN = %q, want empty", cfg.Postgres.DSN)
	}
	if cfg.Observability.OTel.ServiceName != "service" {
		t.Fatalf("Observability.OTel.ServiceName = %q, want service", cfg.Observability.OTel.ServiceName)
	}
	if cfg.Observability.OTel.TracesSampler != "parentbased_traceidratio" {
		t.Fatalf("Observability.OTel.TracesSampler = %q, want parentbased_traceidratio", cfg.Observability.OTel.TracesSampler)
	}
	if report.LoadDuration <= 0 {
		t.Fatalf("LoadDuration = %s, want > 0", report.LoadDuration)
	}
	if report.ValidateDuration <= 0 {
		t.Fatalf("ValidateDuration = %s, want > 0", report.ValidateDuration)
	}
}

func TestPrecedenceNamespaceWinsOverFileAndOverlay(t *testing.T) {
	resetConfigEnv(t)

	basePath := writeTempConfig(t, `
http:
  addr: ":8081"
`)
	overlayPath := writeTempConfig(t, `
http:
  addr: ":8082"
`)

	t.Setenv("APP__HTTP__ADDR", ":8083")

	cfg, _, err := LoadDetailed(LoadOptions{
		ConfigPath:     basePath,
		ConfigOverlays: []string{overlayPath},
	})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}

	if cfg.HTTP.Addr != ":8083" {
		t.Fatalf("HTTP.Addr = %q, want :8083", cfg.HTTP.Addr)
	}
}

func TestFlatEnvKeysAreIgnored(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("HTTP_ADDR", ":9090")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want default :8080", cfg.HTTP.Addr)
	}
}

func TestEnvExampleLoadsThroughConfigLoader(t *testing.T) {
	resetConfigEnv(t)

	for key, value := range readEnvExample(t, filepath.Join("..", "..", "env", ".env.example")) {
		t.Setenv(key, value)
	}

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() with env/.env.example values error = %v", err)
	}
	if cfg.HTTP.ShutdownTimeout != 30*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 30s from env/.env.example", cfg.HTTP.ShutdownTimeout)
	}
}

func TestTST001PrecedenceDeterministicSnapshotAcrossRepeatedLoads(t *testing.T) {
	resetConfigEnv(t)

	basePath := writeTempConfig(t, `
http:
  addr: ":8081"
`)
	overlayPath := writeTempConfig(t, `
http:
  addr: ":8082"
`)

	t.Setenv("APP__HTTP__ADDR", ":8083")

	opts := LoadOptions{
		ConfigPath:     basePath,
		ConfigOverlays: []string{overlayPath},
	}

	cfg1, report1, err := LoadDetailed(opts)
	if err != nil {
		t.Fatalf("first LoadDetailed() error = %v", err)
	}
	cfg2, report2, err := LoadDetailed(opts)
	if err != nil {
		t.Fatalf("second LoadDetailed() error = %v", err)
	}

	if cfg1 != cfg2 {
		t.Fatalf("config snapshots differ between repeated loads: first=%+v second=%+v", cfg1, cfg2)
	}
	if !reflect.DeepEqual(report1.UnknownKeyWarnings, report2.UnknownKeyWarnings) {
		t.Fatalf("UnknownKeyWarnings differs between repeated loads: first=%v second=%v", report1.UnknownKeyWarnings, report2.UnknownKeyWarnings)
	}
}

func TestStrictUnknownKeyRejects(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)

	_, _, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     true,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected strict unknown key error")
	}
	if !errors.Is(err, ErrStrictUnknownKey) {
		t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
	}
	if got := ErrorType(err); got != "strict_unknown_key" {
		t.Fatalf("ErrorType(error) = %q, want strict_unknown_key", got)
	}
}

func TestPermissiveUnknownKeyAllows(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)

	_, report, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     false,
	})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if !containsString(report.UnknownKeyWarnings, "unknown.field") {
		t.Fatalf("UnknownKeyWarnings = %v, want unknown.field", report.UnknownKeyWarnings)
	}
}

func TestRemovedObservabilityKeysRejectInStrictMode(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
observability:
  metrics:
    enabled: true
    path: /internal/metrics
  grafana:
    enabled: true
    cloud_otlp_endpoint: "https://example.invalid"
`)

	_, _, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     true,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected strict unknown key error")
	}
	if !errors.Is(err, ErrStrictUnknownKey) {
		t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
	}
}

func TestRequiredIfEnabledPostgresSecretPolicy(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__ENABLED", "true")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestTST003RequiredIfEnabledContracts(t *testing.T) {
	t.Run("postgres_enabled_without_dsn_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__POSTGRES__ENABLED", "true")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected secret policy error")
		}
		if !errors.Is(err, ErrSecretPolicy) {
			t.Fatalf("error = %v, want ErrSecretPolicy", err)
		}
	})

	t.Run("postgres_enabled_with_dsn_allowed", func(t *testing.T) {
		resetConfigEnv(t)
		dsn := "postgres://app:app@localhost:5432/app?sslmode=disable"
		t.Setenv("APP__POSTGRES__ENABLED", "true")
		t.Setenv("APP__POSTGRES__DSN", dsn)

		cfg, _, err := LoadDetailed(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadDetailed() error = %v", err)
		}
		if !cfg.Postgres.Enabled {
			t.Fatalf("Postgres.Enabled = false, want true")
		}
		if cfg.Postgres.DSN != dsn {
			t.Fatalf("Postgres.DSN = %q, want %q", cfg.Postgres.DSN, dsn)
		}
	})

	t.Run("mongo_enabled_without_uri_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__MONGO__ENABLED", "true")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected secret policy error")
		}
		if !errors.Is(err, ErrSecretPolicy) {
			t.Fatalf("error = %v, want ErrSecretPolicy", err)
		}
	})

	t.Run("mongo_enabled_without_database_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__MONGO__URI", "mongodb://localhost:27017")
		configPath := writeTempConfig(t, `
mongo:
  enabled: true
  database: ""
`)

		_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validation error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
	})

	t.Run("mongo_disabled_without_uri_allowed", func(t *testing.T) {
		resetConfigEnv(t)

		cfg, _, err := LoadDetailed(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadDetailed() error = %v", err)
		}
		if cfg.Mongo.Enabled {
			t.Fatalf("Mongo.Enabled = true, want false")
		}
		if cfg.Mongo.URI != "" {
			t.Fatalf("Mongo.URI = %q, want empty", cfg.Mongo.URI)
		}
	})

	t.Run("redis_enabled_with_empty_addr_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		configPath := writeTempConfig(t, `
redis:
  enabled: true
  addr: ""
`)

		_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validation error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
	})
}

func TestRedisStoreGuardRejectsWithoutAllowFlag(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", "127.0.0.1:6379")
	t.Setenv("APP__REDIS__MODE", "store")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "false")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected redis store guard rejection")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "redis.allow_store_mode=true") {
		t.Fatalf("error = %v, want allow_store_mode hint", err)
	}
}

func TestRedisStoreGuardRequiresZeroStaleWindow(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", "127.0.0.1:6379")
	t.Setenv("APP__REDIS__MODE", "store")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "true")
	t.Setenv("APP__REDIS__STALE_WINDOW", "1s")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected stale-window validation error")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestRedisStoreGuardAllowsConfiguredModeForV1GuardPath(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", "127.0.0.1:6379")
	t.Setenv("APP__REDIS__MODE", "store")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "true")
	t.Setenv("APP__REDIS__STALE_WINDOW", "0s")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Redis.Mode != "store" {
		t.Fatalf("Redis.Mode = %q, want store", cfg.Redis.Mode)
	}
	if !cfg.Redis.AllowStoreMode {
		t.Fatalf("Redis.AllowStoreMode = false, want true")
	}
	if cfg.Redis.StaleWindow != 0 {
		t.Fatalf("Redis.StaleWindow = %s, want 0", cfg.Redis.StaleWindow)
	}
}

func TestNonLocalRejectsSymlinkConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	target := writeTempConfig(t, `
http:
  addr: ":8080"
`)

	tempDir := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", tempDir)

	linkPath := filepath.Join(tempDir, "config-link.yaml")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: linkPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error for non-local symlink config")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestTST005NonLocalRejectsWorldWritableConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
`)
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", filepath.Dir(configPath))

	if err := os.Chmod(configPath, 0o666); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error for world-writable config")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestInvalidDurationParseError(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__READ_TIMEOUT", "oops")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected parse error")
	}
	if !errors.Is(err, ErrParse) {
		t.Fatalf("error = %v, want ErrParse", err)
	}
}

func TestTST005MalformedYAMLReturnsParseError(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
broken: [
`)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected parse error for malformed YAML")
	}
	if !errors.Is(err, ErrParse) {
		t.Fatalf("error = %v, want ErrParse", err)
	}
	if got := ErrorType(err); got != "parse" {
		t.Fatalf("ErrorType(error) = %q, want parse", got)
	}
}

func TestConfigFileWithoutEnvironmentHintFailsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "")

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
`)
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", filepath.Dir(configPath))

	if err := os.Chmod(configPath, 0o666); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected fail-closed hardening error without explicit local env hint")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalRejectsConfigOutsideAllowedRoots(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	allowedRoot := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", allowedRoot)

	otherRoot := t.TempDir()
	path := filepath.Join(otherRoot, "config.yaml")
	if err := os.WriteFile(path, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatalf("LoadDetailed() expected allowed-root policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalDefaultRootsDoNotAllowRepositoryConfigDir(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", "")

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(previousWD, "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(previousWD); chdirErr != nil {
			t.Fatalf("os.Chdir() restore error = %v", chdirErr)
		}
	})

	repoConfigDir := filepath.Join(repoRoot, "env", "config")
	if err := os.MkdirAll(repoConfigDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(repoConfigDir, "nonlocal-default-root-test.yaml")
	content := "app:\n  env: prod\nhttp:\n  addr: \":8080\"\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(configPath)
	})

	_, _, err = LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected allowed-root policy rejection for repository config path in non-local mode")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalRejectsSymlinkPathComponents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	allowedRoot := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", allowedRoot)

	realRoot := t.TempDir()
	configPath := filepath.Join(realRoot, "config.yaml")
	if err := os.WriteFile(configPath, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	linkedDir := filepath.Join(allowedRoot, "linked")
	if err := os.Symlink(realRoot, linkedDir); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	pathViaSymlink := filepath.Join(linkedDir, "config.yaml")
	_, _, err := LoadDetailed(LoadOptions{ConfigPath: pathViaSymlink})
	if err == nil {
		t.Fatalf("LoadDetailed() expected symlink-path rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

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
		t.Fatalf("LoadDetailed() expected secret source policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestLocalRejectsSecretLikeValuesInConfigFile(t *testing.T) {
	resetConfigEnv(t)

	path := writeTempConfig(t, `
postgres:
  enabled: true
  dsn: "postgres://app:secret@localhost:5432/app?sslmode=disable"
`)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret source policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestConfigFileAllowsEmptySecretLikePlaceholders(t *testing.T) {
	resetConfigEnv(t)

	path := writeTempConfig(t, `
postgres:
  dsn: ""
mongo:
  uri: ""
redis:
  password: ""
observability:
  otel:
    exporter:
      otlp_headers: ""
`)

	if _, _, err := LoadDetailed(LoadOptions{ConfigPath: path}); err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil for empty secret-like placeholders", err)
	}
}

func TestConfigFileRejectsCommonFutureSecretLikeKeys(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantKey string
	}{
		{
			name: "client secret",
			content: `
oauth:
  client_secret: "secret"
`,
			wantKey: "oauth.client_secret",
		},
		{
			name: "jwt secret",
			content: `
security:
  jwt_secret: "secret"
`,
			wantKey: "security.jwt_secret",
		},
		{
			name: "api key",
			content: `
provider:
  api_key: "secret"
`,
			wantKey: "provider.api_key",
		},
		{
			name: "private key",
			content: `
tls:
  private_key: "secret"
`,
			wantKey: "tls.private_key",
		},
		{
			name: "top level token",
			content: `
token: "secret"
`,
			wantKey: "token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	keys := []string{
		"http.addr",
		"redis.key_prefix",
		"feature_flags.redis_readiness_probe",
		"metadata.public_key",
	}

	for _, key := range keys {
		if isSecretLikeConfigKey(key) {
			t.Fatalf("isSecretLikeConfigKey(%q) = true, want false", key)
		}
	}
}

func TestParseErrorDoesNotLeakRawValue(t *testing.T) {
	resetConfigEnv(t)

	secretLikeValue := "supersecret-token-value"
	t.Setenv("APP__HTTP__READ_TIMEOUT", secretLikeValue)

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected parse error")
	}
	if !errors.Is(err, ErrParse) {
		t.Fatalf("error = %v, want ErrParse", err)
	}
	if strings.Contains(err.Error(), secretLikeValue) {
		t.Fatalf("error unexpectedly contains raw secret-like value: %v", err)
	}
}

func TestFlatPostgresDSNIsIgnored(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("POSTGRES_DSN", "postgres://app:app@localhost:5432/app?sslmode=disable")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Postgres.Enabled {
		t.Fatalf("Postgres.Enabled = true, want false when only flat key is set")
	}
	if cfg.Postgres.DSN != "" {
		t.Fatalf("Postgres.DSN = %q, want empty when only flat key is set", cfg.Postgres.DSN)
	}
}

func TestErrorTypeMapping(t *testing.T) {
	if got := ErrorType(ErrStrictUnknownKey); got != "strict_unknown_key" {
		t.Fatalf("ErrorType(strict) = %q", got)
	}
	if got := ErrorType(ErrSecretPolicy); got != "secret_policy" {
		t.Fatalf("ErrorType(secret_policy) = %q", got)
	}
	if got := ErrorType(ErrValidate); got != "validate" {
		t.Fatalf("ErrorType(validate) = %q", got)
	}
	if got := ErrorType(ErrParse); got != "parse" {
		t.Fatalf("ErrorType(parse) = %q", got)
	}
	if got := ErrorType(ErrDependencyInit); got != "dependency_init" {
		t.Fatalf("ErrorType(dependency_init) = %q", got)
	}
	if got := ErrorType(ErrLoad); got != "load" {
		t.Fatalf("ErrorType(load) = %q", got)
	}
}

func TestLoadDetailedWithContextCanceled(t *testing.T) {
	resetConfigEnv(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := LoadDetailedWithContext(ctx, LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailedWithContext() expected context cancellation error")
	}
	if !errors.Is(err, ErrLoad) {
		t.Fatalf("error = %v, want ErrLoad", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestLoadDetailedFailedStageReporting(t *testing.T) {
	t.Run("parse_stage", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__HTTP__READ_TIMEOUT", "oops")

		_, report, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected parse error")
		}
		if !errors.Is(err, ErrParse) {
			t.Fatalf("error = %v, want ErrParse", err)
		}
		if report.FailedStage != StageParse {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageParse)
		}
		if report.FailedStageDuration <= 0 {
			t.Fatalf("FailedStageDuration = %s, want > 0", report.FailedStageDuration)
		}
	})

	t.Run("validate_stage", func(t *testing.T) {
		resetConfigEnv(t)
		configPath := writeTempConfig(t, `
unknown:
  field: value
`)

		_, report, err := LoadDetailed(LoadOptions{
			ConfigPath: configPath,
			Strict:     true,
		})
		if err == nil {
			t.Fatalf("LoadDetailed() expected strict unknown key error")
		}
		if !errors.Is(err, ErrStrictUnknownKey) {
			t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
		}
		if report.FailedStage != StageValidate {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageValidate)
		}
		if report.FailedStageDuration <= 0 {
			t.Fatalf("FailedStageDuration = %s, want > 0", report.FailedStageDuration)
		}
	})

	t.Run("load_file_stage", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__APP__ENV", "prod")
		t.Setenv("APP_CONFIG_ALLOWED_ROOTS", t.TempDir())

		_, report, err := LoadDetailed(LoadOptions{ConfigPath: "/nonexistent/config.yaml"})
		if err == nil {
			t.Fatalf("LoadDetailed() expected load error")
		}
		if !errors.Is(err, ErrLoad) && !errors.Is(err, ErrSecretPolicy) {
			t.Fatalf("error = %v, want ErrLoad or ErrSecretPolicy", err)
		}
		if report.FailedStage != StageLoadFile {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageLoadFile)
		}
		if report.FailedStageDuration <= 0 {
			t.Fatalf("FailedStageDuration = %s, want > 0", report.FailedStageDuration)
		}
	})
}

func TestOTLPExporterValuesFromNamespaceEnv(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT", "https://otel.example.com:4318")
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_TRACES_ENDPOINT", "https://otel.example.com:4318/v1/traces")
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_HEADERS", "authorization=Bearer token")
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_PROTOCOL", "http/protobuf")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Observability.OTel.Exporter.OTLPEndpoint != "https://otel.example.com:4318" {
		t.Fatalf("OTLPEndpoint = %q, want %q", cfg.Observability.OTel.Exporter.OTLPEndpoint, "https://otel.example.com:4318")
	}
	if cfg.Observability.OTel.Exporter.OTLPTracesEndpoint != "https://otel.example.com:4318/v1/traces" {
		t.Fatalf("OTLPTracesEndpoint = %q, want %q", cfg.Observability.OTel.Exporter.OTLPTracesEndpoint, "https://otel.example.com:4318/v1/traces")
	}
	if cfg.Observability.OTel.Exporter.OTLPHeaders != "authorization=Bearer token" {
		t.Fatalf("OTLPHeaders = %q, want %q", cfg.Observability.OTel.Exporter.OTLPHeaders, "authorization=Bearer token")
	}
	if cfg.Observability.OTel.Exporter.OTLPProtocol != "http/protobuf" {
		t.Fatalf("OTLPProtocol = %q, want %q", cfg.Observability.OTel.Exporter.OTLPProtocol, "http/protobuf")
	}
}

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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func resetConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range configEnvResetKeys() {
		t.Setenv(key, "")
	}
	t.Setenv("APP__APP__ENV", "local")
}

func configEnvResetKeys() []string {
	knownKeys := knownConfigKeys()
	keySet := make(map[string]struct{}, len(knownKeys)+1)
	for key := range knownKeys {
		keySet[namespaceEnvForConfigKey(key)] = struct{}{}
	}
	keySet[allowedConfigRootsEnvVar] = struct{}{}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func namespaceEnvForConfigKey(key string) string {
	return namespacePrefix + strings.ToUpper(strings.ReplaceAll(key, keyDelimiter, "__"))
}

func TestKnownConfigKeysMatchSnapshotTagsAndDefaults(t *testing.T) {
	defaultKeys := sortedStringSetKeys(defaultValues())
	knownKeys := sortedStringSetKeys(knownConfigKeys())
	if !reflect.DeepEqual(knownKeys, defaultKeys) {
		t.Fatalf("knownConfigKeys() = %v, want default keys %v", knownKeys, defaultKeys)
	}

	tagKeys := configLeafKeysFromType(t, reflect.TypeOf(Config{}), "")
	sort.Strings(tagKeys)
	if !reflect.DeepEqual(tagKeys, defaultKeys) {
		t.Fatalf("Config koanf leaf keys = %v, want default keys %v", tagKeys, defaultKeys)
	}
}

func configLeafKeysFromType(t *testing.T, typ reflect.Type, prefix string) []string {
	t.Helper()

	if typ.Kind() != reflect.Struct {
		t.Fatalf("configLeafKeysFromType(%s) called with non-struct type", typ)
	}

	keys := make([]string, 0)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := strings.TrimSpace(field.Tag.Get("koanf"))
		if tag == "" || tag == "-" {
			t.Fatalf("%s.%s must declare a concrete koanf tag", typ.Name(), field.Name)
		}

		key := tag
		if prefix != "" {
			key = prefix + keyDelimiter + tag
		}

		if hasKoanfTaggedFields(field.Type) {
			keys = append(keys, configLeafKeysFromType(t, field.Type, key)...)
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

func hasKoanfTaggedFields(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < typ.NumField(); i++ {
		if strings.TrimSpace(typ.Field(i).Tag.Get("koanf")) != "" {
			return true
		}
	}
	return false
}

func sortedStringSetKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestPostgresDurationBounds(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__CONNECT_TIMEOUT", "50ms")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for connect timeout")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestShutdownTimeoutCanBeTunedWhenDrainBudgetIsValid(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "45s")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "20s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "10s")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil for tuned shutdown timeout", err)
	}
	if cfg.HTTP.ShutdownTimeout != 45*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 45s", cfg.HTTP.ShutdownTimeout)
	}
}

func TestShutdownTimeoutMustStayWithinRange(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "500ms")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "0s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "100ms")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for shutdown timeout range")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.shutdown_timeout must be in range") {
		t.Fatalf("error = %v, want shutdown timeout range policy", err)
	}
}

func TestHTTPShutdownBudgetMustLeaveWriteDrainTime(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "25s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "10s")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for write timeout beyond drain budget")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.write_timeout must be <= effective drain budget") {
		t.Fatalf("error = %v, want explicit drain budget policy", err)
	}
}

func TestReadinessTimeoutMustCoverAggregateEnabledProbeBudget(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__READINESS_TIMEOUT", "6s")
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://user:pass@localhost:5432/app?sslmode=disable")
	t.Setenv("APP__POSTGRES__HEALTHCHECK_TIMEOUT", "3s")
	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__MODE", "store")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "true")
	t.Setenv("APP__REDIS__DIAL_TIMEOUT", "2s")
	t.Setenv("APP__MONGO__ENABLED", "true")
	t.Setenv("APP__MONGO__URI", "mongodb://localhost:27017/app")
	t.Setenv("APP__MONGO__CONNECT_TIMEOUT", "2s")
	t.Setenv("APP__FEATURE_FLAGS__MONGO_READINESS_PROBE", "true")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for aggregate readiness timeout")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "aggregate sequential readiness probe budget") {
		t.Fatalf("error = %v, want aggregate readiness dependency budget policy", err)
	}
	if !strings.Contains(err.Error(), "postgres.healthcheck_timeout + redis.dial_timeout + mongo.connect_timeout") {
		t.Fatalf("error = %v, want enabled readiness probe names", err)
	}
}

func TestPostgresDSNMustBeParseable(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://%zz")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for unparseable postgres dsn")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestRedisEnabledRequiresHostPortAddress(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", "redis-without-port")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for redis addr without port")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestMongoURIMustBeParseable(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__MONGO__ENABLED", "true")
	t.Setenv("APP__MONGO__URI", "mongo://bad-uri")
	t.Setenv("APP__MONGO__DATABASE", "app")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for unparseable mongo uri")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestMongoDisabledAllowsInvalidURI(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__MONGO__ENABLED", "false")
	t.Setenv("APP__MONGO__URI", "mongo://bad-uri")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil when mongo is disabled", err)
	}
	if cfg.Mongo.Enabled {
		t.Fatalf("Mongo.Enabled = true, want false")
	}
}

func TestMongoProbeAddress(t *testing.T) {
	t.Run("mongodb scheme", func(t *testing.T) {
		address, err := MongoProbeAddress("mongodb://user:pass@localhost:27017/app?replicaSet=rs0")
		if err != nil {
			t.Fatalf("MongoProbeAddress() error = %v", err)
		}
		if address != "localhost:27017" {
			t.Fatalf("MongoProbeAddress() = %q, want localhost:27017", address)
		}
	})

	t.Run("mongodb srv scheme defaults port", func(t *testing.T) {
		address, err := MongoProbeAddress("mongodb+srv://cluster.example.com/app")
		if err != nil {
			t.Fatalf("MongoProbeAddress() error = %v", err)
		}
		if address != "cluster.example.com:27017" {
			t.Fatalf("MongoProbeAddress() = %q, want cluster.example.com:27017", address)
		}
	})
}

func TestReadDurationParsesDefaultDurations(t *testing.T) {
	resetConfigEnv(t)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.ReadTimeout != 5*time.Second {
		t.Fatalf("HTTP.ReadTimeout = %s, want 5s", cfg.HTTP.ReadTimeout)
	}
	if cfg.Postgres.ConnMaxLifetime != 30*time.Minute {
		t.Fatalf("Postgres.ConnMaxLifetime = %s, want 30m", cfg.Postgres.ConnMaxLifetime)
	}
}

func TestParseInt(t *testing.T) {
	t.Run("supports mixed numeric inputs", func(t *testing.T) {
		value, err := parseInt("42")
		if err != nil {
			t.Fatalf("parseInt(string) error = %v", err)
		}
		if value != 42 {
			t.Fatalf("parseInt(string) = %d, want 42", value)
		}

		value, err = parseInt(float64(7))
		if err != nil {
			t.Fatalf("parseInt(float64) error = %v", err)
		}
		if value != 7 {
			t.Fatalf("parseInt(float64) = %d, want 7", value)
		}
	})

	t.Run("rejects non integer floats", func(t *testing.T) {
		if _, err := parseInt(1.25); err == nil {
			t.Fatalf("parseInt() expected non-integer error")
		}
	})

	t.Run("rejects overflow from unsigned values", func(t *testing.T) {
		overflow := uint(math.MaxInt) + 1
		if _, err := parseInt(overflow); err == nil {
			t.Fatalf("parseInt() expected overflow error for uint value")
		}
		if _, err := parseInt(uint64(math.MaxUint64)); err == nil {
			t.Fatalf("parseInt() expected overflow error for uint64 value")
		}
	})
}

func TestParseInt64(t *testing.T) {
	t.Run("supports mixed numeric inputs", func(t *testing.T) {
		value, err := parseInt64("922")
		if err != nil {
			t.Fatalf("parseInt64(string) error = %v", err)
		}
		if value != 922 {
			t.Fatalf("parseInt64(string) = %d, want 922", value)
		}

		value, err = parseInt64(uint32(11))
		if err != nil {
			t.Fatalf("parseInt64(uint32) error = %v", err)
		}
		if value != 11 {
			t.Fatalf("parseInt64(uint32) = %d, want 11", value)
		}
	})

	t.Run("rejects non integer floats", func(t *testing.T) {
		if _, err := parseInt64(float64(2.5)); err == nil {
			t.Fatalf("parseInt64() expected non-integer error")
		}
	})

	t.Run("rejects overflow from unsigned values", func(t *testing.T) {
		if _, err := parseInt64(uint64(math.MaxUint64)); err == nil {
			t.Fatalf("parseInt64() expected overflow error")
		}
	})
}

func TestParseBool(t *testing.T) {
	value, err := parseBool("true")
	if err != nil {
		t.Fatalf("parseBool(true) error = %v", err)
	}
	if !value {
		t.Fatalf("parseBool(true) = false, want true")
	}

	if _, err := parseBool(1); err == nil {
		t.Fatalf("parseBool() expected unsupported type error")
	}
}

func TestValidateRangeHelpers(t *testing.T) {
	t.Run("int range is inclusive", func(t *testing.T) {
		if err := validateIntRange("redis.pool_size", 1, 1, 100); err != nil {
			t.Fatalf("validateIntRange(min) error = %v", err)
		}
		if err := validateIntRange("redis.pool_size", 100, 1, 100); err != nil {
			t.Fatalf("validateIntRange(max) error = %v", err)
		}
	})

	t.Run("int range out of bounds returns ErrValidate", func(t *testing.T) {
		err := validateIntRange("redis.pool_size", 101, 1, 100)
		if err == nil {
			t.Fatalf("validateIntRange() expected error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "redis.pool_size") {
			t.Fatalf("error = %v, want field name in message", err)
		}
	})

	t.Run("duration range is inclusive", func(t *testing.T) {
		if err := validateDurationRange("http.read_timeout", time.Second, time.Second, 10*time.Second); err != nil {
			t.Fatalf("validateDurationRange(min) error = %v", err)
		}
		if err := validateDurationRange("http.read_timeout", 10*time.Second, time.Second, 10*time.Second); err != nil {
			t.Fatalf("validateDurationRange(max) error = %v", err)
		}
	})

	t.Run("duration range out of bounds returns ErrValidate", func(t *testing.T) {
		err := validateDurationRange("http.read_timeout", 11*time.Second, time.Second, 10*time.Second)
		if err == nil {
			t.Fatalf("validateDurationRange() expected error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "http.read_timeout") {
			t.Fatalf("error = %v, want field name in message", err)
		}
	})
}
