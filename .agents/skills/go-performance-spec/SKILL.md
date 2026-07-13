---
name: go-performance-spec
description: "Design measurable performance contracts for Go services before coding. Use when latency, throughput, allocation, contention, memory, or capacity needs explicit workload budgets and benchmark/profile/trace proof. Skip local optimization implementation, generic speed advice, and unrelated API/schema/CI design."
---

# Go Performance Spec

## Outcome

Produce a reproducible performance contract for the affected operation: workload, budget, bottleneck hypothesis, proof protocol, thresholds, runtime signal, and rollout action.

## Method

1. Name the operation class, user/system outcome, workload shape, hot path, and current evidence.
2. Set explicit latency, throughput, allocation, memory, contention, or capacity budgets only where relevant; label unproven numbers as assumptions.
3. Compare options only for a real live fork and prefer the least-complex option that can meet the budget.
4. Specify a reproducible benchmark, profile, trace, or scenario protocol with baseline, target, variance rule, and pass/fail threshold.
5. Add runtime telemetry and rollout/rollback checkpoints proportional to the claim, then hand off semantic changes to their owning domain.

## Decision Rules

- Measure a named hot path under representative input, concurrency, skew, cache, and dependency state; do not optimize a dashboard average or toy fixture.
- Prefer algorithmic, data-flow, payload, and round-trip reductions before micro-optimization, pooling, caching, PGO, or added fan-out.
- Keep concurrency bounded and specify cancellation, queue, contention, saturation, and shutdown behavior when performance work adds parallelism.
- Match the proof to the claim: microbenchmarks do not establish system latency, profiles do not establish correctness, and one best run does not establish a stable improvement.
- Define DB/cache, overload, degradation, or API-visible consequences explicitly when they change behavior; performance does not own those semantics.
- Preserve mixed-version and rollback safety. Do not accept a gain without a detection path and rollback trigger.
- Use `constraint_only`, `proof_only`, or `no new decision required in <domain>` when an adjacent domain needs no new decision now.

## Reference Selector

Load at most one reference by default; load more only for independent decision pressures.

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

Return the relevant hot-path map, workload and budget table, bottleneck hypothesis, option decision, proof protocol and thresholds, runtime/rollout checkpoints, forced handoffs, assumptions, blockers, and reopen conditions.

## Success And Stop

Success means implementation can pursue a measured budget without inventing workload, proof, or semantic policy. Stop when the critical path or workload is unknown, targets have no authority, proof cannot be reproduced, or meeting the budget would require unresolved API, data/cache, reliability, concurrency, observability, or delivery decisions.
