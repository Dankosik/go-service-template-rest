# Reference Selector

State which architecture decision the selected reference can change.

| Pressure | Load | Required effect |
| --- | --- | --- |
| Current package/component boundary, dependency direction, generated/manual owner, or composition root can constrain the target | [Component Boundaries](../../../../docs/architecture/boundaries.md) | Preserve or explicitly replace the current owner and dependency direction instead of inventing a parallel path. |
| A system neighbor, provider contract, or cross-service evidence path changes | [Integration Boundaries](../../../../docs/architecture/integration.md) | Bind the crossing to its real contract, runtime evidence, trust, and accountable owner. |
| New service-to-service call, internal API, gRPC vs REST/OpenAPI, consumer-class change, or protocol migration | [inter-service-protocol-selection.md](inter-service-protocol-selection.md) | Classify the consumers, default a new strictly-internal synchronous contract to native gRPC, and decide its Railway exposure and per-neighbor correlation policy. |

Every other architecture pressure has a nearer owner; route to it rather than deciding here.

| Pressure | Owner |
| --- | --- |
| Whether a boundary, copy, queue, cell, or region is earned; capacity before topology; the cost a boundary introduces | [capacity-and-topology.md](../../../../docs/universal-disciplines/distributed-system-design/references/capacity-and-topology.md) |
| Authority, consistency and freshness per operation, outbox, saga, CQRS, partitioning, ordering | [data-and-coordination.md](../../../../docs/universal-disciplines/distributed-system-design/references/data-and-coordination.md) |
| Deadlines, retry budgets, overload, degradation, failover capacity, retry storms, fallback under load | [resilience-and-load.md](../../../../docs/universal-disciplines/distributed-system-design/references/resilience-and-load.md), then `go-reliability` |
| Mixed-version coexistence, backfill, cutover, the rollback boundary, temporary writers and their removal condition | [evolution-and-multi-region.md](../../../../docs/universal-disciplines/distributed-system-design/references/evolution-and-multi-region.md); this repository's release window is owned by `go-delivery-platform` and the DDL sequence by [postgres-schema-design](../../../../docs/universal-disciplines/postgres-schema-design/SKILL.md) |
| External provider, partner webhook, ambiguous third-party outcome, provider vocabulary | [external-api-integration](../../../../docs/universal-disciplines/external-api-integration/SKILL.md) |
| Which fact is authoritative, projections, exports, search surfaces, retention | `go-data-architecture` |
| Compensation, pivot, replay, redrive, reconciliation across owners | `go-distributed` |
| Go package, file, and dependency placement; enforcing a decided boundary in code | `go-implementation-ownership` |
