package oauth2clientcredentials

import (
	"slices"
	"strings"
)

// Config binds one OAuth client to one token endpoint.
type Config struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

func validateConfig(cfg Config) (Config, error) {
	cfg.TokenURL = strings.TrimSpace(cfg.TokenURL)
	cfg.Scopes = slices.Clone(cfg.Scopes)
	if cfg.TokenURL == "" || cfg.ClientID == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return Config{}, ErrInvalidConfiguration
	}
	return cfg, nil
}
