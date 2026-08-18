package config

import (
	"fmt"
	"time"
)

// HTTPIdempotencyConfig contains the one deployment-owned product quantity.
type HTTPIdempotencyConfig struct {
	Retention time.Duration `koanf:"retention"`
}

// ValidateHTTPIdempotencyActive validates the bootstrap-owned part of an active
// registration. Store-owned quantities are validated by postgresidempotency.
func ValidateHTTPIdempotencyActive(cfg HTTPIdempotencyConfig, postgres PostgresConfig) error {
	if !postgres.Enabled {
		return fmt.Errorf("%w: http_idempotency requires postgres.enabled", ErrValidate)
	}
	if cfg.Retention <= 0 {
		return fmt.Errorf("%w: http_idempotency.retention must be positive", ErrValidate)
	}
	return nil
}
