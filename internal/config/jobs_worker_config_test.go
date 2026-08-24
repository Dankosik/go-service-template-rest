package config

import (
	"context"
	"errors"
	"strings"
	"testing"
)

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestJobsConfigWorkerLoaderIgnoresForeignProfiles(t *testing.T) {
	setJobsWorkerConfigEnv(t)
	t.Setenv("APP__AUTHN__ISSUER", "")
	t.Setenv("APP__OUTBOUND_AUTH__DEPENDENCY", "not a dependency")
	t.Setenv("APP__OUTBOUND_AUTH__ACQUISITION_TIMEOUT", "not a duration")
	t.Setenv("APP__OBJECT_STORAGE__PROVIDER", "not a provider")

	_, _, err := LoadJobsWorkerDetailedWithContext(context.Background(), LoadOptions{})
	if err != nil {
		t.Fatalf("LoadJobsWorkerDetailedWithContext() error = %v", err)
	}
}

//nolint:paralleltest // This test mutates process-global environment.
func TestJobsConfigWorkerLoaderRejectsUnknownRetainedKey(t *testing.T) {
	setJobsWorkerConfigEnv(t)
	t.Setenv("APP__POSTGRES__UNKNOWN", "value")

	_, _, err := LoadJobsWorkerDetailedWithContext(context.Background(), LoadOptions{})
	if !errors.Is(err, ErrUnknownKey) || !strings.Contains(err.Error(), "postgres.unknown") {
		t.Fatalf("LoadJobsWorkerDetailedWithContext() error = %v, want retained-section unknown key", err)
	}
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestJobsConfigWorkerLoaderRejectsInvalidJobsAndPostgres(t *testing.T) {
	for _, test := range []struct {
		name     string
		key      string
		value    string
		contains string
	}{
		{name: "postgres disabled", key: "APP__POSTGRES__ENABLED", value: "false", contains: "requires postgres.enabled"},
		{name: "missing postgres dsn", key: "APP__POSTGRES__DSN", value: "", contains: "postgres.dsn"},
		{name: "invalid jobs workers", key: "APP__JOBS__MAX_WORKERS", value: "501", contains: "jobs.max_workers"},
	} {
		t.Run(test.name, func(t *testing.T) {
			setJobsWorkerConfigEnv(t)
			t.Setenv(test.key, test.value)

			_, _, err := LoadJobsWorkerDetailedWithContext(context.Background(), LoadOptions{})
			if !errors.Is(err, ErrValidate) && !errors.Is(err, ErrSecretPolicy) {
				t.Fatalf("LoadJobsWorkerDetailedWithContext() error = %v, want config validation error", err)
			}
			if !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("LoadJobsWorkerDetailedWithContext() error = %v, want %q", err, test.contains)
			}
		})
	}
}

// profile:webhooks-durable:start
//
//nolint:paralleltest // This test mutates process-global environment.
func TestJobsWorkerRequiresWebhookSecretsWhenEnabled(t *testing.T) {
	setJobsWorkerConfigEnv(t)
	t.Setenv("APP__WEBHOOKS__ENABLED", "true")
	t.Setenv("APP__WEBHOOKS__ENDPOINTS", `{"endpoints":[]}`)
	_, _, err := LoadJobsWorkerDetailedWithContext(context.Background(), LoadOptions{})
	if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), "webhooks.static_secrets") {
		t.Fatalf("LoadJobsWorkerDetailedWithContext() error = %v", err)
	}
}

// profile:webhooks-durable:end

//nolint:paralleltest // This test mutates process-global environment or working directory.
func setJobsWorkerConfigEnv(t *testing.T) {
	t.Helper()
	clearConfigEnv(t)
	for key, value := range map[string]string{
		"APP__APP__ENV":                 "local",
		"APP__POSTGRES__ENABLED":        "true",
		"APP__POSTGRES__DSN":            "postgres://jobs:password@localhost:5432/jobs?sslmode=disable",
		"APP__POSTGRES__MAX_OPEN_CONNS": "2",
		"APP__JOBS__MAX_WORKERS":        "1",
	} {
		t.Setenv(key, value)
	}
}
