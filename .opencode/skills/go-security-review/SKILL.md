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

## Working Rules
1. Confirm the task is code review and identify changed security-sensitive scope.
2. Map changed files/functions to one or more security review axes. If no security-sensitive surface is present, return `No security findings.` with `Residual Risks` explaining why the scope is security-neutral.
3. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
4. Load context using this skill's dynamic loading rules.
5. Evaluate changed code in this order:
   - `Trust Boundary And Input Validation`
   - `AuthN/AuthZ And Tenant Isolation`
   - `Injection And Query Safety`
   - `Outbound Security And SSRF Controls`
   - `Filesystem, Path, And Upload Safety`
   - `Secrets And Sensitive Data Handling`
   - `Abuse Resistance And Resource Controls`
   - `Security Test Traceability`
6. Record only evidence-backed findings and map each finding to explicit approved obligations (prefer `SEC-*` decisions or explicit clauses in `50/60/70/90`).
7. For each finding, make impact concrete with attacker preconditions, affected asset/boundary, and expected security consequence.
8. Classify severity by exploitability, blast radius, and merge safety impact (`critical/high/medium/low`).
9. Provide the smallest safe corrective action for each finding.
10. Keep comments strictly in security-review domain and hand off deep cross-domain risks to the corresponding reviewer role.
11. If safe resolution requires changing approved spec intent, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
12. Do not edit spec files during code review.
13. If no findings exist, state this explicitly and include residual security risks.

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
