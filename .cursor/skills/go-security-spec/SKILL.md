---
name: go-security-spec
description: "Design security-first specifications for Go services in a spec-first workflow. Use when planning or revising security requirements before coding and you need explicit trust boundaries, authn/authz and tenant-isolation rules, threat-class controls, secure defaults, abuse-resistance guardrails, and testable security acceptance criteria. Skip when the task is a local code fix, low-level middleware/handler implementation, pure API/resource modeling, SQL schema-only work, or CI/container execution setup."
---

# Go Security Spec

## Purpose
Create a clear, reviewable security specification package before implementation. Success means trust boundaries, identity/access rules, and threat controls are explicit, defensible, and directly translatable into implementation and tests.

## Scope And Boundaries
In scope:
- define trust boundaries, security assumptions, and threat exposure for affected flows
- define identity and access model requirements (caller/subject separation, authn/authz boundaries, tenant isolation, object-level authorization)
- define secure-by-default controls by threat class (validation, encoding, injection, SSRF, path traversal, deserialization, resource limits)
- define secrets and sensitive-data handling requirements (sanitization/redaction/no-leak rules)
- define abuse-resistance security requirements (timeouts, limits, bounded concurrency, rate controls on expensive paths)
- define fail-closed and deny-by-default behavior requirements
- define security acceptance criteria and negative-path obligations for `70-test-plan.md`
- produce security deliverables that remove hidden "decide later" gaps

Out of scope:
- endpoint/resource modeling and full API semantics outside security boundary implications
- service/module decomposition decisions outside security domain
- physical SQL schema design, DDL details, and migration scripting
- distributed workflow decomposition/orchestration as a primary domain
- detailed cache runtime tuning (exact keys, TTL/jitter, invalidation mechanics)
- full SLI/SLO targets and alert-threshold tuning
- CI/CD implementation details and container/runtime execution setup
- low-level implementation wiring in handlers/middleware/repositories
- benchmark/profiling plans and runtime performance tuning as a primary domain

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets and keep depth proportional to phase:
   - Phase 0: contribute minimum security constraints that keep architecture framing safe; record unresolved assumptions/blockers in `80`
   - Phase 1: refine architecture-shaping security constraints for `20` and rollout-safety constraints for `60`
   - Phase 2 and later: run full security pass; maintain `50/80/90` and update impacted `30/40/55/70`
3. Load context using this skill's dynamic loading rules and stop when four security axes are source-backed: trust boundaries, identity/access model, threat controls, and verification obligations.
4. Normalize affected security surface: entry boundaries, sensitive data classes, side effects, retry behavior, and abuse vectors.
5. For each nontrivial security decision, compare at least two options and select one explicitly.
6. Assign decision ID (`SEC-###`) and owner for each major security decision.
7. Record trade-offs and cross-domain impact (API, data, distributed consistency, reliability, observability, delivery/platform).
8. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or move to `80-open-questions.md` with owner and unblock condition.
9. If uncertainty blocks secure-by-default decisions, record it in `80-open-questions.md` with concrete next step.
10. Keep `50-security-observability-devops.md` as primary artifact and synchronize impacted spec files.
11. Verify internal consistency: no hidden security decisions deferred to coding and no contradictions between `50` and impacted `30/40/55/70/90`.

## Security Decision Protocol
For every major security decision, document:
1. decision ID (`SEC-###`) and current phase
2. owner role
3. context and trust boundary
4. threat scenario and impact
5. options (minimum two)
6. selected option with rationale
7. at least one rejected option with explicit rejection reason
8. control mapping and enforcement points (`contract` / `middleware` / `service` / `repository` / `infra`)
9. fail behavior (`fail-closed`, client-visible error semantics, audit/telemetry obligations)
10. test obligations (negative-path and abuse scenarios)
11. cross-domain impact and trade-offs
12. residual risk, compensating controls, reopen conditions, affected artifacts, and linked open-question IDs (if any)

## Output Expectations
- Primary artifact:
  - `50-security-observability-devops.md` with mandatory sections:
    - `Trust Boundaries And Threat Assumptions`
    - `Identity, AuthN/AuthZ, And Tenant Isolation Requirements`
    - `Threat-Class Control Matrix`
    - `Secrets, Sensitive Data, And Redaction Rules`
    - `Abuse-Resistance And Fail-Closed Policies`
    - `Security Verification And Negative Test Obligations`
    - `Residual Risk, Compensating Controls, And Reopen Criteria`
- Required core artifacts per pass:
  - `80-open-questions.md` with security blockers/uncertainties
  - `90-signoff.md` with accepted security decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
- Conditional artifact status format for `30/40/55/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `SEC-###`
  - for `updated`, list changed sections and linked `SEC-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit controls, enforcement points, and verification requirements.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four security axes are covered with source-backed inputs: trust boundaries, identity/access model, threat controls, verification obligations.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved security decisions require them
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`

Load by trigger:
- API-boundary security semantics (auth contract, error disclosure, idempotency/rate-limit coupling):
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async propagation, cross-service trust boundaries, and workflow failure paths:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/storage/cache security implications:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security observability, delivery gates, and runtime hardening implications:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/llm/platform/10-containerization-and-dockerfile.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `50-security-observability-devops.md` contains all mandatory sections from this skill.
- Trust boundaries, identity/access model, and threat assumptions are explicit for all affected flows.
- All major security decisions include `SEC-###`, owner, selected option, and at least one rejected option with reason.
- Security controls include explicit enforcement points and fail-closed behavior expectations.
- Secrets/redaction/error-disclosure requirements are explicit and consistent across impacted artifacts.
- Negative-path and abuse-path test obligations are defined and synchronized with `70-test-plan.md`.
- Security blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `30/40/55/70` artifacts have explicit status with decision links and no contradictions.
- No hidden security decisions are deferred to coding.

## Anti-Patterns
Use these preferred patterns to prevent anti-pattern drift:
- express decisions in threat-driven form with concrete controls and explicit scope
- define authentication requirements and authorization requirements as separate, explicit sections
- treat internal network traffic as untrusted by default and document any approved trust exception
- define object-level authorization and tenant isolation requirements for each affected resource flow
- cover both happy-path and negative/abuse-path security scenarios in spec and test obligations
- map each control to explicit enforcement points and verification obligations
- close critical security decisions in spec artifacts or track them as explicit open questions with owner
