---
name: go-performance-spec
description: "Design performance-first specifications for Go services in a spec-first workflow. Use when planning or revising latency, throughput, allocation, contention, and capacity behavior before coding and you need explicit hot-path budgets, benchmark/profile/trace acceptance criteria, and performance risk controls. Skip when the task is a local code fix, endpoint-only API payload design, schema-only modeling, CI/container setup, or low-level micro-optimization implementation."
---

# Go Performance Spec

## Purpose
Create a clear, reviewable performance specification package before implementation. Success means performance goals, bottlenecks, and verification criteria are explicit, defensible, and directly translatable into implementation and test work.

## Scope And Boundaries
In scope:
- define performance budgets for affected critical paths (latency, throughput, allocation/resource constraints)
- define bottleneck model across handler, domain, DB/cache, client, and concurrency boundaries
- define measurable acceptance criteria and evidence requirements (benchmark/profile/trace)
- define performance-sensitive constraints for implementation sequencing and rollout safety checks
- define regression-control obligations for `70-test-plan.md` and review readiness
- synchronize performance implications across affected spec artifacts
- produce performance deliverables that remove hidden "decide later" gaps

Out of scope:
- primary ownership of service decomposition and architecture boundaries
- endpoint-level API payload/status/error schema design outside performance implications
- primary ownership of schema ownership, DDL, and migration sequencing
- primary ownership of cache topology/key policy and invalidation mechanics
- primary ownership of timeout/retry/degradation policies and incident runbook governance
- primary ownership of security control design
- implementation-level optimization coding before spec sign-off

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: seed performance assumptions and blockers in `80`.
   - Phase 1: define architecture-shaping performance constraints for `20` and sequencing constraints for `60`.
   - Phase 2 and later: maintain `20/60/70/80/90` and update impacted `30/40/50/55` when required.
3. Load context using this file's dynamic loading rules and stop when four performance axes are source-backed: budget targets, bottleneck map, measurement protocol, and acceptance criteria.
4. Normalize target operations and load shape: which operations are hot paths, what workload class matters, and what user-facing/system-facing metric is authoritative.
5. For each nontrivial performance decision, compare at least two options and select one explicitly.
6. Assign decision ID (`PERF-###`) and owner for each major performance decision.
7. Record trade-offs and cross-domain impact (architecture, API, data/cache, reliability, observability, delivery).
8. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or move to `80-open-questions.md` with owner and unblock condition.
9. If uncertainty blocks a measurable and safe performance decision, record it in `80-open-questions.md` with concrete next step.
10. Keep performance outputs measurement-first: attach an explicit evidence plan and target threshold to each optimization claim.
11. Verify internal consistency: ensure budgets, criteria, and affected artifacts are aligned before closing the pass.

## Performance Decision Protocol
For every major performance decision, document:
1. decision ID (`PERF-###`) and current phase
2. owner role
3. context and target operation/workload
4. bottleneck hypothesis and baseline assumptions
5. options (minimum two)
6. selected option with rationale
7. at least one rejected option with explicit rejection reason
8. measurement protocol (benchmark/profile/trace type, environment, dataset shape)
9. acceptance thresholds and pass/fail criteria
10. trade-offs (`latency`/`throughput`/`allocation`/complexity/cost)
11. cross-domain impact and affected artifacts
12. reopen conditions and linked open-question IDs (if any)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `PERF-###` decisions with rationale and trade-offs
  - `Artifact Update Matrix`: `20/60/70` and conditional `30/40/50/55` with `Status: updated|no changes required` and linked `PERF-###`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved items for `80-open-questions.md` with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
- Primary artifacts:
  - `20-architecture.md`:
    - critical path map
    - budget decomposition by operation class
    - throughput/concurrency assumptions
  - `60-implementation-plan.md`:
    - performance-sensitive sequencing
    - measurement checkpoints and rollback-safe transition criteria
  - `70-test-plan.md`:
    - benchmark/profile/trace coverage plan
    - baseline and target thresholds with reproducible pass/fail rules
- Required core artifacts per pass:
  - `80-open-questions.md` with performance blockers and owners
  - `90-signoff.md` with accepted performance decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
- Conditional artifact status format for `30/40/50/55`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `PERF-###`
  - for `updated`, list changed sections and linked `PERF-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit metrics, thresholds, and validation protocol.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four performance axes are covered with source-backed inputs: budget targets, bottleneck model, measurement protocol, and acceptance criteria.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved performance decisions require them
- `docs/llm/go-instructions/60-go-performance-and-profiling.md`

Load by trigger:
- Concurrency model, lock contention, goroutine lifecycle, or queueing concerns:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Benchmark/test strategy or quality-pipeline implications:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Sync/async interaction and resilience implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- DB/cache bottleneck implications:
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
- API contract implications (payload size, latency-visible behavior, idempotency/retry impacts):
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Observability and release-gate implications for performance acceptance:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- Affected critical paths have explicit performance budget targets.
- Every major decision includes `PERF-###`, owner, selected option, and at least one rejected option with reason.
- `70-test-plan.md` includes reproducible benchmark/profile/trace obligations and pass/fail thresholds.
- Claimed improvements or constraints have explicit measurement protocol.
- Impacted `30/40/50/55` artifacts have explicit status with decision links and no contradictions.
- Performance blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- No critical performance decision is deferred to coding.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- tie every optimization proposal to an explicit metric, threshold, and verification protocol
- combine microbenchmarks with profile/trace or scenario-level evidence before making system-level conclusions
- define operation-level budgets for each affected hot path
- include contention, queueing, and scheduler analysis when concurrency-sensitive paths are involved
- keep ownership boundaries explicit and coordinate cross-domain implications with reliability/observability roles
- define benchmark/profile environment and dataset shape up front for reproducible results
- move unresolved critical uncertainty to `80-open-questions.md` with owner and unblock condition
