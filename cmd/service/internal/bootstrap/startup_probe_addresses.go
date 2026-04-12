package bootstrap

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func postgresStartupProbeAddress(cfg config.PostgresConfig) (string, error) {
	pgxCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return "", fmt.Errorf("%w: parse postgres dsn", errDependencyInit)
	}
	host := strings.TrimSpace(pgxCfg.ConnConfig.Host)
	if host == "" || pgxCfg.ConnConfig.Port == 0 {
		return "", fmt.Errorf("%w: invalid postgres probe address", errDependencyInit)
	}
	return net.JoinHostPort(host, strconv.Itoa(int(pgxCfg.ConnConfig.Port))), nil
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
		return "", fmt.Errorf("%w: resolve mongo probe address", errDependencyInit)
	}
	return address, nil
}
