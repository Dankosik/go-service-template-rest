package config

import (
	"fmt"
	"net/url"
	"strings"
)

// OutboundAuthConfig is the immutable source tuple for one OAuth client.
type OutboundAuthConfig struct {
	TokenURL     string `koanf:"token_url"`
	ClientID     string `koanf:"client_id"`
	ClientSecret string `koanf:"client_secret"`
	Scopes       string `koanf:"scopes"`
	Audience     string `koanf:"audience"`
	Resource     string `koanf:"resource"`
}

func outboundAuthDefaults() map[string]any {
	return map[string]any{
		"outbound_auth.token_url":     "",
		"outbound_auth.client_id":     "",
		"outbound_auth.client_secret": "",
		"outbound_auth.scopes":        "",
		"outbound_auth.audience":      "",
		"outbound_auth.resource":      "",
	}
}

func validateOutboundAuthConfig(cfg *OutboundAuthConfig) error {
	cfg.TokenURL = strings.TrimSpace(cfg.TokenURL)
	cfg.Scopes = strings.Join(strings.Fields(cfg.Scopes), " ")
	if cfg.ClientID == "" {
		return fmt.Errorf("%w: outbound_auth.client_id is required", ErrValidate)
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return fmt.Errorf("%w: outbound_auth.client_secret is required", ErrValidate)
	}
	endpoint, err := url.Parse(cfg.TokenURL)
	if err != nil || endpoint == nil || !endpoint.IsAbs() || endpoint.Opaque != "" ||
		!strings.EqualFold(endpoint.Scheme, "https") || endpoint.Host == "" || endpoint.Hostname() == "" ||
		endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return fmt.Errorf("%w: outbound_auth.token_url must be an absolute HTTPS URL", ErrValidate)
	}
	endpoint.Scheme = "https"
	endpoint.Host = strings.ToLower(endpoint.Host)
	cfg.TokenURL = endpoint.String()
	if cfg.Audience != "" && cfg.Resource != "" {
		return fmt.Errorf("%w: outbound_auth.audience and outbound_auth.resource are mutually exclusive", ErrValidate)
	}
	return nil
}
