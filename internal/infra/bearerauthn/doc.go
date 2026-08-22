// Package bearerauthn is the shared inbound bearer-authentication runtime.
//
// It owns credential grammar and removal, the sanitized failure taxonomy,
// verification telemetry, principal publication, the HTTP OpenAPI resolver,
// native gRPC public-method policy, and stream expiry. Concrete trust engines
// implement [Verifier]; this package never discovers an issuer, parses a JWT,
// calls an introspection endpoint, or selects a profile at runtime.
package bearerauthn
