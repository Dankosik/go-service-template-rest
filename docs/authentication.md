# OIDC/JWT Authentication

The `AUTHN=oidc-jwt` initialization profile gives HTTP and gRPC handlers one
verified `reqctx.Principal`. Feature code does not perform OIDC discovery, fetch
or refresh JWKS, parse JWTs, read credentials, map authentication errors, or
publish authentication telemetry.

The default `AUTHN=none` profile removes the verifier, configuration, tests,
dependencies, guide, OpenAPI security declaration, and bootstrap wiring.

## Initialize and configure

```bash
make template-init \
  MODULE=github.com/your-org/my-service \
  CODEOWNER=@your-org/backend \
  DATABASE=none \
  GRPC=none \
  AUTHN=oidc-jwt
```

Supply only the issuer and this API's audience:

```dotenv
APP__AUTHN__ISSUER=https://identity.example.com
APP__AUTHN__AUDIENCE=https://api.example.com
```

The issuer must be an exact HTTPS URL without user information, query, or
fragment. The audience is a non-empty resource-server identifier. The runtime
TLS trust store, DNS/egress policy, and the ingress that terminates client TLS
remain deployment-owned inputs.

`authn.token_profile` defaults to `resource-server`. Set it to `rfc9068` only
when the authorization server is configured to issue that dialect:

```dotenv
APP__AUTHN__TOKEN_PROFILE=rfc9068
```

## Default resource-server contract

The verifier accepts compact JWT access tokens that satisfy all of these rules:

- `alg=RS256`, selected by service policy rather than trusted from the token;
- a signature from the configured issuer's discovered JWKS;
- a JWK whose optional `use` is absent or `sig`, never `enc`;
- exact `iss` and an `aud` string or array containing the configured audience;
- required `exp`, optional `nbf`, and a 30-second clock-skew allowance;
- at most 32 KiB before parsing;
- at least one stable identity: `sub` or a client identifier.

The default does not require a `typ` header, `iat`, or `jti`. Mainstream issuers
use several JWT access-token dialects. Client identity is read from
`client_id`, `azp`, `appid`, or `cid`; if several are present they must agree.

The explicit `rfc9068` profile additionally requires `typ=at+jwt` (or
`application/at+jwt`), `client_id`, `iat`, and `jti`. It does not impose an
unrelated maximum token lifetime.

The verifier publishes:

```go
reqctx.Principal{
    Issuer:   verifiedIssuer,
    Subject:  subjectWhenPresent,
    ClientID: verifiedClientIDWhenPresent,
}
```

It does not convert roles, tenant IDs, permissions, or token scopes into an
authorization decision. Those remain feature-owned policy.

## HTTP and gRPC

HTTP operations inherit Bearer security from the OpenAPI contract.
`/health/live` and `/health/ready` remain public. The authentication seam removes
`Authorization` before the generated handler runs and stores the principal in
the request context.

Missing or invalid credentials return `401` with a Bearer challenge. Malformed
credentials return `400`, oversized credentials `431`, and a JWKS refresh
failure needed to validate the presented key returns `503`. Responses never
contain token, claim, parser, key, URL, provider body, or raw dependency text.

Native gRPC applies the same verifier to every application RPC.
`grpc.health.v1.Health/Check` remains public; `Health/Watch` is protected.
Authentication failures map to `Unauthenticated`, `ResourceExhausted`, or
`Unavailable`. Protected streams receive a deadline no later than token expiry
plus clock skew, and their handler-visible metadata no longer contains the
credential.

Both handler kinds read identity in the same way:

```go
principal, ok := reqctx.PrincipalFromContext(ctx)
```

On a protected operation, `ok == false` is a wiring defect and should result in
an internal failure, not a second credential parser.

## Discovery, JWKS, readiness, and shutdown

Startup performs OIDC discovery, requires the discovered issuer to match
exactly, validates an HTTPS `jwks_uri`, and installs the first JWKS before
startup succeeds. Discovery and JWKS calls use the repository's fixed-authority
public-HTTPS client with a five-second timeout, 1 MiB body limit, 32 KiB header
limit, refused redirects, post-DNS public-address checks, and one connection.

`github.com/MicahParks/keyfunc/v3` and `github.com/golang-jwt/jwt/v5` own JWT
parsing, key selection, the cached key set, periodic refresh, and unknown-`kid`
recovery. Keys refresh every 15 minutes. An unknown `kid` can trigger one
refresh per 30 seconds; followers do not queue behind the cooldown. Failed
refreshes never replace a valid cached set.

Authentication is an eager startup gate, not a dynamic readiness dependency.
After startup, a provider outage affects only requests that need unavailable new
trust; it does not evict an otherwise healthy instance. Shutdown cancels the
library refresh context and releases idle provider connections after the API
drain and before telemetry flush.

## Replay and revocation

These are bearer access tokens. Anyone holding one can replay it until expiry or
until a signing-key change makes it unverifiable. This profile performs no token
introspection and owns no denylist, so logout or account disablement does not
immediately invalidate an already-issued self-contained token.

If the accepted revocation window is shorter than the issuer's access-token
lifetime, add an explicit introspection or authoritative decision-time policy;
do not infer it from additional JWT claims.

## Signals and local proof

The package publishes `authn.verifications` with bounded `authn.transport`,
`authn.result`, and sanitized `authn.reason` attributes, plus the no-label
`authn.jwks.refresh_failures` counter. A refresh failure logs only
`authn_jwks_refresh_failed` and the component name.

Run:

```bash
go test -vet=off ./internal/infra/oidcjwt
go test -vet=off -race ./internal/infra/oidcjwt
make openapi-check
TEMPLATE_INIT_PROFILE=authn make template-init-check
```

Local tests prove the library composition and repository policy. Production
adoption still needs a real issuer token for the chosen profile and the actual
ingress/TLS/egress boundary.
