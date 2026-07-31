# Optional OIDC JWT Authentication

status: ready

## Scope and non-goals

An initialized service chooses exactly one profile:

- `AUTHN=none` (default) produces the current unauthenticated template without
  authentication-owned source, configuration, dependency, OpenAPI component,
  bootstrap wiring, documentation, or tests.
- `AUTHN=oidc-jwt` produces a single-issuer resource server that accepts only
  strict RFC 9068 signed JWT access tokens and publishes an authenticated
  `reqctx.Principal`.

The capability establishes identity only. Authorization, RBAC, tenant policy,
domain ownership, user provisioning, sessions, introspection, revocation,
token issuance, DPoP/mTLS-bound tokens, and replay storage remain out of scope.
No fake identity, disabled-verification switch, insecure-development mode, or
production bypass is permitted.

## Behavior and contract delta

### Profile and configuration

**AUTHN-01 — Structural choice.** `AUTHN` accepts only `none` or `oidc-jwt`.
Invalid or empty-explicit values are rejected before any destination file is
mutated. The selected value is recorded in `template.lock`; repeating
initialization with the same complete inputs is byte-stable.

**AUTHN-02 — Absence means absence.** `AUTHN=none` physically removes every
capability-owned artifact and dependency. It leaves neither a dormant
authentication framework nor no-op/fake bootstrap wiring. `GRPC` remains an
independent choice.

**AUTHN-02A — Required egress prerequisite.** `AUTHN=oidc-jwt` retains the
existing bounded `internal/infra/httpclient` as an internal Discovery/JWKS
security prerequisite even when the independent `OUTBOUND_HTTP` choice is
`none`. That requested value still means no general service/provider outbound
capability, config, example, or documentation is retained; it does not require
duplicating the repository's fixed-authority network policy inside authn.
`AUTHN=none,OUTBOUND_HTTP=none` still removes the package, while either
`AUTHN=oidc-jwt` or `OUTBOUND_HTTP=bounded` retains it. `template.lock` records
the requested values separately.

**AUTHN-03 — Immutable trust policy.** `AUTHN=oidc-jwt` requires one immutable
runtime policy containing:

- one HTTPS issuer URL with no user information, query, or fragment;
- one non-empty expected audience;
- at least one trusted HTTP proxy CIDR for the TLS-terminated HTTP hop;
- the fixed allowed signing algorithm `RS256`;
- maximum compact-token size 8 KiB;
- clock skew 30 seconds;
- discovery/JWKS request timeout 5 seconds;
- refresh interval 5 minutes;
- maximum key-set age 15 minutes;
- unknown/signature-miss refresh cooldown 30 seconds;
- maximum discovery or JWKS body 1 MiB and maximum 100 total JWK entries.

Missing, malformed, internally inconsistent, unsafe, or incomplete policy
causes startup failure before either application listener serves. These values
are conservative profile policy, not provider-controlled defaults. Changing a
fixed value is a compatibility/security-policy change that reopens
Specification.

**AUTHN-04 — Transport confidentiality.** HTTP authentication is accepted only
when the immediate peer is in the configured trusted-proxy CIDR set and the
proxy supplies exactly one `X-Forwarded-Proto` field with the single
case-insensitive value `https` and no comma-joined value. A direct or spoofed
forwarded indication from any other peer is rejected before credential parsing.
When `GRPC=enabled`, the gRPC server must use its existing native TLS transport
mode; the authn profile and plaintext gRPC configuration are invalid together.

### OpenAPI and HTTP behavior

**AUTHN-05 — Fail-closed OpenAPI default.** In the OIDC profile the canonical
OpenAPI document defines one HTTP Bearer JWT security scheme and selects it as
the single top-level security requirement. Only the existing liveness and
readiness operations override it with `security: []`. No anonymous `{}` OR
alternative exists. A newly added operation is therefore protected unless its
public status is an intentional canonical-contract change.

**AUTHN-06 — Exact Bearer extraction.** A protected HTTP request must contain
exactly one `Authorization` field whose value consists of the case-insensitive
scheme `Bearer`, one separating space, and one non-empty token without leading,
trailing, comma-joined, or additional credential data. Duplicate fields,
alternate schemes, and ambiguous whitespace are malformed. The 8 KiB compact
token limit is applied before decode, cryptography, or outbound work.

