---
name: go-performance
description: "Performance: Use when latency, throughput, allocation, contention, memory, capacity, or workload budgets need a decision, or when changed hot paths and benchmark/profile evidence need review. Own measurable performance policy and conformance; Skip when correctness, reliability, or DB/cache policy is primary."
---

# Go Performance

Load the [shared specialist contract](../specialist-contract.md). Keep workload shape, budgets, hot-path costs, capacity, measurement, optimization complexity, and rollout evidence coherent.

## Choose The Branch

- **Decision** — select when performance policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when workload, budget, option consequence, benchmark/profile proof, and blockers are explicit.
- **Review** — select when changed Go must conform to accepted performance policy. Load the [review selector](references/review/index.md) for the measured risk. Complete when every affected hot path and proof claim is dispositioned as a finding or no finding with the smallest correction and focused measurement; missing policy stays in the decision branch.

Hand concurrency correctness to `go-concurrency`, DB/cache correctness to `go-db-cache`, and overload policy to `go-reliability`.
