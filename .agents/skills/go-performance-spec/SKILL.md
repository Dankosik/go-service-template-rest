---
name: go-performance-spec
description: "Use when latency, throughput, allocation, contention, memory, or capacity needs an explicit workload model, budget, and benchmark/profile/trace proof before coding; Own measurable performance contracts; Skip when the task is optimization implementation, resilience policy, system topology, or generic speed advice."
---

# Go Performance Spec

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Outcome And Boundary

Produce a reproducible contract for one operation: critical path, workload, latency/throughput/resource budgets, bottleneck hypothesis, proof, regression threshold, runtime signal, and rollout action.

Own measurable workload, latency, throughput, allocation/GC, memory, contention, and capacity policy. Do not choose topology, DB/cache mechanism, API or resilience semantics, synchronization, general observability/delivery, or placement; carry forced budgets only and block on unset upstream policy.

## Method

1. Name operation/outcome, critical path, workload, baseline, and bottleneck evidence; set only relevant budgets and label unsupported numbers with their authority or assumption.
2. Prefer the least-complex option that can meet the budget.
3. Specify controlled benchmark/load/profile/trace proof with baseline, target, samples, variance, regression threshold, and oracle.
4. Add proportional bounded telemetry and rollout/rollback checkpoints; hand semantic changes to their owners.

## Decision Rules

- Model representative mix, input size/cardinality/skew, concurrency, rate/burst, tenants, cache/dependency/retry modes, and worst accepted envelope. Keep decisive buckets separate; reject averages, median-only aggregates, and toy fixtures.
- Prefer algorithmic, data-flow, payload, and round-trip reductions before micro-optimization, pooling, caching, PGO, or added fan-out.
- Budget CPU, allocation/live heap, GC/scheduler, locks/pools/queues/goroutines, and downstream capacity when relevant; bound parallelism and expose cancellation, contention, saturation, and shutdown consequences.
- Match proof to claim: microbenchmarks isolate code, scenarios compare operations, load tests prove envelopes, profiles locate cost, and traces expose scheduling/contention. Reject one best run, hidden variance, mismatched fixtures, and aggregate wins that regress a protected bucket.
- State DB/cache, overload/degradation, streaming, or API consequences, but leave semantics and mechanisms to their owners.
- Preserve mixed-version and rollback safety. Do not accept a gain without a detection path and rollback trigger.

## Reference Selector

Load the reference whose symptom sharpens the highest-risk performance decision.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| “Make it faster” has no budgeted hot path. | [budget-modeling-and-hot-path-maps.md](references/budget-modeling-and-hot-path-maps.md) | Choose operation budgets and a bottleneck hypothesis. |
| Traffic mix, skew, cache, dependency, or fixture shape matters. | [workload-profile-and-input-shape.md](references/workload-profile-and-input-shape.md) | Define representative workload buckets. |
| The sufficient proof type is unclear. | [measurement-protocols.md](references/measurement-protocols.md) | Select a symptom-matched protocol and variance rule. |
| Go commands or artifact capture must be executable. | [benchmark-profile-and-trace-plans.md](references/benchmark-profile-and-trace-plans.md) | Write concrete benchmark/profile/trace obligations. |
| An optimization class appears under-justified. | [option-selection-and-complexity-bounds.md](references/option-selection-and-complexity-bounds.md) | Select the least-complex viable option. |
| Fan-out, queues, workers, locks, or capacity changes. | [concurrency-contention-and-capacity.md](references/concurrency-contention-and-capacity.md) | Bound concurrency and saturation proof. |
| DB, cache, pagination, retry, or API semantics are affected. | [db-cache-api-performance-contracts.md](references/db-cache-api-performance-contracts.md) | Expose semantic handoffs and fallback budgets. |
| Canary, telemetry, rollback, or production proof is needed. | [runtime-telemetry-and-rollout-checkpoints.md](references/runtime-telemetry-and-rollout-checkpoints.md) | Tie signals to release actions. |
| Allocation, live heap, GC CPU, or memory limit dominates. | [memory-allocation-and-gc-budgets.md](references/memory-allocation-and-gc-budgets.md) | Specify memory and GC envelopes. |
| PGO/profile lifecycle is proposed. | [pgo-profile-lifecycle.md](references/pgo-profile-lifecycle.md) | Require representative profile provenance and rollback. |
| Overload, shedding, queueing, retries, or tenant fairness matters. | [overload-backpressure-and-load-shedding.md](references/overload-backpressure-and-load-shedding.md) | Define capacity-protection semantics. |
| SLI/SLO, histogram, or percentile thresholds matter. | [latency-sli-slo-and-histogram-thresholds.md](references/latency-sli-slo-and-histogram-thresholds.md) | Set percentile/window-aware acceptance. |
| Payload, JSON, streaming, flushing, or body size dominates. | [payload-serialization-and-streaming-budgets.md](references/payload-serialization-and-streaming-budgets.md) | Bound representation and streaming behavior. |

## Output

Return critical path, workload/budgets, baseline/hypothesis, option, proof and thresholds, runtime/rollout actions, forced constraints, assumptions, blockers, and reopen conditions.

## Success And Stop

Stop when critical path/workload or baseline is unknown, targets lack authority, proof/variance is irreproducible, regression thresholds are absent, or the budget needs unresolved neighboring policy.
