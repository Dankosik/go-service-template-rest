package config

// profile:authn-oidc-jwt:start

import (
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

// validateAuthnConfig applies the shared trust rules at configuration load, so a
// bad issuer, audience, or proxy CIDR stops the process instead of surfacing at
// authn startup. internal/authntrust owns the rules and why they live in a leaf
// package: this one may not import the verifier that applies them too.
//
// It also rewrites what it accepts, which is more than the trimming the other
// validators here do: trusted_proxy_cidrs is stored back in masked canonical
// form, so 10.0.0.5/8 is held as 10.0.0.0/8. That rewritten string is the one
// startup_authn.go hands oidcjwt.NewPolicy, which runs it through the same
// parse and so cannot reach a different set of peers.
func validateAuthnConfig(cfg *AuthnConfig) error {
	cfg.Issuer = strings.TrimSpace(cfg.Issuer)
	cfg.Audience = strings.TrimSpace(cfg.Audience)
	cfg.TrustedProxyCIDRs = strings.TrimSpace(cfg.TrustedProxyCIDRs)

	if !authntrust.ValidIssuerURL(cfg.Issuer) {
		return fmt.Errorf(
			"%w: authn.issuer must be an absolute HTTPS URL without user info, query, or fragment",
			ErrValidate,
		)
	}
	if cfg.Audience == "" {
		return fmt.Errorf("%w: authn.audience cannot be empty", ErrValidate)
	}

	trusted, err := authntrust.ParseProxyCIDRs(cfg.TrustedProxyCIDRs)
	if err != nil {
		return fmt.Errorf("%w: authn.%w", ErrValidate, err)
	}
	canonical := make([]string, 0, len(trusted))
	for _, prefix := range trusted {
		canonical = append(canonical, prefix.String())
	}
	cfg.TrustedProxyCIDRs = strings.Join(canonical, ",")
	return nil
}

// profile:authn-oidc-jwt:end
