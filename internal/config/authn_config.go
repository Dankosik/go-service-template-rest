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
	// profile:authn-oidc-introspection:start
	IntrospectionEndpoint          string `koanf:"introspection_endpoint"`
	IntrospectionTargetClass       string `koanf:"introspection_target_class"`
	IntrospectionPrivateHostSuffix string `koanf:"introspection_private_host_suffix"`
	IntrospectionClientID          string `koanf:"introspection_client_id"`
	IntrospectionClientSecret      string `koanf:"introspection_client_secret"`
	// profile:authn-oidc-introspection:end
}

func authnDefaults() map[string]any {
	// Trust policy has no executable placeholder. OIDC profiles fail config
	// validation until deployment supplies every required value.
	return map[string]any{
		"authn.issuer":        "",
		"authn.audience":      "",
		"authn.token_profile": authntrust.TokenProfileResourceServer,
		// profile:authn-oidc-introspection:start
		"authn.introspection_endpoint":            "",
		"authn.introspection_target_class":        "",
		"authn.introspection_private_host_suffix": "",
		"authn.introspection_client_id":           "",
		"authn.introspection_client_secret":       "",
		// profile:authn-oidc-introspection:end
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
	// profile:authn-oidc-introspection:start
	if introspectionConfigured(cfg) {
		if err := validateIntrospectionConfig(cfg); err != nil {
			return err
		}
	}
	// profile:authn-oidc-introspection:end
	return nil
}

// profile:authn-oidc-introspection:start

func introspectionConfigured(cfg *AuthnConfig) bool {
	return cfg.IntrospectionEndpoint != "" ||
		cfg.IntrospectionTargetClass != "" ||
		cfg.IntrospectionPrivateHostSuffix != "" ||
		cfg.IntrospectionClientID != "" ||
		cfg.IntrospectionClientSecret != ""
}

func validateIntrospectionConfig(cfg *AuthnConfig) error {
	if !authntrust.ValidIntrospectionEndpoint(cfg.IntrospectionEndpoint) {
		return fmt.Errorf(
			"%w: authn.introspection_endpoint must be an absolute HTTPS URL without user info, query, or fragment",
			ErrValidate,
		)
	}
	if !authntrust.ValidIntrospectionTargetClass(cfg.IntrospectionTargetClass) {
		return fmt.Errorf(
			"%w: authn.introspection_target_class must be one of external-https or private-https",
			ErrValidate,
		)
	}
	if cfg.IntrospectionTargetClass == authntrust.TargetClassPrivateHTTPS {
		if strings.TrimSpace(cfg.IntrospectionPrivateHostSuffix) == "" {
			return fmt.Errorf("%w: authn.introspection_private_host_suffix is required for private-https", ErrValidate)
		}
	} else if cfg.IntrospectionPrivateHostSuffix != "" {
		return fmt.Errorf("%w: authn.introspection_private_host_suffix is forbidden for external-https", ErrValidate)
	}
	if cfg.IntrospectionClientID == "" {
		return fmt.Errorf("%w: authn.introspection_client_id cannot be empty", ErrValidate)
	}
	if cfg.IntrospectionClientSecret == "" {
		return fmt.Errorf("%w: authn.introspection_client_secret cannot be empty", ErrValidate)
	}
	return nil
}

// profile:authn-oidc-introspection:end

// profile:authn-oidc-jwt:end
