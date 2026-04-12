package config

// PostgresReadinessProbeRequired reports whether Postgres participates in runtime readiness.
func (cfg Config) PostgresReadinessProbeRequired() bool {
	return cfg.Postgres.Enabled && cfg.FeatureFlags.PostgresReadinessProbe
}

// RedisReadinessProbeRequired reports whether Redis participates in runtime readiness.
func (cfg Config) RedisReadinessProbeRequired() bool {
	return cfg.Redis.Enabled && (cfg.FeatureFlags.RedisReadinessProbe || cfg.Redis.StoreMode())
}

// MongoReadinessProbeRequired reports whether Mongo participates in runtime readiness.
func (cfg Config) MongoReadinessProbeRequired() bool {
	return cfg.Mongo.Enabled && cfg.FeatureFlags.MongoReadinessProbe
}
