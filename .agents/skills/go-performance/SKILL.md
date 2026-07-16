---
name: go-performance
description: "Performance: Use when latency, throughput, allocation, contention, memory, capacity, or workload budgets need a decision, or when changed hot paths and benchmark/profile evidence need review. Own measurable performance policy and conformance; Skip when correctness, reliability, or DB/cache policy is primary."
---

# Go Performance

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected budgets and hot paths from accepted workloads and SLOs, changed execution paths, current measurements, and rollout constraints; bind every budget to a unit, percentile or capacity measure, protocol, and owner before optimizing.

## Choose The Branch

- **Decision** — select when performance policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover each budget, workload, measure, owner, complexity consequence, and rollout proof.
- **Review** — select when changed Go must conform to accepted performance policy. Load the [review selector](references/review/index.md) for the measured risk. Measure every affected hot path into the shared finding envelope, naming any outside boundary or proof blocker with the smallest correction. Missing policy ends this run with a named Performance Decision handoff; conformance Review begins separately after acceptance.

Hand concurrency correctness to `go-concurrency`, DB/cache correctness to `go-db-cache`, and overload policy to `go-reliability`.
