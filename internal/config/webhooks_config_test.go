package config

import (
	"errors"
	"testing"
)

func TestWebhooksConfigContract(t *testing.T) {
	valid := WebhooksConfig{Enabled: true, Endpoints: `{"endpoints":[]}`}
	postgres := PostgresConfig{Enabled: true}
	jobs := JobsConfig{MaxWorkers: 1}
	if err := validateWebhooks(valid, postgres, jobs); err != nil {
		t.Fatalf("validateWebhooks(valid) error = %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*WebhooksConfig, *PostgresConfig, *JobsConfig)
	}{
		{"postgres disabled", func(_ *WebhooksConfig, p *PostgresConfig, _ *JobsConfig) { p.Enabled = false }},
		{"jobs disabled", func(_ *WebhooksConfig, _ *PostgresConfig, j *JobsConfig) { j.MaxWorkers = 0 }},
		{"endpoints missing", func(w *WebhooksConfig, _ *PostgresConfig, _ *JobsConfig) { w.Endpoints = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			webhooks, pg, jobConfig := valid, postgres, jobs
			test.mutate(&webhooks, &pg, &jobConfig)
			if err := validateWebhooks(webhooks, pg, jobConfig); !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
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
	if cfg.Webhooks != (WebhooksConfig{}) {
		t.Fatalf("default webhooks = %+v", cfg.Webhooks)
	}
}
