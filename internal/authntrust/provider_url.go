package authntrust

import (
	"net/url"
	"strings"
)

// ValidProviderURL reports whether raw is a URL this service may fetch an OIDC
// provider document from.
//
// Requiring absolute HTTPS with no user info, query, or fragment keeps a
// provider-supplied endpoint from carrying credentials or redirect parameters
// into a request this service makes on its own behalf.
//
// One predicate serves the configured issuer and the JWKS URI that issuer's own
// Discovery document names, which is deliberate: both are provider-supplied
// endpoints, so a constraint added for one is a constraint the other needs too.
// That second caller is why this is a predicate rather than a validator
// returning an error — it is one term in a conjunction that has no message to
// attach.
//
// A constraint tightened here tightens for every deployment at once, including
// values already held in a running configuration. That is the intended blast
// radius: the alternative is a deployment holding an issuer its own verifier
// would refuse.
func ValidProviderURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	// The parsed nil check is not redundant. url.Parse's contract pairs a nil
	// result with a non-nil error, but nilaway reads the two as independent, and
	// dropping the guard fails the deep lint gate rather than a test.
	return err == nil &&
		parsed != nil &&
		parsed.IsAbs() &&
		strings.EqualFold(parsed.Scheme, "https") &&
		parsed.Host != "" &&
		parsed.Hostname() != "" &&
		parsed.Opaque == "" &&
		parsed.User == nil &&
		parsed.RawQuery == "" &&
		!parsed.ForceQuery &&
		parsed.Fragment == ""
}
