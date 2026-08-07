package authntrust

import (
	"errors"
	"net/netip"
	"strings"
)

// ParseProxyCIDRs parses a comma-separated trusted-proxy list into masked,
// de-duplicated prefixes, keeping the configured order.
//
// Masking is what makes the answer canonical: 10.0.0.5/8 and 10.0.0.0/8 name the
// same network, so they are one entry, and a list holding both is a duplicate
// rather than two rules. Returning prefixes rather than strings is what lets one
// caller store the canonical spelling and the other match a peer address against
// the same parse, with no second interpretation between them.
//
// A blank entry is skipped, so a trailing comma is not an error. An empty result
// is: a deployment naming no trusted peer would have nothing to check a
// forwarded transport claim against, and the fail-closed reading of that is to
// refuse the configuration rather than every request.
func ParseProxyCIDRs(raw string) ([]netip.Prefix, error) {
	trusted := make([]netip.Prefix, 0)
	seen := make(map[netip.Prefix]struct{})
	for value := range strings.SplitSeq(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, errors.New("trusted_proxy_cidrs contains an invalid CIDR")
		}
		prefix = prefix.Masked()
		if _, exists := seen[prefix]; exists {
			return nil, errors.New("trusted_proxy_cidrs contains a duplicate CIDR")
		}
		seen[prefix] = struct{}{}
		trusted = append(trusted, prefix)
	}
	if len(trusted) == 0 {
		return nil, errors.New("trusted_proxy_cidrs must contain at least one CIDR")
	}
	return trusted, nil
}
