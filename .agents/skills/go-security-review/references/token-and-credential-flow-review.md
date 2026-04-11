# Token And Credential Flow Review

## Behavior Change Thesis
When loaded for symptom "identity, token, reset, or password material changes," this file makes the model inspect verification, entropy, storage, replay, enumeration, and password hashing instead of likely mistake only asking for throttling or redaction.

## When To Load
Load this when changed Go code touches JWT or bearer token parsing, header-derived identity, session or API token creation, password reset, invitation links, one-time codes, token persistence, password hashing, credential comparison, or account-recovery responses.

If the main issue is object authorization after identity is verified, load the authz reference. If the main issue is provider cost or repeated reset attempts, load the abuse reference. If the main issue is token value disclosure in logs or errors, load the secrets reference.

## Decision Rubric
- For JWT or bearer tokens, verify signature, algorithm allowlist, issuer, audience, expiry, not-before, key source, token type, and parse-error handling when the change owns those checks.
- Do not trust unverified token claims or client-controlled identity headers such as `X-User-ID`, `X-Tenant-ID`, or `X-Admin` unless an authenticated gateway contract strips inbound copies and sets them.
- Generate reset, invite, API, and session tokens with `crypto/rand` or a vetted token library; reject `math/rand`, timestamps, counters, deterministic user-derived values, or UUID-backed secrets without CSPRNG and entropy evidence.
- Persist reset and API tokens hashed or otherwise non-recoverable when they grant account access, with expiry and single-use or replay controls.
- Treat JWT `kid`, `jku`, and `x5u` as untrusted key-selection inputs; use an allowlisted key source instead of fetching attacker-selected keys.
- Keep account-recovery responses generic enough to avoid account enumeration while still giving safe user guidance.
- Store passwords with an approved adaptive password-hashing scheme; reject plaintext, reversible encryption, fast hashes such as SHA/MD5, and custom password hashing.
- Compare token digests using constant-time comparison when the code does a manual equality check on secret-derived material.

## Imitate
```text
[critical] [go-security-review] internal/infra/http/auth_middleware.go:55
Issue: Axis: Token And Credential Flow; the middleware accepts `X-User-ID` and `X-Tenant-ID` from the request headers before JWT verification succeeds.
Impact: A caller can forge identity and tenant context for downstream authorization checks.
Suggested fix: Ignore client-supplied identity headers at the service boundary and derive subject and tenant only from a fully verified token or authenticated gateway context.
Reference: identity source boundary.
```

Copy this shape when caller-controlled headers become identity.

```text
[high] [go-security-review] internal/app/reset.go:69
Issue: Axis: Token And Credential Flow; password-reset tokens are generated with `math/rand` seeded from time and stored in plaintext.
Impact: An attacker can predict or recover reset tokens and take over accounts before expiry.
Suggested fix: Generate reset tokens with `crypto/rand`, store only a token digest with expiry and single-use invalidation, and compare digests safely.
Reference: account recovery token lifecycle.
```

Copy this shape when the risk is token predictability or recoverability, not just reset abuse volume.

```text
[high] [go-security-review] internal/app/passwords.go:38
Issue: Axis: Token And Credential Flow; new passwords are stored as raw SHA-256 hashes without a password-hashing work factor or per-password salt.
Impact: A database leak enables fast offline cracking of user passwords.
Suggested fix: Use the repo-approved adaptive password hashing library and add a regression test that plaintext or fast hashes are not accepted as stored passwords.
Reference: password storage contract.
```

Copy this shape when password storage is weakened.

## Reject
```text
Issue: Use stronger crypto.
```

Reject because it does not identify the token, secret material, attacker path, or required lifecycle control.

```text
Suggested fix: Rate-limit password reset.
```

Reject when the actual merge blocker is predictable token generation, plaintext token storage, missing expiry, or reusable tokens.

```text
Issue: JWT auth looks okay because the token parsed successfully.
```

Reject because parse success is not signature, algorithm, issuer, audience, expiry, or key validation.

## Agent Traps
- Do not let a successful parser call stand in for full token verification.
- Do not trust JWT header `alg`, `kid`, `jku`, or `x5u` as policy without an allowlisted verifier and key source.
- Do not treat UUIDs, timestamps, or random-looking strings as account-recovery secrets without CSPRNG, entropy, and lifecycle evidence.
- Do not combine password hashing and reset-token findings when they need different fixes and tests.
- Do not require a new auth architecture when the local fix is deriving identity from verified claims instead of request headers.
- Do not log or return raw reset tokens while writing a token lifecycle finding; add a secrets handoff if disclosure is a separate issue.

## Validation Shape
- Add JWT negative tests for altered claims, wrong issuer, wrong audience, expired `exp`, future `nbf`, unknown key ID, unexpected algorithm, missing signature, and parse errors when those controls are local.
- Add reset-token tests for randomness source shape when feasible, digest-only persistence, expiry, single-use invalidation, generic responses, and replay rejection.
- Add password tests that prove plaintext, reversible, or fast-hash stored values are rejected or migrated according to the approved contract.
- Add tests that client-supplied identity headers are ignored unless an authenticated gateway fixture explicitly owns them.

## Repo-Local Anchors
- `api/openapi/service.yaml` defines bearer-auth contract surfaces when protected operations are introduced.
- `make go-security` runs the repo's security tooling, but token lifecycle defects usually need targeted negative tests too.
