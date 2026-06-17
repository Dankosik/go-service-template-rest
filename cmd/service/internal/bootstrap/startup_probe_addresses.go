package bootstrap

import (
	"fmt"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

func postgresStartupProbeAddress(cfg config.PostgresConfig) (string, error) {
	address, err := postgres.ProbeAddress(cfg.DSN)
	if err != nil {
		return "", fmt.Errorf("%w: resolve postgres probe address: %w", errDependencyInit, err)
	}
	return address, nil
}
