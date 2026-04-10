---
name: go-security-spec
description: "Design security requirements for Go services: trust boundaries, identity and access rules, tenant isolation, threat-class controls, abuse resistance, secure defaults, and testable security behavior."
---

# Go Security Spec

## Purpose
Define or review security requirements so trust boundaries, identity rules, authorization boundaries, data protection, and failure behavior are explicit, enforceable, and testable.

## Specialist Stance
- Treat security as explicit trust boundaries, identity rules, denial behavior, and abuse controls.
- Separate authentication, authorization, tenant isolation, sensitive-data handling, and resource-exhaustion defenses.
- Prefer fail-closed, least-privilege, standard-library-friendly controls with clear negative-path proof.
- Hand off architecture, API, physical schema, reliability, observability, or delivery policy when they stop being security-owned decisions.

## Scope
- define trust boundaries, security assumptions, and threat exposure for affected flows
- define identity and access rules, including caller/subject separation, tenant isolation, and object-level authorization
- define secure-by-default controls for untrusted input, outbound access, and sensitive data handling
- define abuse-resistance behavior: limits, bounded concurrency, timeout policy, and safe degradation
- define fail-closed behavior for critical security paths
- define verification obligations for negative and abuse-path security behavior
- surface hidden security decisions instead of leaving them to implementation guesses

## Boundaries
Do not:
- redesign general service architecture, ownership topology, or distributed coordination model as the primary output
- take ownership of full API resource modeling or physical schema design outside their security impact
- prescribe low-level middleware, handler, repository, or CI wiring as the main result
- treat observability, reliability, or delivery policy as the primary domain unless they materially affect security behavior

## Core Defaults
- Use a zero-trust baseline: external and internal traffic are untrusted unless explicitly justified otherwise.
- Keep authentication, authorization, tenant isolation, and data protection as separate decision blocks.
- Prefer deny-by-default and least privilege. Missing policy means deny.
- Prefer standard library and minimal dependencies; security libraries require explicit justification.
- Missing trust-boundary facts, identity-model facts, or enforcement ownership are blockers, not details to improvise later.

## Expertise

### Trust Boundaries And Threat Modeling
- Require an explicit boundary map for affected flows: `external`, `partner`, `internal service`, and `async worker/consumer`.
- Make data classification and leakage exposure assumptions explicit for each boundary.
- Classify side effects and retry behavior before choosing controls.
- Define outbound access policy explicitly: allowed schemes, hosts, ports, redirect behavior, and egress assumptions.
- Reject generic threat statements that do not identify an attacker path and concrete impact.

### Identity, Authorization, And Tenant Isolation
- Require one explicit `AuthContext` model with caller/subject separation and tenant binding.
- Define authentication requirements per boundary: token or certificate verification, audience/issuer/algorithm checks, lifetime checks, and trusted key-source policy.
- Keep enforcement layered:
  - boundary/auth middleware authenticates and builds context
  - service layer enforces object-level authorization before side effects
  - repository/data access enforces tenant scoping
- Define propagation policy per hop (`forward_token`, `token_exchange`, `internal_token`), including async envelopes.
- Never treat unsigned headers such as `X-User-Id`, `X-Tenant-Id`, or role headers as a trusted identity source.

### Threat-Class Control Design
- Cover the core threat classes for every affected boundary:
  - input validation and strict decoding
  - output encoding and error sanitization
  - injection controls
  - SSRF policy
  - path traversal controls
  - deserialization safety
  - resource-exhaustion controls
- Use explicit request-boundary defaults:
  - bounded header, URI, body, and multipart limits
  - unknown-field rejection and trailing-token rejection for mutable JSON writes
- Treat command execution and `unsafe` usage as forbidden by default and require explicit approval.
- Reject blacklist-only validation and “we validate later in business logic” as primary controls.

