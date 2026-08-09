---
name: go-performance
description: "Performance: Use for latency, throughput, allocation, contention, capacity, workload amplification/budgets, hot paths, or benchmarks. Own measurable policy; Skip correctness, reliability, or DB/cache policy."
---

# Go Performance

Performance decisions and optimizations use different evidence loops:

`decision: accepted workload -> multiplier or ceiling -> viable mechanisms -> structural acceptance boundary -> planned measurement`

`optimization: accepted budget -> comparable baseline -> attribution -> smallest change -> comparable delta`

A budget without a unit, percentile, and owner is a mood. Before implementation, reject mechanisms whose amplification or ceiling cannot satisfy the accepted envelope. After implementation, claim an improvement only from comparable measurements under that workload.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected budgets and hot paths from accepted workloads and SLOs, changed execution paths, current measurements, and rollout constraints; bind every budget to a unit, percentile or capacity measure, protocol, and owner before optimizing.

[Benchmarking](../../../docs/benchmarking.md) owns proof level, workload definition, capture, comparison, PGO lifecycle, remote execution, and completion policy. Read it for measurement; the references below cover decisions.

## Choose The Branch

- **Decision** — select when performance policy is absent or changing. Load [amplification and scaling](references/amplification-and-scaling.md) before closing a scale-sensitive mechanism. Complete when shared Decision dispositions cover each workload and budget or structural constraint, multiplier or ceiling, measure and owner, selected mechanism and decision-changing rejected alternative, complexity consequence, planned proof, and reopen condition. Load [runtime limits and capacity](references/decision/runtime-limits-and-capacity.md) when the decision is about the process's own envelope — container memory, GC, `GOMAXPROCS`, pool size, or admission concurrency — because this repository already sets or deliberately declines several of those knobs.
- **Review** — select when changed Go must conform to accepted performance policy. Load the [review selector](references/review/index.md) for the measured risk. Measure every affected hot path into the shared finding envelope.

Hand concurrency correctness to `go-concurrency`, DB/cache correctness to `go-db-cache`, and overload policy to `go-reliability`. Load [`postgres-performance`](../../../docs/universal-disciplines/postgres-performance/SKILL.md) when the measurement points at the database: it forces the bottleneck to be attributed from a baseline before any intervention, and accepts evidence placing the constraint outside PostgreSQL as a result, instead of tuning settings toward a suspected cause.
