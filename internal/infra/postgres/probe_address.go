package postgres

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProbeAddress extracts a probe-ready host:port from a PostgreSQL DSN.
func ProbeAddress(rawDSN string) (string, error) {
	pgxCfg, err := pgxpool.ParseConfig(rawDSN)
	if err != nil {
		// pgx parse errors can echo the input DSN, including credentials.
		return "", fmt.Errorf("%w: parse postgres dsn: invalid value redacted", ErrConfig)
	}
	host := strings.TrimSpace(pgxCfg.ConnConfig.Host)
	if host == "" || pgxCfg.ConnConfig.Port == 0 || strings.ContainsAny(host, `/\`) {
		return "", fmt.Errorf("%w: invalid postgres probe address", ErrConfig)
	}
	return net.JoinHostPort(host, strconv.Itoa(int(pgxCfg.ConnConfig.Port))), nil
}
