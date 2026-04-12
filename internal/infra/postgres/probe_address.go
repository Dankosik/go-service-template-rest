package postgres

// ProbeAddress extracts a probe-ready host:port from a PostgreSQL DSN.
func ProbeAddress(rawDSN string) (string, error) {
	pgxCfg, err := parsePoolConfig(rawDSN)
	if err != nil {
		return "", err
	}
	target, err := postgresTargetFromPoolConfig(pgxCfg)
	if err != nil {
		return "", err
	}
	return target.address(), nil
}
