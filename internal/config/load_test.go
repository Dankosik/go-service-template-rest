package config

import (
	"context"
	"errors"
	"testing"
)

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
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

// profile:database-postgres:start
//
//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
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
	})

	//nolint:paralleltest // Subtests reset process-wide configuration environment.
	t.Run("validate_stage", func(t *testing.T) {
		resetConfigEnv(t)
		configPath := writeTempConfig(t, `
unknown:
  field: value
`)

		_, report, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
		if err == nil {
			t.Fatalf("LoadDetailed() expected unknown key error")
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
			t.Fatalf("LoadDetailed() expected load error")
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
