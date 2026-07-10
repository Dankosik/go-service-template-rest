---
name: go-architect-spec
description: "Design architecture-first specifications for Go services. Use when planning new features, refactors, service extractions, or behavior changes before coding and you need explicit target-state boundary ownership, workload-driven topology, source-of-truth, sync/async flow, consistency, failure, migration, and operability decisions. Skip local implementation, endpoint payload detail, SQL migration mechanics, or post-code review."
---

# Go Architect Spec

## Trigger And Scope

Use this skill before coding when a Go service needs a material boundary, topology, runtime, source-of-truth, sync/async, projection, provider, distributed-flow, resilience, or extraction decision. Produce architecture decisions that domain, API, data, reliability, delivery, technical design, and planning can consume.

Own system boundary and interaction choices, not endpoint payload catalogs, SQL/DDL mechanics, package/file ownership, low-level tuning, CI/container detail, or implementation code. Route those consequences to the matching specialist.

## Approved Input And Decision Boundary

Inspect the approved problem frame, current repository/service boundaries, invariant and write authority, real workload evidence, critical path, team/runtime ownership, external dependencies, failure and degradation expectations, data/read topology, and rollout/mixed-version constraints. Ground external-platform behavior in current official docs/source and credible real implementations; do not infer it from model memory. Mark invented latency, scale, growth, freshness, RPO/RTO, or cost numbers as assumptions.

Architecture may choose among real live forks; it must not manufacture options for completeness. When a real choice exists, compare the viable stdlib, repository, mature-OSS, and custom options against repository, operational, proof, and idiomatic-Go fit. Record only rejections that explain the selected design.

## Architecture Invariants And Defaults

1. **Invariant and write ownership define boundaries.** Prefer one explicit source of truth per invariant-bearing entity or process; reject service-per-table, shared-schema writes, direct cross-service DB reads, and ambiguous process ownership.
2. **A modular monolith is the default boundary.** Extract a service only when domain/data/team/transaction ownership, independent deployability, runtime isolation or scaling, operational readiness, and accepted consistency costs all justify it. A separate worker/runtime may solve isolation without a new service.
3. **Workload shapes topology.** Classify request/response, long-running, bursty fan-out, stream, reconciliation, or operator-driven work; model hot keys/tenants, backlog, payloads, and read/write pressure before choosing a broker, cache, projection, or split.
4. **Sync is earned by finality and budget.** Keep request paths short, name end-to-end/per-hop deadlines and retry/idempotency classes, and move variable, fan-out, or deferrable work behind an honest job/operation boundary.
5. **Async has one process owner.** Distinguish commands from events, queue from pub/sub, orchestration from choreography, and internal state machine from workflow engine. Correctness-bearing flows need atomic message linkage, idempotent handling, bounded retries, poison/DLQ ownership, durable state, and reconciliation.
6. **Hard invariants stay local when possible.** Identify the irreversible pivot, keep local ACID around owned state, and define compensable-before/retryable-after or forward-recovery behavior for cross-process steps. Reject unscoped exactly-once and distributed-lock shortcuts.
7. **Read scale never moves write truth accidentally.** Replicas, caches, search, exports, aggregators, and CQRS projections are derived with freshness, rebuild, bypass, and correction rules.
8. **Target-state evolution is bounded and operable.** Prefer a direct target state when it can land safely; use expand/migrate/verify/contract, canary/shadow/dual-read, or temporary bridges only when live constraints require them, with authority, reconciliation, exit criteria, removal proof, rollback limits, and owner.

Operational overhead, observability, on-call, graceful lifecycle, release coordination, and repair tooling are first-class costs. Do not select technology because it is already fashionable or available.

## Symptom-Driven Reference Selector

Load at most one reference by default. Load more only for independent architecture pressures. State which choice the reference is expected to change.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| Boundary placement, write ownership, shared data, team seams, or Go package layout is being mistaken for service architecture. | [boundary-decomposition-examples.md](references/boundary-decomposition-examples.md) | Choose invariant/ownership boundaries and dependency direction instead of entity services or generic packages. |
| Modular monolith, internal module, separate worker/runtime, or true service extraction is disputed. | [modular-monolith-vs-service-extraction.md](references/modular-monolith-vs-service-extraction.md) | Apply an all-conditions extraction test instead of treating traffic or team preference as sufficient. |
| Request path versus queue, saga/process manager, orchestration/choreography, or workflow engine is unclear. | [sync-async-workflow-ownership.md](references/sync-async-workflow-ownership.md) | Name process owner, pivot, durable state, and client completion model before choosing tools. |
| CQRS, replicas, read services, projections, search, dashboards, exports, aggregators, or stale reads are proposed. | [read-write-topology-and-projections.md](references/read-write-topology-and-projections.md) | Preserve write authority and freshness/bypass/rebuild rules instead of promoting a convenient query view to truth. |
| External provider, webhook state, vendor vocabulary, or ambiguous partner results affect lifecycle. | [external-provider-anti-corruption.md](references/external-provider-anti-corruption.md) | Normalize semi-trusted provider evidence behind a local owner instead of importing vendor state into domain truth. |
| Ownership/source-of-truth move, service extraction, mixed versions, canary, shadow read, compatibility bridge, or rollback boundary is live. | [rollout-and-migration-patterns.md](references/rollout-and-migration-patterns.md) | Select target-state authority first and bound unavoidable transition machinery with exit proof. |
| Premature microservices, distributed monolith, shared DB, direct DB reads, dual writes, retry storms, fragile fallback, or permanent shim smells appear. | [architecture-anti-patterns.md](references/architecture-anti-patterns.md) | Turn the smell into a concrete failure consequence, blocker, accepted risk, or reopen condition. |

## Required Evidence And Deliverable

For each material architecture decision, record the problem and constraints, invariant/write owner, dominant workload, critical path, whether a real live fork exists, selected option, rejected viable options and patterns, consistency/failure model, operational and Go/repository fit, measurable acceptance boundary, rollout/rollback only when triggered, assumptions, and reopen or extraction criteria.

Return a compact architecture packet with:

- context, scope, non-goals, boundary/ownership model, and dependency direction where needed;
- workload, request/background topology, critical path, and sync/async interaction;
- command/query authority, projection/cache/provider anti-corruption rules;
- invariant, pivot, state, idempotency, recovery, degradation, lifecycle, and operability model;
- live-alternative rationale and bounded rollout/migration consequences;
- only downstream API/data/security/operability/delivery decisions or proof obligations that must act now; otherwise state `no new decision required`.

## Success, Escalation, And Stop Conditions

Success means technical design can assign concrete package/file ownership and planning can order implementation without rediscovering service ownership, runtime topology, write truth, completion semantics, failure recovery, projection authority, or rollout constraints.

Stop or escalate when ownership, invariant, pivot, workload, failure/degradation, consistency, provider normalization, migration compatibility, or operational readiness is unresolved; when a public API, data, security, reliability, delivery, or domain owner must make the decisive choice; or when a material design choice lacks evidence. Reject new services without the all-conditions case, sync chains without budgets/retry/idempotency, correctness-bearing async without durable linkage/recovery, projections as truth, indefinite dual writes/shims, and architecture left for coding to discover.
