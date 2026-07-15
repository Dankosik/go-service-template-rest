# Reference Selector

Load the reference whose symptom sharpens the highest-risk performance decision.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| “Make it faster” has no budgeted hot path. | [budget-modeling-and-hot-path-maps.md](budget-modeling-and-hot-path-maps.md) | Choose operation budgets and a bottleneck hypothesis. |
| Traffic mix, skew, cache, dependency, or fixture shape matters. | [workload-profile-and-input-shape.md](workload-profile-and-input-shape.md) | Define representative workload buckets. |
| The sufficient proof type is unclear. | [measurement-protocols.md](measurement-protocols.md) | Select a symptom-matched protocol and variance rule. |
| Go commands or artifact capture must be executable. | [benchmark-profile-and-trace-plans.md](benchmark-profile-and-trace-plans.md) | Write concrete benchmark/profile/trace obligations. |
| An optimization class appears under-justified. | [option-selection-and-complexity-bounds.md](option-selection-and-complexity-bounds.md) | Select the least-complex viable option. |
| Fan-out, queues, workers, locks, or capacity changes. | [concurrency-contention-and-capacity.md](concurrency-contention-and-capacity.md) | Bound concurrency and saturation proof. |
| DB, cache, pagination, retry, or API semantics are affected. | [db-cache-api-performance-contracts.md](db-cache-api-performance-contracts.md) | Expose semantic handoffs and fallback budgets. |
| Canary, telemetry, rollback, or production proof is needed. | [runtime-telemetry-and-rollout-checkpoints.md](runtime-telemetry-and-rollout-checkpoints.md) | Tie signals to release actions. |
| Allocation, live heap, GC CPU, or memory limit dominates. | [memory-allocation-and-gc-budgets.md](memory-allocation-and-gc-budgets.md) | Specify memory and GC envelopes. |
| PGO/profile lifecycle is proposed. | [pgo-profile-lifecycle.md](pgo-profile-lifecycle.md) | Require representative profile provenance and rollback. |
| Overload, shedding, queueing, retries, or tenant fairness matters. | [overload-backpressure-and-load-shedding.md](overload-backpressure-and-load-shedding.md) | Define capacity-protection semantics. |
| SLI/SLO, histogram, or percentile thresholds matter. | [latency-sli-slo-and-histogram-thresholds.md](latency-sli-slo-and-histogram-thresholds.md) | Set percentile/window-aware acceptance. |
| Payload, JSON, streaming, flushing, or body size dominates. | [payload-serialization-and-streaming-budgets.md](payload-serialization-and-streaming-budgets.md) | Bound representation and streaming behavior. |
