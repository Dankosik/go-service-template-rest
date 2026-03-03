---
name: go-security-review
description: "Review Go code changes for security correctness in a spec-first workflow. Use when auditing pull requests or diffs for trust-boundary enforcement, authn/authz and tenant isolation, injection/SSRF/path traversal/file-upload risks, secret leakage, and abuse-resistance controls against approved security contracts. Skip when designing specifications, implementing features, or performing primary architecture/performance/concurrency/DB/reliability/QA reviews."
---

# Go Security Review

## Purpose
Deliver domain-scoped code review findings for security correctness during Phase 4 review. Success means changed code preserves approved security intent, high-risk vulnerabilities are surfaced before `Gate G4`, and spec mismatches are escalated explicitly.

## Scope And Boundaries
In scope:
- review changed code against approved security contracts in `specs/<feature-id>/50-security-observability-devops.md`
- review trust-boundary enforcement for untrusted input and side-effecting operations
- review `AuthN/AuthZ` and tenant/object-level access control in changed paths
- review injection defenses (`SQL/NoSQL/command/template`) and query-safety discipline
- review outbound call security and SSRF controls for user-influenced targets
- review filesystem/path/upload controls against traversal and unsafe file handling
- review secrets and sensitive-data handling in responses, errors, logs, and traces
- review abuse-resistance controls (limits, time budgets, bounded resource usage)
- review security negative-path test traceability against approved `70-test-plan.md`
- produce actionable findings with exact `file:line`, impact, and minimal safe fix
- escalate spec-level conflicts through `Spec Reopen`

Out of scope:
- redesigning architecture during code review without explicit `Spec Reopen`
- editing spec artifacts in Phase 4
- performing primary-domain idiomatic/style, architecture integrity, performance proof, concurrency mechanics, DB/cache correctness, reliability policy, QA strategy, or domain-invariant review
- blocking PRs with preference-only comments without concrete security impact

## Hard Skills
### Security Review Core Instructions

#### Mission
- Protect merge safety by surfacing exploitable vulnerabilities and security-contract regressions in changed code before `Gate G4`.
- Keep security review evidence-based, threat-specific, and bounded to Phase 4 reviewer ownership.
- Convert each confirmed security risk into minimal safe fix guidance aligned with approved spec intent.

#### Default Posture
- Treat external and internal inputs as untrusted unless approved spec explicitly narrows trust.
- Prefer fail-closed and deny-by-default behavior for identity, authorization, and side-effecting operations.
- Treat missing limits, missing timeout budgets, and missing tenant/object checks as defects until proven safe.
- Prefer boundary-first controls and explicit runtime enforcement over best-effort hardening after side effects.
- Keep security ownership strict; hand off primary non-security root causes while preserving security impact notes.

#### Spec-First Review Competency
- Enforce `docs/spec-first-workflow.md` Phase 4 constraints:
  - domain-scoped findings only;
  - exact `file:line` references;
  - practical fix path;
  - explicit `Spec Reopen` for spec-intent conflicts.
- Treat open `critical/high` security findings as merge blockers for `Gate G4`.
- Never redefine approved security behavior implicitly through review comments.
- Map each finding to approved obligations (prefer `SEC-*`, then explicit clauses in `50/55/70/90`).

#### Trust Boundary And Input Validation Competency
- Require boundary-first validation before business logic.
- Require strict JSON discipline on mutable endpoints:
  - size limit before decode;
  - `DisallowUnknownFields`;
  - reject trailing tokens.
- Require allowlist validation for enums, ranges, formats, sortable/filterable fields, and state transitions.
- Reject blacklist-only validation and validation that starts after side effects.
- Require explicit limits for headers, URI/query, body, multipart, and filter complexity.

#### AuthN, AuthZ, And Tenant Isolation Competency
- Keep authentication and authorization findings separate.
- Require complete AuthN validation before business logic (token/cert validity, issuer/audience/lifetime/alg checks where applicable).
- Require fail-closed authorization and object-level checks on resource-by-ID flows.
- Require caller-vs-subject separation in identity context for mixed service/end-user flows.
- Require tenant scope enforcement across service logic, repositories, caches, and async handlers.
- Treat missing tenant binding, implicit superuser paths, or default-allow behavior as high-risk defects.

#### Injection And Query Safety Competency
- Require SQL/NoSQL parameterization for values and allowlisted dynamic identifiers.
- Reject raw client JSON filters mapped directly into datastore operators.
- Reject command execution with shell expansion or user-influenced command strings.
- Require template-safe rendering (`html/template` for HTML) and no unsafe escaping bypass without explicit review.
- Treat user-influenced query/command/path construction without allowlist controls as exploitable until disproven.

