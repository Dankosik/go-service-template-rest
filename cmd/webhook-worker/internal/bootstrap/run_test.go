package bootstrap

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
)

func TestWebhookWorkerStartupRejectsDisabledProfile(t *testing.T) {
	err := validateRuntimeConfig(config.Config{})
	if !errors.Is(err, postgreswebhook.ErrConfig) {
		t.Fatalf("validateRuntimeConfig() error = %v", err)
	}
}

func TestWebhookWorkerConfigHelpers(t *testing.T) {
	options, err := parseLoadOptions([]string{"--config", "base.yaml", "--config-overlay", "one.yaml", "--config-overlay=two.yaml"})
	if err != nil || options.ConfigPath != "base.yaml" || len(options.ConfigOverlays) != 2 || options.ConfigOverlays[1] != "two.yaml" {
		t.Fatalf("parseLoadOptions() = %+v, %v", options, err)
	}
	for _, args := range [][]string{{"--config", ""}, {"--config-overlay", ""}, {"unexpected"}} {
		if _, err := parseLoadOptions(args); err == nil {
			t.Fatalf("parseLoadOptions(%q) error = nil", args)
		}
	}

	valid := config.Config{}
	valid.Webhooks.Enabled = true
	valid.Postgres.Enabled = true
	valid.Observability.Metrics.Addr = ":9090"
	valid.Webhooks.DrainTimeout = time.Second
	valid.HTTP.GracePeriod = 11 * time.Second
	if err := validateRuntimeConfig(valid); err != nil {
		t.Fatalf("validateRuntimeConfig(valid) error = %v", err)
	}
	for _, test := range []struct {
		name     string
		mutate   func(*config.Config)
		contains string
	}{
		{name: "postgres disabled", mutate: func(cfg *config.Config) { cfg.Postgres.Enabled = false }, contains: "webhooks and postgres"},
		{name: "diagnostics missing", mutate: func(cfg *config.Config) { cfg.Observability.Metrics.Addr = " " }, contains: "diagnostics address"},
		{name: "grace too short", mutate: func(cfg *config.Config) { cfg.HTTP.GracePeriod = 9 * time.Second }, contains: "http.grace_period"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid
			test.mutate(&cfg)
			if err := validateRuntimeConfig(cfg); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("validateRuntimeConfig() error = %v, want %q", err, test.contains)
			}
		})
	}

	worker, err := workerConfig(valid)
	host, hostErr := os.Hostname()
	if err != nil || hostErr != nil || worker.WorkerID != host || worker.DrainTimeout != valid.Webhooks.DrainTimeout {
		t.Fatalf("workerConfig() = %+v, %v", worker, err)
	}
}
