package oidcjwt

import (
	"errors"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	MaxTokenBytes       = 8 << 10
	MaxProviderBody     = 1 << 20
	MaxJWKEntries       = 100
	AllowedAlgorithm    = "RS256"
	ClockSkew           = 30 * time.Second
	ProviderTimeout     = 5 * time.Second
	RefreshInterval     = 5 * time.Minute
	MaxKeySetAge        = 15 * time.Minute
	RefreshCooldown     = 30 * time.Second
	providerHeaderLimit = 32 << 10
)

// Policy is the immutable trust configuration shared by both transports.
type Policy struct {
	issuer         string
	audience       string
	trustedProxies []netip.Prefix
}

// NewPolicy validates and copies the complete trust policy.
func NewPolicy(issuer, audience, trustedProxyCIDRs string) (Policy, error) {
	issuer = strings.TrimSpace(issuer)
	audience = strings.TrimSpace(audience)
	parsedIssuer, err := url.Parse(issuer)
	if err != nil ||
		!parsedIssuer.IsAbs() ||
		!strings.EqualFold(parsedIssuer.Scheme, "https") ||
		parsedIssuer.Host == "" ||
		parsedIssuer.Hostname() == "" ||
		parsedIssuer.Opaque != "" ||
		parsedIssuer.User != nil ||
		parsedIssuer.RawQuery != "" ||
		parsedIssuer.ForceQuery ||
		parsedIssuer.Fragment != "" {
		return Policy{}, errors.New("authn issuer must be an absolute HTTPS URL without user info, query, or fragment")
	}
	if audience == "" {
		return Policy{}, errors.New("authn audience cannot be empty")
	}

	parts := strings.Split(trustedProxyCIDRs, ",")
	trusted := make([]netip.Prefix, 0, len(parts))
	seen := make(map[netip.Prefix]struct{})
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		prefix, parseErr := netip.ParsePrefix(value)
		if parseErr != nil {
			return Policy{}, errors.New("authn trusted_proxy_cidrs contains an invalid CIDR")
		}
		prefix = prefix.Masked()
		if _, exists := seen[prefix]; exists {
			return Policy{}, errors.New("authn trusted_proxy_cidrs contains a duplicate CIDR")
		}
		seen[prefix] = struct{}{}
		trusted = append(trusted, prefix)
	}
	if len(trusted) == 0 {
		return Policy{}, errors.New("authn trusted_proxy_cidrs must contain at least one CIDR")
	}

	return Policy{
		issuer:         issuer,
		audience:       audience,
		trustedProxies: slices.Clone(trusted),
	}, nil
}

func (p Policy) trustedProxy(address netip.Addr) bool {
	for _, prefix := range p.trustedProxies {
		if prefix.Contains(address.Unmap()) {
			return true
		}
	}
	return false
}
