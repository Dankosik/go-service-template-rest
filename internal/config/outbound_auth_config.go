package config

import (
	"fmt"
	"net/url"
	"strings"
)

// OutboundAuthConfig is the immutable source tuple for one named OAuth client.
type OutboundAuthConfig struct {
	TokenURL     string `koanf:"token_url"`
	ClientID     string `koanf:"client_id"`
	ClientSecret string `koanf:"client_secret"`
	Scopes       string `koanf:"scopes"`
}

func validateOutboundAuthConfig(cfg *OutboundAuthConfig, prefix string) error {
	cfg.TokenURL = strings.TrimSpace(cfg.TokenURL)
	cfg.Scopes = strings.Join(strings.Fields(cfg.Scopes), " ")
	if cfg.ClientID == "" {
		return fmt.Errorf("%w: %s.client_id is required", ErrValidate, prefix)
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return fmt.Errorf("%w: %s.client_secret is required", ErrValidate, prefix)
	}
	endpoint, err := url.Parse(cfg.TokenURL)
	if err != nil || endpoint == nil || !endpoint.IsAbs() || endpoint.Opaque != "" ||
		!strings.EqualFold(endpoint.Scheme, "https") || endpoint.Host == "" || endpoint.Hostname() == "" ||
		endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return fmt.Errorf("%w: %s.token_url must be an absolute HTTPS URL", ErrValidate, prefix)
	}
	endpoint.Scheme = "https"
	endpoint.Host = strings.ToLower(endpoint.Host)
	cfg.TokenURL = endpoint.String()
	return nil
}
