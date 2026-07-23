package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/knadh/koanf/v2"
)

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
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

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
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

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
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
	if !slices.Contains(report.UnknownKeyWarnings, "unknown.field") {
		t.Fatalf("UnknownKeyWarnings = %v, want unknown.field", report.UnknownKeyWarnings)
	}
}

func TestPermissiveUnknownKeyWarningsPreservedOnValidationError(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)
	t.Setenv("APP__HTTP__ADDR", "")

	_, report, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     false,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !slices.Contains(report.UnknownKeyWarnings, "unknown.field") {
		t.Fatalf("UnknownKeyWarnings = %v, want unknown.field", report.UnknownKeyWarnings)
	}
}

func TestStrictUnknownKeyRejectsScalarSectionKeys(t *testing.T) {
	for _, tc := range []struct {
		name    string
		envKey  string
		wantKey string
	}{
		{name: "root section", envKey: "APP__HTTP", wantKey: "http"},
		{name: "nested section", envKey: "APP__OBSERVABILITY__OTEL", wantKey: "observability.otel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(tc.envKey, "oops")

			_, report, err := LoadDetailed(LoadOptions{Strict: true})
			if err == nil {
				t.Fatalf("LoadDetailed() expected strict unknown key error")
			}
			if !errors.Is(err, ErrStrictUnknownKey) {
				t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Fatalf("error = %v, want unknown section key %q", err, tc.wantKey)
			}
			if report.FailedStage != StageValidate {
				t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageValidate)
			}
		})
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestPermissiveUnknownKeyWarnsAndIgnoresScalarSectionKey(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
http: oops
`)

	cfg, report, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if !slices.Contains(report.UnknownKeyWarnings, "http") {
		t.Fatalf("UnknownKeyWarnings = %v, want http", report.UnknownKeyWarnings)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want default :8080 after ignored section scalar", cfg.HTTP.Addr)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
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
}

func TestConfigReadinessProbeRequiredPolicyHelpers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		cfg          Config
		wantPostgres bool
	}{
		{
			name: "disabled dependencies ignore readiness flags",
			cfg: Config{
				FeatureFlags: FeatureFlagsConfig{
					PostgresReadinessProbe: true,
				},
			},
		},
		{
			name: "enabled postgres without readiness flag",
			cfg: Config{
				Postgres: PostgresConfig{Enabled: true},
			},
		},
		{
			name: "enabled postgres with readiness flag",
			cfg: Config{
				Postgres: PostgresConfig{Enabled: true},
				FeatureFlags: FeatureFlagsConfig{
					PostgresReadinessProbe: true,
				},
			},
			wantPostgres: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.cfg.PostgresReadinessProbeRequired(); got != tc.wantPostgres {
				t.Fatalf("PostgresReadinessProbeRequired() = %v, want %v", got, tc.wantPostgres)
			}
		})
	}
}

func TestConfigReadinessProbeBudgetsUseRequiredRuntimeProbes(t *testing.T) {
	t.Parallel()

	cfg := Config{
		HTTP: HTTPConfig{
			ReadinessTimeout: 10 * time.Second,
		},
		Postgres: PostgresConfig{
			Enabled:            true,
			HealthcheckTimeout: 2 * time.Second,
		},
		FeatureFlags: FeatureFlagsConfig{
			PostgresReadinessProbe: true,
		},
	}

	budgets := cfg.ReadinessProbeBudgets()
	want := []ReadinessProbeBudget{
		{ConfigKey: "postgres.healthcheck_timeout", Budget: 2 * time.Second},
	}
	if len(budgets) != len(want) {
		t.Fatalf("ReadinessProbeBudgets() len = %d, want %d", len(budgets), len(want))
	}
	for i := range want {
		if budgets[i] != want[i] {
			t.Fatalf("ReadinessProbeBudgets()[%d] = %+v, want %+v", i, budgets[i], want[i])
		}
	}

	budgets[0].Budget = time.Nanosecond
	if got := cfg.ReadinessProbeBudgets()[0].Budget; got != 2*time.Second {
		t.Fatalf("ReadinessProbeBudgets() returned aliased slice; first budget = %s, want 2s", got)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestLocalAllowsSymlinkConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)

	target := writeTempConfig(t, `
http:
  addr: ":18080"
`)

	linkPath := filepath.Join(t.TempDir(), "config-link.yaml")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	cfg, _, err := LoadDetailed(LoadOptions{ConfigPath: linkPath})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.Addr != ":18080" {
		t.Fatalf("HTTP.Addr = %q, want :18080", cfg.HTTP.Addr)
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
	if !strings.Contains(err.Error(), "invalid duration syntax") {
		t.Fatalf("error = %v, want sanitized duration parse detail", err)
	}
}

func TestParseErrorsExposeSanitizedDetail(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envValue   string
		wantDetail string
	}{
		{
			name:       "duration missing unit",
			envKey:     "APP__HTTP__READ_TIMEOUT",
			envValue:   "150",
			wantDetail: "missing duration unit",
		},
		{
			name:       "int format",
			envKey:     "APP__HTTP__MAX_HEADER_BYTES",
			envValue:   "many",
			wantDetail: "invalid integer format",
		},
		{
			name:       "float finite check",
			envKey:     "APP__OBSERVABILITY__OTEL__TRACES_SAMPLER_ARG",
			envValue:   "NaN",
			wantDetail: "non-finite numeric value",
		},
		{
			name:       "bool format",
			envKey:     "APP__FEATURE_FLAGS__POSTGRES_READINESS_PROBE",
			envValue:   "maybe",
			wantDetail: "invalid boolean format",
		},
		{
			name:       "log level",
			envKey:     "APP__LOG__LEVEL",
			envValue:   "secret-level",
			wantDetail: "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(tt.envKey, tt.envValue)

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want parse error")
			}
			if !errors.Is(err, ErrParse) {
				t.Fatalf("error = %v, want ErrParse", err)
			}
			if !strings.Contains(err.Error(), tt.wantDetail) {
				t.Fatalf("error = %v, want sanitized detail %q", err, tt.wantDetail)
			}
			if strings.Contains(err.Error(), tt.envValue) {
				t.Fatalf("error = %v, leaked raw value %q", err, tt.envValue)
			}
		})
	}
}

func TestNonFiniteSamplerArgReturnsParseError(t *testing.T) {
	for _, value := range []string{"NaN", "+Inf"} {
		t.Run(value, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__OBSERVABILITY__OTEL__TRACES_SAMPLER_ARG", value)

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want parse error")
			}
			if !errors.Is(err, ErrParse) {
				t.Fatalf("error = %v, want ErrParse", err)
			}
			if got := ErrorType(err); got != "parse" {
				t.Fatalf("ErrorType(error) = %q, want parse", got)
			}
		})
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestMalformedYAMLReturnsParseError(t *testing.T) {
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

func TestLoadConfigFileRejectsWhitespaceOnlyPath(t *testing.T) {
	t.Parallel()

	err := loadConfigFile(context.Background(), koanf.New(keyDelimiter), " \t\n ", configFilePolicyLocal)
	if err == nil {
		t.Fatal("loadConfigFile() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrLoad) {
		t.Fatalf("error = %v, want ErrLoad", err)
	}
	if !strings.Contains(err.Error(), "empty config path") {
		t.Fatalf("error = %v, want empty config path detail", err)
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

	repoRoot := filepath.Join(t.TempDir(), "repo")
	repoConfigDir := filepath.Join(repoRoot, "env", "config")
	if err := os.MkdirAll(repoConfigDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(repoConfigDir, "nonlocal-default-root-test.yaml")
	content := "app:\n  env: prod\nhttp:\n  addr: \":8080\"\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected allowed-root policy rejection for repository config path in non-local mode")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalAllowedRootsRejectsUnsafeInputs(t *testing.T) {
	testCases := []struct {
		name             string
		allowedRoots     string
		wantErrorMessage string
	}{
		{
			name:             "relative root",
			allowedRoots:     "relative-config-root",
			wantErrorMessage: "APP_CONFIG_ALLOWED_ROOTS entries must be absolute paths",
		},
		{
			name:             "empty value uses default roots",
			allowedRoots:     "",
			wantErrorMessage: "outside allowed roots",
		},
		{
			name:             "delimiter only value produces no roots",
			allowedRoots:     ",;;",
			wantErrorMessage: "outside allowed roots",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__APP__ENV", "prod")
			t.Setenv("APP_CONFIG_ALLOWED_ROOTS", tc.allowedRoots)

			configPath := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(configPath, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
				t.Fatalf("os.WriteFile() error = %v", err)
			}

			_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want allowed-root policy rejection")
			}
			if !errors.Is(err, ErrSecretPolicy) {
				t.Fatalf("error = %v, want ErrSecretPolicy", err)
			}
			if !strings.Contains(err.Error(), tc.wantErrorMessage) {
				t.Fatalf("error = %v, want %q", err, tc.wantErrorMessage)
			}
		})
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

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
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

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestConfigFileAllowsEmptySecretLikePlaceholders(t *testing.T) {
	resetConfigEnv(t)

	path := writeTempConfig(t, `
postgres:
  dsn: ""
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

	//nolint:paralleltest // Subtests reset process-wide configuration environment.
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
	t.Parallel()

	keys := []string{
		"http.addr",
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
	if strings.Contains(err.Error(), "time: invalid duration") {
		t.Fatalf("error unexpectedly wraps raw time.ParseDuration detail: %v", err)
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
	t.Parallel()

	if got := ErrorType(nil); got != "" {
		t.Fatalf("ErrorType(nil) = %q, want empty", got)
	}
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
	if got := ErrorType(ErrLoad); got != "load" {
		t.Fatalf("ErrorType(load) = %q", got)
	}
	if got := ErrorType(errors.New("new config error class")); got != "unknown" {
		t.Fatalf("ErrorType(unknown) = %q, want unknown", got)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
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

	//nolint:paralleltest // Subtests reset process-wide configuration environment.
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

	//nolint:paralleltest // Subtests reset process-wide configuration environment.
	t.Run("validate_context_stage", func(t *testing.T) {
		resetConfigEnv(t)

		_, report, err := LoadDetailed(LoadOptions{ValidateBudget: time.Nanosecond})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validate context deadline error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want context.DeadlineExceeded", err)
		}
		if got := ErrorType(err); got != "validate" {
			t.Fatalf("ErrorType(error) = %q, want validate", got)
		}
		if report.FailedStage != StageValidate {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageValidate)
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

func resetConfigEnv(t *testing.T) {
	t.Helper()

	previousValues := make(map[string]string)
	for _, key := range configEnvResetKeys() {
		if value, ok := os.LookupEnv(key); ok {
			previousValues[key] = value
			t.Setenv(key, value)
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
	t.Cleanup(func() {
		for _, key := range configEnvResetKeys() {
			if _, ok := previousValues[key]; !ok {
				_ = os.Unsetenv(key)
			}
		}
	})
	t.Setenv("APP__APP__ENV", "local")
}

func configEnvResetKeys() []string {
	knownKeys := knownConfigKeys()
	knownSections := knownConfigSections()
	keySet := make(map[string]struct{}, len(knownKeys)+len(knownSections)+1)
	for key := range knownKeys {
		keySet[namespaceEnvForConfigKey(key)] = struct{}{}
	}
	for key := range knownSections {
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
