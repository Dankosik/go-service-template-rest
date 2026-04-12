package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func parsePoolConfig(rawDSN string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(rawDSN)
	if err != nil {
		return nil, fmt.Errorf("%w: parse postgres dsn: invalid value redacted", ErrConfig)
	}
	return config, nil
}
