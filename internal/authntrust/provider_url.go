package authntrust

import (
	"net/url"
	"strings"
)

// ValidIssuerURL reports whether raw is a configured OIDC issuer this service
// may trust. Query and fragment are forbidden because issuer comparison and the
// derived Discovery URL must each have one exact reading.
func ValidIssuerURL(raw string) bool {
	parsed, ok := validHTTPSURL(raw)
	return ok && parsed.RawQuery == "" && !parsed.ForceQuery
}

// ValidJWKSURL reports whether raw is a provider-discovered JWKS endpoint this
// service may fetch. Provider-owned query parameters are part of that endpoint;
// user info and fragments still have no place in an outbound trust request.
func ValidJWKSURL(raw string) bool {
	_, ok := validHTTPSURL(raw)
	return ok
}

func validHTTPSURL(raw string) (url.URL, bool) {
	if strings.TrimSpace(raw) != raw {
		return url.URL{}, false
	}
	parsed, err := url.Parse(raw)
	// The parsed nil check is not redundant. url.Parse's contract pairs a nil
	// result with a non-nil error, but nilaway reads the two as independent, and
	// dropping the guard fails the deep lint gate rather than a test.
	if err != nil || parsed == nil {
		return url.URL{}, false
	}
	return *parsed, parsed.IsAbs() &&
		strings.EqualFold(parsed.Scheme, "https") &&
		parsed.Host != "" &&
		parsed.Hostname() != "" &&
		parsed.Opaque == "" &&
		parsed.User == nil &&
		parsed.Fragment == ""
}
