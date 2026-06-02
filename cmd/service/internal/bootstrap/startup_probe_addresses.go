package bootstrap

import (
	"fmt"
	"strings"

	"github.com/Dankosik/billing-service/internal/config"
	"github.com/Dankosik/billing-service/internal/infra/postgres"
)

func postgresStartupProbeAddress(cfg config.PostgresConfig) (string, error) {
	address, err := postgres.ProbeAddress(cfg.DSN)
	if err != nil {
		return "", fmt.Errorf("%w: resolve postgres probe address: %w", errDependencyInit, err)
	}
	return address, nil
}

func redisStartupProbeAddress(cfg config.RedisConfig) (string, error) {
	address := strings.TrimSpace(cfg.Addr)
	if address == "" {
		return "", fmt.Errorf("%w: empty redis probe address", errDependencyInit)
	}
	return address, nil
}

func mongoStartupProbeAddress(cfg config.MongoConfig) (string, error) {
	address, err := config.MongoProbeAddress(cfg.URI)
	if err != nil {
		return "", fmt.Errorf("%w: resolve mongo probe address: %w", errDependencyInit, err)
	}
	return address, nil
}
