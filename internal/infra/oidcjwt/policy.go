package oidcjwt

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/example/go-service-template-rest/internal/authntrust"
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
// The rules are internal/authntrust's, and the configuration loader applies the
// same ones earlier: a deployment value refused here should already have stopped
// the process at configuration load, so reaching an error from this function
// means the input came from somewhere other than a loaded snapshot. That package
// owns why the rules cannot live in either caller.
func NewPolicy(input PolicyInput) (Policy, error) {
	issuer := strings.TrimSpace(input.Issuer)
	audience := strings.TrimSpace(input.Audience)
	if !authntrust.ValidIssuerURL(issuer) {
		return Policy{}, errors.New("authn issuer must be an absolute HTTPS URL without user info, query, or fragment")
	}
	if audience == "" {
		return Policy{}, errors.New("authn audience cannot be empty")
	}
	trusted, err := authntrust.ParseProxyCIDRs(input.TrustedProxyCIDRs)
	if err != nil {
		return Policy{}, fmt.Errorf("authn %w", err)
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
