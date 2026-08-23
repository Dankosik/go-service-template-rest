package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/samber/lo"
)

func buildJobsWorkerSnapshot(source *koanf.Koanf) (Config, []string, error) {
	// profile:inbound-webhooks-standard:start
	if hasNonEmptyConfigValue(source.Get("inbound_webhooks.static_secrets")) {
		return Config{}, nil, fmt.Errorf("%w: inbound_webhooks.static_secrets is not allowed in the jobs worker", ErrValidate)
	}
	// profile:inbound-webhooks-standard:end
	values := lo.PickBy(source.All(), func(key string, _ any) bool {
		return jobsWorkerConfigKey(key)
	})

	worker := koanf.New(keyDelimiter)
	if err := worker.Load(confmap.Provider(values, keyDelimiter), nil); err != nil {
		return Config{}, nil, fmt.Errorf("%w: load jobs worker configuration: %w", ErrLoad, err)
	}
	return buildSnapshot(worker)
}

func jobsWorkerConfigKey(key string) bool {
	sections := []string{"app", "http", "log", "observability", "postgres", "jobs"}
	// profile:webhooks-durable:start
	sections = append(sections, "webhooks")
	// profile:webhooks-durable:end
	for _, section := range sections {
		if key == section || strings.HasPrefix(key, section+keyDelimiter) {
			return true
		}
	}
	// profile:inbound-webhooks-standard:start
	if key == "inbound_webhooks.endpoints" {
		return true
	}
	// profile:inbound-webhooks-standard:end
	return false
}

func validateJobsWorkerConfig(cfg *Config, unknownKeys []string) error {
	if unknown := findUnknownKeys(unknownKeys); len(unknown) > 0 {
		return fmt.Errorf("%w: unknown keys: %s", ErrUnknownKey, strings.Join(unknown, ", "))
	}
	if err := validateAppConfig(&cfg.App); err != nil {
		return err
	}
	if err := validateHTTPConfig(&cfg.HTTP); err != nil {
		return err
	}
	if err := validatePostgres(cfg.Postgres, cfg.HTTP); err != nil {
		return err
	}
	if err := validateJobs(cfg.Jobs, cfg.Postgres); err != nil {
		return err
	}
	// profile:webhooks-durable:start
	if cfg.Webhooks.Enabled && cfg.Webhooks.StaticSecrets == "" {
		return fmt.Errorf("%w: webhooks.static_secrets must be supplied through environment", ErrValidate)
	}
	// profile:webhooks-durable:end
	// profile:inbound-webhooks-standard:start
	if err := validateInboundWebhooksWorker(cfg.InboundWebhooks); err != nil {
		return err
	}
	// profile:inbound-webhooks-standard:end
	return validateObservabilityConfig(&cfg.Observability)
}
