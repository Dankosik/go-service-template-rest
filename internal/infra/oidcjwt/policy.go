package oidcjwt

import (
	"errors"
	"net/netip"
	"strings"
)

// Policy is the immutable trust configuration shared by both transports. The
// capability limits no deployment may change are in trust_envelope.go.
type Policy struct {
	issuer         string
	audience       string
	trustedProxies []netip.Prefix
}

// PolicyInput carries the configured trust values one deployment holds. doc.go's
// "Adding a configured trust value" owns the full list of what a new field
// touches.
//
// The exhaustruct include entry for this type in .golangci.yml is load-bearing: a
// field added here fails lint at every production call site that does not set it.
// Remove this type from that entry and a forgotten call site becomes a zero value
// at an authentication boundary instead of a failed build.
//
// Test call sites are exempt, because that config excludes exhaustruct from
// _test.go repository-wide. That is the right trade here: a test that omits a
// new field is stating it does not vary that value, while a composition root
// that omits one is shipping it unset.
type PolicyInput struct {
	Issuer            string
	Audience          string
	TrustedProxyCIDRs string
}

// NewPolicy validates and copies the complete trust policy.
//
// The issuer is held to validProviderURL, the same shape every URL this package
// will fetch from must have. One owner for that shape inside this package is
// deliberate: the issuer and the discovered JWKS URI are both provider-supplied
// endpoints, so a constraint added for one is a constraint the other needs too.
//
// internal/config's validateAuthnConfig restates these rules so a bad value
// fails at configuration load instead of at authn startup. validProviderURL owns
// why that copy cannot be removed and what holds the two sides to one answer.
func NewPolicy(input PolicyInput) (Policy, error) {
	issuer := strings.TrimSpace(input.Issuer)
	audience := strings.TrimSpace(input.Audience)
	if !validProviderURL(issuer) {
		return Policy{}, errors.New("authn issuer must be an absolute HTTPS URL without user info, query, or fragment")
	}
	if audience == "" {
		return Policy{}, errors.New("authn audience cannot be empty")
	}

	parts := strings.Split(input.TrustedProxyCIDRs, ",")
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
		trustedProxies: trusted,
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
