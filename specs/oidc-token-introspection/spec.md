# Add a strict uncached token introspection authentication profile

status: ready

Independent Specification Review: PASS on 2026-08-22 after one bounded delta
recheck; no findings survive.

Problem: `AUTHN=oidc-jwt` can authenticate only locally verifiable JWT access
tokens and cannot observe revocation before token or signing-key expiry. A
service whose authorization server issues opaque access tokens or owns a
shorter revocation requirement needs an authoritative request-time decision
without changing feature handlers or introducing a second identity path.

## Outcome

Add one mutually exclusive initializer choice:

```text
AUTHN=none|oidc-jwt|oidc-introspection
```

`AUTHN=oidc-introspection` treats the bearer access token as opaque, submits it
once to one configured RFC 7662 endpoint authenticated with
`client_secret_basic`, and publishes the same `reqctx.Principal` only after a
strict active-token response. It performs no response caching. Feature code,
OpenAPI security declarations, HTTP/gRPC identity access, and authorization
ownership do not change when switching between the two authentication
profiles.

## Scope and non-goals

The profile owns bearer extraction, the authoritative introspection decision,
strict response admission, principal normalization, transport error parity,
bounded outbound work, secret-safe telemetry, startup configuration admission,
shutdown, and initializer presence/absence.

It does not own token issuance, refresh, revocation, logout, userinfo, sessions,
roles, permissions, tenant selection, policy decisions, provider registration,
network/TLS/egress provisioning, or authorization-server availability. It does
not parse a presented token as JWT, fall back to local JWT verification, query
more than one endpoint, discover metadata, accept per-request destinations,
forward the token to business code, or introspect refresh tokens.

The first version supports only `client_secret_basic` with dedicated
resource-server credentials. It does not add `client_secret_post`, bearer
acquisition, `private_key_jwt`, mTLS, DPoP, a credential plugin interface, or a
generic OAuth endpoint client. A provider requiring another method reopens
Specification with that exact credential lifecycle and proof boundary.

## Behavioral contract

### Selection, absence, and compatibility

- The three `AUTHN` values are mutually exclusive. There is no runtime fallback
  or mixed JWT/introspection mode.
- `AUTHN=none` remains the default and removes all authentication behavior.
  `AUTHN=oidc-jwt` remains behaviorally unchanged.
- Selecting `oidc-introspection` retains the existing OpenAPI bearer security,
  HTTP/gRPC authentication seams, failure renderers, and `reqctx.Principal`.
  Existing feature handlers and their tests require no authentication-specific
  change.
- An unselected introspection capability is physically absent from generated
  source, configuration examples, tests, documentation, dependencies, and
  bootstrap wiring. `template.lock` records the selected value.
- The profile has no database, jobs, messaging, object-storage, outbound-auth,
  or generic outbound-HTTP selector dependency. Its one fixed provider client
  is part of authentication trust, as Discovery/JWKS access is for the JWT
  profile.

### Configuration and startup admission

Before serving, the selected profile requires one complete immutable tuple:

- exact expected issuer;
- non-empty API audience;
- exact absolute HTTPS introspection endpoint with no user information, query,
  or fragment;
- endpoint target classification needed by the repository's fixed-authority
  egress policy;
- non-empty introspection client ID; and
- non-empty introspection client secret from an approved secret input, never a
  committed YAML default.

Missing, partial, unknown, or invalid values fail configuration/startup closed.
The endpoint authority cannot be changed by a request, response, redirect,
proxy, or ambient environment. Startup validates configuration and constructs
the bounded client but does not send a synthetic token or infer provider health
from an unauthorized probe.

Liveness is never gated by the provider. After successful startup, a dynamic
provider outage does not evict the instance from readiness; affected protected
requests fail as unavailable trust. This matches the existing authentication
health contract and avoids moving every replica out of service for one shared
dependency outage. Provider availability remains an alertable authentication
signal.

### Credential admission and outbound request

- HTTP and gRPC retain the current bearer-header count, whitespace, scheme, and
  32 KiB token limits and remove `Authorization` before any handler, access log,
  or downstream client can observe it.
- Every syntactically accepted token causes at most one RFC 7662 HTTP POST with
  `application/x-www-form-urlencoded` data, `token_type_hint=access_token`, and
  the dedicated endpoint credential. The presented token never appears in the
  URL.
- Each call is bounded by the caller context, one fixed provider timeout,
  response-header and response-body limits, and the fixed endpoint authority.
  Redirects, proxy routing, and retries are forbidden.
- Introspection runs inside the queue-free business admission slot the current
  transport already holds before authentication: HTTP uses the configured
  positive `http.max_in_flight` value (default 256), while gRPC uses its fixed
  256-RPC business limit. With both transports enabled, the exact per-process
  provider-call ceiling is their sum. The profile adds no second limiter or
  queue.
