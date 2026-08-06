package oidcjwt

import (
	"errors"
	"net/netip"
	"slices"
	"strings"
	"time"
)

// Fixed capability policy — the half of this file that a deployment cannot
// change, as against [Policy] below, which is entirely what a deployment
// supplies. Two things named policy live here for that reason: this block is the
// envelope, and that type is the trust values inside it.
//
// These are deliberately constants rather than configuration: a provider that
// could widen the accepted algorithm, the size caps, or the staleness limit could
// weaken authentication from outside the service. They are collected here so the
// whole trust envelope reads in one place, and docs/authentication.md publishes
// the same values to operators. The consumers are elsewhere — timing in
// lifecycle.go and refresh.go, size caps in token.go, keyset.go, and provider.go.
const (
	// MaxTokenBytes bounds the credential before any parsing. Access tokens with
	// ordinary claim sets are well under 2 KiB, so this leaves room for large
	// provider claims while keeping an oversized header cheap to reject.
	MaxTokenBytes = 8 << 10
	// MaxProviderBody bounds a Discovery or JWKS response body. It is enforced
	// both by the HTTP client and again after reading, so a provider that
	// ignores the transport limit still cannot grow the parse.
	MaxProviderBody = 1 << 20
	// MaxJWKEntries bounds how many JWK entries one key set may declare, which
	// bounds the RSA parsing a single refresh can be made to do.
	MaxJWKEntries = 100
	// AllowedAlgorithm is the only accepted signature algorithm, in the protected
	// header and in JWK metadata alike. A single value is the point: with nothing
	// to negotiate there is no algorithm substitution to defend against.
	AllowedAlgorithm = "RS256"
	// ClockSkew is the allowance applied to exp, iat, and nbf in both directions.
	// It absorbs ordinary drift between the provider's clock and this host
	// without extending a token's life in any way an operator would notice.
	ClockSkew = 30 * time.Second
	// ProviderTimeout bounds one Discovery or JWKS request, both as the response
	// header timeout and as the whole-request budget. Startup spends at most two
	// of these before failing closed.
	ProviderTimeout = 5 * time.Second
	// RefreshInterval is the scheduled refresh cadence on the success path. It is
	// deliberately one third of MaxKeySetAge, so a healthy provider replaces the
	// set three times over before it could go stale and a single missed refresh
	// never reaches requests. Changing either value without the other changes
	// how much provider outage the service absorbs.
	RefreshInterval = 5 * time.Minute
	// MaxKeySetAge is how long a completely validated key set stays usable
	// without replacement. Past it, verification and readiness both fail closed,
	// because a key revoked at the provider can no longer be observed here.
	MaxKeySetAge = 15 * time.Minute
	// RefreshCooldown rate-limits the token-driven refresh, so a stream of
	// unknown key ids from an attacker cannot turn into provider load. It doubles
	// as the retry backoff after a failed scheduled refresh: recovery is retried
	// at this cadence rather than waiting out another RefreshInterval, which is
	// what lets a short outage heal well inside MaxKeySetAge.
	RefreshCooldown = 30 * time.Second
	// providerHeaderLimit bounds response headers from the provider. It is
	// unexported because no caller outside the provider client can act on it.
	providerHeaderLimit = 32 << 10
)

// Policy is the immutable trust configuration shared by both transports.
type Policy struct {
	issuer         string
	audience       string
	trustedProxies []netip.Prefix
}

// PolicyInput carries the configured trust values one deployment holds.
//
// It is a struct rather than positional parameters because every field is a
// string, so the pair most worth protecting — issuer and audience — would
// transpose and still compile. The exhaustruct include entry for this type in
// .golangci.yml preserves the one property positional parameters gave for free: a
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
