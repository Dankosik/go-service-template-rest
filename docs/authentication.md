# Authentication

The selected `AUTHN` profile gives HTTP and gRPC handlers one verified
`reqctx.Principal`. Feature code does not parse credentials, call a provider,
map authentication errors, or publish authentication telemetry.

The default `AUTHN=none` profile removes the verifier, configuration, tests,
dependencies, guide, OpenAPI security declaration, and bootstrap wiring.

<!-- profile:authn-oidc-jwt:start -->
## Initialize OIDC JWT

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

`authn.token_profile` defaults to `resource-server`. Set it to `rfc9068` only
when the authorization server issues that dialect:

```dotenv
APP__AUTHN__TOKEN_PROFILE=rfc9068
```
<!-- profile:authn-oidc-jwt:end -->

<!-- profile:authn-oidc-introspection:start -->
## Initialize OIDC token introspection

```bash
make template-init \
  MODULE=github.com/your-org/my-service \
  CODEOWNER=@your-org/backend \
  DATABASE=none \
  GRPC=none \
  AUTHN=oidc-introspection
```

Supply the complete immutable tuple. The client secret is environment-only:

```dotenv
APP__AUTHN__ISSUER=https://identity.example.com
APP__AUTHN__AUDIENCE=https://api.example.com
APP__AUTHN__INTROSPECTION_ENDPOINT=https://identity.example.com/oauth/introspect
APP__AUTHN__INTROSPECTION_TARGET_CLASS=external-https
APP__AUTHN__INTROSPECTION_CLIENT_ID=resource-server
APP__AUTHN__INTROSPECTION_CLIENT_SECRET=
```

`APP__AUTHN__INTROSPECTION_CLIENT_SECRET` must be non-empty at runtime and must
not appear in YAML. For a private fixed-authority endpoint set
`APP__AUTHN__INTROSPECTION_TARGET_CLASS=private-https` and
`APP__AUTHN__INTROSPECTION_PRIVATE_HOST_SUFFIX` to the required DNS suffix.
<!-- profile:authn-oidc-introspection:end -->

The issuer must be an exact HTTPS URL without user information, query, or
fragment. The audience is a non-empty resource-server identifier. The runtime
TLS trust store, DNS/egress policy, and the ingress that terminates client TLS
remain deployment-owned inputs.

## HTTP and gRPC

HTTP operations inherit Bearer security from the OpenAPI contract.
`/health/live` and `/health/ready` remain public. The authentication seam removes
`Authorization` before the generated handler runs and stores the principal in
the request context.

Missing or invalid credentials return `401` with a Bearer challenge. Malformed
credentials return `400`, oversized credentials `431`, caller cancellation or
deadline expiry `504`, and unavailable trust `503`. Responses never contain
token, claim, parser, key, URL, provider body, or raw dependency text.

Native gRPC applies the same verifier to every application RPC.
`grpc.health.v1.Health/Check` remains public; `Health/Watch` is protected.
Authentication failures map to `Unauthenticated`, `ResourceExhausted`,
`Canceled`, `DeadlineExceeded`, or `Unavailable`. Protected streams receive a
deadline no later than token expiry plus clock skew, and their handler-visible
metadata no longer contains the credential.

Both handler kinds read identity in the same way:

```go
principal, ok := reqctx.PrincipalFromContext(ctx)
```

On a protected operation, `ok == false` is a wiring defect and should result in
an internal failure, not a second credential parser.

<!-- profile:authn-oidc-jwt:start -->
## JWT verification

The verifier accepts compact JWT access tokens that satisfy all of these rules:

- `alg=RS256`, selected by service policy rather than trusted from the token;
- a signature from the configured issuer's discovered JWKS;
- a JWK whose optional `use` is absent or `sig`, never `enc`;
- exact `iss` and an `aud` string or array containing the configured audience;
- required `exp`, optional `nbf`, and a 30-second clock-skew allowance;
- at most 32 KiB before parsing;
- at least one stable identity: `sub` or a client identifier.

The default does not require a `typ` header, `iat`, or `jti`. The explicit
`rfc9068` profile additionally requires `typ=at+jwt` (or `application/at+jwt`),
`sub`, `client_id`, `iat`, and `jti`.

Startup performs OIDC discovery and installs the first JWKS before serving.
Authentication is an eager startup gate, not a dynamic readiness dependency.

These are bearer access tokens. This profile performs no token introspection
and owns no denylist.
<!-- profile:authn-oidc-jwt:end -->

<!-- profile:authn-oidc-introspection:start -->
## Token introspection

The presented bearer value is opaque. Every syntactically accepted credential
causes exactly one RFC 7662 POST to the configured HTTPS endpoint with
`client_secret_basic`, `token_type_hint=access_token`, and no token in the URL.

Only HTTP 200 `application/json` with a strict object, boolean `active`, and
the required issuer, audience, expiry, and identity members is usable.
`active=false` is a generic invalid credential. Provider, transport, and
response-contract failures are unavailable trust for that request. There is no
cache, retry, redirect, proxy, or remembered failure.

Startup constructs the bounded client without provider I/O. A later provider
outage does not change readiness.
<!-- profile:authn-oidc-introspection:end -->

The verifier publishes:

```go
reqctx.Principal{
    Issuer:   verifiedIssuer,
    Subject:  subjectWhenPresent,
    ClientID: verifiedClientIDWhenPresent,
}
```

It does not convert roles, tenant IDs, permissions, or token scopes into an
authorization decision.

## Signals and local proof

The package publishes `authn.verifications` with bounded `authn.transport`,
`authn.result`, and sanitized `authn.reason` attributes.

Run:

```bash
go test -vet=off ./internal/infra/bearerauthn
<!-- profile:authn-oidc-jwt:start -->
go test -vet=off ./internal/infra/oidcjwt
<!-- profile:authn-oidc-jwt:end -->
<!-- profile:authn-oidc-introspection:start -->
go test -vet=off ./internal/infra/oauthintrospection
<!-- profile:authn-oidc-introspection:end -->
make openapi-check
go test ./internal/infra/oidcjwt ./internal/config ./internal/infra/http
```
