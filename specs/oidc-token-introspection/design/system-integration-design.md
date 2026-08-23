# OIDC token introspection System / Integration Design

status: ready for independent Technical Design Review

Authoritative behavior and reopen authority:

- [`../spec.md`](../spec.md)
- [`../research/synthesis.md`](../research/synthesis.md)

This artifact selects mechanism only. The Specification remains authoritative
for behavior, failure categories, proof expectations, and its exhaustive
Specification reopen conditions.

## Refreshed baseline and evidence boundary

- Current `HEAD` is `8967a4ac06d4fce0515703b15ffa5db35e5378ae`.
- The checkout is seven commits behind the locally known `origin/main`. Those
  commits change agent-harness and workflow material, not authentication,
  configuration, bootstrap, fixed-authority HTTP, or initializer behavior.
- The dirty checkout contains an unfinished, separately owned
  `internal/infra/httpclient` refactor and matching OIDC edits. A compile-only
  probe currently fails on undefined `attemptAuthorizationError` and
  `ResponseTooLargeError`; this design therefore makes no current-tree build
  claim and does not treat the provisional API as accepted.
- The stable current owner remains `internal/infra/httpclient`: one HTTPS
  authority, post-DNS address admission, no ambient proxy, refused redirects,
  sanitized propagation, and explicit caller-owned response/budget policy.
  Its accepted constructors do not expose a response-header timeout or byte
  cap, so they cannot realize the complete introspection edge unchanged.
- This design owns the smallest additive repair to that accepted client: a
  caller-supplied `ResponseLimits` value and external/private constructors that
  apply only `ResponseHeaderTimeout` and `MaxResponseHeaderBytes` to the cloned
  transport. Existing constructors and behavior remain unchanged. The provider
  adapter still chooses the values and owns operation/body policy.
- Implementation must refresh that owner before editing it. If the concurrent
  candidate has become accepted and exposes an equivalent surface, use that
  surface with retry/instrumentation disabled instead of adding parallel
  constructors. An API name change alone does not reopen this design; loss of a
  semantic guarantee does.

No refreshed evidence changes behavior accepted by the Specification, so
Specification stays closed.

## Selected mechanism

Introduce one narrow shared bearer-authentication runtime shell and keep the
two trust engines concrete:

```text
HTTP OpenAPI / native gRPC
        |
        v
internal/infra/bearerauthn
  carrier removal, failure taxonomy, metrics, principal publication,
  public Health/Check rule, stream expiry
        |
        +-- internal/infra/oidcjwt
        |     OIDC Discovery, JWKS lifecycle, JWT verification
        |
        `-- internal/infra/oauthintrospection
              one RFC 7662 request per accepted credential
