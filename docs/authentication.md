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
  audience, and non-empty `sub`, `client_id`, and `jti`;
- JSON-number `exp` and `iat` NumericDate values with `exp` after `iat`, optional
  `nbf` before `exp`, a fixed 15-minute maximum issued lifetime, and a fixed
  30-second clock-skew allowance. Integer, fractional, and exponent notation are
  accepted and compared at nanosecond resolution.

Unsigned tokens, algorithm substitution, tokens signed by an untrusted key,
unknown claims encodings, missing required claims, and incomplete trust configuration
are rejected. The published identity is `reqctx.Principal{Issuer, Subject,
ClientID}`. `(Issuer, Subject)` is the stable caller identity; `ClientID` names
the OAuth client to which the token was issued. All three values are
correlatable identity data; this pack does not put them in logs or telemetry.
Token scopes, roles, tenant IDs, `jti`, and other provider claims are
deliberately not converted into authorization.

HTTP operations use OpenAPI Bearer security. `/health/live` and `/health/ready`
remain public. Missing or invalid credentials return `401` with a Bearer
challenge; malformed or duplicated headers return `400`; oversized credentials
return `431`; unavailable current trust returns `503`. This pack never owns
`403`. A canceled or expired authentication check returns `504` rather than
being misclassified as a bad credential. The shared HTTP/gRPC parser accepts
RFC 6750's one-or-more ASCII spaces after the Bearer scheme and rejects other
surrounding or internal whitespace.

Native gRPC `Health/Check` remains public for platform probes. `Health/Watch`
and every application RPC require a credential; protected streams receive a
deadline no later than the token's `exp` plus the 30-second skew and consume the
process-wide RPC admission budget. Authentication failures map to
`Unauthenticated`, `ResourceExhausted`, or `Unavailable`. Both transports remove
the credential before invoking application handlers and publish the principal
through `internal/reqctx`.

## Discovery, keys, and readiness

Startup performs OIDC Discovery, requires the discovered issuer to match
exactly, validates an HTTPS `jwks_uri`, and must install a completely valid JWKS
before either listener is admitted. A provider-owned query is allowed on
`jwks_uri`; user information, fragments, redirects, and authority widening are
not. Discovery and JWKS requests have a five-second timeout, a 1 MiB response
limit, and bounded connections.

The verifier accepts at most 100 JWK entries and only public RSA signing keys of
at least 2048 bits whose metadata is compatible with RS256 verification. It
refreshes on a five-minute cadence, coalesces concurrent refreshes, and
rate-limits caller-driven unknown-key refreshes for a fixed 30 seconds. The
scheduled cadence and its 30-second failure retry receive bounded ±10% jitter so
replicas do not synchronize provider load; the fixed 15-minute key-set age limit
is never jittered. A failed refresh may continue using the last completely
validated set until that limit. Once it is reached, authentication and readiness
fail closed until a full replacement set is installed. Partial or malformed key
sets never replace the current set.

Shutdown cancels and joins any in-flight refresh before closing the owned HTTP
connection pool. If the shared background-shutdown budget expires first, the
process reports the timeout and proceeds to telemetry flush instead of starting
a second unbounded join.

## Bearer replay and revocation

Access tokens are bearer credentials: anyone who obtains one can replay it
against its configured audience until verification stops accepting it. This
pack has no introspection call or per-token denylist, so issuer logout, account
disablement, and credential revocation do not invalidate an already issued
token immediately. An already-open protected gRPC stream is canceled at the
same `exp` plus skew boundary rather than retaining the startup identity
indefinitely.

| Event | What stops old access | Maximum delay |
| --- | --- | --- |
| Logout | Access-token expiry | 15 minutes 30 seconds from issue |
| Credential compromise | Access-token expiry, or removal of its signing key after JWKS refresh | 15 minutes 30 seconds from issue when the signing key remains valid |
| Role or scope downgrade | Not consumed by this pack; any future token-cached authorization waits for access-token expiry | 15 minutes 30 seconds from issue if such claims are added |
| Tenant or membership removal | Not consumed by this pack; a feature must re-read its authority or accept token staleness | Feature-owned; at most 15 minutes 30 seconds if encoded into this token |

A healthy signing-key removal is normally observed at the next five-minute JWKS
refresh; provider failure makes the verifier fail closed once the installed key
set reaches 15 minutes. Features needing a shorter user, permission, or tenant
revocation window must add an authoritative decision-time lookup or an accepted
deny event rather than copying more claims into `Principal`.

Sender-constrained access tokens such as DPoP or mutual-TLS tokens are not
implemented. Add that as a separate IdP, ingress, client, and service protocol
design when replay of a stolen bearer token is outside the accepted threat
model; enabling it inside this verifier alone would not constrain the other
hops.

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
  `scheduled`, `key_miss`), `authn.result`, and on failure `authn.reason`, one
  of `request`, `transport`, `status`, `body`, `oversize`, `invalid_document`,
  `panic`, or `unknown`;
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

`key_miss` counts every refresh a verification asked for, which is wider than an
absent key id: a key rotated in place, under a name the service already holds,
also fails to verify and is recovered by the same fetch. Read it as "the
installed key set could not answer", and expect a burst of it around a provider
rotation. It is the one trigger a caller can drive, so it is rate-limited to one
fetch per 30 seconds; a sustained rate with no rotation behind it means requests
are arriving with key ids this service has never held. Those requests wait on
the fetch, so tune it together with `http.max_in_flight`, which caps how many
can be waiting at once.

A recovered panic is reported as `authn_panic_recovered` with its Go type and
stack and never its value. It is a defect in the service rather than a provider
problem; the refresh metric records the closed `panic` reason and the log
locates it.

No metric, log message, trace attribute, returned error, or panic recovery
contains a token, JWT segment, claim value, subject, issuer URL, JWKS URL, or
provider response body. Provider failures retain only the closed phase above;
status codes, bodies, URLs, and transport error text remain redacted. Investigate
availability with the startup stage or refresh reason, key age, and provider-side
request logs rather than enabling credential logging.
