---
name: go-security-spec
description: "Design security-first specifications for Go services in a spec-first workflow. Use when planning or revising security requirements before coding and you need explicit trust boundaries, authn/authz and tenant-isolation rules, threat-class controls, secure defaults, abuse-resistance guardrails, and testable security acceptance criteria. Skip when the task is a local code fix, low-level middleware/handler implementation, pure API/resource modeling, SQL schema-only work, or CI/container execution setup."
---

# Go Security Spec

## Purpose
Create a clear, reviewable security specification package before implementation. Success means trust boundaries, identity/access rules, and threat controls are explicit, defensible, and directly translatable into implementation and tests.
Use `Hard Skills` as the normative domain baseline for decision quality and security-risk controls; use workflow sections below for execution sequence and artifact synchronization.

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

## Hard Skills
### Security Spec Core Instructions

#### Mission
- Convert security intent into enforceable pre-coding contracts for trust boundaries, identity, and threat controls.
- Protect `Gate G2` readiness by removing hidden security decisions and implicit trust assumptions.
- Ensure selected controls are fail-closed, testable, and operationally observable.

#### Default Posture
- Use zero-trust baseline: treat external and internal traffic as untrusted unless explicitly proven otherwise.
- Keep authentication, authorization, tenant isolation, and data protection as separate explicit decision blocks.
- Prefer deny-by-default and least privilege; missing policy equals deny.
- Prefer standard library and minimal dependencies; security libraries require explicit justification.
- Treat missing threat inputs, identity-model facts, or enforcement ownership as blockers until bounded as `[assumption]`.

#### Spec-First Workflow Competency
- Enforce phase-aware behavior from `docs/spec-first-workflow.md`; security decisions are finalized before coding.
- Keep `50-security-observability-devops.md` as primary security artifact and synchronize impacts into `30/40/55/70/80/90`.
- Treat unresolved trust-boundary, AuthN/AuthZ, tenant-isolation, or threat-control decisions as blockers for `Gate G2`.
- Require `SEC-###` linkage for every material decision and affected artifact section.

#### Trust Boundary And Threat Modeling Competency
- Require explicit boundary map for each affected flow: `external`, `partner`, `internal service`, `async worker/consumer`.
- Require explicit data classification and leakage exposure assumptions per boundary.
- Require side-effect and retry model classification (`idempotent`/`non-idempotent`) before selecting controls.
- Require explicit outbound access policy: allowed schemes/hosts/ports, redirect policy, and egress expectations.
- Reject generic threat statements without concrete attacker path and impact.

#### Identity And Access Control Competency
- Require one explicit `AuthContext` model with caller/subject separation and tenant binding.
- Require complete AuthN contract per boundary:
  - token/cert verification rules
  - audience/issuer/algorithm/lifetime checks
  - trusted key-source policy.
- Require layered AuthZ boundaries:
  - middleware authenticates and builds context
  - service layer enforces object-level authorization before side effects
  - repository/data path enforces tenant scope.
- Require explicit propagation policy per hop (`forward_token`, `token_exchange`, `internal_token`), including async envelopes.
- Reject designs that trust unsigned headers (`X-User-Id`, `X-Tenant-Id`, roles) as identity source.

#### Threat-Class Control Matrix Competency
- Require threat-class controls for every affected boundary:
  - input validation and strict decode
  - output encoding and error sanitization
  - injection controls (SQL/NoSQL/command/template)
  - SSRF policy
  - path traversal controls
  - deserialization safety
  - resource-exhaustion controls.
- Enforce strict request-boundary defaults:
  - bounded headers/URI/body/multipart limits
  - unknown-field rejection and trailing-token rejection for mutable JSON writes.
- Require dangerous-path exception handling:
  - command execution and `unsafe` usage are forbidden by default and require explicit approval.
- Reject "validate later in business logic" and blacklist-only validation as primary controls.

#### API Security Contract Competency
- Require security-critical contract semantics to be explicit in API artifacts:
  - auth error semantics (`401`/`403`)
  - size/media constraints (`413`/`415`/`422`)
  - rate-limit semantics (`429`, `Retry-After`)
  - idempotency and retry class.
- Require retry-unsafe operations to declare idempotency-key policy (scope, TTL, conflict behavior).
- Require long-running side effects to use explicit async semantics (`202` + operation resource) rather than fake synchronous success.
- Require stable sanitized error model (`application/problem+json`) with no internal leak.
- Reject contract/runtime drift where controls exist in docs but not enforceable in middleware/service flow.

#### Async And Distributed Security Competency
- Require message-authenticity model for async paths: signed envelope, integrity checks, replay window, and dedup boundary.
- Require explicit prohibition of raw bearer-token propagation through async payloads.
- Require outbox/inbox-idempotency alignment for side-effecting async flows and saga steps.
- Require security invariants across workflow states:
  - who may trigger each step
  - compensation authorization semantics
  - stuck/timeout handling and escalation ownership.
- Reject "eventual consistency will fix access control later" assumptions.

#### Data, Storage, And Cache Security Competency
- Require service-owned data-boundary discipline; no implicit cross-service DB trust.
- Require parameterized SQL and allowlisted dynamic identifiers in Go SQL access decisions.
- Require least-privilege DB role split (runtime vs migration) and no sensitive query leakage in logs.
- Require migration security controls:
  - mixed-version compatibility window
  - no cross-system dual writes
  - rollback/recovery limitations made explicit.
- Require cache safety rules when cache is affected:
  - tenant/scope/version-safe keys
  - no shared-cache secret storage by default
  - explicit fail-open/fail-closed policy per data class.

