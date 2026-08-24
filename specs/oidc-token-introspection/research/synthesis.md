# OAuth 2.0 token introspection research synthesis

Status: supporting Research complete; ready for Specification

Evidence snapshot: 2026-08-22

Repository baseline: `8967a4ac06d4fce0515703b15ffa5db35e5378ae`
plus the current dirty worktree. Existing unrelated changes were inspected only
where they affected the authentication baseline and were not modified.

## Research boundary

This note closes four decisions for an `AUTHN=oidc-introspection` profile:
what an RFC 7662 response proves, how the endpoint is located and authenticated,
which returned fields can form the existing `reqctx.Principal`, and whether a
positive result may be cached. It does not choose Go packages, file placement,
wiring, test fixtures, rollout, provider registration, credentials, or a live
deployment.

Although the requested profile is named OIDC for symmetry with
`AUTHN=oidc-jwt`, token introspection is the OAuth 2.0 protocol defined by RFC
7662. The profile therefore treats every presented bearer access token as
opaque and never infers trust from its local shape.

## Claim synthesis

### `active` is authoritative but not a complete local principal contract

**Fact.** RFC 7662 requires the boolean `active` member. A properly authorized
query for an inactive, unknown, or undisclosable token returns HTTP 200 with
`active=false`; it is not a provider error. The definition of active is owned by
the authorization server and is expected to include applicable expiry,
not-before, revocation, signature, and resource-server checks. `iss`, `aud`,
`exp`, `sub`, and `client_id` are registered but optional response members.
([RFC 7662 sections 2.2 and 4](https://www.rfc-editor.org/rfc/rfc7662.html#section-2.2))

**Inference and decision effect.** `active=false` is an invalid caller
credential. `active=true` alone is insufficient for this repository: the
profile must additionally receive exact `iss`, matching `aud`, required `exp`,
and at least one stable `sub` or `client_id` before publishing the existing
`reqctx.Principal`. This is intentionally stricter than bare RFC 7662 and makes
the issuer, resource, identity, and stream-expiry boundaries equivalent to the
current JWT profile.

**Counter-evidence disposition.** A compliant provider may omit those optional
members. Such a provider does not satisfy this profile; silently substituting
registration-time assumptions would let two providers publish observably
different principals. Reopen Specification for one provider-specific mapping
instead of weakening the shared profile.

### Endpoint discovery and authentication are separate contracts

**Fact.** RFC 7662 requires TLS, POST form encoding, and authorization of the
resource server, but leaves endpoint discovery and the authentication method
out of scope. RFC 8414 defines optional `introspection_endpoint` and
`introspection_endpoint_auth_methods_supported` metadata and explicitly gives
the latter no default when omitted.
([RFC 7662 section 2.1](https://www.rfc-editor.org/rfc/rfc7662.html#section-2.1),
[RFC 8414 section 2](https://www.rfc-editor.org/rfc/rfc8414.html#section-2))

**Inference and decision effect.** The minimum deterministic v1 contract is one
explicit immutable HTTPS endpoint plus one authentication method:
`client_secret_basic`. It does not add metadata discovery, a credential-mode
registry, bearer acquisition, mTLS, or JWT client assertions. The client ID and
secret are dedicated resource-server credentials supplied through the
repository's secret-input boundary.

**Counter-evidence disposition.** RFC 8414 can advertise other methods and some
providers require them. Supporting them without a named provider would add key,
certificate, token-acquisition, rotation, and proof contracts that v1 does not
need. A provider that cannot issue `client_secret_basic` credentials reopens
Specification with its exact method and lifecycle.

### Introspection is a bearer-token exfiltration and work-amplification boundary

**Fact.** RFC 7662 requires protected and authorized endpoint access to prevent
token scanning, requires TLS certificate validation, recommends POST to keep
tokens out of query logs, and treats response identity as privacy-sensitive.
([RFC 7662 sections 2.1, 4, and 5](https://www.rfc-editor.org/rfc/rfc7662.html#section-4))

**Inference and decision effect.** The existing bearer grammar, size bound,
credential removal, fixed-authority HTTP client, sanitized error taxonomy, and
HTTP/gRPC principal seam remain authoritative. Existing queue-free transport
admission already runs before authentication: HTTP uses the positive
`http.max_in_flight` value (default 256), and gRPC uses its fixed 256-business-
RPC budget. The profile adds no second limiter or queue; it makes at most one
provider attempt inside the slot already held. With both transports enabled,
the per-process provider-call ceiling is the sum of their business ceilings.
Tokens, endpoint credentials, provider bodies, and returned identity never
become logs, traces, metric attributes, or caller-visible errors.

### No positive cache is currently earned

**Fact.** RFC 7662 permits response caching but makes the stale-revocation
window an explicit security tradeoff and forbids caching beyond `exp` when it
is returned.
([RFC 7662 section 4](https://www.rfc-editor.org/rfc/rfc7662.html#section-4))

**Inference and decision effect.** v1 caches neither positive nor negative
responses and never serves stale identity after an introspection failure. This
is the only policy consistent with an unspecified provider propagation SLA,
end-to-end revocation objective, and request distribution.

**Reopen condition.** Positive caching may be reconsidered only when a named
adopter supplies all three: an accepted end-to-end revocation bound, the
provider's measured or contractual revocation propagation bound, and
representative load showing that uncached introspection misses an accepted
latency or capacity target. Any future cache TTL must be no greater than the
remaining revocation budget, token lifetime, and configured cap; a non-positive
remainder means no cache. Negative caching and stale-on-error remain separate
Specification decisions.

## Repository baseline and downstream effect

The current `AUTHN=oidc-jwt` profile already owns the required outer contract:
OpenAPI bearer security, HTTP and gRPC credential removal, one
`reqctx.Principal`, sanitized `missing|malformed|oversize|invalid|unavailable`
outcomes, public HTTP probes and only public gRPC `Health/Check`, protected
stream expiry, optional-profile removal, and bootstrap-owned shutdown. The new
profile should substitute behind that behavior rather than create a second
handler-facing credential path.

The current checkout has no introspection selector, package, configuration,
profile markers, or initializer proof. Technical Design must therefore choose
a runtime boundary and generated-profile ownership, but it need not invent the
behavior above.

## Unknowns and refresh conditions

- No provider, registration, endpoint, target class, credential, revocation
  SLA, request distribution, or capacity entitlement was supplied. Those are
  adopter and deployment inputs, not facts inferred from the RFCs. A provider
  whose concurrency entitlement is below the sum of enabled transport ceilings
  requires a reopened provider-budget decision before this profile can be
  enabled.
- Live provider compatibility remains unproved until an adopter supplies a
  real active token, inactive/revoked token, and authorized endpoint client in
  its actual TLS and egress environment.
- Refresh the repository baseline before Technical Design if authentication,
  fixed-authority HTTP, configuration, bootstrap, or initializer surfaces
  change. Refresh the external contract if RFC 7662 or RFC 8414 is updated or a
  provider-specific profile is proposed.
