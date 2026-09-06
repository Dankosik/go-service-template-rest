---
name: go-performance
description: "Measured performance decisions. Use when a workload or budget can change the mechanism, or when an optimization claim needs a comparable baseline, attribution, and delta."
metadata:
  invocation: model
  kind: method
---

# Go Performance

Performance decisions and optimizations use different evidence loops:

`decision: accepted workload -> multiplier or ceiling -> viable mechanisms -> structural acceptance boundary -> planned measurement`

`optimization: accepted budget -> comparable baseline -> attribution -> smallest change -> comparable delta`

A budget without a unit, percentile, and owner is a mood. Before implementation, reject mechanisms whose amplification or ceiling cannot satisfy the accepted envelope. After implementation, claim an improvement only from comparable measurements under that workload.

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
Ground every changed mechanism or claim in accepted workloads, SLOs, execution
paths, measurements, and rollout constraints. For comparing mechanisms,
interacting hot paths, or a decision/review handoff, record
`PerformancePath{workload, budget, unit, percentile_or_capacity, multiplier,
mechanism, baseline, attribution, delta, owner, proof}` per path. A single local
claim can use the existing benchmark result and task artifact; record only the
fields applicable to the selected evidence loop.

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

Complete when every mechanism fits the accepted ceiling and every optimization
claim has comparable workload identity, baseline, attribution, and delta. Use
[PostgreSQL performance](../../../docs/universal-disciplines/postgres-performance/SKILL.md)
for a PostgreSQL-shaped risk or measurement.
