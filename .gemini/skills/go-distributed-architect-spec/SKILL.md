---
name: go-distributed-architect-spec
description: "Design distributed-consistency-first specifications for Go services in a spec-first workflow. Use when planning or revising cross-service workflows before coding and you need explicit saga/orchestration-choreography decisions, invariant ownership, outbox/inbox idempotency contracts, compensation/forward-recovery policies, and reconciliation strategy. Skip when the task is a local code fix, endpoint-level API contract design, physical SQL schema/migration scripting, CI/container setup, or low-level resilience/observability tuning."
---

# Go Distributed Architect Spec

## Purpose
Create a clear, reviewable distributed-consistency specification package before implementation. Success means cross-service workflow ownership, consistency semantics, and failure-handling contracts are explicit, defensible, and directly translatable into implementation and tests.

## Scope And Boundaries
In scope:
- decompose cross-service business flows into explicit steps, owners, and local transaction boundaries
- define invariant register with explicit split between `local_hard_invariant` and `cross_service_process_invariant`
- choose workflow style (orchestration vs choreography) and define valid boundaries and handoff rules
- define per-step contract: trigger, local transaction scope, idempotency key, timeout/retry class, success transition, compensation or forward recovery
- define pivot transaction and pre-pivot/post-pivot policies
- define outbox/inbox/dedup requirements and durable commit-before-ack ordering
- define replay/out-of-order/duplicate handling under at-least-once delivery
- define reconciliation ownership/cadence and read-model freshness constraints
- define race controls (single active workflow per business key, CAS/version transitions, serialization boundaries)

Out of scope:
- baseline service/module topology decisions outside distributed-consistency domain
- endpoint-level HTTP/JSON payload, status, and error contract design
- physical SQL schema design, DDL details, migration scripts, and backfill mechanics
- concrete cache runtime tuning (exact keys, TTL/jitter, invalidation implementation)
- detailed resilience tuning thresholds (bulkhead/circuit/load-shedding specifics)
- detailed observability schemas, SLI/SLO targets, and alert-threshold tuning
- full security-control catalogs and authn/authz hardening specifics
- CI/CD gate design and container/runtime hardening setup
- low-level test implementation details for specific test suites

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: seed distributed assumptions/blockers in `80` and distributed constraints for `40`
   - Phase 1: define distributed constraints that must shape `20` and `60`
   - Phase 2 and later: maintain `40/80/90` and update impacted `20/30/55/70`
3. Load context using this skill's dynamic loading rules and stop when four distributed axes are source-backed: workflow ownership, invariant classification, delivery/idempotency semantics, and reconciliation/freshness policy.
4. Normalize each affected flow into explicit entities, owners, invariants, state transitions, and failure classes.
5. For each nontrivial distributed decision, compare at least two options and select one explicitly.
6. Assign decision ID (`DIST-###`) and owner for each major distributed decision.
7. Record trade-offs and cross-domain impact (architecture, API, data, reliability, observability, security).
8. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate them in the current pass or move them to `80-open-questions.md` with owner and unblock condition.
9. Record any blocking uncertainty in `80-open-questions.md` with concrete next step.
10. Keep `40-data-consistency-cache.md` as primary artifact and synchronize distributed implications in impacted artifacts.
11. Verify internal consistency: no hidden global-atomicity assumptions and no critical distributed decisions deferred to coding.

## Decision Heuristics
- keep behavior inside one local transaction boundary when an invariant must hold at commit time and compensation is unacceptable
- choose orchestration when one business outcome spans multiple services and needs explicit operational control of retries/timeouts/compensation
- choose choreography when reactions are independent and do not require centralized process state
- place non-compensable steps after the pivot and enforce idempotent forward recovery for post-pivot completion
- require one active workflow per business key and version-checked state transitions before approving the flow design

## Distributed Decision Protocol
For every major distributed decision, document:
1. decision ID (`DIST-###`) and current phase
2. owner role
3. context and problem
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. trade-offs (gains and losses)
8. invariant and consistency impact
9. idempotency/dedup/outbox-inbox implications
10. compensation/forward-recovery and reconciliation implications
11. impact on architecture, API, data, reliability, observability, and security
12. reopen conditions, affected artifacts, and linked open-question IDs (if any)

## Output Expectations
- Primary artifact:
  - `40-data-consistency-cache.md` with mandatory sections:
    - `Distributed Flow Inventory`
    - `Invariant Register And Ownership`
    - `Workflow State Models And Step Contracts`
    - `Consistency And Staleness Contracts`
    - `Outbox/Inbox/Dedup And Idempotency Policy`
    - `Compensation, Pivot, And Forward Recovery`
    - `Reconciliation Ownership And Cadence`
  - `40-data-consistency-cache.md` must include these checkable tables:
    - flow table: `flow_id`, `business_outcome`, `owner_service`, `interaction_style`, `consistency_type`, `max_staleness`, `reconciliation_owner`
    - step table: `flow_id`, `step_id`, `step_owner`, `trigger`, `local_tx_boundary`, `idempotency_key_source`, `timeout_budget`, `retry_class`, `success_transition`, `recovery_mode`
    - reconciliation table: `flow_id`, `job_owner`, `cadence`, `watermark_strategy`, `repair_action`, `escalation_path`
- Required core artifacts per pass:
  - `80-open-questions.md` with distributed blockers/uncertainties
  - `90-signoff.md` with accepted distributed decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `20-architecture.md`
  - `30-api-contract.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
- Conditional artifact status format for `20/30/55/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DIST-###`
  - for `updated`, list changed sections and linked `DIST-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit owners, transitions, and failure-path semantics.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four distributed axes are covered with source-backed inputs: workflow ownership, invariant classification, delivery/idempotency semantics, reconciliation/freshness policy.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved decisions require them
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`

Load by trigger:
- Failure/degradation/system-evolution implications:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Sync request-reply constraints and deadline coupling:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Schema evolution, migration sequencing, replay/reliability implications:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- API-visible idempotency/retry/async semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Async observability coverage (lag/retry/dlq/reconciliation):
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `40-data-consistency-cache.md` includes all mandatory sections and required tables from this skill.
- Every affected cross-service flow has explicit owner, state model, and transitions.
- Every critical invariant is classified with owner and enforcement point.
- Every workflow step defines trigger, transaction scope, idempotency key policy, timeout/retry class, and compensation or forward-recovery rule.
- Outbox/inbox/dedup and durable commit-before-ack ordering are defined for side-effecting flows.
- Replay/out-of-order/duplicate handling and reconciliation strategy are explicit and testable.
- Staleness/freshness contracts are defined where eventual consistency is used.
- Distributed blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- No hidden assumptions of global ACID semantics remain.

## Anti-Patterns
- assuming implicit global ACID behavior across services
- accepting dual-write (`db + publish`) without outbox-equivalent atomic linkage
- leaving workflow behavior without explicit state model and step contracts
- allowing retries without idempotency and dedup policy
- enforcing hard write invariants from stale projections/read models
- leaving compensation or forward recovery undefined for failure paths
- deferring critical distributed-consistency decisions to coding phase
