---
name: go-system-architecture-spec
description: "Use when a feature, refactor, or extraction needs service or component boundaries, topology, source-of-truth, sync/async flow, consistency, failure, migration, or operability decisions before coding; Own target-state system architecture; Skip when the primary decision is endpoint detail, physical data design, Go package placement, or post-code review."
---

# Go System Architecture Spec

Load the [shared specialist contract](../specialist-contract.md), then apply this system-decision boundary.

## Outcome And Boundary

Choose target-state service/component boundaries, runtime topology, invariant and write authority, sync/async flow, workload and critical path, consistency and failure/degradation model, migration, and operability. Do not decide endpoint payloads/statuses, physical schema/DDL, Go package/file placement, low-level tuning, CI/container details, or implementation code.

Inspect the approved problem, current boundaries, invariant and write authority, measured workload and critical path, team/runtime ownership, dependencies, read topology, failure/degradation expectations, and mixed-version rollout constraints. Ground unfamiliar external behavior in current official sources and credible implementations; label invented latency, scale, growth, freshness, RPO/RTO, or cost as assumptions. Use the canonical [research solution-discovery method](../../../docs/spec-first-workflow/phases/research.md#method) when evidence could change a material architecture choice.

## Architecture Core

1. Put one explicit source of truth behind each invariant-bearing entity or process. Reject service-per-table, shared-schema writes, direct cross-service DB reads, and ambiguous process ownership.
2. Default to a modular monolith. Extract only when domain/data/team/transaction ownership, independent deployment, runtime isolation or scaling, operational readiness, and accepted consistency cost justify it; use a separate runtime when isolation alone is enough.
3. Let request/response, long-running, fan-out, stream, reconciliation, operator work, hot keys/tenants, backlog, payload, and read/write pressure shape topology before selecting brokers, caches, projections, or splits.
4. Keep synchronous critical paths within finality and deadline budgets. Move variable, fan-out, or deferrable work behind an honest job/operation boundary with named retry/idempotency classes.
5. Give async work one process owner. Distinguish command/event, queue/pub-sub, orchestration/choreography, and state machine/workflow engine; require durable linkage/state, idempotent handling, bounded retry, poison/DLQ ownership, and reconciliation for correctness-bearing flows.
6. Keep hard invariants local when possible. Name the irreversible pivot and local ACID boundary; define compensation before it and retry/forward recovery after it. Reject unscoped exactly-once and distributed-lock shortcuts.
7. Keep replicas, caches, search, exports, aggregators, and CQRS projections derived, with freshness, rebuild, bypass, and correction rules; read scale must not move write truth accidentally.
8. Bound target-state evolution. Use expand/migrate/verify/contract, canary/shadow/dual-read, or temporary bridges only when live constraints require them, with authority, reconciliation, exit/removal proof, rollback limits, and owner.

Count observability, on-call, graceful lifecycle, release coordination, repair tooling, and technology ownership as architecture cost; reject fashion or availability as sufficient selection evidence.

## Symptom-Driven References

State which architecture decision the selected reference can change.

| Pressure | Load | Required effect |
| --- | --- | --- |
| Boundary/write ownership, shared data, team seams, or package layout is confused with service architecture | [boundary-decomposition-examples.md](references/boundary-decomposition-examples.md) | Choose invariant and authority boundaries, leaving package placement downstream. |
| Modular monolith, separate runtime, or service extraction | [modular-monolith-vs-service-extraction.md](references/modular-monolith-vs-service-extraction.md) | Apply the complete extraction test. |
| Request path/queue, saga, orchestration/choreography, or workflow engine | [sync-async-workflow-ownership.md](references/sync-async-workflow-ownership.md) | Name process owner, pivot, durable state, and completion model before tools. |
| CQRS, replicas, read service, projection, search, dashboard, export, or aggregator | [read-write-topology-and-projections.md](references/read-write-topology-and-projections.md) | Preserve write authority and define freshness, bypass, rebuild, and correction. |
| Provider/webhook vocabulary or ambiguous partner result affects lifecycle | [external-provider-anti-corruption.md](references/external-provider-anti-corruption.md) | Normalize provider evidence behind a local owner. |
| Authority move, extraction, mixed versions, canary, shadow, bridge, or rollback boundary | [rollout-and-migration-patterns.md](references/rollout-and-migration-patterns.md) | Select target authority first and bound transition machinery. |
| Microservice, distributed-monolith, shared-DB, direct-read, dual-write, retry-storm, fallback, or permanent-shim smell | [architecture-anti-patterns.md](references/architecture-anti-patterns.md) | Convert the smell into a failure consequence, blocker, risk, or reopen condition. |

## Return And Stop

Return context/non-goals; boundary, invariant/write/process authority, and dependency direction; workload, critical path, runtime and sync/async topology; command/query and derived-read rules; pivot, consistency, idempotency, recovery, degradation, lifecycle, and operability; triggered migration/rollback constraints; measurable proof; assumptions; and reopen criteria. Leave endpoint, schema, security, delivery, and Go-placement consequences with their owners.

Stop when domain/invariant authority, workload, pivot, consistency, failure/degradation, provider normalization, migration compatibility, or operational readiness is unresolved, or another policy owner must decide first. Reject unjustified services, unbudgeted sync chains, correctness-bearing async without durable recovery, projections as truth, indefinite dual writes/shims, and architecture left for coding to discover.