### API-Facing Security Semantics
- Make auth failure behavior explicit (`401` vs `403`).
- Make size, media, and validation security semantics explicit (`413`, `415`, `422`, and related cases).
- Make rate-limit behavior explicit, including `429` and `Retry-After`.
- Retry-unsafe operations must declare idempotency-key behavior: scope, TTL, and conflict semantics.
- Long-running side effects should use explicit async semantics rather than fake synchronous success.
- Error responses must be stable, sanitized, and free of sensitive implementation detail.

### Async And Distributed Security
- Define message authenticity for async paths: signed envelope, integrity checks, replay window, and dedup boundary.
- Prohibit raw bearer-token propagation through async payloads.
- Align async side effects with outbox/inbox or equivalent durability and dedup guarantees.
- Make step-level authorization explicit:
  - who may trigger each step
  - who may compensate or retry
  - how stuck/timeout paths are escalated
- Never rely on eventual consistency to “fix access control later.”

### Data, Storage, And Cache Security
- Keep service-owned data boundaries explicit; no implicit cross-service DB trust.
- Require parameterized SQL and allowlisted dynamic identifiers.
- Split DB privileges between runtime and migration responsibilities.
- Do not leak sensitive query text or credentials into logs or traces.
- Make migration security implications explicit: mixed-version compatibility window, rollback limits, and recovery assumptions.
- If cache is involved, require tenant-safe and scope-safe keys, explicit fail-open vs fail-closed policy, and a clear rule for what data may never be stored in shared cache.

### Abuse Resistance And Fail Behavior
- Bound timeouts, retries, queues, concurrency, and request size on security-sensitive paths.
- Keep retry policy idempotency-aware and bounded with jitter.
- Preserve security invariants under overload: critical dependencies fail closed unless a safer contract is explicitly proven.
- Fallback and degradation modes must never bypass authorization or tenant isolation.
- Reject infinite timeout, infinite retry, and unbounded buffering patterns.

### Security Observability And Privacy
- Make audit-relevant events observable with a bounded taxonomy:
  - authentication failures
  - authorization denials
  - tenant-scope violations
  - idempotency conflicts
  - abuse-control triggers
- Require structured logs and traces with correlation IDs while prohibiting leakage of secrets, tokens, and sensitive personal data.
- Keep telemetry labels and dimensions low-cardinality and operationally useful.
- Isolate debug, admin, and profiling surfaces; any incident-mode visibility should be time-bounded and access-controlled.

### Runtime Hardening And Verification
- Translate security decisions into concrete verification:
  - negative-path tests
  - abuse-path tests
  - security scanning where relevant
  - runtime hardening expectations
- Container and runtime expectations for security-sensitive services should include non-root execution, minimal runtime image, no embedded secrets, and correct trust-store behavior.
- Required negative classes typically include forged token, wrong tenant, insufficient scope, object-level deny, injection/SSRF/path traversal attempts, and payload abuse.
- For async paths, test signature, replay, dedup, and tenant checks when those paths are in scope.

## Decision Quality Bar
Major security recommendations should make the following explicit:
- the trust boundary and threat scenario
- at least two viable options when the decision is nontrivial
- the selected control and at least one rejected alternative
- enforcement points
- fail behavior
- verification obligations
- residual risk and reopen conditions

Security claims without enforcement and verification are incomplete.

## Deliverable Shape
Return security work in a compact, reviewable form:
- `Security Decisions`
- `Threat-Control Matrix`
- `Sensitive Data And Redaction Rules`
- `Abuse Resistance And Fail Behavior`
- `Verification Obligations`
- `Assumptions And Residual Risks`

## Escalate When
Escalate if:
- trust boundaries or identity model are ambiguous
- object-level authorization or tenant isolation lacks an explicit enforcement point
- untrusted input lacks threat-class control coverage
- retry-unsafe behavior has no idempotency contract
- async paths lack authenticity, replay, or dedup rules
- sensitive-data handling lacks sanitization or redaction rules
- abuse-prone paths have no bounded timeout, limit, or concurrency strategy
- runtime hardening assumptions materially affect safety but remain undefined
