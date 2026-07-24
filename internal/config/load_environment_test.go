package config

import (
	"errors"
	"path/filepath"
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
	if cfg.HTTP.ReadinessTimeout != 4*time.Second {
		t.Fatalf("HTTP.ReadinessTimeout = %s, want 4s", cfg.HTTP.ReadinessTimeout)
	}
	if cfg.HTTP.ReadinessPropagationDelay != 15*time.Second {
		t.Fatalf("HTTP.ReadinessPropagationDelay = %s, want 15s", cfg.HTTP.ReadinessPropagationDelay)
	}
	if cfg.Postgres.Enabled {
		t.Fatalf("Postgres.Enabled = true, want false")
	}
	if cfg.Postgres.DSN != "" {
		t.Fatalf("Postgres.DSN = %q, want empty", cfg.Postgres.DSN)
	}
	if cfg.Observability.Metrics.Addr != "127.0.0.1:9090" {
		t.Fatalf("Observability.Metrics.Addr = %q, want 127.0.0.1:9090", cfg.Observability.Metrics.Addr)
	}
	if cfg.Observability.OTel.ServiceName == "" {
		t.Fatal("Observability.OTel.ServiceName is empty")
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

func TestEmptyNamespaceEnvOverridesRequiredDefault(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__ADDR", "")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for empty env override")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestResourceIdentityFieldsCannotBeEmpty(t *testing.T) {
	for _, tc := range []struct {
		name       string
		envKey     string
		wantDetail string
	}{
		{
			name:       "app version",
			envKey:     "APP__APP__VERSION",
			wantDetail: "app.version cannot be empty",
		},
		{
			name:       "otel service name",
			envKey:     "APP__OBSERVABILITY__OTEL__SERVICE_NAME",
			wantDetail: "observability.otel.service_name cannot be empty",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(tc.envKey, "")

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want validation error")
			}
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), tc.wantDetail) {
				t.Fatalf("error = %v, want %q", err, tc.wantDetail)
			}
		})
	}
}

func TestEmptyNamespaceEnvOverridesConfigFileValue(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
observability:
  otel:
    exporter:
      otlp_endpoint: "https://otel.example.com:4318"
`)
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT", "")

	cfg, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Observability.OTel.Exporter.OTLPEndpoint != "" {
		t.Fatalf("OTLPEndpoint = %q, want empty env override", cfg.Observability.OTel.Exporter.OTLPEndpoint)
	}
}

func TestNamespaceEnvPreservesRawDataBearingStrings(t *testing.T) {
	resetConfigEnv(t)

	postgresDSN := " postgres://user:pass@localhost:5432/app?sslmode=disable "
	headers := " authorization=Bearer token, x-trace= spaced value "
	t.Setenv("APP__POSTGRES__DSN", postgresDSN)
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_HEADERS", headers)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Postgres.DSN != postgresDSN {
		t.Fatalf("Postgres.DSN = %q, want exact env value %q", cfg.Postgres.DSN, postgresDSN)
	}
	if cfg.Observability.OTel.Exporter.OTLPHeaders != headers {
		t.Fatalf("OTLPHeaders = %q, want exact env value %q", cfg.Observability.OTel.Exporter.OTLPHeaders, headers)
	}
}

func TestNamespaceEnvTrimsSyntaxFields(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__APP__ENV", " local ")
	t.Setenv("APP__OBSERVABILITY__OTEL__SERVICE_NAME", " service ")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.App.Env != "local" {
		t.Fatalf("App.Env = %q, want local", cfg.App.Env)
	}
	if cfg.Observability.OTel.ServiceName != "service" {
		t.Fatalf("Observability.OTel.ServiceName = %q, want service", cfg.Observability.OTel.ServiceName)
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

func TestNamespaceEnvForConfigKey(t *testing.T) {
	t.Parallel()

	if got := namespaceEnvForConfigKey("app.env"); got != "APP__APP__ENV" {
		t.Fatalf("namespaceEnvForConfigKey(app.env) = %q, want APP__APP__ENV", got)
	}
}

//nolint:paralleltest // Reads env/.env.example through process-wide environment overrides.
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

	cfg1, _, err := LoadDetailed(opts)
	if err != nil {
		t.Fatalf("first LoadDetailed() error = %v", err)
	}
	cfg2, _, err := LoadDetailed(opts)
	if err != nil {
		t.Fatalf("second LoadDetailed() error = %v", err)
	}

	if cfg1 != cfg2 {
		t.Fatalf("config snapshots differ between repeated loads: first=%+v second=%+v", cfg1, cfg2)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
