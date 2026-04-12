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

var recognizedPostgresEnvVars = []string{
	"PGHOST",
	"PGPORT",
	"PGDATABASE",
	"PGUSER",
	"PGPASSWORD",
	"PGPASSFILE",
	"PGSERVICE",
	"PGSERVICEFILE",
	"PGSSLMODE",
	"PGSSLCERT",
	"PGSSLKEY",
	"PGSSLROOTCERT",
	"PGSSLPASSWORD",
	"PGSSLSNI",
	"PGAPPNAME",
	"PGCONNECT_TIMEOUT",
	"PGTARGETSESSIONATTRS",
}

var requiredPostgresDSNSettings = []string{
	"host",
	"port",
	"user",
	"password",
	"database",
	"sslmode",
}

var disallowedPostgresDSNKeys = map[string]string{
	"service":     "postgres dsn uses unsupported service/passfile source",
	"servicefile": "postgres dsn uses unsupported service/passfile source",
	"passfile":    "postgres dsn uses unsupported service/passfile source",
	"sslcert":     "postgres dsn uses unsupported TLS file source",
	"sslkey":      "postgres dsn uses unsupported TLS file source",
	"sslpassword": "postgres dsn uses unsupported TLS file source",
	"sslrootcert": "postgres dsn uses unsupported TLS file source",
}

var allowedPostgresSSLModes = map[string]struct{}{
	"disable":     {},
	"require":     {},
	"verify-ca":   {},
	"verify-full": {},
}

var pgxFileDefaultOverrideKeys = []string{
	"passfile",
	"sslcert",
	"sslkey",
	"sslrootcert",
	"sslpassword",
}

type postgresTarget struct {
	host string
	port uint16
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
	if _, err := postgresTargetFromPoolConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

func preflightPostgresDSN(rawDSN string) (string, error) {
	dsn := strings.TrimSpace(rawDSN)
	if dsn == "" {
		return "", fmt.Errorf("%w: postgres dsn is empty", ErrConfig)
	}
	for _, name := range recognizedPostgresEnvVars {
		if os.Getenv(name) != "" {
			return "", fmt.Errorf("%w: postgres dsn uses unsupported ambient PG environment", ErrConfig)
		}
	}

	settings, isURL, err := parsePostgresDSNSettings(dsn)
	if err != nil {
		return "", fmt.Errorf("%w: parse postgres dsn: invalid value redacted", ErrConfig)
	}
	if err := validatePostgresDSNSettings(settings); err != nil {
		return "", err
	}
	if isURL {
		return normalizePostgresURLDSN(dsn)
	}
	return normalizePostgresKeywordValueDSN(dsn), nil
}

func parsePostgresDSNSettings(dsn string) (map[string]string, bool, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		settings, err := parsePostgresURLDSNSettings(dsn)
		return settings, true, err
	}
	settings, err := parsePostgresKeywordValueDSNSettings(dsn)
	return settings, false, err
}

func parsePostgresURLDSNSettings(dsn string) (map[string]string, error) {
	settings := make(map[string]string)

	parsedURL, err := url.Parse(dsn)
	if err != nil {
		if urlErr := new(url.Error); errors.As(err, &urlErr) {
			return nil, urlErr.Err
		}
		return nil, err
	}
	if parsedURL.User != nil {
		settings["user"] = parsedURL.User.Username()
		if password, present := parsedURL.User.Password(); present {
			settings["password"] = password
		}
	}

	var hosts []string
	var ports []string
	for _, host := range strings.Split(parsedURL.Host, ",") {
		if host == "" {
			continue
		}
		if postgresURLHostIsIPOnly(host) {
			hosts = append(hosts, strings.Trim(host, "[]"))
			continue
		}
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			return nil, err
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

func postgresURLHostIsIPOnly(host string) bool {
	return net.ParseIP(strings.Trim(host, "[]")) != nil || !strings.Contains(host, ":")
}

func parsePostgresKeywordValueDSNSettings(dsn string) (map[string]string, error) {
	settings := make(map[string]string)
	remaining := dsn

	for len(remaining) > 0 {
		eqIdx := strings.IndexRune(remaining, '=')
		if eqIdx < 0 {
			return nil, errors.New("invalid keyword/value")
		}

		key := strings.Trim(remaining[:eqIdx], " \t\n\r\v\f")
		remaining = strings.TrimLeft(remaining[eqIdx+1:], " \t\n\r\v\f")
		value, rest, err := nextPostgresKeywordValue(remaining)
		if err != nil {
			return nil, err
		}
		if key == "" {
			return nil, errors.New("invalid keyword/value")
		}
		if key == "dbname" {
			key = "database"
		}
		settings[key] = value
		remaining = rest
	}
	return settings, nil
}

func nextPostgresKeywordValue(s string) (string, string, error) {
	if s == "" {
		return "", "", nil
	}
	if s[0] == '\'' {
		return nextQuotedPostgresKeywordValue(s[1:])
	}
	end := 0
	for end < len(s) {
		if asciiSpace[s[end]] == 1 {
			break
		}
		if s[end] == '\\' {
			end++
			if end == len(s) {
				return "", "", errors.New("invalid backslash")
			}
		}
		end++
	}
	value := unescapePostgresKeywordValue(s[:end])
	if end == len(s) {
		return value, "", nil
	}
	return value, s[end+1:], nil
}

func nextQuotedPostgresKeywordValue(s string) (string, string, error) {
	end := 0
	for end < len(s) {
		if s[end] == '\'' {
			value := unescapePostgresKeywordValue(s[:end])
			if end == len(s) {
				return value, "", nil
			}
			return value, s[end+1:], nil
		}
		if s[end] == '\\' {
			end++
		}
		end++
	}
	return "", "", errors.New("unterminated quoted string in connection info string")
}

func unescapePostgresKeywordValue(value string) string {
	value = strings.ReplaceAll(value, `\\`, `\`)
	return strings.ReplaceAll(value, `\'`, `'`)
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

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
		return "", err
	}
	query := parsedURL.Query()
	for _, key := range pgxFileDefaultOverrideKeys {
		query.Set(key, "")
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func normalizePostgresKeywordValueDSN(dsn string) string {
	var normalized strings.Builder
	normalized.Grow(len(dsn) + len(pgxFileDefaultOverrideKeys)*16)
	normalized.WriteString(dsn)
	for _, key := range pgxFileDefaultOverrideKeys {
		normalized.WriteByte(' ')
		normalized.WriteString(key)
		normalized.WriteString("=''")
	}
	return normalized.String()
}

func postgresTargetFromPoolConfig(config *pgxpool.Config) (postgresTarget, error) {
	if config == nil || config.ConnConfig == nil {
		return postgresTarget{}, fmt.Errorf("%w: invalid postgres pool config", ErrConfig)
	}
	if len(config.ConnConfig.Fallbacks) > 0 {
		return postgresTarget{}, fmt.Errorf("%w: postgres dsn fallback targets are not supported", ErrConfig)
	}

	host := strings.TrimSpace(config.ConnConfig.Host)
	port := config.ConnConfig.Port
	if host == "" || port == 0 {
		return postgresTarget{}, fmt.Errorf("%w: postgres dsn requires valid single tcp host and port", ErrConfig)
	}
	if network, _ := pgconn.NetworkAddress(host, port); network != "tcp" {
		return postgresTarget{}, fmt.Errorf("%w: postgres dsn requires valid single tcp host and port", ErrConfig)
	}
	return postgresTarget{host: host, port: port}, nil
}

func (target postgresTarget) address() string {
	return net.JoinHostPort(target.host, strconv.Itoa(int(target.port)))
}
