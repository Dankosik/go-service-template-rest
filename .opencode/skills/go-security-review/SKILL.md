---
name: go-security-review
description: "Review Go code changes for trust-boundary enforcement, authorization and tenant isolation, injection and SSRF risk, secret handling, and abuse resistance."
---

# Go Security Review

## Purpose
Protect changed code from exploitable vulnerabilities and security-contract drift at trust boundaries, data boundaries, and side-effecting operations.

## Scope
- review untrusted-input handling and strict boundary validation
- review authentication, authorization, tenant isolation, and object-level access checks
- review injection, query safety, SSRF, path traversal, and unsafe file or upload handling
- review secret, token, PII, and sensitive-data handling in code, errors, logs, traces, and metrics
- review abuse resistance: time budgets, concurrency, queue bounds, and expensive-path controls
- review async identity and replay-sensitive flows when touched
- review runtime and verification signals relevant to the changed security surface
- review whether negative-path behavior remains testable and explicit

## Boundaries
Do not:
- redesign the entire security model during code review unless local correction is impossible
- absorb primary ownership of performance, QA, or reliability depth when they are only supporting signals
- accept internal traffic or internal callers as trusted by default
- treat generic hardening advice as a finding without a concrete exploit or contract risk

## Core Defaults
- Inputs are untrusted unless an explicit trust contract says otherwise.
- Fail closed on identity, authorization, and hard validation boundaries.
- Prefer boundary-first controls before business logic and before side effects.
- Missing limits, missing timeout budgets, and missing tenant checks are defects until proven safe.
- Prefer the smallest safe correction that closes the exploit or removes the unsafe assumption.

## Expertise

### Trust Boundary And Input Validation
- Require strict parsing, size limits, and allowlist validation before business logic.
- Reject blacklist-only validation and validation that begins after side effects.
- Require explicit bounds on body, multipart, headers, query parameters, and filter complexity when relevant.
- Treat unknown-field tolerance on mutable inputs as risky unless deliberate.

### Authentication, Authorization, And Tenant Isolation
- Keep authentication defects separate from authorization defects.
- Require complete identity validation before side effects.
- Require object-level checks and tenant binding on resource-by-ID flows.
- Flag default-allow behavior, implicit superuser paths, and missing tenant propagation as high-risk.
- Require service and user identity to stay distinct in mixed flows.

### Injection And Query Safety
- Require parameterization for values and allowlisting for dynamic identifiers.
- Reject raw user-controlled filters mapped straight into datastore or query operators.
- Reject command execution with shell expansion or user-influenced command strings.
- Require safe template use and no unchecked escape bypass.
- Treat user-influenced query, command, or path construction without guardrails as exploitable until disproven.

### Outbound Security And SSRF
- Require explicit outbound timeout budgets and context propagation.
- When targets are user-influenced, require scheme, host, port, and post-resolution network controls that make SSRF meaningfully bounded.
- Reject `http.Get` or default clients on security-sensitive outbound flows.
- Treat redirect handling and DNS-resolution behavior as part of the security boundary.

### Filesystem, Path, And Upload Safety
- Require root-constrained or otherwise safe file access for user-influenced paths.
- Reject trusting user-provided filenames as durable storage keys.
- Require upload size limits, streaming where possible, extension plus content validation, and storage isolation from direct public serving when relevant.
- Require explicit scan or publish gating if malware or unsafe content checks are part of the contract.

### Secrets, Error Disclosure, And Telemetry
- Reject secrets, tokens, raw auth headers, DSNs, stack traces, SQL text, or sensitive payloads in responses and diagnostics.
- Require redaction discipline in logs, traces, and metrics.
- Require correlation identifiers for triage, but never as auth input.
- Treat public debug or admin exposure as a security finding, not just operability drift.

### Abuse Resistance And Resource Control
- Require explicit timeouts, bounded concurrency, bounded queues, and size limits on expensive or security-sensitive paths.
- Reject unbounded `io.ReadAll`, unbounded retries, and unbounded fan-out on untrusted input paths.
- Keep overload behavior explicit enough that it does not silently weaken security posture.
- Treat fail-open fallback on critical security dependencies as unsafe unless explicitly justified.

### Async Identity And Distributed Security
- Reject raw bearer token propagation in async messages.
- Require authenticity checks or signed envelopes where async identity matters.
- Require replay, dedup, and ack-order safety where messages trigger side effects.
- Keep forensic traceability explicit across retries and dead-letter paths.

### Data, Cache, And Runtime Hardening
- Require least-privilege data access and tenant-safe cache keys when auth context affects results.
- Reject caching secrets or private per-user results under shared keys.
- Review migration or backfill behavior when it can break retention, deletion, or audit guarantees.
- Expect security-sensitive runtime surfaces to keep non-root, minimal-privilege, and no-trust-bypass defaults when those surfaces are touched.

### Cross-Domain Handoffs
- Hand off timeout, retry, overload, and degraded-mode policy depth to `go-reliability-review`.
- Hand off DB/query/cache correctness depth to `go-db-cache-review`.
- Hand off race or shared-state control defects to `go-concurrency-review`.
- Hand off coverage completeness to `go-qa-review`.
- Hand off broader structure and ownership drift to `go-design-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the security axis
- the violated control or unsafe assumption
- realistic attacker preconditions
- affected trust boundary or data asset
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: confirmed exploitable high-impact vulnerability
- `high`: strong evidence of significant security-contract breach
- `medium`: bounded but meaningful security weakness
- `low`: local hardening improvement

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-security-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

In `Issue`, start with the axis context, for example `Axis: AuthZ And Tenant Isolation; ...`.

## Escalate When
Escalate when:
- safe correction changes trust-boundary policy, auth model, or tenant-isolation contract (`go-security-spec`)
- API-visible auth, error, limit, or idempotency semantics must change (`api-contract-designer-spec`)
- the fix needs new timeout, retry, or overload policy to stay safe under abuse (`go-reliability-spec`)
- data ownership, cache key contract, or deletion and audit behavior must change (`go-db-cache-spec`)
- local repair exposes broader architecture or platform-hardening drift (`go-design-spec` or `go-devops-spec`)
