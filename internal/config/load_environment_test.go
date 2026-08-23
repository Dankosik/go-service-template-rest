package config

import (
	"errors"
	// profile:object-storage:start
	"path/filepath"
	// profile:object-storage:end
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
	if cfg.HTTP.GracePeriod != 45*time.Second {
		t.Fatalf("HTTP.GracePeriod = %s, want 45s", cfg.HTTP.GracePeriod)
	}
	if cfg.HTTP.ShutdownTimeout != 25*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 25s", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.HTTP.ReadinessTimeout != 4*time.Second {
		t.Fatalf("HTTP.ReadinessTimeout = %s, want 4s", cfg.HTTP.ReadinessTimeout)
	}
	if cfg.HTTP.ReadinessPropagationDelay != 15*time.Second {
		t.Fatalf("HTTP.ReadinessPropagationDelay = %s, want 15s", cfg.HTTP.ReadinessPropagationDelay)
	}
	// profile:database-postgres:start
	if cfg.Postgres.Enabled {
		t.Fatal("Postgres.Enabled = true, want false")
	}
	if cfg.Postgres.DSN != "" {
		t.Fatalf("Postgres.DSN = %q, want empty", cfg.Postgres.DSN)
	}
	// profile:database-postgres:end
	if cfg.Observability.Metrics.Addr != ":9090" {
		t.Fatalf("Observability.Metrics.Addr = %q, want :9090", cfg.Observability.Metrics.Addr)
	}
	if cfg.Observability.Pprof.Enabled {
		t.Fatal("Observability.Pprof.Enabled = true, want false by default")
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
	lastOverlayPath := writeTempConfig(t, `
http:
  addr: ":8083"
`)

	cfg, _, err := LoadDetailed(LoadOptions{
		ConfigPath:     basePath,
		ConfigOverlays: []string{overlayPath, lastOverlayPath},
	})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.Addr != ":8083" {
		t.Fatalf("HTTP.Addr = %q, want last overlay :8083", cfg.HTTP.Addr)
	}

	t.Setenv("APP__HTTP__ADDR", ":8084")
	cfg, _, err = LoadDetailed(LoadOptions{
		ConfigPath:     basePath,
		ConfigOverlays: []string{overlayPath, lastOverlayPath},
	})
	if err != nil {
		t.Fatalf("LoadDetailed() with namespace override error = %v", err)
	}
	if cfg.HTTP.Addr != ":8084" {
		t.Fatalf("HTTP.Addr = %q, want namespace override :8084", cfg.HTTP.Addr)
	}
}

func TestMalformedNamespaceEnvRejectsWithoutDisclosingValue(t *testing.T) {
	for _, envKey := range []string{"APP__", "APP____HTTP__ADDR", "APP__HTTP____ADDR"} {
		t.Run(envKey, func(t *testing.T) {
			resetConfigEnv(t)
			const canary = "malformed-env-secret-canary"
			t.Setenv(envKey, canary)

			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrUnknownKey) || !strings.Contains(err.Error(), envKey) {
				t.Fatalf("LoadDetailed() error = %v, want malformed environment key rejection", err)
			}
			if strings.Contains(err.Error(), canary) {
				t.Fatalf("LoadDetailed() error disclosed raw environment value: %v", err)
			}
		})
	}
}

func TestEmptyNamespaceEnvOverridesRequiredDefault(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__ADDR", "")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() expected validation error for empty env override")
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

	headers := " authorization=Bearer token, x-trace= spaced value "
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_HEADERS", headers)
	// profile:database-postgres:start
	postgresDSN := " postgres://user:pass@localhost:5432/app?sslmode=disable "
	t.Setenv("APP__POSTGRES__DSN", postgresDSN)
	// profile:database-postgres:end
	// profile:outbound-auth-oauth2-client-credentials:start
	clientID := " client:id "
	clientSecret := " client secret "
	t.Setenv("APP__OUTBOUND_AUTH__CLIENT_ID", clientID)
	t.Setenv("APP__OUTBOUND_AUTH__CLIENT_SECRET", clientSecret)
	// profile:outbound-auth-oauth2-client-credentials:end

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	// profile:database-postgres:start
	if cfg.Postgres.DSN != postgresDSN {
		t.Fatalf("Postgres.DSN = %q, want exact env value %q", cfg.Postgres.DSN, postgresDSN)
	}
	// profile:database-postgres:end
	if cfg.Observability.OTel.Exporter.OTLPHeaders != headers {
		t.Fatalf("OTLPHeaders = %q, want exact env value %q", cfg.Observability.OTel.Exporter.OTLPHeaders, headers)
	}
	// profile:outbound-auth-oauth2-client-credentials:start
	if cfg.OutboundAuth.ClientID != clientID {
		t.Fatalf("OutboundAuth.ClientID = %q, want exact env value %q", cfg.OutboundAuth.ClientID, clientID)
	}
	if cfg.OutboundAuth.ClientSecret != clientSecret {
		t.Fatalf("OutboundAuth.ClientSecret = %q, want exact env value %q", cfg.OutboundAuth.ClientSecret, clientSecret)
	}
	// profile:outbound-auth-oauth2-client-credentials:end
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

// profile:object-storage:start
//
//nolint:paralleltest // Reads env/.env.example through process-wide environment overrides.
func TestEnvExampleIsFailClosedUntilObjectStorageIsConfigured(t *testing.T) {
	resetConfigEnv(t)

	for key, value := range readEnvExample(t, filepath.Join("..", "..", "env", ".env.example")) {
		t.Setenv(key, value)
	}
	// profile:outbound-auth-oauth2-client-credentials:start
	setOutboundAuthTestEnv(t)
	// profile:outbound-auth-oauth2-client-credentials:end

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("LoadDetailed() error = %v, want incomplete object-storage validation", err)
	}
	if !strings.Contains(err.Error(), "object_storage.provider") {
		t.Fatalf("LoadDetailed() error = %v, want object storage placeholder rejection", err)
	}
}

// profile:object-storage:end

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
