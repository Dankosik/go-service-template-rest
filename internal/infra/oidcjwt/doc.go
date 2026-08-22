// Package oidcjwt is the JWT/JWKS trust engine for the shared bearer runtime.
//
// golang-jwt validates JWT structure, signatures, issuer, audience, and time;
// keyfunc owns cached JWKS selection and refresh. This package retains only the
// repository policy around discovery/egress, principal normalization, and JWKS
// refresh telemetry. Inbound bearer grammar, transport adapters, and the shared
// verification counter live in internal/infra/bearerauthn.
//
// The default resource-server profile accepts mainstream JWT access-token
// dialects. The explicit RFC 9068 profile additionally requires at+jwt typing,
// client_id, iat, and jti. Neither profile authorizes roles, scopes, or tenants.
package oidcjwt
