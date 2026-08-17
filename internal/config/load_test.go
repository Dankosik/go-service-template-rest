package //nolint:paralleltest // This test mutates process-global environment or working directory.

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
//nolint:paralleltest // This test mutates process-global environment or working directory.

// profile:database-postgres:start
//
//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
config

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestLoadNormalizesStringsAtSemanticValidationOwners(t *testing.T) {

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", " local ")
	t.Setenv("APP__APP__VERSION", " v1.2.3 ")
	t.Setenv("APP__HTTP__ADDR", " :8081 ")
	t.Setenv("APP__OBSERVABILITY__METRICS__ADDR", " 127.0.0.1:9091 ")
	t.Setenv("APP__OBSERVABILITY__OTEL__SERVICE_NAME", " reference ")
	t.Setenv("APP__OBSERVABILITY__OTEL__TRACES_SAMPLER", " parentbased_traceidratio ")
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT", " https://otel.example.com/v1/traces ")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}

	if cfg.App.Env != "local" || cfg.App.Version != "v1.2.3" {
		t.Fatalf("App = %+v, want normalized strings", cfg.App)
	}
	if cfg.HTTP.Addr != ":8081" {
		t.Fatalf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":8081")
	}
	if cfg.Observability.Metrics.Addr != "127.0.0.1:9091" {
		t.Fatalf("Metrics.Addr = %q, want %q", cfg.Observability.Metrics.Addr, "127.0.0.1:9091")
	}
	if cfg.Observability.OTel.ServiceName != "reference" {
		t.Fatalf("ServiceName = %q, want %q", cfg.Observability.OTel.ServiceName, "reference")
	}
	if cfg.Observability.OTel.TracesSampler != "parentbased_traceidratio" {
		t.Fatalf("TracesSampler = %q, want normalized sampler", cfg.Observability.OTel.TracesSampler)
	}
	if cfg.Observability.OTel.Exporter.OTLPEndpoint != "https://otel.example.com/v1/traces" {
		t.Fatalf("OTLPEndpoint = %q, want normalized endpoint", cfg.Observability.OTel.Exporter.OTLPEndpoint)
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
		t.Fatal("Postgres.Enabled = true, want false when only flat key is set")
	}
	if cfg.Postgres.DSN != "" {
		t.Fatalf("Postgres.DSN = %q, want empty when only flat key is set", cfg.Postgres.DSN)
	}
}

// profile:database-postgres:end

func TestErrorTypeMapping(t *testing.T) {
	t.Parallel()

	if got := ErrorType(nil); got != "" {
		t.Fatalf("ErrorType(nil) = %q, want empty", got)
	}
	if got := ErrorType(ErrUnknownKey); got != "unknown_key" {
		t.Fatalf("ErrorType(unknown_key) = %q", got)
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
	t.Parallel()
	resetConfigEnv(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := LoadDetailedWithContext(ctx, LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailedWithContext() expected context cancellation error")
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
			t.Fatal("LoadDetailed() expected parse error")
		}
		if !errors.Is(err, ErrParse) {
			t.Fatalf("error = %v, want ErrParse", err)
		}
		if report.FailedStage != StageParse {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageParse)
		}
	})

	//nolint:paralleltest // Subtests reset process-wide configuration environment.
	t.Run("validate_stage", func(t *testing.T) {
		t.Parallel()
		resetConfigEnv(t)
		configPath := writeTempConfig(t, `
unknown:
  field: value
`)

		_, report, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
		if err == nil {
			t.Fatal("LoadDetailed() expected unknown key error")
		}
		if !errors.Is(err, ErrUnknownKey) {
			t.Fatalf("error = %v, want ErrUnknownKey", err)
		}
		if report.FailedStage != StageValidate {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageValidate)
		}
	})

	t.Run("load_file_stage", func(t *testing.T) {

		resetConfigEnv(t)
		t.Setenv("APP__APP__ENV", "prod")
		t.Setenv("APP_CONFIG_ALLOWED_ROOTS", t.TempDir())

		_, report, err := LoadDetailed(LoadOptions{ConfigPath: "/nonexistent/config.yaml"})
		if err == nil {
			t.Fatal("LoadDetailed() expected load error")
		}
		if !errors.Is(err, ErrLoad) && !errors.Is(err, ErrSecretPolicy) {
			t.Fatalf("error = %v, want ErrLoad or ErrSecretPolicy", err)
		}
		if report.FailedStage != StageLoadFile {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageLoadFile)
		}
	})
}

func TestLoadDetailedRejectsEmptyExplicitPaths(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		opts LoadOptions
	}{
		{
			name: "whitespace base path",
			opts: LoadOptions{ConfigPath: " \t\n "},
		},
		{
			name: "empty overlay path",
			opts: LoadOptions{ConfigOverlays: []string{""}},
		},
		{
			name: "whitespace overlay path",
			opts: LoadOptions{ConfigOverlays: []string{" \t\n "}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resetConfigEnv(t)

			_, report, err := LoadDetailed(tc.opts)
			if !errors.Is(err, ErrLoad) {
				t.Fatalf("LoadDetailed() error = %v, want ErrLoad", err)
			}
			if report.FailedStage != StageLoadFile {
				t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageLoadFile)
			}
		})
	}
}

func TestOTLPExporterValuesFromNamespaceEnv(t *testing.T) {

	resetConfigEnv(t)

	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT", "https://otel.example.com:4318")
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_HEADERS", "authorization=Bearer token")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Observability.OTel.Exporter.OTLPEndpoint != "https://otel.example.com:4318" {
		t.Fatalf("OTLPEndpoint = %q, want %q", cfg.Observability.OTel.Exporter.OTLPEndpoint, "https://otel.example.com:4318")
	}
	if cfg.Observability.OTel.Exporter.OTLPHeaders != "authorization=Bearer token" {
		t.Fatalf("OTLPHeaders = %q, want %q", cfg.Observability.OTel.Exporter.OTLPHeaders, "authorization=Bearer token")
	}
}

func TestLoadInvalidDurationReturnsParseError(t *testing.T) {

	resetConfigEnv(t)
	t.Setenv("APP__HTTP__READ_TIMEOUT", "oops")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() expected parse error")
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
		{name: "duration missing unit", envKey: "APP__HTTP__READ_TIMEOUT", envValue: "150", wantDetail: "missing duration unit"},
		{name: "int format", envKey: "APP__HTTP__MAX_HEADER_BYTES", envValue: "many", wantDetail: "invalid integer format"},
		{name: "float finite check", envKey: "APP__OBSERVABILITY__OTEL__TRACES_SAMPLER_ARG", envValue: "NaN", wantDetail: "non-finite numeric value"},
		{name: "bool format", envKey: "APP__HTTP__ACCESS_LOG_HEALTH_PROBES", envValue: "maybe", wantDetail: "invalid boolean format"},
		{name: "log level", envKey: "APP__LOG__LEVEL", envValue: "secret-level", wantDetail: "invalid log level"},
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
	t.Parallel()
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
broken: [
`)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatal("LoadDetailed() expected parse error for malformed YAML")
	}
	if !errors.Is(err, ErrParse) {
		t.Fatalf("error = %v, want ErrParse", err)
	}
	if got := ErrorType(err); got != "parse" {
		t.Fatalf("ErrorType(error) = %q, want parse", got)
	}
}

func TestParseErrorDoesNotLeakRawValue(t *testing.T) {

	resetConfigEnv(t)

	secretLikeValue := "supersecret-token-value"
	t.Setenv("APP__HTTP__READ_TIMEOUT", secretLikeValue)

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() expected parse error")
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
