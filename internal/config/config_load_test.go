package config

import (
	"context"
	"errors"
	"testing"
	"time"
)

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
