package authntrust

import (
	"strings"

	"github.com/example/go-service-template-rest/internal/outboundtrust"
)

// ValidIssuerURL reports whether raw is a configured OIDC issuer this service
// may trust. Query and fragment are forbidden because issuer comparison and the
// derived Discovery URL must each have one exact reading.
func ValidIssuerURL(raw string) bool {
	return validHTTPSTarget(raw, false)
}

// ValidJWKSURL reports whether raw is a provider-discovered JWKS endpoint this
// service may fetch. Provider-owned query parameters are part of that endpoint;
// user info and fragments still have no place in an outbound trust request.
func ValidJWKSURL(raw string) bool {
	return validHTTPSTarget(raw, true)
}

// validHTTPSTarget applies outboundtrust.HTTPSTarget without repairing raw: a
// trust value with surrounding whitespace is rejected, not trimmed.
func validHTTPSTarget(raw string, allowQuery bool) bool {
	if strings.TrimSpace(raw) != raw {
		return false
	}
	_, issue := outboundtrust.HTTPSTarget(raw)
	return issue == outboundtrust.TargetOK || allowQuery && issue == outboundtrust.TargetHasQuery
}