- Existing outer admission saturation remains an ordinary transport-capacity
  rejection before credential processing: HTTP returns its current `503 at
  capacity`, and gRPC returns `ResourceExhausted`. Deployment owns the enabled
  transports and HTTP setting; the gRPC transport owns its fixed ceiling. If a
  provider cannot accept the resulting sum, reopen Specification for a distinct
  provider budget before enabling the profile.
- Caller cancellation and caller deadline expiry preserve the current
  transport outcomes even when observed during provider I/O. A provider-owned
  timeout while the caller context is still live, connection/TLS/DNS failure,
  invalid endpoint credential, non-success status, forbidden redirect,
  oversized response, wrong media type, malformed JSON, or unusable active
  response is unavailable trust.

### Response and principal admission

Only a bounded HTTP 200 JSON object is an introspection result. Unknown members
are ignored after the enclosing object passes the bounds. Duplicate top-level
members, trailing JSON values, or malformed registered members make the
provider response unusable. The following rules then apply in order:

1. `active` must exist and be a JSON boolean. Missing or ill-typed `active` is
   unavailable trust.
2. `active=false` is an invalid caller credential regardless of any additional
   members. The service never reports whether the token is unknown, expired,
   revoked, unauthorized for this resource server, or otherwise inactive.
3. `active=true` must include string `iss`, an `aud` string or string array, a
   finite integral `exp`, and must yield at least one stable identity under step
   5. A missing or structurally invalid required member is unavailable trust
   because the authoritative provider did not produce this profile's response
   contract.
4. `exp` must still be valid under the existing 30-second clock-skew policy. If
   `nbf` is present, the same policy must admit it. A response that is active
   but expired or not yet usable is an invalid caller credential.
5. Returned values receive no trimming or coercion. An issuer other than the
   configured issuer or an audience that does not contain the configured API is
   an invalid caller credential. `sub` and `client_id` keep their distinct
   meanings and may both be present. An absent or empty optional identity member
   contributes no identity and is accepted when the other member is valid; a
   supplied non-empty member must already equal its trimmed value. A wrong type,
   whitespace-padded value, or absence of both stable identities is unavailable
   trust.
6. A successful result publishes only:

   ```go
   reqctx.Principal{
       Issuer:   configuredIssuer,
       Subject:  returnedSubjectWhenPresent,
       ClientID: returnedClientIDWhenPresent,
   }
   ```

   `username`, `scope`, `jti`, provider extensions, and every other response
   member do not enter request context or authorization decisions.

The authorization server is authoritative for revocation and token state, but
the service remains authoritative for its issuer, audience, identity, time, and
response-shape admission. No provider `active=true` can weaken those local
rules.

### Caller-visible failure parity

HTTP retains the current authentication taxonomy:

| Condition | HTTP outcome |
| --- | --- |
| Missing or inactive/invalid credential | `401` with the existing Bearer challenge |
| Malformed bearer carrier | `400` |
| Oversized credential | `431` |
| Caller cancellation or caller deadline expiry | existing `504` authentication-verification timeout |
| Existing outer HTTP admission saturated | existing `503 at capacity` |
| Provider timeout, endpoint-auth, or response-contract failure | `503` |

Native gRPC retains the corresponding existing `Unauthenticated`,
`ResourceExhausted`, `Canceled`, `DeadlineExceeded`, and `Unavailable`
categories. Caller cancellation is `Canceled`; caller deadline expiry is
`DeadlineExceeded`; a provider-owned timeout while the caller context remains
live is `Unavailable`. No response includes a token, endpoint, client
identifier, subject, raw provider status/body, parser text, network address, or
secret.

HTTP public probes remain public. Native gRPC keeps only exact
`grpc.health.v1.Health/Check` public; `Health/Watch` and application RPCs remain
protected. A protected stream receives a handler-visible deadline no later
than the introspected `exp` plus the existing clock skew, with expiry enforced
at message I/O as in the JWT profile.

### Replay, revocation, failure, and recovery

- Every accepted protected request is based on a successful request-time
  introspection result. The profile caches neither active nor inactive results,
  coalesces no equal tokens, and never serves stale identity after provider
  failure.
- The observable revocation delay is therefore owned by the authorization
  server's propagation contract plus an in-flight request already admitted
  before revocation. This repository adds no cache window.
- A provider outage cannot turn a previously active token into an accepted
  request. Recovery is the next successful introspection call; there is no
  background retry, stale fallback, local allowlist, or operator bypass.
- Shutdown stops new admission, lets serving transports drain within their
  existing budgets, cancels provider calls through their request contexts, and
  releases idle provider connections before telemetry flush. There is no
  process-lifetime refresh or cache goroutine to join.

### Privacy, telemetry, and abuse bounds

- The presented access token and introspection client secret are credentials.
  The introspection response and resulting principal are sensitive identity
  data. None may appear in logs, traces, metrics, errors, panic text, config
  dumps, test failure output, or generated documentation values.
- Reuse the bounded `authn.verifications` transport/result/reason vocabulary.
  Introspection adds no endpoint, status-code, issuer, audience, subject,
  client-ID, token hash, or response-field metric labels.