#### Abuse-Resistance And Resilience Competency
- Require bounded timeouts, retries, queues, concurrency, and request limits for security-sensitive paths.
- Require retry policy to remain idempotency-aware and bounded with jitter.
- Require overload behavior to preserve security invariants (`fail_closed` for critical dependency classes).
- Require explicit degradation policy so fallback modes cannot bypass authorization or tenant isolation.
- Reject infinite timeout/retry/unbounded buffering patterns.

#### Security Observability And Privacy Competency
- Require audit-relevant security events to be observable with bounded taxonomy:
  - authn failures
  - authz denies
  - tenant-scope violations
  - idempotency conflicts
  - abuse-control triggers.
- Require structured logs/traces with correlation IDs while prohibiting secrets/tokens/PII leakage.
- Require telemetry cardinality discipline and sanitized error exposure.
- Require debug/pprof/admin endpoint isolation and TTL-bounded incident-mode instrumentation.
- Reject observability plans that cannot support incident triage without exposing sensitive data.

#### Delivery And Runtime Hardening Competency
- Require security impact to map to blocking quality gates (contract checks, `govulncheck`, `gosec`, container scan policy).
- Require docs/codegen/migration drift controls when security-related behavior changes.
- Require container hardening baseline for security-sensitive services:
  - non-root runtime
  - minimal runtime image
  - no embedded secrets
  - TLS trust-store correctness.
- Reject merge readiness claims without explicit gate evidence path.

#### Verification And Test Obligations Competency
- Require `70-test-plan.md` to include negative and abuse-path coverage tied to each major `SEC-###` decision.
- Mandatory negative classes when applicable:
  - forged/invalid token
  - wrong tenant
  - insufficient scope/role
  - object-level access denial
  - injection/SSRF/path traversal attempts
  - payload/limit abuse.
- Require async security tests for signature/replay/dedup/tenant checks when async path is in scope.
- Require verification commands to include security/tooling path relevant to changed scope.
- Reject sign-off if security controls are specified but untestable.

#### Evidence Threshold Competency
- Every major security decision must include:
  1. decision ID (`SEC-###`) and owner
  2. trust boundary and threat scenario
  3. at least two options
  4. selected option and at least one rejected option with explicit reason
  5. control mapping and enforcement points (`contract`/`middleware`/`service`/`repository`/`infra`)
  6. fail behavior (`fail_closed`, error semantics, audit obligations)
  7. cross-domain impact summary (API/data/distributed/reliability/observability/delivery/platform)
  8. verification obligations (`70-test-plan.md` + runtime evidence path)
  9. residual risk and reopen criteria.
- Security claims without explicit enforcement and verification are invalid.
- Decision quality is measured by enforceability and incident survivability, not document length.

#### Assumption And Uncertainty Discipline
- Mark unknown critical facts as `[assumption]` immediately.
- Keep assumptions bounded, testable, and linked to decisions.
- Resolve assumptions in current pass when source-backed validation is possible.
- Promote unresolved critical assumptions to `80-open-questions.md` with owner and unblock condition.
- Never hide uncertainty in generic wording or defer it to coding phase.

#### Review Blockers For This Skill
- Trust boundary or identity model ambiguity for changed critical flows.
- Missing object-level authorization or tenant isolation enforcement points.
- Missing threat-class control coverage for changed untrusted inputs.
- Retry-unsafe operation without idempotency and conflict semantics.
- Async security path without authenticity/replay/dedup/tenant checks.
- Sensitive-data handling without redaction/sanitization rules.
- Abuse-prone path without bounded timeout/limit/concurrency/rate controls.
- Security-critical delivery/runtime hardening implications not mapped to gates.
- Major decision missing `SEC-###`, alternative comparison, or verification obligations.
- Critical security uncertainty deferred to coding instead of blocker tracking.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets and keep depth proportional to phase:
   - Phase 0: contribute minimum security constraints that keep architecture framing safe; record unresolved assumptions/blockers in `80`.
   - Phase 1: refine architecture-shaping security constraints for `20` and rollout-safety constraints for `60`.
   - Phase 2 and later: run full security pass; maintain `50/80/90` and update impacted `30/40/55/70`.
3. Apply `Hard Skills` defaults from this file. Any deviation must be explicit, justified, linked to decision ID (`SEC-###`), and carry reopen criteria.
4. Load context using this skill's dynamic loading rules and stop when four security axes are source-backed: trust boundaries, identity/access model, threat controls, and verification obligations.
5. Normalize affected security surface: entry boundaries, sensitive data classes, side effects, retry behavior, and abuse vectors.
6. For each nontrivial security decision, compare at least two options and select one explicitly.
7. Assign decision ID (`SEC-###`) and owner for each major security decision.
8. Record trade-offs and cross-domain impact (API, data, distributed consistency, reliability, observability, delivery/platform).
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or move to `80-open-questions.md` with owner and unblock condition.
10. If uncertainty blocks secure-by-default decisions, record it in `80-open-questions.md` with concrete next step.
11. Keep `50-security-observability-devops.md` as primary artifact and synchronize impacted spec files.
12. Verify internal consistency: no hidden security decisions deferred to coding and no contradictions between `50` and impacted `30/40/55/70/90`.
13. Run final blocker check against `Hard Skills -> Review Blockers For This Skill` before closing a pass.

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
- Response format:
  - `Decision Register`: accepted `SEC-###` decisions with rationale and trade-offs
  - `Threat-Control Matrix`: controls by boundary/threat class with enforcement points
  - `Artifact Update Matrix`: `50/80/90` and conditional `30/40/55/70` with `Status: updated|no changes required` and linked `SEC-###`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved security blockers with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
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
- No active item from `Hard Skills -> Review Blockers For This Skill` remains unresolved.
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
