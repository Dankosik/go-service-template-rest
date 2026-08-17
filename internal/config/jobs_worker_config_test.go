package config

import (
	"context"
	"errors"
	"os"
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
		{name: "invalid jobs poll interval", key: "APP__JOBS__POLL_INTERVAL", value: "0s", contains: "jobs.poll_interval"},
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

//nolint:paralleltest // This test mutates process-global environment or working directory.
func setJobsWorkerConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range configEnvResetKeys(t) {
		if value, ok := os.LookupEnv(key); ok {
			t.Setenv(key, value)
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
	for key, value := range map[string]string{
		"APP__APP__ENV":                      "local",
		"APP__POSTGRES__ENABLED":             "true",
		"APP__POSTGRES__DSN":                 "postgres://jobs:password@localhost:5432/jobs?sslmode=disable",
		"APP__POSTGRES__MAX_OPEN_CONNS":      "2",
		"APP__JOBS__ENABLED":                 "true",
		"APP__JOBS__POLL_INTERVAL":           "1s",
		"APP__JOBS__MAX_CONCURRENCY":         "1",
		"APP__JOBS__LEASE_DURATION":          "6s",
		"APP__JOBS__STORE_OPERATION_TIMEOUT": "1s",
		"APP__JOBS__OBSERVATION_INTERVAL":    "1s",
		"APP__JOBS__DRAIN_TIMEOUT":           "1s",
	} {
		t.Setenv(key, value)
	}
}
