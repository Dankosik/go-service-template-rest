# OIDC/JWT Authentication

The `AUTHN=oidc-jwt` initialization profile adds one conservative authentication
boundary for services that receive signed JWT access tokens from an OpenID
Provider. The default `AUTHN=none` profile deletes this guide, the verifier,
configuration, dependency, tests, OpenAPI security declaration, and bootstrap
wiring. It does not leave a dormant authentication framework.

This pack authenticates a caller. Authorization, roles, tenant policy, sessions,
user provisioning, and domain ownership remain feature-owned work.

There is no registration point for extra claim rules. Adding one is an edit
inside `internal/infra/oidcjwt`, and that package's documentation
(`go doc ./internal/infra/oidcjwt`) names each site and what the edit costs at
the next template sync.

## Initialize the profile

```bash
make template-init \
  MODULE=github.com/your-org/my-service \
  CODEOWNER=@your-org/backend \
  DATABASE=none \
  GRPC=none \
  AUTHN=oidc-jwt \
  OUTBOUND_HTTP=none
```

`OUTBOUND_HTTP=none` still retains the bounded HTTP client package because OIDC
discovery and JWKS retrieval require it. `GRPC=none` removes only the gRPC
authentication adapter; `GRPC=enabled` applies the same verifier to every
non-health unary and streaming RPC.

## Configure trust

Set all three values to operator-owned values:

```dotenv
APP__AUTHN__ISSUER=https://identity.example.com
APP__AUTHN__AUDIENCE=https://api.example.com
APP__AUTHN__TRUSTED_PROXY_CIDRS=10.20.0.0/16
```

The issuer must be an exact HTTPS URL without query, fragment, or user
information. The audience is one exact, non-empty resource-server identifier.
Trusted proxy CIDRs name only the immediate TLS-terminating peers allowed to
assert exactly one `X-Forwarded-Proto: https` value. Direct connection state,
comma-joined values, plaintext requests, and assertions from any other peer fail
closed. This keeps the transport decision at the configured deployment boundary
instead of trusting a request-local signal.

The example issuer and audience in local configuration are reserved
documentation values. They cannot establish initial trust; replace them before
starting the service.

When native gRPC is enabled with this profile, its server transport must be TLS.
The profile rejects the plaintext development switch because bearer credentials
must not cross an unprotected RPC transport.

## Accepted token contract

The verifier accepts only compact, signed JWT access tokens with:

- exactly three unpadded base64url segments and a total size no greater than
  8 KiB;
- protected `alg=RS256`, a non-empty protected `kid`, and
  `typ=at+jwt` or `typ=application/at+jwt`;
- no `crit`, `b64`, `jku`, `x5u`, or embedded `jwk` header;
- an exact issuer, an audience array or string containing the configured
  audience, and non-empty `sub`;
- integral `exp` and `iat` NumericDate values, optional integral `nbf`, and a
  fixed 30-second clock-skew allowance.

Unsigned tokens, algorithm substitution, tokens signed by an untrusted key,
unknown claims encodings, missing subjects, and incomplete trust configuration
are rejected. The only published identity is
`reqctx.Principal{Subject: <opaque sub>}`. Token scopes, roles, client IDs, and
other provider claims are deliberately not converted into authorization.

HTTP operations use OpenAPI Bearer security. `/health/live` and `/health/ready`
remain public. Missing or invalid credentials return `401` with a Bearer
challenge; malformed or duplicated headers return `400`; oversized credentials
return `431`; unavailable current trust returns `503`. This pack never owns
`403`.

Native gRPC health remains public. Other RPCs map the corresponding failures to
`Unauthenticated`, `ResourceExhausted`, or `Unavailable`. Both transports remove
the credential before invoking application handlers and publish the principal
through `internal/reqctx`.

## Discovery, keys, and readiness

Startup performs OIDC Discovery, requires the discovered issuer to match
exactly, validates an HTTPS `jwks_uri`, and must install a completely valid JWKS
before either listener is admitted. Discovery and JWKS requests have a
five-second timeout, a 1 MiB response limit, bounded connections, and no ambient
redirect or authority widening.

The verifier accepts at most 100 JWK entries and only public RSA signing keys of
at least 2048 bits whose metadata is compatible with RS256 verification. It
refreshes every five minutes, coalesces concurrent refreshes, and rate-limits
unknown-key refreshes for 30 seconds. A failed refresh may continue using the
last completely validated set until its fixed 15-minute age limit. Once that
limit is reached, authentication and readiness fail closed until a full
replacement set is installed. Partial or malformed key sets never replace the
current set.

Shutdown cancels and joins any in-flight refresh before closing the owned HTTP
connection pool.

## Local development and tests

There is no bypass, fake principal mode, or accept-all switch. The production
egress policy intentionally rejects loopback and private provider addresses. For
interactive service runs, use a real publicly reachable HTTPS test issuer and
mint RS256 access tokens with its registered test key. For deterministic local
development, run the package harness: it injects a bounded in-process provider
transport, serves real Discovery/JWKS documents, and verifies real signatures
without weakening production network policy.

Package tests under `internal/infra/oidcjwt` load fixed test-only RSA keys from
`testdata`, serve deterministic Discovery/JWKS responses, and validate real
signatures. The testdata keys are not reachable from the production package
graph. Run:

```bash
go test -vet=off ./internal/infra/oidcjwt
go test -vet=off -race ./internal/infra/oidcjwt
make openapi-check
TEMPLATE_INIT_PROFILE=authn make template-init-check
```

## Operational signals and redaction

The pack emits bounded-cardinality metrics:

- `authn.verifications`, labelled `authn.transport` (`http`, `grpc`),
  `authn.result` (`success`, `failure`), and on failure `authn.reason`, one of
  `missing`, `malformed`, `untrusted_transport`, `oversize`, `invalid`,
  `unavailable`, or `canceled`;
- `authn.jwks.refreshes`, labelled `authn.refresh.trigger` (`startup`,
  `scheduled`, `key_miss`) and `authn.result`;
- `authn.jwks.age` in seconds, unlabelled.

The reasons are separate because they call for different responses.
`untrusted_transport` means the peer did not match `authn.trusted_proxy_cidrs`
or did not assert exactly one `X-Forwarded-Proto: https`, so it usually reports
a deployment that moved rather than a misbehaving client, and it typically
fails every request at once. `unavailable` means current trust is gone: check
`authn.jwks.age` against the fixed 15-minute limit and the refresh result.
`canceled` means the caller gave up while the service was fetching keys to
answer it, so read it against `authn.jwks.refreshes` and provider latency rather
than as anything the caller sent; it is deliberately not counted as `invalid`,
because a client that hung up is not a client with a bad credential. `invalid`
and `missing` are ordinary client outcomes and are expected to be non-zero.

No metric, log message, trace attribute, returned error, or panic recovery
contains a token, JWT segment, claim value, subject, issuer URL, JWKS URL, or
provider response body. Provider failures are intentionally stage-specific but
sanitized. Investigate availability with the stage, refresh result, key age, and
provider-side request logs rather than enabling credential logging.