```

`bearerauthn` is earned by behavior that must be identical across both selected
profiles. It owns no issuer, endpoint, client credential, JWT parser, provider
response, cache, authorization rule, or runtime profile switch. Each generated
service contains the shared shell and exactly one concrete trust engine.

The concrete engine contract is deliberately small: verify one already parsed
bearer value under the caller context and return a `reqctx.Principal` plus its
expiry, or one sanitized `invalid`/`unavailable`/context failure. It has exactly
two source-template implementations and one generated-service implementation.

## Drivers and alternative dispositions

| Driver | Forced consequence |
| --- | --- |
| Request-time revocation authority | Every accepted protected request performs one authoritative introspection; no local allow result survives the call. |
| Transport and handler parity | Bearer grammar, removal, error rendering, principal publication, gRPC public method, and stream expiry have one shared runtime owner. |
| Fixed external trust boundary | One immutable endpoint is built at startup through `internal/infra/httpclient`; requests cannot select or redirect it. |
| Queue-free existing admission | Provider work runs only inside the HTTP or gRPC business slot already held; the provider client adds neither a limiter nor a connection queue. |
| Optional-profile physical absence | Initializer generation keeps one common bearer pack and exactly one concrete trust pack; `AUTHN=none` removes all three. |
| Secret and identity privacy | Raw client/provider errors never cross the adapter; the provider client emits no endpoint-bearing spans or labels. |
| No evidence for more machinery | Standard-library form, Basic authentication, JSON, time, and HTTP are sufficient; no dependency, cache, retry, breaker, discovery, or credential interface is added. |

| Alternative | Disposition |
| --- | --- |
| Add introspection inside `oidcjwt` | Rejected: it makes a JWT/JWKS owner a generic authentication framework and leaves package identity false. |
| Duplicate HTTP/gRPC adapters and failure taxonomy in both engines | Rejected: the generated service is small, but the source template would have two security seams whose parity depends on duplicate edits. |
| Runtime-select JWT versus introspection from configuration | Rejected: `AUTHN` is an initializer choice; a runtime switch retains an unselected trust path and dependencies. |
| Add a generic OAuth endpoint client or credential plugin | Rejected by the locked v1 `client_secret_basic` boundary and lack of a second consumer. |
| Add cache, coalescing, retry, limiter, breaker, readiness probe, or failover | Rejected by the Specification and by the absence of a useful degraded identity result. |

## Configuration and startup contract

The generated introspection profile has one immutable `authn` tuple:

| Key | Admission |
| --- | --- |
| `authn.issuer` | Existing exact HTTPS issuer rule. |
| `authn.audience` | Existing non-empty API audience rule. |
| `authn.introspection_endpoint` | Absolute HTTPS URL with host and path allowed, but no user information, query, forced query, or fragment. |
| `authn.introspection_target_class` | Required exact value `external-https` or `private-https`; no inferred default. |
| `authn.introspection_private_host_suffix` | Required only for `private-https`; forbidden for `external-https`; the fixed-authority client remains final admission authority. |
| `authn.introspection_client_id` | Non-empty exact credential component; never trimmed or logged. |
| `authn.introspection_client_secret` | Non-empty exact credential component; non-empty file/YAML values remain forbidden and deployment supplies `APP__AUTHN__INTROSPECTION_CLIENT_SECRET`. |

`internal/config` rejects missing, partial, unknown, and inconsistent tuples
before serving. `internal/authntrust` holds only the pure endpoint and target
class predicates shared by configuration and the adapter; the configured
values remain in the immutable config snapshot.

Bootstrap constructs, in order:

1. the concrete `oauthintrospection` policy and fixed-authority client without
   provider I/O;
2. the shared `bearerauthn` runtime around that engine;
3. the existing HTTP router and optional gRPC server using the runtime's
   current resolver/interceptors.

Ownership remains with the constructing stage until the complete runtime is
returned. Any later startup failure closes it through the existing deferred
authentication cleanup. There is no startup probe, synthetic token, background
goroutine, cache, or readiness dependency.

## Fixed provider edge

`oauthintrospection` owns one `internal/infra/httpclient` instance and chooses
these non-configurable operation bounds:

| Property | Selected value |
| --- | --- |
| Provider sub-timeout | 5 seconds, always below the default 8-second HTTP request budget; a shorter caller budget wins. |
| Response-header timeout | 5 seconds, matching the provider sub-timeout; a shorter caller budget still wins. |
| Response-header size | 32 KiB maximum, enforced by the fixed-authority transport through the design-owned `ResponseLimits` seam. |
| Decoded response body | 1 MiB maximum, enforced by `oauthintrospection` while reading the response. |
| Redirects | Refused; every non-200 response, including 3xx, is unavailable trust. |
| Proxy | Disabled by the fixed-authority client. |
| Attempts | Exactly one; retry policy disabled. |
| Active-connection cap | None inside the client; the outer HTTP and gRPC business slots are the only provider-call ceiling and therefore create no second wait queue. |
| Outbound instrumentation | None in the accepted fixed-authority client. If an accepted concurrent client adds instrumentation, this provider must disable it; only bounded authentication verification telemetry is emitted. |

The additive `httpclient` surface is deliberately narrower than the dirty
candidate:

```text
ResponseLimits {
    ResponseHeaderTimeout
    MaxResponseHeaderBytes
}

NewExternalHTTPSWithLimits(baseURL, limits)
NewPrivateHTTPSWithLimits(baseURL, privateHostSuffix, limits)
```

Both fields must be positive. These constructors reuse the existing target
validation, post-DNS gate, proxy prohibition, redirect refusal, propagation
sanitizer, pool, `Do`, and close behavior; they only set the two transport
header guards. `oauthintrospection` supplies the fixed five-second header
timeout and 32 KiB header maximum. The existing constructors keep their current
behavior for current callers. Request timeout remains a derived context in
`oauthintrospection`, decoded-body admission remains beside its parser, and no
retry, metric, dependency name, target enum, connection limit, or generic
provider configuration is added to `httpclient` by this design.

TLS roots, private DNS/route existence, egress policy, credential provisioning,
and provider capacity remain deployment-owned inputs. Enabling the profile is
blocked until deployment certifies provider capacity for the sum of enabled
HTTP and gRPC business ceilings.

### Canonical request

For every syntactically accepted bearer value, the engine builds exactly one
new request from the immutable endpoint:

```text
POST <configured introspection endpoint>
Content-Type: application/x-www-form-urlencoded
Accept: application/json
Authorization: Basic <client_secret_basic credential>

