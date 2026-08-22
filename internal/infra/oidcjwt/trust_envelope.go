package oidcjwt

import "time"

// Fixed safe defaults. Issuer and audience remain deployment inputs; changing
// the algorithm or provider-work bounds is a code-reviewed trust decision.
// Shared token size and clock-skew bounds live in bearerauthn.
const (
	MaxProviderBody  = 1 << 20
	AllowedAlgorithm = "RS256"
	ProviderTimeout  = 5 * time.Second
	RefreshInterval  = 15 * time.Minute
	RefreshCooldown  = 30 * time.Second
)