- Provider failures use bounded semantic reasons only; raw dependency text is
  never caller-visible or attached to telemetry.
- The existing queue-free HTTP/gRPC business admission is required to remain
  outside authentication and bounds provider calls before identity is known.
  Deployment still owns edge rate limiting and must certify that provider
  capacity covers the sum of enabled transport ceilings; this profile does not
  claim per-principal fairness before identity is known.

## Deliberately unchanged behavior

- `reqctx.Principal` and its handler-facing accessors are unchanged.
- Authentication proves identity only. Roles, scopes, permissions, and tenant
  policy remain outside this capability even when the provider returns them.
- OpenAPI security inheritance and fail-closed OR/AND validation remain the
  contract authority; the profile does not create new anonymous operations.
- HTTP/gRPC error bodies and status mapping remain transport-owned and
  sanitized.
- TLS roots, DNS, egress, proxy policy, ingress TLS termination, provider
  registration, secret distribution/rotation, and alert thresholds remain
  deployment-owned inputs.

## Positive-cache decision

The accepted v1 cache policy is **no cache**. A configurable zero, cache
interface, token-keyed map, negative cache, `singleflight`, and stale-on-error
path are all rejected because they add code without changing that policy.

Positive caching reopens Specification only after a named adopter supplies:

1. an accepted end-to-end maximum revocation delay `R`;
2. a measured or contractual provider propagation bound `P`; and
3. representative evidence that uncached introspection misses an accepted
   latency or capacity target.

Any future positive TTL `C` must satisfy:

```text
0 < C <= min(token_expiry - now, configured_cap, R - P)
```

If `R - P <= 0`, if `exp` is absent, or if any input is not accepted, caching
remains disabled. Negative caching, stale serving, cross-replica cache state,
and token-derived cache identifiers are not implied by a future positive cache
decision.

## Success criteria and proof expectations

The Specification succeeds when a later implementation can demonstrate all of
the following without inventing behavior:

1. Both HTTP and gRPC publish the same principal for one strict active response
   and preserve the existing public/protected route policy and stream expiry.
2. Missing, malformed, oversized, inactive, wrong-issuer, wrong-audience,
   expired/not-yet-valid, identity-less, malformed-provider, endpoint-auth,
   timeout, capacity, and cancellation paths fail in the declared category with
   no authentication bypass.
3. A TLS fake provider proves one fixed endpoint, POST form semantics, Basic
   endpoint authentication, one attempt, limits, context cancellation, refused
   redirects/proxy, and no token in URL or diagnostics.
4. Focused negative disclosure proof finds no token, secret, principal, endpoint,
   or provider body in HTTP/gRPC responses, logs, traces, metrics, config output,
   or test diagnostics.
5. Initializer proof shows the selected profile is complete, the two other
   authentication choices are unchanged, and every unselected introspection
   source/config/test/doc/dependency/profile marker is absent with the exact
   `template.lock` value.
6. Startup/shutdown proof covers invalid configuration, partial-construction
   cleanup, bounded in-flight cancellation, idle-connection release, and no
   leaked goroutine.

These are claim boundaries, not a Test Design matrix. Technical Design must
select the runtime and ownership mechanism first; Test Design is triggered
afterward because the provider/cancellation/disclosure/profile oracles are
material and cross several proving layers.

## Risks, assumptions, and reopen conditions

- No named authorization server, registration, endpoint, credential, target
  class, revocation SLA, traffic distribution, capacity, or deployment was
  supplied. The accepted template contract assumes a provider that supports
  RFC 7662, `client_secret_basic`, and the strict active-response subset above.
  Live compatibility remains an adopter gate.
- If the real provider omits required identity/binding fields, uses another
  endpoint authentication method, requires request extensions, returns signed
  introspection responses, or binds tokens to mTLS/DPoP, reopen Specification;
  do not add a fallback branch inside this profile.
- Refresh the current-tree baseline before Technical Design because the
  checkout is dirty and `main` was six commits behind `origin/main` at the
  evidence snapshot.
- Reopen Specification if authorization begins consuming introspection scopes,
  roles, tenant data, or extensions; that changes the principal and policy
  contract rather than only the verifier mechanism.
- Reopen Specification for any positive/negative cache, stale fallback,
  readiness transition, provider failover, retry, or multi-endpoint behavior.

## Phase disposition

Technical Design is required because implementation would otherwise choose the
outbound trust boundary, provider-call lifecycle, concurrency/admission shape,
runtime composition, profile-marker ownership, failure plumbing, and shutdown
mechanism. Go Code / Ownership Design is also expected because a second
verifier must share the transport contract without turning `oidcjwt` into a
generic authentication framework.

Planning and Implementation are not authorized by this Specification result.

Supporting evidence: [research synthesis](research/synthesis.md),
[current authentication contract](../../docs/authentication.md), and
[RFC 7662](https://www.rfc-editor.org/rfc/rfc7662.html).