#### Outbound Security And SSRF Competency
- Require explicit outbound timeout budgets and context propagation.
- Require SSRF policy when outbound target is user-influenced:
  - scheme/host/port allowlists;
  - private/loopback/link-local/multicast blocking after DNS resolution;
  - redirect target re-check.
- Reject security-sensitive use of `http.Get`/`http.DefaultClient` and implicit infinite timeouts.
- Require egress-policy assumptions to be explicit; code-only SSRF controls are insufficient defense-in-depth.

#### Filesystem, Path, And Upload Competency
- Require root-constrained file access for user-influenced paths (`os.OpenInRoot` or equivalent safe boundary).
- Reject trust in client filenames/paths as storage keys.
- Require upload controls:
  - body size limits before parse;
  - streaming over full-memory buffering;
  - extension allowlist plus content sniffing;
  - storage isolation outside webroot.
- Require explicit scan/publish gating when malware/content validation is part of contract.

#### Secrets, Error Disclosure, And Telemetry Competency
- Require sanitized client-facing errors; no stack traces, SQL text, topology, tokens, or secrets in responses.
- Require redaction discipline in logs/traces/metrics:
  - never emit credentials, raw authorization headers, DSNs, or unrestricted PII payloads.
- Require correlation fields (`request_id`/`correlation_id`) for incident triage but never as auth/authz input.
- Require debug/admin endpoint isolation from public ingress and explicit kill-switch policy for pprof/expvar.
- Treat telemetry or diagnostics paths that leak sensitive data as security findings, not observability-only notes.

#### Abuse Resistance And Resource Control Competency
- Require explicit timeouts, bounded concurrency, queue bounds, and rate-limit semantics on expensive/security-sensitive paths.
- Require retry classification aligned with idempotency policy; reject retries for auth/validation/conflict/not-found classes.
- Require overload semantics (`429` vs `503`) to be explicit and safe for caller behavior.
- Flag fail-open fallback where dependency class is `critical_fail_closed` (authz, payments, hard validation).
- Treat unbounded memory (`io.ReadAll` on untrusted streams), unbounded retries, or unbounded fan-out as abuse-risk defects.

#### Async Identity And Distributed Security Competency
- Require no raw bearer token propagation in async messages.
- Require signed/verified identity envelope or equivalent authenticity checks for async processing.
- Require dedup/idempotency and durable ack ordering (ack only after durable side effects).
- Require stable correlation through retries and DLQ transitions for forensic traceability.
- Flag hidden dual-write consistency patterns that bypass security controls or auditability.

#### Data, Cache, And Migration Security Competency
- Require least-privilege DB access and no sensitive interpolated SQL logging.
- Require tenant-safe cache keys including tenant/scope/version dimensions when response varies by auth context.
- Reject caching secrets or private per-user responses in shared keys.
- Require migration/backfill plans to preserve tenant boundaries, PII lifecycle, and rollback realism.
- Flag data-evolution changes that can break deletion, retention, or audit guarantees as security-impacting issues.

#### Delivery And Runtime Hardening Competency
- Require merge-safety evidence for security-sensitive changes through repository-aligned checks:
  - `go test ./...`;
  - `go test -race ./...` when concurrency-sensitive;
  - `go vet ./...`;
  - `govulncheck ./...`;
  - `gosec ./...` where configured.
- For container/runtime-surface changes, require non-root runtime, minimal image profile, and no TLS trust bypass.
- Treat downgraded or bypassed blocking security gates as review blockers unless explicitly approved and time-bounded.

#### Security Test Traceability Competency
- Require mapping findings to negative-path obligations in `70-test-plan.md`.
- For changed security-critical paths, require at least one realistic abuse/failure scenario per affected axis.
- Mandatory categories when applicable:
  - wrong tenant or object ownership;
  - insufficient scope/role;
  - malformed or oversized input;
  - forged or invalid token/signature;
  - retry/idempotency conflict;
  - SSRF, path traversal, or injection attempts.
- Treat missing negative-path coverage for high-risk changed paths as finding-worthy or residual-risk-worthy.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - security axis context;
  - violated control/contract reference;
  - realistic attacker preconditions;
  - affected trust boundary/data asset;
  - smallest safe corrective action.
- Severity is assigned by exploitability, blast radius, and merge safety:
  - `critical`: confirmed exploitable high-impact vulnerability;
  - `high`: strong evidence of significant security contract breach likely to cause incident;
  - `medium`: meaningful bounded weakness;
  - `low`: local hardening improvement.