**AUTHN-07 — HTTP outcomes.**

| Condition | Outcome | Challenge / retry contract |
| --- | --- | --- |
| No Authorization field | `401 Unauthorized` | `WWW-Authenticate: Bearer` |
| Well-formed Bearer credential but token is cryptographically or semantically invalid | `401 Unauthorized` | `WWW-Authenticate: Bearer error="invalid_token"` |
| Duplicate or malformed Authorization credential | `400 Bad Request` | `WWW-Authenticate: Bearer error="invalid_request"` |
| Compact token exceeds 8 KiB | `431 Request Header Fields Too Large` | no token content and no handler call |
| Trust is temporarily unavailable beyond the permitted key age | `503 Service Unavailable` | sanitized fixed response; `Retry-After: 30` |
| Immediate peer is not the trusted TLS boundary | `400 Bad Request` | sanitized fixed response; credential is not parsed |

Authentication never returns `403`; authorization owns that status. Bodies use
the repository's stable sanitized problem shape and reveal no token, claim,
key, issuer endpoint, parser text, or dependency text. A rejection never invokes
the operation handler.

### JWT access-token contract

**AUTHN-08 — Accepted serialization.** The token is exactly one signed compact
JWS with three base64url segments and one protected header. JWE, nested JWT,
multiple signatures, detached/unencoded payload, unsecured JWT, padded or
non-canonical base64url, duplicate JSON members, and trailing data are rejected.
Any `crit` or `b64` header and any security-relevant unprotected header are
rejected. `jku`, `x5u`, and embedded `jwk` never influence trust.

**AUTHN-09 — Header policy.** The protected `alg` is exactly `RS256`. The
protected `typ`, compared by media-type case-insensitive rules, is exactly
`at+jwt` or `application/at+jwt` with no parameter. A protected non-empty
`kid` is required as an explicit stricter local interoperability policy.

**AUTHN-10 — Signature and key policy.** Verification succeeds only when the
current trusted JWKS contains exactly one usable RSA signing key with the exact
`kid`, compatible algorithm/use/key-operations metadata, and acceptable public
key parameters. Missing, duplicate, ambiguous, incompatible, private/symmetric,
or undersized matching keys are rejected. A JWKS may contain well-formed keys
for other algorithms or uses; they are ignored. A candidate JWKS is installed
atomically only when the JSON and every JWK are structurally valid, the total
set contains no more than 100 entries, and at least one usable RS256 signing key
remains.

**AUTHN-11 — Required claims.** After signature verification, `iss`, `sub`,
`client_id`, `jti`, `exp`, and `iat` must each be present exactly once with the
RFC 9068 type; `aud` is one non-empty string or a non-empty array of unique
non-empty strings. `iss` exactly equals the configured issuer and `aud`
contains the configured audience exactly. `sub`, `client_id`, and `jti` are
non-empty. Unknown claims are non-authoritative and ignored.

**AUTHN-12 — Time policy.** At one injected current time:

- `exp` must be later than current time minus 30 seconds;
- optional `nbf` must not be later than current time plus 30 seconds;
- `iat` must not be later than current time plus 30 seconds.

No local maximum token age is imposed beyond mandatory `exp`. NumericDate must
be finite, integral, and representable without overflow. Time is checked again
on every request; cache freshness never extends token validity.

**AUTHN-13 — Principal mapping.** A valid token produces
`reqctx.Principal{Subject: sub}` with no derived scopes. The subject remains the
provider's opaque value and is meaningful only within the one configured
issuer. `client_id`, tenant, email, display name, roles, permissions, and scope
claims do not become identity or authorization. HTTP and gRPC publish the same
principal value and never publish the raw token or claims object.

### Discovery, key lifecycle, and recovery

