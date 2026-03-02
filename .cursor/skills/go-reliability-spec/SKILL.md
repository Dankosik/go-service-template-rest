---
name: go-reliability-spec
description: "Design reliability-first specifications for Go services in a spec-first workflow. Use when planning or revising timeout/deadline, retry budget, backpressure, degradation, startup/shutdown, and rollout/rollback safety behavior before coding and you need explicit failure contracts and resilience acceptance criteria. Skip when the task is a local code fix, endpoint-only payload/schema design, SQL schema-only modeling, CI/container setup, or low-level implementation of middleware/worker/runtime code."
---

# Go Reliability Spec

## Purpose
Create a clear, reviewable reliability specification package before implementation. Success means failure behavior and resilience controls are explicit, defensible, and directly translatable into implementation and test work.

## Scope And Boundaries
In scope:
- define per-dependency failure contracts and criticality classes (`critical_fail_closed`, `critical_fail_degraded`, `optional_fail_open`)
- define timeout and deadline policy (end-to-end budget, per-hop caps, propagation rules, fail-fast thresholds)
- define retry eligibility, retry budgets, jitter policy, and never-retry categories
- define overload containment policy (bounded queues, bulkheads, load shedding, rejection semantics)
- define circuit-breaking policy and containment escalation rules
- define graceful lifecycle policy (startup/readiness/liveness responsibilities and shutdown draining semantics)
- define degradation/fallback mode model, activation and recovery criteria
- define rollout and rollback reliability gates for risky changes
- define reliability acceptance obligations for `70-test-plan.md`
- synchronize reliability implications across affected spec artifacts
- produce reliability deliverables that remove hidden "decide later" gaps

Out of scope:
- primary ownership of service decomposition and ownership topology
- endpoint-level API payload and error-schema design beyond reliability semantics
- primary ownership of distributed workflow topology and saga decomposition
- primary ownership of SQL ownership/DDL/migration implementation mechanics
- primary ownership of cache topology, keying, and invalidation strategy
- primary ownership of SLI/SLO target governance and alert routing
- primary ownership of secure-coding controls and threat catalog
- primary ownership of CI/CD implementation mechanics and container hardening details
- implementation-level coding of middleware, retry wrappers, worker pools, or shutdown hooks before spec sign-off

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: establish reliability baseline in `55-reliability-and-resilience.md` and seed blockers in `80-open-questions.md`
   - Phase 1: define architecture-shaping reliability constraints for `20-architecture.md` and sequencing constraints for `60-implementation-plan.md`
   - Phase 2 and later: maintain `55/80/90` and update impacted `20/30/40/50/60/70` as needed
3. Load context using this skill's dynamic loading rules and stop when five reliability axes are source-backed: dependency criticality, timeout/retry contract, overload containment, degradation lifecycle, rollout/rollback safety.
4. Classify each critical dependency first, then define explicit contract fields: timeout, retry class/budget, bulkhead bound, fallback mode, and observability trigger.
5. For each nontrivial reliability decision, compare at least two options and select one explicitly.
6. Assign decision ID (`REL-###`) and owner for each major reliability decision.
7. Record trade-offs and cross-domain impact (architecture, API, data/cache, security, observability, delivery, performance).
8. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or move them to `80-open-questions.md` with owner and unblock condition.
9. If uncertainty blocks a safe reliability decision, record it in `80-open-questions.md` with concrete next step.
10. Keep `55-reliability-and-resilience.md` as primary artifact and synchronize reliability implications in affected artifacts.
11. Verify internal consistency: no contradictory timeout/retry/degradation policy and no critical reliability decisions deferred to coding.

## Reliability Decision Protocol
For every major reliability decision, document:
1. decision ID (`REL-###`) and current phase
2. owner role
3. context and failure scenario
4. dependency criticality class and invariant impact
5. options (minimum two)
6. selected option with rationale
7. at least one rejected option with explicit rejection reason
8. contract details:
   - timeout/deadline budget and propagation
   - retry eligibility, attempts, budget, and jitter
   - queue bounds, bulkhead isolation, and shedding behavior
   - fallback/degradation mode entry and exit criteria
   - startup/readiness/liveness/shutdown behavior
   - rollout promotion/rollback triggers and authority
9. verification obligations (tests and required signals)
10. cross-domain impact and affected artifacts
11. reopen conditions and linked open-question IDs (if any)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `REL-###` decisions with rationale and trade-offs
  - `Artifact Update Matrix`: required updates for `55/80/90` and status for impacted `20/30/40/50/60/70`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved reliability items for `80-open-questions.md` with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
- Primary artifact:
  - `55-reliability-and-resilience.md` with mandatory reliability sections:
    - `Dependency Criticality And Failure Contracts`
    - `Timeout, Deadline, And Retry Policy`
    - `Backpressure, Bulkheads, And Overload Response`
    - `Degradation Modes And Fallback Policy`
    - `Startup, Readiness, Liveness, And Shutdown`
    - `Rollout, Rollback, And Reliability Gates`
- Required core artifacts per pass:
  - `80-open-questions.md` with reliability blockers/uncertainties
  - `90-signoff.md` with accepted reliability decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `20-architecture.md`
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `50-security-observability-devops.md`
  - `60-implementation-plan.md`
  - `70-test-plan.md`
- Conditional artifact status format for `20/30/40/50/60/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `REL-###`
  - for `updated`, list changed sections and linked `REL-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit reliability policy semantics and verification criteria.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when five reliability axes are covered with source-backed inputs: dependency criticality, timeout/retry policy, overload containment, degradation lifecycle, rollout/rollback safety.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved reliability decisions require them
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

Load by trigger:
- Error wrapping, cancellation semantics, and context deadline behavior:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Goroutine lifecycle, bounded queues/channels, worker pools, and shutdown coordination:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- API-visible reliability semantics (`429`/`503`, `Retry-After`, idempotency/retry, `202` fallback):
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async and distributed workflow reliability implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Observability and budget-aware release implications:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
- Data evolution/reconciliation implications:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `55-reliability-and-resilience.md` contains all mandatory reliability sections from this skill.
- Every major reliability decision includes `REL-###`, owner, selected option, and at least one rejected option with reason.
- Every critical dependency has explicit timeout/retry/bulkhead/fallback contract and owner.
- Overload, degradation, and shutdown behavior is explicit and testable.
- Rollout and rollback reliability gates are explicit with trigger and authority semantics.
- Every `[assumption]` is either source-validated in the current pass or tracked in `80-open-questions.md` with owner and unblock condition.
- Reliability blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `20/30/40/50/60/70` artifacts have explicit status with decision links and no contradictions.
- No hidden reliability decisions are deferred to coding.

## Anti-Patterns
Use these preferred patterns to keep reliability decisions reviewable and implementation-ready:
- define timeout/retry behavior with explicit budget and eligibility policy
- keep queue/concurrency bounds explicit and finite
- define degradation with explicit entry and recovery criteria
- apply soft controls first and introduce state-machine circuit breakers only with incident evidence
- keep ownership boundaries explicit and coordinate with observability/devops through interface contracts
- define rollback behavior with explicit authority and trigger rules
- move unresolved reliability uncertainty to `80-open-questions.md` with owner and unblock condition
