package authntrust

// profile:authn-oidc-jwt:start

const (
	TokenProfileResourceServer = "resource-server"
	TokenProfileRFC9068        = "rfc9068"
)

// ValidTokenProfile reports whether raw names one supported access-token
// contract. The ordinary resource-server profile is the default; RFC 9068 is
// explicit because mainstream issuers still ship other JWT dialects.
func ValidTokenProfile(raw string) bool {
	return raw == TokenProfileResourceServer || raw == TokenProfileRFC9068
}

// profile:authn-oidc-jwt:end
