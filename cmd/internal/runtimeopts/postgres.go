package runtimeopts

// profile:database-postgres:start

import (
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

// Postgres builds the pool options the API process, jobs worker, and outbox
// relay open their pools with. They size their pools from the same configuration
// section, so a budget added to it reaches all three or none.
func Postgres(cfg config.PostgresConfig) postgres.Options {
	return postgres.Options{
		DSN:          cfg.DSN,
		MaxOpenConns: cfg.MaxOpenConns,
	}
}

// profile:database-postgres:end
