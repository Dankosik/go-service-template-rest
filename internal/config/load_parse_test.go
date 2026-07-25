package config

import (
	"errors"
	"strings"
	"testing"
)

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
