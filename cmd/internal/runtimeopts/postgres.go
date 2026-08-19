package runtimeopts

// profile:database-postgres:start

import (
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

// Postgres builds the pool options both the API process and the outbox relay
// open their pool with. The two size their pools from the same configuration
// section, so a budget added to it reaches both or neither.
func Postgres(cfg config.PostgresConfig) postgres.Options {
	return postgres.Options{
		DSN:          cfg.DSN,
		MaxOpenConns: cfg.MaxOpenConns,
	}
}

// profile:database-postgres:end
