# Authentication, authorization, and service identity instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing `AuthN`/`AuthZ` behavior in HTTP/gRPC handlers, middleware, interceptors, or service-layer policies
  - Implementing OAuth2/OIDC resource-server behavior, JWT validation, token exchange, or service-to-service identity
  - Designing tenant isolation, object-level authorization, or permission models (`RBAC`, `ABAC`)
  - Designing identity propagation across synchronous and asynchronous interactions
  - Reviewing security-sensitive code paths for broken access control, BOLA/IDOR, or privilege escalation risks
- Do not load when: The task is documentation-only or pure internal refactor with no auth/identity/runtime behavior impact

## Purpose
- This document defines repository defaults for authentication, authorization, service identity, and tenant isolation.
- Goal: make auth behavior predictable, fail-closed, and reviewable across services and transports.
- Defaults here are mandatory unless an ADR explicitly approves an exception.

## Baseline assumptions
- Zero-trust baseline: internal traffic is untrusted by default.
- The service can receive two identity types in one request:
  - `workload/service identity` (who is calling this service)
  - `end-user identity` (on whose behalf the action is requested)
- Authentication and authorization are separate concerns and must stay separate in code.
- Object-level authorization is mandatory for resource-by-ID operations.

## Required inputs before changing auth behavior
Resolve first. If unknown, apply defaults and document assumptions.

- Entry boundary: external/public API, partner API, internal service API, async consumer.
- Identity mix: end-user only, service only, or both.
- Tenant model: single-tenant or multi-tenant; source of `tenant_id`.
- Token model: JWT offline validation, opaque token introspection, or mixed.
- Propagation model: forward token, token exchange, or internal token mint.
- Authorization model: RBAC baseline or ABAC/policy engine requirement.
- Revocation requirement: eventual revocation acceptable vs near-immediate revocation required.

## Identity model and trust boundaries

### Principal model (mandatory)
- Every authenticated request MUST build one `AuthContext` with at least:
  - `subject_id` (end-user or client identity)
  - `subject_type` (`end_user` or `service`)
  - `caller_service` (workload identity of direct caller when available)
  - `tenant_id` (for multi-tenant systems)
  - `roles` and/or `scopes`
  - `auth_method` (`bearer`, `mtls`, `token_exchange`, `internal_token`)
  - `request_id`, trace identifiers
- Keep caller and subject identities separate; never collapse them into one field.
- Never trust identity from arbitrary headers (`X-User-Id`, `X-Tenant-Id`, `X-Roles`) unless issued by a trusted boundary and cryptographically protected.

### End-user AuthN defaults (OIDC/OAuth2)
- Default: OAuth2/OIDC bearer access token, service acts as resource server.
- Bearer tokens MUST be accepted only over TLS.
- For browser/login flows, default is Authorization Code + PKCE (client-side concern).
- Never recommend implicit grant or ROPC (`password`) grant.

### JWT validation defaults and pitfalls
- JWT validation MUST be complete, fail-closed, and done before business logic:
  - verify signature
  - verify `iss`
  - verify `aud` (service-specific audience)
  - verify lifetime claims (`exp`, `nbf`, `iat`) with bounded skew (default: `30s`)
  - allowlist algorithms; reject unknown algorithms and `alg=none`
  - if using RFC 9068 profile, verify access-token type (`typ: at+jwt` or `application/at+jwt`)
- Key management rules:
  - trust keys only from issuer-bound metadata (`iss -> jwks_uri`)
  - do not trust token-supplied key locations (`jku`/`x5u`) by default
  - cache verifier/JWKS clients as long-lived components; do not recreate per request
- Revocation decision:
  - default: JWT offline validation
  - if near-immediate revocation is mandatory, require introspection/online checks and document latency tradeoff

## Service-to-service authentication defaults
- Internal service calls MUST use mTLS by default.
- Authorization decisions for service-to-service calls MUST be based on workload identity, not IP/hostname.
- Prefer SPIFFE/SPIRE-compatible workload identity when available.
- Identity certificate policy defaults:
  - short-lived certs (target lifetime in hours, not days)
  - automated rotation
  - connection refresh strategy during rotation
- `InsecureSkipVerify` or equivalent trust-chain bypass is forbidden outside isolated tests.

## Authorization model and boundaries

### Default model: deny by default + least privilege
- Missing policy/role/scope MUST result in deny.
- Default permissions for new roles/principals are empty.
- Privilege changes must be explicit, reviewed, and auditable.

### RBAC vs ABAC decision rules
- Start with RBAC when:
  - role set is stable and coarse-grained
  - permissions can be reviewed via explicit role matrix
