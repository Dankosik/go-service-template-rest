package oauth2clientcredentials

import (
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

// Config binds one OAuth client to one token endpoint.
type Config struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string

	// Audience and Resource are optional, mutually exclusive provider extensions.
	Audience string
	Resource string

	// TokenTargetClass defaults to ExternalHTTPS. PrivateHTTPS requires the
	// dependency-owned private suffix.
	TokenTargetClass       httpclient.TargetClass
	TokenPrivateHostSuffix string
}

func validateConfig(cfg Config) (Config, error) {
	cfg.TokenURL = strings.TrimSpace(cfg.TokenURL)
	cfg.Scopes = slices.Clone(cfg.Scopes)
	if cfg.TokenTargetClass == 0 {
		cfg.TokenTargetClass = httpclient.ExternalHTTPS
	}
	if cfg.TokenURL == "" || cfg.ClientID == "" || strings.TrimSpace(cfg.ClientSecret) == "" ||
		cfg.Audience != "" && cfg.Resource != "" {
		return Config{}, ErrInvalidConfiguration
	}
	return cfg, nil
}
