---
name: go-performance
description: "Performance: Use for latency, throughput, allocation, contention, capacity, workload amplification/budgets, hot paths, or benchmarks. Own measurable policy; Skip correctness, reliability, or DB/cache policy."
---

# Go Performance

Performance work moves between two **measurements**: a baseline that shows the bottleneck and a comparable after that shows the delta — everything in between is hypothesis.

`accepted budget -> workload model -> baseline -> attribution -> smallest change -> comparable delta`

A budget without a unit, percentile, and owner is a mood; a hot path earns optimization through profile evidence rather than intuition; and amplification — N+1 calls, fan-out, retry storms, allocation inside loops — usually dominates any micro-cost. An optimization that cannot state what it measured before and after, under which workload, did not happen.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct affected budgets and hot paths from accepted workloads and SLOs, changed execution paths, current measurements, and rollout constraints; bind every budget to a unit, percentile or capacity measure, protocol, and owner before optimizing.

[Benchmarking](../../../docs/benchmarking.md) owns proof level, workload definition, capture, comparison, PGO lifecycle, remote execution, and completion policy, and `AGENTS.md` already loads it for any performance claim. Read it for how to measure; the references below cover only what it does not decide.

## Choose The Branch

- **Decision** — select when performance policy is absent or changing. Complete when shared Decision dispositions cover each budget, workload, measure, owner, complexity consequence, and rollout proof. Load [runtime limits and capacity](references/decision/runtime-limits-and-capacity.md) when the decision is about the process's own envelope — container memory, GC, `GOMAXPROCS`, pool size, or admission concurrency — because this repository already sets or deliberately declines several of those knobs.
- **Review** — select when changed Go must conform to accepted performance policy. Load the [review selector](references/review/index.md) for the measured risk. Measure every affected hot path into the shared finding envelope.

Hand concurrency correctness to `go-concurrency`, DB/cache correctness to `go-db-cache`, and overload policy to `go-reliability`. Load [`postgres-performance`](../../../docs/universal-disciplines/postgres-performance/SKILL.md) when the measurement points at the database: it forces the bottleneck to be attributed from a baseline before any intervention, and accepts evidence placing the constraint outside PostgreSQL as a result, instead of tuning settings toward a suspected cause.