- Move to ABAC when one or more is true:
  - access depends on resource attributes, ownership, geography, data class, or runtime context
  - RBAC role explosion appears
  - multi-tenant entitlements require per-resource conditions
- Hybrid is allowed: RBAC coarse gate + ABAC fine-grained object decision.

### Authorization boundaries (mandatory)
- Middleware/interceptor:
  - authenticate credential and build `AuthContext`
  - do not perform domain-specific authorization decisions
- Handler:
  - validate request shape
  - bind route/body identifiers + `AuthContext`
  - call service with explicit context
- Service/use-case layer:
  - enforce authorization rules before side effects
  - enforce object-level checks (`owner_id`, ACL, tenant boundary)
- Repository/data access:
  - enforce tenant scoping in queries
  - never broaden scope because caller has a role string alone

### Tenant isolation defaults
- `tenant_id` MUST come from verified identity or trusted signed internal credential.
- If request tenant and identity tenant mismatch, deny (`403` / `PERMISSION_DENIED`).
- Tenant scope must be applied consistently:
  - service logic
  - repository filters
  - cache keys
  - async messages/jobs
  - audit logs
- Default deny for cross-tenant reads/writes unless endpoint contract explicitly allows and review approved.

## Identity propagation rules

### Synchronous interactions (HTTP/gRPC)
- Propagate request context end-to-end (deadline/cancel + identity metadata).
- Required observability propagation:
  - HTTP: `traceparent`, optional `baggage`, `X-Request-ID`
  - gRPC: metadata equivalents for trace/request ID
- Choose one identity propagation mode per call path and document it:
  - `forward_token`
  - `token_exchange` (recommended for per-service audience/downscoping)
  - `internal_token` (signed internal credential with minimal claims)
- Decision defaults:
  - same trust zone + correct audience: forwarding may be allowed
  - cross-service with different audiences or high lateral-movement risk: use token exchange/downscoped token
  - internal-only machine calls: workload identity + service principal scopes
- Never propagate raw end-user token to every downstream by default.

### Asynchronous interactions (events/queues/workflows/webhooks)
- Do not put raw bearer tokens into messages.
- Use signed event/task identity envelope with minimal required claims:
  - `subject_id`
  - `subject_type`
  - `tenant_id`
  - `initiator_service`
  - `scopes_or_entitlements_snapshot` (minimal)
  - `issued_at` / `expires_at`
  - `request_id` / `trace_id` / `event_id`
- Enforce consumer-side checks:
  - verify message authenticity/integrity
  - verify tenant scope before data access
  - deduplicate by stable event/request ID for at-least-once delivery
- For webhooks/callbacks:
  - verify signature and replay window
  - do not treat source IP or unsigned headers as identity proof

## Decision rules (apply in order)
1. Determine identity types involved (`end_user`, `service`, or both).
2. Authenticate first (token/cert validation), fail-closed on any auth error.
3. Build explicit `AuthContext` with tenant and caller/subject separation.
4. Choose propagation mode (`forward`, `exchange`, `internal`) for each hop explicitly.
5. Enforce authorization at service boundary and object level before side effects.
6. Enforce tenant scope in every data path (DB/cache/async).
7. Emit auditable decision metadata (without leaking tokens/PII).

## Anti-patterns to reject
- Trusting unverified claims or unsigned headers as identity.
- Parsing JWT payload without signature/issuer/audience verification.
- Accepting generic audience tokens for all services.
- Mixing authentication and authorization logic in one implicit block.
- Handler-only authorization without service-layer object checks.
- Implicit superuser paths:
  - hidden `is_admin` query flags
  - environment backdoors
  - default allow rules
- mTLS enabled but with trust checks disabled (`InsecureSkipVerify` or permissive peer match).
- Using service identity as a substitute for end-user authorization.
- Propagating raw user token through async pipelines.
- Tenant-agnostic cache keys or repository queries in multi-tenant flows.

## Review checklist (merge gate)
- Identity model is explicit: caller vs subject identities are separate in context.
- AuthN checks are complete:
  - JWT signature/iss/aud/alg/lifetime (+ `typ` when required) validated
  - key trust chain bound to issuer metadata
- Service-to-service auth is explicit:
  - mTLS/workload identity enforced for internal calls
  - no TLS verification bypass in production paths
- Authorization is fail-closed and layered:
  - middleware authenticates
  - service layer enforces object-level authorization
  - repository enforces tenant scoping
- RBAC/ABAC choice is documented with rationale and least-privilege posture.
- Propagation policy is documented and consistent across sync and async paths.
- Async consumers verify identity envelope integrity and enforce dedup + tenant checks.
- No anti-patterns from this document are present.
- Tests include negative access cases:
  - wrong tenant
  - insufficient scope/role
  - forged/invalid token
  - confused deputy / cross-service audience misuse

