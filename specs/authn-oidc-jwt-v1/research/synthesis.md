# OIDC/JWT Authentication Research

status: ready

Valid as of: 2026-07-31

## Scope and method

This synthesis closes the evidence questions that can change the authentication
specification or design. It is organized by claim rather than by source and
distinguishes:

- **Normative**: an applicable standards requirement.
- **Practice**: a pattern observed in maintained production Go code, not a
  requirement.
- **Library**: behavior of a named release, including its limits.
- **Repository**: a current local ownership or profile constraint.
- **Disputed**: credible alternatives that Research cannot select.
- **Inference**: a conservative implication to be decided by the named
  downstream owner.

All external sources were accessed on 2026-07-31. Release and commit references
are immutable; living registries are valid only as of that date.

## Decision-changing conclusions

### 1. The accepted credential is an RFC 9068 JWT access token

**Normative.** [RFC 9068](https://www.rfc-editor.org/rfc/rfc9068.html)
(Proposed Standard, October 2021) defines the applicable access-token profile.
The token is a signed compact JWS with media type `application/at+jwt`; its JOSE
header `typ` is `at+jwt`, compared using media-type case-insensitive rules. The
profile requires `iss`, `sub`, `aud`, `exp`, `iat`, `jti`, and `client_id`.
Encrypted, nested, unsecured, generic `JWT`, absent-type, and
parameterized-type tokens are outside this capability.

**Normative.** [RFC 8725](https://www.rfc-editor.org/rfc/rfc8725.html)
(BCP 225, February 2020) requires callers to allow only explicitly configured
algorithms, validate all cryptographic operations, prevent cross-JWT confusion
with mutually exclusive validation rules, and validate issuer and audience.
[RFC 7515](https://www.rfc-editor.org/rfc/rfc7515.html),
[RFC 7517](https://www.rfc-editor.org/rfc/rfc7517.html),
[RFC 7518](https://www.rfc-editor.org/rfc/rfc7518.html), and
[RFC 7519](https://www.rfc-editor.org/rfc/rfc7519.html) (Proposed Standards,
May 2015) own JWS, JWK, algorithms, and JWT syntax. The
[IANA JOSE registry](https://www.iana.org/assignments/jose/jose.xhtml), last
updated 2026-05-22 when inspected, is the current algorithm-name authority.

**Normative.** RFC 9068 requires support for `RS256` by conforming issuers and
resource servers. [RFC 9864](https://www.rfc-editor.org/rfc/rfc9864.html)
(Proposed Standard, October 2025) updates the JOSE algorithm registry and
deprecates the polymorphic `EdDSA` identifier in favor of curve-specific
identifiers. It does not remove the RFC 9068 `RS256` interoperability floor.

**Normative.** `iss` must exactly equal configured issuer policy; `aud` must
contain the configured resource audience as an exact string, while additional
audiences are permitted. `exp` is mandatory and must not have passed, `nbf`
when present must not be in the future, and `iat` is mandatory. JWT NumericDate
comparisons may apply a small configured allowance for clock skew.

**Disputed.** A reported RFC 9068 erratum proposes weakening mandatory `iss`.
It was not incorporated into the published RFC at the evidence date. The
published profile remains the authority until an applicable approved update
reopens Specification.

**Inference for Specification.** Require the complete RFC 9068 claim set,
exact issuer and expected-audience membership, `RS256` as the initial explicit
algorithm policy, and a small bounded skew. Whether to reject old-but-unexpired
tokens by maximum `iat` age is local policy rather than a standards requirement
and remains open for Specification.

### 2. Discovery bootstraps one fixed issuer; it is not issuer selection

**Normative.** [OpenID Connect Discovery 1.0 incorporating errata set 2](https://openid.net/specs/openid-connect-discovery-1_0.html)
(Final, 2023-12-15) defines issuer-derived discovery and requires the returned
`issuer` value to exactly match the issuer used to construct the request.
[OpenID Connect Core 1.0 incorporating errata set 2](https://openid.net/specs/openid-connect-core-1_0.html)
(Final, 2023-12-15) supplies OIDC issuer and ID-token context; its ID-token
validation rules are not a substitute for RFC 9068 access-token validation.
[RFC 8414](https://www.rfc-editor.org/rfc/rfc8414.html) (Proposed Standard,
June 2018) defines OAuth authorization-server metadata but is not an
authorization to probe multiple metadata conventions after an OIDC failure.

**Normative.** Signing keys come from the discovered `jwks_uri`. Attacker-owned
`jku`, `x5u`, or embedded `jwk` header values must not become trust authorities.
The selected JWK must be structurally compatible with the allowed signature
algorithm and signing use/key operations. RFC 9068 recommends, but does not
universally require, a `kid` header.

**Inference for Specification.** This single-issuer rotating-key profile may
adopt the stricter local policy that every token and every usable JWK has a
non-empty `kid` and that exactly one compatible JWK matches it. If selected,
that is an explicit local interoperability restriction, not a normative claim.

**Repository.** `internal/infra/httpclient.ExternalHTTPS` already owns
authority-pinned public-HTTPS egress, redirects, DNS/address policy, bounded
headers/bodies/connections, cancellation, and idle-connection cleanup. OIDC
discovery and JWKS access must reuse that owner rather than creating a second
network-security policy.

**Inference for System Design.** Configuration names exactly one issuer.
Startup derives exactly its OIDC discovery URL, requires exact metadata issuer
equality, builds a separately authority-pinned JWKS client for the discovered
HTTPS URI, and establishes initial keys before serving. There is no fallback
to caller-supplied key URLs or opportunistic metadata probing.

### 3. Bearer extraction is a transport security boundary

**Normative.** [RFC 6750](https://www.rfc-editor.org/rfc/rfc6750.html)
(Proposed Standard, October 2012, updated by RFCs 8996 and 9700) defines HTTP
Bearer credentials and requires TLS. A missing credential challenges with
`401` and `WWW-Authenticate: Bearer`; an invalid token is `401
invalid_token`; malformed or multiple authentication mechanisms are
`invalid_request`. [RFC 9700](https://www.rfc-editor.org/rfc/rfc9700.html)
(BCP 240, January 2025) reinforces current OAuth security practice.

**Normative.** gRPC authentication metadata is transported as HTTP/2 metadata.
Official [gRPC authentication](https://grpc.io/docs/guides/auth/),
[metadata](https://grpc.io/docs/guides/metadata/), and
[status-code](https://grpc.io/docs/guides/status-codes/) documentation requires
transport credentials for bearer use and distinguishes `UNAUTHENTICATED` from
authorization-owned `PERMISSION_DENIED`.

**Practice.** Kubernetes v1.36.3 (commit `0f29094`, 2026-07-23) parses a single
Bearer credential, removes spoofable/raw authentication headers before calling
the application, and publishes typed user context. Auth0
`go-jwt-middleware` v3.2.0 (commit `fb96150`, 2026-05-15) uses equivalent unary
and wrapped-stream gRPC interceptors, but leaves original metadata reachable
and maps some attacker-controlled key misses to `Internal`.

**Repository.** `internal/infra/http.RouterConfig.Authenticate` is the existing
fail-closed OpenAPI authentication seam. `http.Authenticated` maps a resolved
non-empty identity into `reqctx.Principal`. `internal/infra/grpc.NewServer`
already accepts unary and streaming policy interceptors in the correct
post-recovery/admission, pre-domain-error boundary. Health RPCs are current
infrastructure endpoints and need explicit public treatment.

**Disputed.** RFC 6750's `400 invalid_request` for malformed requests conflicts
with the current HTTP seam's uniform `401` rejection path. API Design must
decide whether the capability adds a narrow typed rejection mapping or
documents a conservative uniform `401`; authentication must never emit `403`.
Dependency/trust unavailability is neither malformed nor an invalid token and
needs a distinct service-unavailable outcome.

**Inference for API and gRPC Design.** Accept exactly one Authorization
field/value and one `Bearer` credential. Reject duplicates, comma-joined
credentials, empty tokens, alternate schemes, and oversize input before JWT
parsing or network work. HTTP and gRPC must call the same verifier and publish
the same principal. Raw credentials must not cross into handlers. Malformed
input, invalid credentials, and unavailable trust remain distinct finite
classes through the adapter.

### 4. The principal is issuer-scoped even if the repository stores `sub`

**Normative.** OIDC and JWT subject identifiers are locally unique within an
issuer, not globally unique. RFC 9068 also requires `client_id`, but it does not
turn the client into the resource-owner subject.

**Repository.** `internal/reqctx.Principal` currently has `Subject` and
`Scopes`. The accepted outcome requires this existing path, while
authorization, tenant interpretation, and domain ownership are out of scope.
Only one immutable issuer is configured per initialized service.

**Disputed.** Extending the shared principal with issuer/client fields produces
more explicit identity but leaves capability-shaped residue in `AUTHN=none`.
Encoding issuer and subject into an invented string risks an unstable,
non-standard application identifier.

**Inference for Specification.** Preserve the provider's opaque `sub` exactly
as `Principal.Subject`, with the invariant that it is meaningful only under
the verifier's single configured issuer. Do not map `client_id`, scopes, tenant
claims, email, or display names into authority-bearing identity. Reopen this
decision if multi-issuer authentication becomes an accepted outcome.

### 5. No inspected implementation supplies the complete key lifecycle

Evidence set:

- `coreos/go-oidc` v3.20.0, commit `75dfa5c`, released 2026-07-08.
- Kubernetes v1.36.3, commit `0f29094`, released 2026-07-23.
- Auth0 `go-jwt-middleware` v3.2.0, commit `fb96150`, released 2026-05-15.
- `oauth2-proxy` v7.15.2, commit `5961fd9`, released 2026-04-14.

**Library/practice.** `go-oidc.RemoteKeySet` verifies cached keys first, then
coalesces concurrent refresh after signature/key miss. Its shared fetch uses
`context.WithoutCancel`, waiters may independently cancel, sequential random
`kid` values can trigger sequential fetches, and cached known keys have no
TTL/max-stale cutoff.

**Practice.** Kubernetes adds a bounded HTTP client, lifecycle-owned discovery
retry, initialization health, and fail-closed verification, but inherits
`go-oidc` stale-key and sequential-refresh behavior.

**Practice.** Auth0's provider bounds JWKS response size, fetch time, and cache
TTL; proactively refreshes around 80% of TTL; serializes per-URI refresh; and
fails closed once keys expire. It deliberately does not immediately refresh a
still-valid set for an unknown `kid`, which limits attacker-driven fetches but
can delay an unannounced non-overlapping rotation.

**Practice/counterexample.** `oauth2-proxy` retains raw credentials in its
session because forwarding them is a gateway responsibility. That boundary is
not applicable to an in-service verifier.

**Disputed.** Prompt unknown-`kid` refresh and refresh-abuse resistance pull in
opposite directions. The last-known-good allowance, refresh-ahead point,
maximum stale interval, retry cooldown, and readiness effect are local
reliability policies. Research found no implementation that closes all of
them without additional ownership.

**Inference for System/Reliability Design.** One lifecycle-owned cache per
configured issuer should perform cache-first verification, scheduled
refresh-ahead, one coalesced bounded refresh for an unknown `kid`, and a global
cooldown/negative result for sequential misses. Known keys may remain usable
only through a fixed maximum age; initial trust and trust beyond that age fail
closed. A canceled request may stop waiting but must not create duplicate
refresh work.

### 6. Observability must describe outcomes, never credentials or claims

**Normative/inference.** Bearer possession authorizes use, so the raw
Authorization value, compact token, JOSE parts, decoded claims, signature,
subject, arbitrary `kid`, and upstream response bodies are secrets or
attacker-controlled sensitive values. They cannot appear in errors, logs,
traces, metrics, or panic output.

**Practice.** Kubernetes OIDC metrics record coarse results and hashes of a
bounded configured issuer/server set. Hashing alone does not bound cardinality;
configuration bounds it. Auth0 interceptors omit token contents from returned
errors.

**Repository.** Existing HTTP and gRPC telemetry can observe handler/interceptor
outcomes. The verifier must expose a finite error taxonomy rather than raw
parser/network errors. The repository's outbound client already bounds and
redacts transport diagnostics.

**Inference for Observability Design.** Record only transport, coarse result,
and fixed reason classes such as missing, malformed, invalid, unavailable, or
oversize. Never label with subject, `kid`, issuer discovered from input, claim,
or token. Preserve causes only inside non-exported classifications when doing
so cannot render secret material.

### 7. Structural optionality is a generated-service property

**Repository.**

- `scripts/init-module.sh` validates profile values before mutation, removes
  marker-delimited source blocks and capability files, regenerates canonical
  artifacts, tidies modules, and records choices in `template.lock`.
- Existing profiles are `DATABASE`, `GRPC`, `OUTBOUND_HTTP`, and
  `REFERENCE_EXAMPLE`; there is no current `AUTHN` profile or JWT dependency.
- Canonical OpenAPI is `api/openapi/service.yaml`; generated Go is derived by
  `internal/openapi/oapi-codegen.yaml`.
- The service OpenAPI currently contains only public liveness/readiness
  operations and top-level `security: []`. There is no dormant bearer scheme.
- Bootstrap composes HTTP in `cmd/service/internal/bootstrap/run.go` and optional
  gRPC through `startup_grpc.go`.
- `scripts/ci/template-init-check.sh` owns profile purity, repeatability, and
  invalid-value behavior.

**Inference for Design.** `AUTHN=none` remains the initializer default and
removes the concrete verifier, config fields/schema/defaults/validation,
dependency, OpenAPI bearer scheme, bootstrap, documentation, and tests. The
OIDC profile retains the HTTP pack. Its gRPC adapter is independently removed
when `GRPC=none`. No generic authentication abstraction remains in the base.
The OIDC profile must make authentication the OpenAPI default and override only
the two explicit health operations with `security: []`; an anonymous `{}` OR
alternative is forbidden. This makes newly added operations fail closed unless
their public status is an intentional canonical-contract edit.

**Inference for Test Design.** Required profile proof covers:

- `AUTHN=none` with both gRPC profiles, no authn paths/symbols/config/dependency;
- `AUTHN=oidc-jwt,GRPC=none`, HTTP verifier retained and gRPC adapter absent;
- `AUTHN=oidc-jwt,GRPC=enabled`, both adapters compile and use one verifier;
- invalid `AUTHN` rejected before mutation;
- repeated initialization byte stability;
- canonical OpenAPI/protobuf generation and module cleanup.

## Candidate-library evidence

No reviewed maintained module implements the whole access-token contract
unchanged. Exact current revisions and roles:

| Candidate | Current revision and fit | Missing or conflicting ownership |
| --- | --- | --- |
| `golang-jwt/jwt/v5` | v5.3.1, commit `7ceae619`, Go 1.21, MIT, no module dependencies. Exact method allowlist; issuer/audience/time validators; required `exp`/`nbf`; injectable clock; custom claims. | No discovery, JWKS lifecycle, `typ` policy, transport extraction, or principal mapping. |
| `go-jose/go-jose/v4` | v4.1.4, commit `0e598766`, Go 1.24, Apache-2.0, no module dependencies. Caller-supplied algorithm allowlist; exposed protected headers; deterministic claim-validation time. | No discovery or remote JWKS cache. Missing time claims are not rejected by its ordinary validation helper, so complete required-claim policy remains local. |
| `lestrrat-go/jwx/v3` | v3.2.0, commit `94006b67`, Go 1.25, MIT. JWT/JWS/JWK primitives, context-aware fetch, bounded response, HTTP-cache-aware periodic cache, deterministic clock and custom validators. | Exact application algorithm and `typ` policies remain local; required time claims and synchronous unknown-`kid` refresh are not supplied by the cache; cache shutdown is mandatory. |
| `lestrrat-go/jwx/v4` plus `jwkfetch` | jwx v4.2.0 commit `dba5b170`; jwkfetch v4.0.4 commit `447b00d8`; Go 1.26, MIT. Separates JOSE from HTTP/cache companion. | Both currently require `GOEXPERIMENT=jsonv2`, which the repository does not enable. Cache lifecycle and access-token policy still remain local. |
| Auth0 `go-jwt-middleware/v3` | v3.2.0, commit `fb961506`, Go 1.25, MIT. The only reviewed package combining discovery, TTL cache, validator, HTTP and gRPC adapters. | No access-token `typ` enforcement; required `exp` and deterministic clock need extra policy; no synchronous unknown-`kid` refresh; fetch redirects/body limits are weaker than the repository owner; output still needs a principal adapter. It pins jwx/v3 v3.0.13 rather than current v3.2.0. |
| `coreos/go-oidc/v3` | v3.20.0, commit `75dfa5c0`, Go 1.25, Apache-2.0. Exact discovery and coalesced cached-key retry. | Its public verifier explicitly validates **ID tokens** and consumes ID-token signing algorithms. It lacks RFC 9068 `typ` policy, bounded key lifetime/body, request-bound fetch, and lifecycle shutdown. |
| `MicahParks/keyfunc/v3` | v3.8.1, commit `39974f23`, Go 1.25, Apache-2.0. A `golang-jwt` key adapter with cancel-owned periodic refresh, rate-limited unknown-`kid` refresh, and current `golang-jwt` v5.3.1. | No discovery, claim, `typ`, transport, or principal policy. It pins `jwkset` v0.11.1, which contains the oversized ECDSA JWK panic fix but predates v0.11.2 base64url/canonicalization fixes. |
| `zitadel/oidc/v3` | v3.48.1, commit `3c8b7045`, Go 1.25, Apache-2.0. Contains an OP-side access-token verifier and a resource-server introspection client. | The local verifier belongs to the provider boundary and omits audience, `typ`, `nbf`, `iat`, deterministic clock, and hardened JWKS lifecycle. The resource-server package performs remote introspection, not local JWT verification. |

**Library.** Current `go-jose` v4.1.4 contains the fix for
[GHSA-78h2-9frx-2jm8](https://github.com/go-jose/go-jose/security/advisories/GHSA-78h2-9frx-2jm8).
Current `golang-jwt` v5.3.1 is newer than the v5.2.2 fix for
[GO-2025-3553](https://pkg.go.dev/vuln/GO-2025-3553). Current Go vulnerability
index results contained only fixed historical entries for the inspected
versions of `go-jose`, `golang-jwt`, and older jwx majors. Lack of an index
entry is not proof of no vulnerability.

**Candidate-space disposition.**

- `go-oidc` and Zitadel are eliminated as high-level verifiers because their
  owned token profiles do not match an RFC 9068 resource server.
- jwx/v4 is eliminated for this version because adopting a repository-wide
  experimental JSON build contract is disproportionate and unnecessary.
- Auth0 remains a full-stack substitute, but its missing policies and weaker
  egress boundary mean an adapter would still own most security decisions.
- `go-jose`, `golang-jwt`, jwx/v3, and the keyfunc composition remain focused
  primitive substitutes.
  Technical Design must compare dependency surface, protected-header access,
  claim validation, fetch-policy overlap, and lifecycle cost; popularity is not
  decision evidence.

No additional maintained candidate discovered in the current Go ecosystem
occupies the same decision slot with a materially stronger complete contract.
Before dependency readiness, the selected composition must be tested in this
module graph, scanned with current vulnerability data, and exercised against
adversarial tokens and deterministic rotation/outage.

## Adversarial reachability

| Reachable path | Consequence | Required enforcement / proof owner |
| --- | --- | --- |
| `alg=none`, algorithm omitted, or an HMAC key interpreted through RSA public-key bytes | Forged token accepted through algorithm confusion | Reject before key lookup using an exact non-empty asymmetric algorithm allowlist; adversarial parser tests. |
| Correct signature under a permitted algorithm but absent, generic, parameterized, or wrong `typ` | ID token or another JWT kind confused with an access token | Validate the protected `typ` header using RFC 9068 media-type equivalence; reject an unprotected or duplicate value. |
| Attacker supplies `jku`, `x5u`, or embedded `jwk` | SSRF or attacker key becomes trust anchor | Ignore these headers for key resolution; only discovered configured-issuer JWKS is authoritative. |
| Duplicate JSON members, malformed base64url/JSON, multiple signatures, JWE, nested JWT, or detached payload | Parser differential or validation bypass | Accept exactly three compact JWS segments and one signature; strict-decode protected header, claims, discovery, and JWKS; reject duplicate members and unsupported forms before semantic use. |
| Unknown critical JOSE header, `b64=false`, or unprotected security parameter | Verifier ignores semantics that change how the signing input must be interpreted | Reject every non-empty `crit`, every `b64` header, and security-relevant values outside the protected header. |
| Missing, wrong-type, empty, or conflicting required claim | Zero/default value bypass or identity ambiguity | Explicit presence/type/value checks for every RFC 9068 required claim after signature verification; table-driven conformance tests. |
| Wrong issuer or expected audience hidden among malformed/mixed values | Cross-issuer/resource replay | Exact configured issuer equality and exact expected-audience membership over only the permitted JSON forms. |
| Expired, premature, future-issued, extreme NumericDate, or large clock manipulation | Replayed or not-yet-valid token accepted | Inject one clock; overflow-safe NumericDate parsing; bounded symmetric skew for `exp`/`nbf` and explicit future-`iat` policy. |
| Empty `sub`, mutable display/email claim substituted as subject, or subject interpreted outside its issuer | Anonymous or cross-domain identity | Require non-empty opaque `sub`; one configured issuer scopes it; map no mutable/display/tenant claim. |
| Duplicate, comma-joined, folded, alternate-scheme, empty, or whitespace-obscured credentials | Proxy/parser disagreement or credential smuggling | Parse exactly one HTTP Authorization value or one gRPC metadata value; accept exactly one case-insensitive Bearer scheme and non-empty token. |
| Oversize credential/JWT, huge JWKS, excessive key count, or expensive key types | Memory, CPU, or upstream egress exhaustion | Enforce transport token limit before decode; bounded discovery/JWKS bodies, JSON depth/key count, supported key families/sizes, and connection/request timeouts. |
| Random unknown-`kid` spray | One upstream request per attacker request; issuer or service DoS | Cache-first lookup, a single coalesced refresh, global cooldown/negative result, no attacker-derived metric labels; concurrent and sequential spray tests. |
| Missing `kid`, duplicate matching `kid`, incompatible key metadata, or a JWKS with zero usable keys | Ambiguous key selection, downgrade, or cache poisoning | Treat required non-empty `kid` as an explicit strict local profile; require exactly one compatible signing key; reject the complete candidate JWKS atomically and retain last-known-good trust rather than partially replacing it. Initial invalid JWKS fails startup. |
| Rotation reuses the same `kid` with new key material | Valid new token rejected until periodic refresh | On signature failure for an otherwise compatible cached `kid`, permit the same bounded coalesced refresh policy once, then retry once. |
| Issuer removes or revokes an old key while the service keeps a cache | Revoked credentials remain accepted indefinitely | Scheduled refresh plus a fixed last-known-good maximum age; after cutoff return unavailable/fail closed. Rotation-overlap and removed-key tests. |
| Discovery/JWKS outage before any successful bootstrap | Service starts with no trustworthy keys or silently bypasses | Startup fails before listeners serve; no readiness and no empty/fake trust. |
| Outage during a valid cache interval, then beyond maximum age | Needless outage or unbounded stale trust | Continue verifying known keys only inside the explicit cached-trust interval; fail unavailable after it. Deterministic clock/outage tests. |
| Redirect, DNS rebinding, private/loopback resolution, authority change, or slow upstream | SSRF, credential infrastructure pivot, hung startup/request | Reuse `ExternalHTTPS` authority pinning and DNS/address policy separately for issuer and discovered JWKS; bounded timeout; no redirects. |
| Bearer credential crosses a plaintext direct HTTP or gRPC hop | Passive observer obtains a replayable credential | Architecture must identify and enforce the trusted TLS-termination boundary for each enabled transport. Direct exposure without TLS is invalid; a proxy-terminated hop is acceptable only under an explicit trusted-proxy/deployment contract rather than attacker-controlled forwarded headers. Configuration/bootstrap and integration proof own this boundary. |
| Request cancellation starts abandoned duplicate refreshes or cancels a refresh shared by healthy waiters | Resource leak or avoidable authentication outage | Lifecycle owns shared refresh; waiter cancellation only detaches that waiter; shutdown cancels the component. Cancellation, timeout, race, and liveness proof. |
| Token/claims/parser errors escape through logs, traces, metrics, errors, or panic | Bearer credential disclosure and unbounded telemetry cardinality | Finite opaque error taxonomy and bounded attributes; no raw error formatting or attacker-controlled labels; capture/redaction tests. |
| HTTP and gRPC differ on credential parsing, validation, principal mapping, or errors | Weaker transport bypasses policy | One verifier and principal result, transport-local exact extraction, `401`/`UNAUTHENTICATED` for authn, `403`/`PERMISSION_DENIED` reserved for authz; parity tests. |
| JWKS/discovery is unavailable after cached trust expires but the adapter reports an invalid credential | Operator outage is hidden as caller fault and pollutes authentication telemetry | Preserve an unavailable class: normally HTTP `503` and gRPC `UNAVAILABLE`; malformed and cryptographically invalid credentials remain client authentication failures. |
| gRPC streaming authenticates after the first application message or exposes mutable authentication state | Unauthenticated work or mid-stream identity drift | Authenticate once before handler invocation; wrap stream context with immutable principal; no per-message token selection. |
| Broad bypass list accidentally includes application methods | Unauthenticated domain entry point | Explicit full-method allowlist for infrastructure health only; unknown/application RPCs require authentication. |
| OpenAPI operation omits security or includes anonymous `{}` as an OR alternative | HTTP request bypasses the authentication callback completely | In the OIDC profile set one top-level bearer requirement; only named health operations carry an explicit empty override; canonical and generated-contract tests prove the verifier gates every other operation. |
| Raw token remains in handler-visible HTTP headers or gRPC metadata | Downstream logging/forwarding leaks bearer capability | Scrub credential from the context/request/stream passed to application code after verification; deterministic visibility test. |
| Possession token is replayed intact inside its validity window | Replay succeeds because `jti` is descriptive, not a nonce store | Record as residual risk. DPoP, mTLS-bound tokens, revocation/introspection, and replay storage are separate accepted outcomes. |

**Security disposition.** The challenged material paths now have a named
pre-application enforcement point and deterministic proof owner. The verifier
cannot convert network unavailability into an invalid-credential fact, and
attacker input cannot produce `Internal`/raw errors or select an outbound
authority. Concrete TLS-boundary enforcement, limits, and status mapping remain
Specification/Design decisions rather than Research conclusions.

## Downstream dispositions

| Owner | Evidence handed forward | Still owned downstream |
| --- | --- | --- |
| Specification | RFC 9068 strict access-token profile plus explicit required-`kid` local restriction; exact single issuer and audience; explicit algorithm/type/time policy; one subject path; no authz. | Concrete clock skew, token-size limit, HTTP malformed/unavailable status, readiness and TLS-boundary promises. |
| System / Integration Design | Fixed discovery authority, repository egress client, one atomic cache owner, initial-trust requirement, HTTP/gRPC/OpenAPI seams, TLS boundary, profile composition. | Library/mechanism selection and concrete lifecycle/TLS enforcement values. |
| Go Ownership Design | Concrete verifier and adapters; consumer-local substitution only; no framework or raw-token propagation. | Exact package/file/API ownership. |
| Test Design | Required conformance, failure, outage, rotation, concurrency, redaction, transport, race, and profile cases. | Deterministic harness and oracle placement. |

## Research stop rationale

Standards, library, production, repository, and adversarial evidence have
reached source saturation: additional examples are unlikely to change the
recorded alternatives. The first independent challenge returned `CONCERNS` for
TLS-boundary ownership, OpenAPI default protection, JOSE/JWKS ambiguity,
normative `kid` wording, and unavailable-error classification; all five are
accepted and repaired above. The fresh re-review found only stale routing in
`workflow-plan.md`; that coordination concern was accepted and repaired without
changing this synthesis. No material research finding remains open. A new
standards revision, selected-library release/advisory, multi-issuer scope, or
transport-termination model reopens the corresponding smallest section.
