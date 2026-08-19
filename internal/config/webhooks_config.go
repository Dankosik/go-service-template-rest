package config

// profile:webhooks-durable:start

import (
	"fmt"
	"time"
)

type WebhooksConfig struct {
	Enabled               bool          `koanf:"enabled"`
	CapacityRevision      int64         `koanf:"capacity_revision"`
	GlobalConcurrency     int           `koanf:"global_concurrency"`
	ClaimScanPage         int           `koanf:"claim_scan_page"`
	PollInterval          time.Duration `koanf:"poll_interval"`
	ObservationInterval   time.Duration `koanf:"observation_interval"`
	StoreOperationTimeout time.Duration `koanf:"store_operation_timeout"`
	AttemptTimeout        time.Duration `koanf:"attempt_timeout"`
	ResponseHeaderTimeout time.Duration `koanf:"response_header_timeout"`
	ResponseHeaderBytes   int           `koanf:"response_header_bytes"`
	ResponseBodyBytes     int           `koanf:"response_body_bytes"`
	DrainTimeout          time.Duration `koanf:"drain_timeout"`
	MaintenanceInterval   time.Duration `koanf:"maintenance_interval"`
	MaintenanceBatch      int           `koanf:"maintenance_batch"`
	StaticSecrets         string        `koanf:"static_secrets"`
}

func webhooksDefaults() map[string]any {
	return map[string]any{
		"webhooks.enabled": false, "webhooks.capacity_revision": 0, "webhooks.global_concurrency": 0,
		"webhooks.claim_scan_page": 0, "webhooks.poll_interval": "0s", "webhooks.observation_interval": "0s",
		"webhooks.store_operation_timeout": "0s", "webhooks.attempt_timeout": "0s",
		"webhooks.response_header_timeout": "0s", "webhooks.response_header_bytes": 0,
		"webhooks.response_body_bytes": 0, "webhooks.drain_timeout": "0s",
		"webhooks.maintenance_interval": "0s", "webhooks.maintenance_batch": 0, "webhooks.static_secrets": "",
	}
}

func validateWebhooks(cfg WebhooksConfig, postgres PostgresConfig, http HTTPConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if !postgres.Enabled {
		return fmt.Errorf("%w: webhooks.enabled requires postgres.enabled", ErrValidate)
	}
	if cfg.CapacityRevision <= 0 {
		return fmt.Errorf("%w: webhooks.capacity_revision must be positive", ErrValidate)
	}
	if err := validateIntRange("webhooks.global_concurrency", cfg.GlobalConcurrency, 1, 256); err != nil {
		return err
	}
	if err := validateIntRange("webhooks.claim_scan_page", cfg.ClaimScanPage, 1, 256); err != nil {
		return err
	}
	if err := validateIntRange("webhooks.maintenance_batch", cfg.MaintenanceBatch, 1, 1000); err != nil {
		return err
	}
	if err := validateIntRange("webhooks.response_header_bytes", cfg.ResponseHeaderBytes, 1, 1<<20); err != nil {
		return err
	}
	if err := validateIntRange("webhooks.response_body_bytes", cfg.ResponseBodyBytes, 1, 1<<20); err != nil {
		return err
	}
	for name, value := range map[string]time.Duration{
		"poll_interval": cfg.PollInterval, "observation_interval": cfg.ObservationInterval,
		"store_operation_timeout": cfg.StoreOperationTimeout, "attempt_timeout": cfg.AttemptTimeout,
		"response_header_timeout": cfg.ResponseHeaderTimeout, "drain_timeout": cfg.DrainTimeout,
		"maintenance_interval": cfg.MaintenanceInterval,
	} {
		if value <= 0 {
			return fmt.Errorf("%w: webhooks.%s must be positive", ErrValidate, name)
		}
	}
	if cfg.StoreOperationTimeout > 30*time.Second || cfg.StoreOperationTimeout > postgresStatementTimeout {
		return fmt.Errorf("%w: webhooks.store_operation_timeout must be at most the 8s Postgres statement budget", ErrValidate)
	}
	if cfg.ResponseHeaderTimeout > cfg.AttemptTimeout || cfg.AttemptTimeout > cfg.DrainTimeout {
		return fmt.Errorf("%w: webhook header, attempt, and drain budgets must nest", ErrValidate)
	}
	if cfg.AttemptTimeout > 10*time.Minute || cfg.DrainTimeout > 30*time.Minute {
		return fmt.Errorf("%w: webhook attempt and drain budgets exceed engine ceilings", ErrValidate)
	}
	if http.GracePeriod <= cfg.DrainTimeout {
		return fmt.Errorf("%w: http.grace_period must exceed webhooks.drain_timeout", ErrValidate)
	}
	if cfg.StaticSecrets == "" {
		return fmt.Errorf("%w: webhooks.static_secrets must be supplied through environment", ErrValidate)
	}
	return nil
}

// profile:webhooks-durable:end
