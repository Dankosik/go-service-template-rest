package config

import (
	"errors"
	"strings"
	"testing"
)

func TestWebhooksConfigContract(t *testing.T) {
	valid := OutboundWebhooksConfig{Enabled: true, Endpoints: `{"endpoints":[]}`}
	postgres := PostgresConfig{Enabled: true}
	jobs := JobsConfig{MaxWorkers: 1}
	if err := validateOutboundWebhooks(valid, postgres, jobs); err != nil {
		t.Fatalf("validateOutboundWebhooks(valid) error = %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*OutboundWebhooksConfig, *PostgresConfig, *JobsConfig)
	}{
		{"postgres disabled", func(_ *OutboundWebhooksConfig, p *PostgresConfig, _ *JobsConfig) { p.Enabled = false }},
		{"jobs disabled", func(_ *OutboundWebhooksConfig, _ *PostgresConfig, j *JobsConfig) { j.MaxWorkers = 0 }},
		{"endpoints missing", func(w *OutboundWebhooksConfig, _ *PostgresConfig, _ *JobsConfig) { w.Endpoints = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			webhooks, pg, jobConfig := valid, postgres, jobs
			test.mutate(&webhooks, &pg, &jobConfig)
			err := validateOutboundWebhooks(webhooks, pg, jobConfig)
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
			if test.name == "jobs disabled" && !strings.Contains(err.Error(), "jobs.max_workers") {
				t.Fatalf("error = %v, want jobs.max_workers prerequisite", err)
			}
		})
	}
}

func TestWebhooksConfigDefaultsDisabled(t *testing.T) {
	resetConfigEnv(t)
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OutboundWebhooks != (OutboundWebhooksConfig{}) {
		t.Fatalf("default webhooks = %+v", cfg.OutboundWebhooks)
	}
}
