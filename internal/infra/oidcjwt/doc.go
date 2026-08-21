// Package oidcjwt turns one configured OIDC issuer and audience into a verified
// request principal for the service's HTTP and gRPC adapters.
//
// golang-jwt validates JWT structure, signatures, issuer, audience, and time;
// keyfunc owns cached JWKS selection and refresh. This package retains only the
// repository policy around discovery/egress, principal normalization, carrier
// removal, private error mapping, stream expiry, and bounded telemetry.
//
// The default resource-server profile accepts mainstream JWT access-token
// dialects. The explicit RFC 9068 profile additionally requires at+jwt typing,
// client_id, iat, and jti. Neither profile authorizes roles, scopes, or tenants.
package oidcjwt
