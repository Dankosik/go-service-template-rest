package config

import "strings"

const (
	RedisModeCache = "cache"
	RedisModeStore = "store"
)

// ModeValue returns the normalized Redis mode for config policy decisions.
func (cfg RedisConfig) ModeValue() string {
	return normalizeRedisMode(cfg.Mode)
}

func normalizeRedisMode(mode string) string {
	return strings.ToLower(strings.TrimSpace(mode))
}

// StoreMode reports whether Redis is configured for store-mode guard behavior.
func (cfg RedisConfig) StoreMode() bool {
	return cfg.ModeValue() == RedisModeStore
}
