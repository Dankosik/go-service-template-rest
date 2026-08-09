package oidcjwt

import "time"

// The fixed trust envelope: the capability limits a deployment cannot change,
// as against [Policy] in policy.go, which is entirely what a deployment
// supplies.
//
// These are deliberately constants rather than configuration: a provider that
// could widen the accepted algorithm, the size caps, or the staleness limit could
// weaken authentication from outside the service. They are collected here so the
// whole envelope reads in one place, and docs/authentication.md publishes the
// same values to operators. The consumers are elsewhere — timing in
// lifecycle.go, size caps in token.go, keyset.go, and provider.go.
const (
	// MaxTokenBytes bounds the credential before any parsing. Access tokens with
	// ordinary claim sets are well under 2 KiB, so this leaves room for large
	// provider claims while keeping an oversized header cheap to reject.
	MaxTokenBytes = 8 << 10
	// maxNumericDateLiteral and maxNumericDateExponent bound one NumericDate
	// claim before math/big is asked to expand it, which MaxTokenBytes cannot do
	// on its own: an exponent is a handful of bytes that names an arbitrarily
	// large number. math/big refuses only a base-10 exponent above a million, so
	// without these the nine-byte literal 1e1000000 buys a million-digit
	// exponentiation — and parseAccessTokenClaims runs before any signature has
	// been checked, so that is work an unauthenticated caller can ask for. A date
	// this service can act on names a second within int64 nanoseconds of the
	// epoch, which no legal spelling needs more room than this to express. They
	// are unexported because no caller outside token.go can act on them.
	maxNumericDateLiteral  = 40
	maxNumericDateExponent = 30
	// MaxProviderBody bounds a Discovery or JWKS response body. It is enforced
	// both by the HTTP client and again after reading, so a provider that
	// ignores the transport limit still cannot grow the parse.
	MaxProviderBody = 1 << 20
	// MaxJWKEntries bounds how many JWK entries one key set may declare, which
	// bounds the RSA parsing a single refresh can be made to do.
	MaxJWKEntries = 100
	// AllowedAlgorithm is the only accepted signature algorithm, and the only
	// spelling of it: parseProtectedHeader, parseToken's jose parse, and
	// compatibleRSAKey all compare against this constant, which is untyped so the
	// jose one needs no conversion. A single value is the point — with nothing to
	// negotiate there is no algorithm substitution to defend against. doc.go's
	// "Extending it" owns what a second algorithm costs.
	AllowedAlgorithm = "RS256"
	// ClockSkew is the allowance applied to exp, iat, and nbf in both directions.
	// It absorbs ordinary drift between the provider's clock and this host
	// without extending a token's life in any way an operator would notice.
	ClockSkew = 30 * time.Second
	// MaxTokenLifetime bounds how long an issued access token may remain valid.
	// Without it, a provider mistake could turn one signed token into a credential
	// that outlives every key-refresh and revocation signal this service observes.
	MaxTokenLifetime = 15 * time.Minute
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
	// keysCurrent in verifier.go is the predicate every caller asks.
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