**AUTHN-14 — Initial trust.** Startup derives the OIDC Discovery location from
the configured issuer, fetches it through the repository's public-HTTPS egress
policy, requires exact metadata issuer equality and one HTTPS `jwks_uri`, then
fetches and validates an initial JWKS. There is no OAuth-metadata fallback,
redirect, caller-supplied JWKS URL, stale disk state, empty trust, or delayed
ready-without-keys state. Discovery and JWKS JSON reject duplicate members at
every object depth, trailing data, and values of the wrong type rather than
using parser-dependent first/last-member behavior. Cancellation or any failure
aborts startup.

**AUTHN-15 — Bounded current trust.** The last completely validated JWKS is the
only runtime key authority. It is current for at most 15 minutes after its last
successful fetch. Refresh occurs at least every 5 minutes. A failed refresh
inside that interval does not invalidate known keys; after the interval no
token can be authenticated until a successful refresh installs a new set.
Readiness is false whenever no current trusted set exists and recovers only
after successful refresh.

**AUTHN-16 — Rotation and abuse resistance.** The first token with an unknown
`kid`, or a signature miss for an otherwise compatible known `kid`, outside the
cooldown triggers exactly one shared refresh and then exactly one verification
retry. Concurrent waiters coalesce onto that refresh. Sequential misses during
the next 30 seconds do not start another refresh, regardless of
attacker-selected `kid`; they are answered from the then-current trust state.
Scheduled refresh remains eligible during the miss cooldown. A canceled request
stops waiting without canceling work shared by live waiters or starting
duplicate work. Shutdown cancels refresh work and releases network resources.

**AUTHN-17 — Failure precedence.** Local syntax/header/claim checks that need no
key run before any attacker-triggered refresh. With a current key set, a token
that is provably invalid remains `invalid_token` even during an issuer outage.
When a required key fact cannot be established and no current trust can answer
it, the result is `unavailable`, never a fabricated invalid-token fact.

### gRPC parity

**AUTHN-18 — RPC boundary.** When `GRPC=enabled`, unary and streaming RPCs
extract exactly one `authorization` metadata value using AUTHN-06 semantics,
authenticate once before the application handler, and publish the principal in
the immutable RPC/stream context. Only the standard gRPC health service methods
are public; every other current or future full method is authenticated.

**AUTHN-19 — gRPC outcomes.**

| Condition | gRPC status |
| --- | --- |
| Missing, malformed, duplicate, or invalid credential | `UNAUTHENTICATED` |
| Token exceeds 8 KiB | `RESOURCE_EXHAUSTED` |
| Trust is unavailable | `UNAVAILABLE` |
| Authorization denial (outside this capability) | `PERMISSION_DENIED` |

All messages are fixed and sanitized. The application receives neither raw
authentication metadata nor a claims object.

### Observability and secret handling

**AUTHN-20 — Bounded telemetry.** Authentication may emit only transport,
success/failure, and a fixed finite reason class (`missing`, `malformed`,
`oversize`, `invalid`, or `unavailable`). Tokens, Authorization values, compact
segments, claims, subjects, signatures, attacker-controlled `kid`, discovered
endpoint, response body, and parser/network text never appear in logs, traces,
metrics, returned errors, or panic output. Configured issuer/audience are not
metric labels.

**AUTHN-21 — Credential lifetime.** The raw token exists only long enough for
transport extraction and verification. Before application dispatch, the HTTP
Authorization field or gRPC metadata credential is removed from the
handler-visible request/context. The verifier stores neither tokens nor claims.

**AUTHN-22 — Required authentication signals.** The OIDC profile exports:

- a verification counter labeled only by `transport`, `result`, and the fixed
  reason classes in AUTHN-20;
- a JWKS refresh counter labeled only by fixed trigger
  (`startup`, `scheduled`, or `key_miss`) and result;
- an unlabeled age gauge for the current trusted key set.

No authentication metric exists in `AUTHN=none`. Logs remain event-oriented
and sanitized; they do not duplicate every per-request metric.

**AUTHN-23 — Local development and examples.** Repository tests and
documentation demonstrate real RS256-signed access tokens and deterministic
test JWKS/discovery data. Production verification has no test-key fallback.
Running a generated service locally therefore requires an actual HTTPS issuer
and the same complete trust/TLS configuration as any other environment.

## Invariants and edge cases

