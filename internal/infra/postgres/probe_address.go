package postgres

// ProbeAddress extracts a probe-ready host:port from a PostgreSQL DSN.
func ProbeAddress(rawDSN string) (string, error) {
	pgxCfg, err := parsePoolConfig(rawDSN)
	if err != nil {
		return "", err
	}
	return postgresProbeAddressFromPoolConfig(pgxCfg)
}
