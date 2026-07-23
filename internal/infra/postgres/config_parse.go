package postgres

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var requiredPostgresDSNSettings = []string{
	"host",
	"port",
	"user",
	"password",
	"database",
	"sslmode",
}

var allowedPostgresSSLModes = map[string]struct{}{
	"disable":     {},
	"require":     {},
	"verify-ca":   {},
	"verify-full": {},
}

const (
	postgresServicePassfileDSNSourceError = "postgres dsn uses unsupported service/passfile source" // #nosec G101 -- diagnostic text, not a credential.
	postgresTLSFileDSNSourceError         = "postgres dsn uses unsupported TLS file source"
)

type postgresFileDefaultDSNKey struct {
	name              string
	validationMessage string
}

var postgresFileDefaultDSNKeys = []postgresFileDefaultDSNKey{
	{name: "passfile", validationMessage: postgresServicePassfileDSNSourceError},
	{name: "sslcert", validationMessage: postgresTLSFileDSNSourceError},
	{name: "sslkey", validationMessage: postgresTLSFileDSNSourceError},
	{name: "sslrootcert", validationMessage: postgresTLSFileDSNSourceError},
	{name: "sslpassword", validationMessage: postgresTLSFileDSNSourceError},
}

var disallowedPostgresDSNKeys = postgresDisallowedDSNKeys()

func postgresDisallowedDSNKeys() map[string]string {
	keys := map[string]string{
		"service":     postgresServicePassfileDSNSourceError,
		"servicefile": postgresServicePassfileDSNSourceError,
	}
	for _, key := range postgresFileDefaultDSNKeys {
		keys[key.name] = key.validationMessage
	}
	return keys
}

func parsePoolConfig(rawDSN string) (*pgxpool.Config, error) {
	normalizedDSN, err := preflightPostgresDSN(rawDSN)
	if err != nil {
		return nil, err
	}

	config, err := pgxpool.ParseConfig(normalizedDSN)
	if err != nil {
		return nil, fmt.Errorf("%w: parse postgres dsn: invalid value redacted", ErrConfig)
	}
	if _, err := postgresProbeAddressFromPoolConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

// ProbeAddress extracts a probe-ready host:port from a PostgreSQL DSN.
func ProbeAddress(rawDSN string) (string, error) {
	pgxCfg, err := parsePoolConfig(rawDSN)
	if err != nil {
		return "", err
	}
	return postgresProbeAddressFromPoolConfig(pgxCfg)
}

func preflightPostgresDSN(rawDSN string) (string, error) {
	dsn := strings.TrimSpace(rawDSN)
	if dsn == "" {
		return "", fmt.Errorf("%w: postgres dsn is empty", ErrConfig)
	}
	for _, env := range os.Environ() {
		name, value, ok := strings.Cut(env, "=")
		if ok && strings.HasPrefix(name, "PG") && value != "" {
			return "", fmt.Errorf("%w: postgres dsn uses unsupported ambient PG environment", ErrConfig)
		}
	}

	if !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		return "", fmt.Errorf("%w: postgres dsn must use postgres:// or postgresql:// URL format", ErrConfig)
	}
	settings, err := parsePostgresURLDSNSettings(dsn)
	if err != nil {
		return "", fmt.Errorf("%w: parse postgres dsn: invalid value redacted", ErrConfig)
	}
	if err := validatePostgresDSNSettings(settings); err != nil {
		return "", err
	}
	return normalizePostgresURLDSN(dsn)
}

func parsePostgresURLDSNSettings(dsn string) (map[string]string, error) {
	settings := make(map[string]string)

	parsedURL, err := url.Parse(dsn)
	if err != nil {
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			return nil, fmt.Errorf("parse postgres url settings: %w", urlErr.Err)
		}
		return nil, fmt.Errorf("parse postgres url settings: %w", err)
	}
	if parsedURL.User != nil {
		settings["user"] = parsedURL.User.Username()
		if password, present := parsedURL.User.Password(); present {
			settings["password"] = password
		}
	}

	var hosts []string
	var ports []string
	for host := range strings.SplitSeq(parsedURL.Host, ",") {
		if host == "" {
			continue
		}
		if postgresURLHostDoesNotNeedSplitHostPort(host) {
			hosts = append(hosts, strings.Trim(host, "[]"))
			continue
		}
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			return nil, fmt.Errorf("parse postgres url host: %w", err)
		}
		if h != "" {
			hosts = append(hosts, h)
		}
		if p != "" {
			ports = append(ports, p)
		}
	}
	if len(hosts) > 0 {
		settings["host"] = strings.Join(hosts, ",")
	}
	if len(ports) > 0 {
		settings["port"] = strings.Join(ports, ",")
	}
	if database := strings.TrimLeft(parsedURL.Path, "/"); database != "" {
		settings["database"] = database
	}
	for key, value := range parsedURL.Query() {
		if key == "dbname" {
			key = "database"
		}
		if len(value) > 0 {
			settings[key] = value[0]
		}
	}
	return settings, nil
}

func postgresURLHostDoesNotNeedSplitHostPort(host string) bool {
	return net.ParseIP(strings.Trim(host, "[]")) != nil || !strings.Contains(host, ":")
}

func validatePostgresDSNSettings(settings map[string]string) error {
	for key, message := range disallowedPostgresDSNKeys {
		if _, present := settings[key]; present {
			return fmt.Errorf("%w: %s", ErrConfig, message)
		}
	}
	for _, key := range requiredPostgresDSNSettings {
		if strings.TrimSpace(settings[key]) == "" {
			return fmt.Errorf("%w: postgres dsn requires explicit host, port, user, password, database, and sslmode", ErrConfig)
		}
	}
	if _, ok := allowedPostgresSSLModes[settings["sslmode"]]; !ok {
		return fmt.Errorf("%w: postgres dsn fallback targets are not supported", ErrConfig)
	}
	return nil
}

func normalizePostgresURLDSN(dsn string) (string, error) {
	parsedURL, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("%w: parse postgres dsn: invalid value redacted", ErrConfig)
	}
	query := parsedURL.Query()
	for _, key := range postgresFileDefaultDSNKeys {
		query.Set(key.name, "")
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

// postgresProbeAddressFromPoolConfig extracts the single tcp host:port target.
func postgresProbeAddressFromPoolConfig(config *pgxpool.Config) (string, error) {
	if config == nil || config.ConnConfig == nil {
		return "", fmt.Errorf("%w: invalid postgres pool config", ErrConfig)
	}
	if len(config.ConnConfig.Fallbacks) > 0 {
		return "", fmt.Errorf("%w: postgres dsn fallback targets are not supported", ErrConfig)
	}

	host := strings.TrimSpace(config.ConnConfig.Host)
	port := config.ConnConfig.Port
	if host == "" || port == 0 {
		return "", fmt.Errorf("%w: postgres dsn requires valid single tcp host and port", ErrConfig)
	}
	if network, _ := pgconn.NetworkAddress(host, port); network != "tcp" {
		return "", fmt.Errorf("%w: postgres dsn requires valid single tcp host and port", ErrConfig)
	}
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}
