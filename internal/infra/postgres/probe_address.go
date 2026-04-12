package postgres

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ProbeAddress extracts a probe-ready host:port from a PostgreSQL DSN.
func ProbeAddress(rawDSN string) (string, error) {
	pgxCfg, err := parsePoolConfig(rawDSN)
	if err != nil {
		return "", err
	}
	host := strings.TrimSpace(pgxCfg.ConnConfig.Host)
	if host == "" || pgxCfg.ConnConfig.Port == 0 || strings.ContainsAny(host, `/\`) {
		return "", fmt.Errorf("%w: invalid postgres probe address", ErrConfig)
	}
	return net.JoinHostPort(host, strconv.Itoa(int(pgxCfg.ConnConfig.Port))), nil
}