- Authentication is fail closed: incomplete trust blocks startup, expired
  trust blocks authentication/readiness, and malformed input never degrades to
  anonymous access.
- A dependency outage can preserve authentication only for known keys inside
  the 15-minute current-trust interval; it cannot extend token expiration.
- JWKS refresh is atomic. A malformed, empty, excessive, or ambiguous candidate
  cannot erase or partially replace last-known-good keys.
- Health endpoints remain public and unchanged except that their transport is
  still subject to the service's deployment/TLS topology.
- The RFC 9068 `jti` claim does not provide replay prevention. A stolen valid
  bearer token remains replayable within its validity window; this is an
  explicitly retained risk bounded by TLS and token expiry.
- Existing `AUTHN=none` runtime and public health behavior is deliberately
  unchanged.

## Decisions, constraints, and authorities

- RFC 9068 and RFC 8725 own the token-profile and JWT-security baseline; OIDC
  Discovery owns issuer bootstrap; RFC 6750 owns HTTP Bearer challenge
  semantics; gRPC owns transport status categories.
- Canonical OpenAPI, not generated Go, owns HTTP security requirements.
- Config is immutable after validated startup. The configured issuer and
  audience are trust authority; discovery/JWKS are derived trust observations
  and never select a different issuer.
- The last wholly validated, not-older-than-15-minutes JWKS is authoritative
  for current signing keys. A partially parsed response is never authority.
- `reqctx.Principal` is the sole application identity path. The verifier and
  carrier adapters are concrete capability code, not a generic authentication
  framework.

## Success criteria and proof expectations

The capability is acceptable only when deterministic proof falsifies the rules
above at the nearest owner:

- real signed-token cases cover valid, malformed, unsigned, wrong signature,
  issuer, audience, type, algorithm, expired, premature, future-issued,
  missing-subject/required-claim, unknown/duplicate `kid`, critical headers,
  duplicate JSON, and oversize tokens;
- deterministic discovery/JWKS cases cover initial outage, strict metadata,
  atomic invalid-set rejection, rotation including same-`kid`, removed keys,
  cached outage before/after cutoff, concurrent refresh, sequential-miss
  cooldown, waiter cancellation, timeout, readiness recovery, and shutdown;
- HTTP contract proof covers missing/malformed/duplicate/oversize/unavailable
  outcomes, challenge headers, trusted-proxy enforcement, identity reaching a
  protected handler, raw-header removal, public health, and fail-closed
  OpenAPI default;
- when enabled, equivalent unary/stream gRPC proof covers native TLS startup,
  health exception, metadata handling/removal, identity, status parity, and
  application non-invocation;
- captured logs, spans, metrics, errors, and panics contain no token, claim,
  subject, `kid`, endpoint body, or dependency/parser text;
- race/liveness proof covers verification, refresh coalescing, cancellation,
  readiness updates, and shutdown;
- initializer proof covers `AUTHN=none` purity, OIDC HTTP-only and HTTP+gRPC
  compilation, the shared egress prerequisite under both `OUTBOUND_HTTP`
  choices, canonical generation, module cleanup, invalid-value before-mutation
  behavior, and repeated byte stability.

Completion evidence is scoped to the recorded toolchain, dependency versions,
profiles, and tests; it does not claim universal security.

## Risks, assumptions, and reopen conditions

- The HTTP runtime is behind a trusted TLS-terminating proxy whose source
  networks are expressible as fixed CIDRs. Reopen System Design before
  implementation if the target deployment cannot provide that boundary; do not
  silently trust forwarded headers or add an insecure mode.
- One issuer per process keeps `sub` unambiguous in `reqctx.Principal`. A
  multi-issuer outcome reopens identity representation and policy.
- `RS256` is the only initial algorithm to minimize algorithm/key-family
  complexity while satisfying RFC 9068 interoperability. A provider that
  cannot issue RS256 access tokens reopens Specification rather than widening
  the list ad hoc.
- The 15-minute trust interval balances bounded outage tolerance with bounded
  key-removal lag. An issuer contract requiring faster revocation or longer
  offline trust reopens Specification.
- Replay resistance beyond TLS and expiry requires a separately accepted
  sender-constrained-token or stateful policy.
