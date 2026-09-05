# Verified Principal

## Load When

Load this when a change touches bearer-token verification, an identity header,
or what a handler is allowed to conclude about who is calling.

## Decide

- `internal/infra/oidcjwt` is the only credential verifier. It reaches handlers
  through `httpx.Authenticated`, which publishes a `reqctx.Principal` onto the
  request; a handler that re-reads `Authorization` is the second path, and the
  one nobody audits.
- The default resource-server profile enforces RS256, exact `iss`, `aud`
  containing the configured audience, required `exp`, optional `nbf`, 30s skew,
  and at least one stable subject or client identity. `golang-jwt` owns parsing
  and signature/claim validation; `keyfunc` owns cached JWKS key selection,
  periodic refresh, and cooldown-limited unknown-`kid` recovery.
- Mainstream client identifiers use `client_id`, `azp`, `appid`, or `cid`. The
  verifier accepts one value or several identical values and rejects a conflict.
  The explicit `rfc9068` token profile additionally requires `typ=at+jwt`,
  `sub`, `client_id`, `iat`, and `jti`; those are not ordinary-profile defaults.
- Initial Discovery and JWKS admission fail startup closed. Later refresh
  failures never replace the last valid set; a request that needs the failed
  refresh answers `KindUnavailable`, while dynamic provider health does not
  evict an otherwise healthy instance from readiness.
- Scope, role, and tenant are not carried by this credential. Authorizing on any
  of them needs its own source and its own decision, derived from the verified
  subject rather than from a request field.
- HTTP and gRPC remove the credential before the handler runs. TLS termination,
  CA roots, DNS/egress, and public-edge rate limiting are deployment controls;
  the verifier does not infer them from forwarding headers.
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
- Accepting an audience-less token because one issuer omits `aud`: configure a
  resource binding at the issuer instead of removing the API boundary.

## Prove

`internal/infra/oidcjwt/verifier_test.go` owns the default/strict claim matrix,
identity aliases, issuer/audience/time failures, and unavailable refresh;
`http_test.go` and `grpc_tls_contract_test.go` prove carrier removal and the
principal at the real transport seams. Extend the earliest owner rather than
re-proving verifier behavior in a handler.

Credential lifecycle, permission-model shape, tenant partitioning, and the
revocation window belong to
[`auth-access-control`](../../../../docs/universal-disciplines/auth-access-control/SKILL.md).