token=<form-encoded opaque bearer>&token_type_hint=access_token
```

The form is produced by `net/url.Values`. `client_secret_basic` first applies
the same `application/x-www-form-urlencoded` escaping to the exact client ID and
secret required by OAuth 2.0, then uses standard-library Basic header encoding;
raw `SetBasicAuth(clientID, clientSecret)` is not equivalent for reserved
characters and is not selected. The bearer is present only in the request body,
never in URL, context values, logs, spans, metrics, or returned errors.

## Response and local trust admission

The adapter accepts only HTTP 200 whose parsed media type is exactly
`application/json`; syntactically valid media-type parameters are allowed and
ignored. The body limit applies to bytes delivered to the parser, including
transparent decompression.

Use the standard `encoding/json` token decoder with `UseNumber`, a top-level
member seen-set, and per-member raw values. This is the minimum mechanism that
can simultaneously reject duplicate top-level names and trailing values while
ignoring unknown members. No generic schema or reflection framework is added.

Admission follows the Specification's order:

1. establish one bounded JSON object, unique top-level names, and no trailing
   value;
2. require `active` to be a JSON boolean;
3. return sanitized invalid immediately for `active=false`, without deriving or
   disclosing why and without interpreting optional members;
4. for `active=true`, decode registered members with exact types, parse `exp`
   and optional `nbf` as exact finite integral NumericDate values without float
   truncation, and apply the shared 30-second skew;
5. enforce exact returned issuer, audience containment, and untrimmed stable
   `sub`/`client_id` rules;
6. return only the configured issuer, admitted subject/client ID, and expiry to
   `bearerauthn`.

Provider state is authoritative for `active`; the service remains authoritative
for response shape, issuer, audience, time, and principal construction. Unknown
members are discarded. Scope, username, token ID, extensions, provider body,
and raw status never leave the concrete engine.

## Material flows and finality

### HTTP protected request

1. Existing `RequestTimeout` and queue-free `MaxInFlight` admit or reject the
   request before authentication.
2. OpenAPI selects the existing Bearer scheme. `bearerauthn` reads the carrier,
   deletes `Authorization`, and rejects missing/malformed/oversized input before
   any provider effect.
3. `oauthintrospection` spends at most one five-second attempt under the caller
   context.
4. An inactive or locally invalid result is final invalid caller credential; an
   unusable provider interaction is final unavailable trust for this request.
5. Only a strict active result is published through the existing
   `httpx.Authenticated`/`reqctx.Principal` seam and reaches the handler.

There is no possibly committed local effect, replay, reconciliation, or stale
identity. The next request starts a new independent decision.

### Native gRPC request or stream

The existing 256-RPC business admission remains outside authentication.
`bearerauthn` removes incoming authorization metadata, leaves only exact
`grpc.health.v1.Health/Check` public, and applies the same concrete engine.
Unary finality is the returned status or handler admission. A protected stream
receives the admitted principal and a deadline no later than `exp + 30s`, with
message I/O checks preserved from the JWT profile.

### Cancellation and provider timeout

The provider sub-context derives from the caller context. After any provider
I/O failure, the engine checks the caller context first:

- caller canceled -> existing HTTP timeout rendering or gRPC `Canceled`, as
  owned by the current transport contract;
- caller deadline expired -> existing HTTP timeout rendering or gRPC
  `DeadlineExceeded`;
- five-second provider sub-timeout while the caller remains live -> sanitized
  unavailable trust;
- every other connection, TLS, DNS, endpoint-auth, status, media-type, size,
  or response-contract failure -> sanitized unavailable trust.

No raw dependency error is wrapped into the shared authentication error.

### Shutdown and recovery

Readiness is set draining and serving transports drain through the existing
shared shutdown budget. New admission stops before the authentication runtime
is closed. Request contexts cancel any forced-out provider I/O; the concrete
engine's idempotent close then releases idle connections before dependency
cleanup and telemetry flush. There is no worker to join.

Recovery from provider failure is the next successful request-time
introspection. There is no remembered failure, half-open state, retry wave, or
operator bypass.

## Authority and observability map

| Fact or effect | Authority | Consumer / signal |
| --- | --- | --- |
| Active, revoked, or otherwise inactive token state | Authorization server response for this request | `oauthintrospection` admission only |
| Expected issuer, API audience, endpoint, target class, endpoint credential | Immutable `internal/config` snapshot supplied by deployment | Bootstrap and concrete engine |
| Fixed destination and resolved-address admission | `internal/infra/httpclient` plus deployment network/TLS policy | One provider attempt |
| Bearer grammar, removal, failure taxonomy, principal publication, stream expiry | `internal/infra/bearerauthn` | HTTP/gRPC transports and handlers |
| Handler-visible identity | `reqctx.Principal` produced only after strict admission | Feature and authorization owners |
| HTTP/gRPC capacity ceiling | Existing transport owners and deployment-enabled transport set | Provider concurrency bound |
| Verification signal | Existing `authn.verifications` transport/result/reason vocabulary | Metrics/alerts |
| Provider availability | Per-request `unavailable` verification outcomes | Deployment-owned alert policy; never readiness |

No endpoint, status, issuer, audience, subject, client ID, token-derived value,
or response field becomes a metric attribute. Introspection adds no per-request
log or trace span. Startup reporting names only configuration keys/reason
classes, never values.

## Initializer and generated-source mechanism

Use three source markers:

- `authn-bearer` for the OpenAPI/HTTP/gRPC/principal/error/bootstrap surfaces
  common to both selected profiles;
- `authn-oidc-jwt` for JWT policy, Discovery/JWKS engine, token-profile config,
  dependencies, docs, and proof;
- `authn-oidc-introspection` for the RFC 7662 engine, its config fields, docs,
  and proof.

The source template keeps JWT as its concrete bootstrap/config implementation
and also compiles the dormant introspection engine. Static files under
`scripts/profiles/authn-oidc-introspection/` replace only the concrete authn
config validator and bootstrap constructor during introspection initialization;
the common bearer runtime is not copied or duplicated.

| `AUTHN` | Generated result |
| --- | --- |
| unset or `none` | Remove common bearer, both concrete engines, authn config/docs/tests, Bearer OpenAPI security, and authn-only dependencies; record `authn = "none"`. |
| `oidc-jwt` | Keep common bearer plus JWT engine; remove introspection package/config/docs/tests/template sources; record `authn = "oidc-jwt"`. |
| `oidc-introspection` | Keep common bearer plus introspection engine; replace concrete config/bootstrap files, remove JWT engine/token-profile/docs/tests/dependencies, and record `authn = "oidc-introspection"`. |

Every branch removes all profile markers and `scripts/profiles`, regenerates the
OpenAPI output when present, runs `go mod tidy`, and remains idempotent against
the exact `template.lock` choice. The shared OpenAPI scheme becomes neutral
Bearer access-token wording; it does not claim that the token is a JWT or expose
an introspection-specific contract.

## Proof boundary handed to Test Design

Technical Design fixes these proving owners without selecting a scenario
matrix:

- `bearerauthn` owns carrier, transport, taxonomy, metric, principal, public
  probe, and stream-expiry proof;
- `oauthintrospection` owns the real TLS provider crossing, exact request,
  limits, no-retry/no-redirect/no-proxy behavior, strict response admission,
  cancellation distinction, cleanup, and disclosure proof;
- `internal/infra/httpclient` owns direct enforcement proof for the two
  caller-supplied response-header guards, independent of provider semantics;
- `internal/config` owns immutable tuple, secret-source, and validation proof;
- bootstrap owns construction order, partial-startup cleanup, drain, and close;
- initializer checks own three-way physical presence/absence, dependency graph,
  generated OpenAPI, `template.lock`, and idempotence.

Local proof cannot establish live provider compatibility, credentials,
network/TLS/egress, revocation propagation, capacity entitlement, deployment
rate limiting, or alert thresholds. Test Design remains required next and is
not entered here.

## Reopen conditions

Reopen Specification only under the explicit conditions in `../spec.md`,
including a different endpoint credential lifecycle, provider-specific response
mapping or request extensions, signed responses, mTLS/DPoP binding,
authorization consumption of additional fields, cache/stale/retry/failover/
multi-endpoint/readiness behavior, or a distinct provider budget. Positive
caching additionally requires every accepted `R`, `P`, and representative-load
input named there.

Reopen System / Integration Design, without reopening Specification, if the
accepted fixed-authority client can no longer supply destination pinning,
post-DNS admission, no proxy/redirect/retry, required bounds, or queue-free
provider concurrency; if transport admission moves inside authentication; or if
initializer generation cannot make unselected trust paths physically absent.

Reopen Go Code / Ownership Design if current code changes the composition root,
shared authentication seam, generated/manual owner, or acyclic import placement.
An implementation-start refresh that only changes concrete constructor names is
mechanical and does not reopen either design.
