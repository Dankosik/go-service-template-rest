package config

import "strings"

const (
	RedisModeCache = "cache"
	RedisModeStore = "store"
)

// ModeValue returns the normalized Redis mode for config policy decisions.
func (cfg RedisConfig) ModeValue() string {
	return strings.ToLower(strings.TrimSpace(cfg.Mode))
}

// StoreMode reports whether Redis is configured for store-mode guard behavior.
func (cfg RedisConfig) StoreMode() bool {
	return cfg.ModeValue() == RedisModeStore
}

// RedisReadinessProbeRequired reports whether Redis participates in runtime readiness.
func (cfg Config) RedisReadinessProbeRequired() bool {
	return cfg.Redis.Enabled && (cfg.FeatureFlags.RedisReadinessProbe || cfg.Redis.StoreMode())
}
