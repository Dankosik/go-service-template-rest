// Package authntrust owns the two deployment trust rules an OIDC JWT
// authentication boundary is configured with: which URLs this service may fetch
// provider documents from, and which peers may state the transport a credential
// arrived over.
//
// It is a leaf on purpose, and it exists because two owners need one answer at
// two different times. internal/config must refuse a bad value at configuration
// load, before any verifier exists, so a mistyped key fails the process instead
// of the authentication boundary. internal/infra/oidcjwt must enforce the same
// rules when it builds its policy, and provider_url.go's rule again at runtime,
// against the JWKS URI the provider's own Discovery document names. The
// depguard rule config_no_runtime_owners stops internal/config from importing
// runtime adapters, so neither owner may import the other and the rules would
// otherwise live in two copies held in step by a parity test.
//
// Each rule keeps its own file because the two change for unrelated reasons:
// one governs requests this service makes, the other governs requests it
// receives. Messages name the configuration setting rather than the Go
// parameter, so each caller prefixes its own section and still produces an
// operator-actionable key.
//
// What stays out: this package holds no configured value and builds no policy
// object. It answers about strings, so a caller may ask before it has anywhere
// to put the answer.
package authntrust