- Reject generic best-practice comments without concrete exploit or contract impact.

#### Assumption And Uncertainty Discipline
- Mark unknown critical facts as bounded `[assumption]`.
- If required artifacts are missing, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated via `Spec Reopen`.
- Unknowns are explicit risk statements, not closure.

#### Review Blockers For This Skill
- Missing trust-boundary validation or missing strict parsing/size limits on untrusted inputs.
- Broken or absent AuthN/AuthZ/tenant/object-level enforcement in changed paths.
- Exploitable injection, SSRF, path traversal, or unsafe upload handling patterns.
- Secret/PII leakage in responses, logs, traces, metrics, or debug endpoints.
- Unbounded resource-abuse vectors (timeouts/retries/concurrency/queue/memory) on security-sensitive operations.
- Security-critical async/dedup/ack-order defects that permit replay or inconsistent side effects.
- Missing or weakened security gates (`govulncheck`, `gosec`, container hardening) for changed risk surface.
- Spec-conflicting security correction path without explicit `Spec Reopen`.

## Working Rules
1. Confirm review unit from context (`single task` or `bounded task scope`), then identify changed security-sensitive scope.
2. Map changed files/functions to one or more security review axes. If no security-sensitive surface is present, return `No security findings.` with `Residual Risks` explaining why the scope is security-neutral.
3. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
4. Load context using this skill's dynamic loading rules.
5. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
6. Evaluate changed code in this order:
   - `Trust Boundary And Input Validation`
   - `AuthN/AuthZ And Tenant Isolation`
   - `Injection And Query Safety`
   - `Outbound Security And SSRF Controls`
   - `Filesystem, Path, And Upload Safety`
   - `Secrets And Sensitive Data Handling`
   - `Abuse Resistance And Resource Controls`
   - `Security Test Traceability`
7. Record only evidence-backed findings and map each finding to explicit approved obligations (prefer `SEC-*` decisions or explicit clauses in `50/55/60/70/90`).
8. For each finding, make impact concrete with attacker preconditions, affected asset/boundary, and expected security consequence.
9. Classify severity by exploitability, blast radius, and merge safety impact (`critical/high/medium/low`).
10. Provide the smallest safe corrective action for each finding.
11. Keep comments strictly in security-review domain and hand off deep cross-domain risks to the corresponding reviewer role.
12. If safe resolution requires changing approved spec intent, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
13. Do not edit spec files during code review.
14. If no findings exist, state this explicitly and include residual security risks.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-security-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking security risks, assumptions, or verification gaps.
- Start each `Issue` with axis context: `Axis: <one of the eight axes>; ...`.
- In `Impact`, include realistic exploit preconditions and affected trust boundary or data asset.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No security findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: confirmed high-impact vulnerability (for example broken access control, tenant escape, credential/secret leakage, or exploitable injection path) that makes merge unsafe.
- `high`: strong evidence of significant security contract mismatch likely to lead to incident under realistic conditions.
- `medium`: bounded but meaningful security weakness with limited blast radius.
- `low`: local hardening improvement with non-blocking impact.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all security review axes are assessable with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`
- review artifacts:
  - `specs/<feature-id>/50-security-observability-devops.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- API-visible security semantics (auth errors, tenant/idempotency/rate semantics, request limits):
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/query/cache security implications:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Timeout/retry/degradation behavior creates security impact (fail-open, replay, abuse windows):
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Sync/async identity propagation or cross-service trust assumptions:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Concurrency behavior may affect security controls (race in auth paths, uncontrolled fan-out on protected operations):
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Verification and test obligations:
  - `specs/<feature-id>/70-test-plan.md`
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Observability, redaction, and debug-surface implications:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Delivery/runtime hardening implications:
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/llm/platform/10-containerization-and-dockerfile.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.

## Definition Of Done
- Review output stays within security-review domain boundaries.
- Every finding is mapped to one explicit security review axis.
- Findings are evidence-backed and use exact `file:line` references.
- Every finding has impact, fix path, and spec reference.
- All `critical/high` security findings are either resolved or clearly escalated.
- No spec-level mismatch remains implicit.
- If no findings, output explicitly states `No security findings.` and includes residual risk note.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- define issues with concrete threat and exploit impact, not generic hardening advice
- separate authentication findings from authorization findings explicitly
- treat internal traffic as untrusted by default unless approved spec says otherwise
- require fail-closed behavior for security-critical paths
- keep security-domain ownership explicit and hand off deep cross-domain issues
- escalate spec-intent conflicts via `Spec Reopen` instead of implicit requirement changes
