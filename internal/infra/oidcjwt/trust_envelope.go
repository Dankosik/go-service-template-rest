package oidcjwt

import "time"

// Fixed safe defaults. Issuer and audience remain deployment inputs; changing
// the algorithm or provider-work bounds is a code-reviewed trust decision.
// Shared token size and clock-skew bounds live in bearerauthn.
const (
	maxProviderBody        = 1 << 20
	maxProviderHeaderBytes = 32 << 10
	maxProviderInFlight    = 1
	allowedAlgorithm       = "RS256"
	providerTimeout        = 5 * time.Second
	refreshInterval        = 15 * time.Minute
	refreshCooldown        = 30 * time.Second
)
