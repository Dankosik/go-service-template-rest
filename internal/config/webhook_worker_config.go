package config

// profile:webhooks-durable:start

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/samber/lo"
)

func buildWebhookWorkerSnapshot(source *koanf.Koanf) (Config, []string, error) {
	values := lo.PickBy(source.All(), func(key string, _ any) bool {
		return webhookWorkerConfigKey(key)
	})

	worker := koanf.New(keyDelimiter)
	if err := worker.Load(confmap.Provider(values, keyDelimiter), nil); err != nil {
		return Config{}, nil, fmt.Errorf("%w: load webhook worker configuration: %w", ErrLoad, err)
	}
	return buildSnapshot(worker)
}

func webhookWorkerConfigKey(key string) bool {
	for _, section := range []string{"app", "http", "log", "observability", "postgres", "webhooks"} {
		if key == section || strings.HasPrefix(key, section+keyDelimiter) {
			return true
		}
	}
	return false
}

func validateWebhookWorkerConfig(cfg *Config, unknownKeys []string) error {
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
	if err := validateWebhooks(cfg.Webhooks, cfg.Postgres, cfg.HTTP); err != nil {
		return err
	}
	return validateObservabilityConfig(&cfg.Observability)
}

// profile:webhooks-durable:end
