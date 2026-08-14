package config

import (
	"fmt"
	"time"
)

// HTTPIdempotencyConfig contains only process-wide runtime quantities. Endpoint
// contracts and retention policy remain with each registered operation.
type HTTPIdempotencyConfig struct {
	OwnerRecoveryDelay     time.Duration `koanf:"owner_recovery_delay"`
	MaintenanceInterval    time.Duration `koanf:"maintenance_interval"`
	CleanupBatchSize       int           `koanf:"cleanup_batch_size"`
	MaxMaintenanceLag      time.Duration `koanf:"max_maintenance_lag"`
	MaxRelationBytes       int64         `koanf:"max_relation_bytes"`
	AdmissionHeadroomBytes int64         `koanf:"admission_headroom_bytes"`
}

// ValidateHTTPIdempotencyActive validates the bootstrap-owned part of an active
// registration. Store-owned quantities are validated by postgresidempotency.
func ValidateHTTPIdempotencyActive(cfg HTTPIdempotencyConfig, postgres PostgresConfig) error {
	if !postgres.Enabled {
		return fmt.Errorf("%w: http_idempotency requires postgres.enabled", ErrValidate)
	}
	if cfg.MaintenanceInterval <= 0 {
		return fmt.Errorf("%w: http_idempotency.maintenance_interval must be positive", ErrValidate)
	}
	return nil
}
