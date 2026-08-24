package config

// profile:webhooks-durable:start

import "fmt"

type WebhooksConfig struct {
	Enabled       bool   `koanf:"enabled"`
	Endpoints     string `koanf:"endpoints"`
	StaticSecrets string `koanf:"static_secrets"`
}

func webhooksDefaults() map[string]any {
	return map[string]any{
		"webhooks.enabled": false, "webhooks.endpoints": "", "webhooks.static_secrets": "",
	}
}

func validateWebhooks(cfg WebhooksConfig, postgres PostgresConfig, jobs JobsConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if !postgres.Enabled || jobs.MaxWorkers <= 0 {
		return fmt.Errorf("%w: webhooks.enabled requires postgres.enabled and jobs.enabled", ErrValidate)
	}
	if cfg.Endpoints == "" {
		return fmt.Errorf("%w: webhooks.endpoints are required", ErrValidate)
	}
	return nil
}

// profile:webhooks-durable:end
