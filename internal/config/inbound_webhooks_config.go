// profile:inbound-webhooks-standard:start
package config

import "fmt"

// InboundWebhooksConfig is the removable inbound webhook config section.
type InboundWebhooksConfig struct {
	Endpoints     string `koanf:"endpoints"`
	StaticSecrets string `koanf:"static_secrets"`
}

func inboundWebhooksDefaults() map[string]any {
	return map[string]any{
		"inbound_webhooks.endpoints":      "",
		"inbound_webhooks.static_secrets": "",
	}
}

func validateInboundWebhooks(cfg InboundWebhooksConfig) error {
	if (cfg.Endpoints == "") != (cfg.StaticSecrets == "") {
		return fmt.Errorf("%w: inbound_webhooks.endpoints and inbound_webhooks.static_secrets must be supplied together", ErrValidate)
	}
	return nil
}

func validateInboundWebhooksWorker(cfg InboundWebhooksConfig) error {
	if cfg.StaticSecrets != "" {
		return fmt.Errorf("%w: inbound_webhooks.static_secrets is not allowed in the jobs worker", ErrValidate)
	}
	return nil
}

// profile:inbound-webhooks-standard:end
