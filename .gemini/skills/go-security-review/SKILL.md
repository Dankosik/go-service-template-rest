---
name: go-security-review
description: "Review Go code changes for trust-boundary enforcement, authn/authz and tenant isolation, browser session/CORS/CSRF risk, token and credential flows, injection/SSRF/path risk, secret handling, and abuse resistance."
---

# Go Security Review

## Purpose
Protect changed code from exploitable vulnerabilities and security-contract drift at trust boundaries, data boundaries, and side-effecting operations.

## Specialist Stance
- Review exploitability and fail-closed behavior before generic hardening advice.
- Prioritize trust-boundary validation, object-level authorization, tenant isolation, injection/SSRF/path risks, secret exposure, and abuse controls.
- Treat internal callers, async messages, and caches as untrusted unless a clear trust contract proves otherwise.
- Hand off performance, QA, reliability, or data depth when they support security but do not own the primary risk.

## Scope
- review untrusted-input handling and strict boundary validation
- review authentication, authorization, tenant isolation, and object-level access checks
- review browser session controls, CSRF, credentialed CORS, and cookie hardening when touched
- review credential and token flows: JWT verification, header-derived identity, password reset, token storage, and password hashing
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

## Reference Files Selector
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default: choose the file whose symptom matches the strongest review pressure. Load multiple references only when the diff spans independent security decisions, such as an authz defect plus a separate SSRF defect.

Choose references by symptom and expected behavior change:

| Symptom in the diff | Load | Behavior change |
| --- | --- | --- |
| Inbound HTTP, generated handler, config, env, CLI, async message, cache payload, or partner-feed data is normalized before action | `references/trust-boundary-and-input-validation-review.md` | Choose boundary-first typed parsing, allowlists, and pre-side-effect rejection instead of "sanitize later" or relying on internal caller trust. |
| Authentication, caller identity, tenant propagation, object-by-ID lookup, admin checks, or access-control failure handling changes | `references/authz-tenant-and-object-access-review.md` | Separate authn, authz, tenant binding, and object ownership instead of treating login, role strings, or object IDs as enough. |
| Browser-callable routes, cookie sessions, credentialed CORS, CSRF exposure, or session cookie attributes change | `references/browser-session-cors-and-csrf-review.md` | Review browser-state attack paths instead of treating CORS as authorization or relying on server authz alone. |
| JWT parsing, header-derived identity, session/API/reset tokens, password reset, invitation links, or password hashing changes | `references/token-and-credential-flow-review.md` | Inspect token verification, entropy, storage, replay, and password hashing instead of only asking for throttling or redaction. |
| SQL, query builders, filter DSLs, templates, subprocesses, or interpreter-like strings use caller-influenced values | `references/injection-query-and-command-safety.md` | Trace attacker-controlled data into interpreter syntax and choose bind/allowlist/no-shell fixes instead of generic injection warnings. |
| User, tenant, partner, webhook, preview, import, redirect, callback, or config input influences outbound HTTP/network targets | `references/ssrf-outbound-and-redirect-safety.md` | Review scheme, host, redirect, DNS, IP-class, timeout, and response limits instead of just saying "validate URL." |
| User-controlled paths, upload names/content, multipart data, archive members, static serving, downloads, temp files, or config file paths change | `references/path-upload-and-filesystem-safety.md` | Choose root-constrained file access and storage isolation instead of lexical cleanup or trusting uploaded filenames. |
| Credentials, PII, auth headers, DSNs, config values, error payloads, logs, traces, metrics labels, panic recovery, debug endpoints, or deployment policy files change | `references/secrets-pii-and-telemetry-disclosure.md` | Review exact disclosure sinks and low-cardinality redaction instead of broad "do not log secrets" advice. |
| Request size, pagination, filters, retries, fan-out, queues, file processing, reset/OTP/webhook/provider cost, or rate-limit semantics change | `references/abuse-resistance-and-resource-bounds.md` | Name the exhausted resource and enforce pre-work bounds instead of vague DoS or rate-limit comments. |

Do not load a reference just because it mentions a keyword; load it when its examples would change the finding you write. Escalate instead of solving locally when the smallest safe correction changes the security contract, identity model, API-visible semantics, or rollout policy.

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

### Browser Session, CORS, And CSRF
- Treat credentialed browser requests as their own trust boundary, especially cookie-authenticated state changes.
- Reject reflective or wildcard credentialed CORS for sensitive APIs.
- Require CSRF defenses or an explicit same-site browser policy before side effects on cookie-authenticated routes.
- Require session cookies to keep `Secure`, `HttpOnly`, and appropriate `SameSite`, `Path`, and `Domain` constraints when touched.
- Do not treat CORS as authorization; server-side identity and authorization still own access.

### Tokens, Credentials, And Password Reset
- Require JWT or bearer token validation to cover signature, algorithm allowlist, issuer, audience, expiry/not-before, key source, and parse errors when those controls are local to the change.
- Reject client-controlled identity headers unless an authenticated gateway contract strips and sets them before the service boundary.
- Require reset, invitation, and API tokens to use cryptographic randomness, enough entropy, expiry, replay or single-use controls, and hashed-at-rest storage when persisted.
- Reject password storage with plaintext, reversible encryption, fast hashes, or custom hashing.
- Keep enumeration, token leakage, and reset throttling distinct when reviewing account recovery flows.

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
