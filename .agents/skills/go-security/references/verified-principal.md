# Verified Principal

## When To Load

Load this when a change touches bearer-token verification, an identity header,
or what a handler is allowed to conclude about who is calling.

## Behavior Change Thesis

Without this file, an identity requirement is written as if this service had
none: a fresh JWT policy naming issuer, audience, and an algorithm allowlist,
and a `Principal.HasScope` check to authorize on. Both are defects here. The
first is a second credential path beside the one that already runs, and the
second denies every caller — `internal/infra/oidcjwt` fills only
`Principal.Subject` and leaves `Scopes` empty.

## Decision Rubric

- `internal/infra/oidcjwt` is the only credential verifier. It reaches handlers
  through `httpx.Authenticated`, which publishes a `reqctx.Principal` onto the
  request; a handler that re-reads `Authorization` is the second path, and the
  one nobody audits.
- What it already enforces, and what a change must not weaken: RS256 only,
  checked in the protected header and again as the sole algorithm passed to
  `jose.ParseSignedCompact`, which go-jose v4 requires the caller to name;
  `crit`, `b64`, `jku`, `x5u`, and `jwk` headers rejected outright; an RFC 9068
  `typ` of `at+jwt`; exact `iss`; `aud` containing the configured audience;
  `sub`, `client_id`, and `jti` present; `exp` and `iat` required within 30s
  skew; and keys selected by `kid` from the fetched JWKS alone.
- Trust is time-bounded. A key set older than `MaxKeySetAge` verifies nothing
  and answers `KindUnavailable`, and a key miss triggers at most one refresh per
  `RefreshCooldown` so an invalid-`kid` flood cannot drive JWKS traffic. Both
  fail closed; a fallback that accepts a token because refresh failed inverts
  them.
- Scope, role, and tenant are not carried by this credential. Authorizing on any
  of them needs its own source and its own decision, derived from the verified
  subject rather than from a request field.
- HTTP identity additionally requires the peer address inside
  `authn.trusted_proxy_cidrs` and exactly one `X-Forwarded-Proto: https`;
  anything else is `KindUntrustedTransport` before the token is parsed. A new
  ingress path inherits that requirement or states why it does not.
- A secured operation reaching a handler with no principal is a wiring defect,
  not an anonymous caller: `reqctx` documents 500 for it, never 401.

## Reject

- Reading `X-User-Id`, `X-Tenant-Id`, or an admin header as identity:
  `ResolveHTTP` deletes `Authorization` after reading it so that no later stage
  can re-derive a caller, and an inbound identity header is caller-controlled
  unless an edge both strips and sets it.
- Widening the allowlist to whatever the issuer signs with: the header `alg` is
  attacker-supplied input, and RFC 8725 puts algorithm selection on the
  verifier.

## Validation Shape

The deny-path matrix spans two files: `internal/infra/oidcjwt/token_test.go`
holds what one token can be wrong about — tampered header and payload,
duplicate members, wrong issuer and audience, expiry and skew — and
`verifier_test.go` holds what needs a live key set, unknown `kid` and a stale
set. Extend those tables rather than re-proving a verifier control at the
handler; add a `middleware_auth_test.go` case only for what the transport seam
itself decides.

Credential lifecycle, permission-model shape, tenant partitioning, and the
revocation window belong to
[`auth-access-control`](../../../../docs/universal-disciplines/auth-access-control/SKILL.md).
