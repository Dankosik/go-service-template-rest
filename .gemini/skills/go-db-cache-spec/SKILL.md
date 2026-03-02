---
name: go-db-cache-spec
description: "Design DB-access-and-cache-first specifications for Go services in a spec-first workflow. Use when planning or revising SQL access discipline, query and transaction risk controls, cache strategy, staleness/fallback semantics, and DB/cache observability-test obligations before coding. Skip when the task is a local code fix, primary schema ownership/migration design, endpoint-only API contract work, CI/container setup, or low-level implementation tuning."
---

# Go DB Cache Spec

## Purpose
Create a clear, reviewable DB/cache specification package before implementation. Success means SQL access and cache decisions are explicit, defensible, measurable, and directly translatable into implementation and test work without hidden runtime-risk assumptions.

## Scope And Boundaries
In scope:
- define SQL access discipline for runtime paths (query shape, round-trip budget, N+1/chatty prevention)
- define transaction and retry/idempotency guardrails at specification level
- define timeout/context/pooling constraints and connection-budget assumptions for DB paths
- decide whether cache is justified based on measured bottleneck evidence
- choose cache topology and pattern with explicit trade-offs (`local`/`distributed`/`hybrid`, `cache-aside` default)
- define staleness and consistency contract per operation class
- define key safety requirements (tenant/scope/version), TTL/jitter/invalidation strategy, and stampede controls
- define failure behavior and degradation policy (fail-open/fallback/bypass/origin protection)
- define DB/cache observability and test obligations required for merge readiness
- produce DB/cache deliverables that remove "decide later" gaps

Out of scope:
- primary data ownership, logical schema modeling, DDL, and migration sequencing leadership
- service/module decomposition and global interaction-style architecture decisions
- endpoint-only REST payload/status/error design details
- full security architecture and authn/authz policy outside DB/cache surface
- SLI/SLO governance and incident process as primary domain
- CI/CD pipeline and container runtime hardening decisions
- low-level repository/cache client code implementation and micro-optimizations

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: seed DB/cache assumptions and blockers in `80`.
   - Phase 1: define DB/cache constraints that must shape `20` and `60`.
   - Phase 2 and later: update `40/80/90` plus impacted `30/50/55/60/70`.
3. Load context using this file's dynamic loading rules and stop when four DB/cache axes are source-backed: SQL access discipline, cache design, degradation behavior, and observability/test obligations.
4. Normalize the decision problem: bottleneck evidence, consistency/staleness tolerance, tenant/scope safety, and failure envelope.
5. For each nontrivial DB/cache decision, compare at least two options and select one explicitly.
6. Assign decision ID (`DBC-###`) and owner for each major DB/cache decision.
7. Record trade-offs and cross-domain impact (API, data, reliability, security, operability).
8. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or move to `80-open-questions.md` with owner and unblock condition.
9. If a DB/cache uncertainty blocks safe implementation, record it in `80-open-questions.md` with concrete next step.
10. Keep `40-data-consistency-cache.md` as the primary artifact and maintain explicit boundary with data-ownership/migration responsibilities.
11. Verify internal consistency: no contradictions across impacted artifacts and no critical DB/cache decisions deferred to coding.
12. Close each pass with a concise readiness summary mapped to `Definition Of Done` so downstream roles can execute without reinterpretation.

## DB/Cache Decision Protocol
For every major DB/cache decision, document:
1. decision ID (`DBC-###`) and current phase
2. owner role
3. context and bottleneck/risk evidence
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. consistency/staleness semantics and transaction implications
8. failure policy (timeout hierarchy, fallback mode, bypass behavior)
9. observability and test obligations
10. impact on API, data, security, and reliability
11. reopen conditions, affected artifacts, and linked open-question IDs (if any)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `DBC-###` decisions with rationale and trade-offs
  - `Artifact Update Matrix`: `40/30/50/55/60/70` with `Status: updated|no changes required` and linked `DBC-###`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved items for `80-open-questions.md` with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
- Primary artifact:
  - `40-data-consistency-cache.md` containing:
    - `SQL Access Risk Profile`
    - `Cache Necessity Decision`
    - `Topology And Pattern Choice`
    - `Staleness And Consistency Contract`
    - `Key/Tenant/Version Safety Requirements`
    - `Invalidation/TTL/Jitter/Stampede Controls`
    - `Failure And Degradation Policy`
- Required core artifacts per pass:
  - `80-open-questions.md` with DB/cache blockers and owners
  - `90-signoff.md` with accepted DB/cache decisions and reopen criteria
- Conditional alignment artifacts (update when impacted by DB/cache decisions):
  - `30-api-contract.md`
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `60-implementation-plan.md`
  - `70-test-plan.md`
- Conditional artifact status format for `30/50/55/60/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DBC-###`
  - for `updated`, list changed sections and linked `DBC-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit trade-offs, assumptions, and safety constraints.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four DB/cache axes are covered with source-backed inputs: SQL access discipline, cache strategy correctness, degradation behavior, and observability/test obligations.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only if unresolved DB/cache decisions require them
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/50-caching-strategy.md`

Load by trigger:
- Timeout/cancellation/retry semantics require tighter error-context framing:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Test obligations need deeper quality framing:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
- Data ownership/schema evolution impact:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- API-visible consistency/idempotency impact:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Cross-service invalidation, async processing, or distributed consistency impact:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Reliability/degradation policy shaping:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- DB/cache telemetry or diagnostics requirements:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Security-sensitive cache surface:
  - `docs/llm/security/10-secure-coding.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `40-data-consistency-cache.md` explicitly defines SQL-access and cache decisions for all affected operation classes.
- Every major DB/cache decision includes `DBC-###`, owner, selected option, and at least one rejected option with reason.
- Cache is introduced only where bottleneck evidence or explicit rationale is documented.
- Staleness, key safety, invalidation, and fallback rules are explicit and testable.
- DB-path guardrails for query/transaction/timeouts/pooling are explicitly documented.
- Cache-path stampede controls and outage behavior are explicitly documented.
- Impacted `30/50/55/60/70` artifacts have explicit status with decision links and no contradictions.
- DB/cache blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- No critical DB/cache decision is deferred to coding.

## Anti-Patterns
Prefer these execution patterns to avoid common failure modes:
- anchor cache decisions in measured bottleneck evidence
- treat cache as an accelerator with explicit source-of-truth boundary
- specify concrete guardrails for query shape, pool budget, and N+1 prevention
- document complete cache contract: staleness, key safety, invalidation, fallback
- coordinate schema/migration leadership with the data-architecture role
- move unresolved critical uncertainty into `80-open-questions.md` with owner and unblock condition
