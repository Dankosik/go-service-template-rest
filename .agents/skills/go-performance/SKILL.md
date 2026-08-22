---
name: go-performance
description: "Performance: Use for latency, throughput, allocs, contention, capacity, complexity, scaling, benchmarks. Own policy; Skip correctness."
metadata:
  invocation: model
  kind: method
---

# Go Performance

Performance decisions and optimizations use different evidence loops:

`decision: accepted workload -> multiplier or ceiling -> viable mechanisms -> structural acceptance boundary -> planned measurement`

`optimization: accepted budget -> comparable baseline -> attribution -> smallest change -> comparable delta`

A budget without a unit, percentile, and owner is a mood. Before implementation, reject mechanisms whose amplification or ceiling cannot satisfy the accepted envelope. After implementation, claim an improvement only from comparable measurements under that workload.

Load the [shared specialist contract](../../contracts/specialist-contract.md). Reconstruct
budgets and hot paths from accepted workloads, SLOs, execution paths,
measurements, and rollout constraints. Bind every budget to a unit, percentile
or capacity measure, protocol, and owner.

[Benchmarking](../../../docs/benchmarking.md) owns proof level, workload
identity, comparable evidence, and completion policy. Load one matching leaf
for capture. Read that owner for measurement; the references below cover
decisions.

## Choose The Branch

- **Decision** — load [amplification and scaling](references/amplification-and-scaling.md)
  for a scale-sensitive mechanism and [runtime limits](references/decision/runtime-limits-and-capacity.md)
  for memory, GC, `GOMAXPROCS`, pools, or admission. Cover the workload,
  multiplier or ceiling, mechanism, dominant complexity, planned proof, and
  reopen condition.
- **Review** — load the [review selector](references/review/index.md) for the
  measured risk and place every affected hot path in the finding envelope.

Hand concurrency correctness to `go-concurrency`, data access to `go-db-cache`,
overload to `go-reliability`, and a PostgreSQL-shaped risk or measurement to
[PostgreSQL performance](../../../docs/universal-disciplines/postgres-performance/SKILL.md).
