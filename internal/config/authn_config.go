package config

// profile:authn-oidc-jwt:start

import (
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

// AuthnConfig names the immutable issuer boundary. Cryptographic, cache, size,
// and time policy is fixed by the capability rather than provider-controlled
// configuration.
type AuthnConfig struct {
	Issuer       string `koanf:"issuer"`
	Audience     string `koanf:"audience"`
	TokenProfile string `koanf:"token_profile"`
}

func authnDefaults() map[string]any {
	// Trust policy has no executable placeholder. OIDC profiles fail config
	// validation until deployment supplies every required value.
	return map[string]any{
		"authn.issuer":        "",
		"authn.audience":      "",
		"authn.token_profile": authntrust.TokenProfileResourceServer,
	}
}

// validateAuthnConfig applies the shared trust rules at configuration load, so a
// bad issuer, audience, or token profile stops the process instead of surfacing at
// authn startup. internal/authntrust owns the rules and why they live in a leaf
// package: this one may not import the verifier that applies them too.
func validateAuthnConfig(cfg *AuthnConfig) error {
	cfg.Issuer = strings.TrimSpace(cfg.Issuer)
	cfg.Audience = strings.TrimSpace(cfg.Audience)
	cfg.TokenProfile = strings.ToLower(strings.TrimSpace(cfg.TokenProfile))

	if !authntrust.ValidIssuerURL(cfg.Issuer) {
		return fmt.Errorf(
			"%w: authn.issuer must be an absolute HTTPS URL without user info, query, or fragment",
			ErrValidate,
		)
	}
	if cfg.Audience == "" {
		return fmt.Errorf("%w: authn.audience cannot be empty", ErrValidate)
	}

	if !authntrust.ValidTokenProfile(cfg.TokenProfile) {
		return fmt.Errorf(
			"%w: authn.token_profile must be one of resource-server or rfc9068",
			ErrValidate,
		)
	}
	return nil
}

// profile:authn-oidc-jwt:end
