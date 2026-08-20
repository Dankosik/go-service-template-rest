---
name: distributed-system-design
description: "Cross-component topology, state, coordination, resilience, capacity, and evolution depth reference."
---

# Distributed System Design

Use only after the active phase identifies a cross-component decision. Inherit
its authority, artifact, review, proof, output, and completion contract; do not
select design depth, work mode, or production action here.

Core invariant: every component, edge, copy, partition, or control plane is
earned by a named force, and the selected topology survives its required load,
failures, and evolution with one explicit authority per fact.

Load one branch:

- capacity, decomposition, sharding, cells, or topology ->
  [capacity-and-topology.md](references/capacity-and-topology.md);
- state authority, consistency, ordering, or coordination ->
  [data-and-coordination.md](references/data-and-coordination.md);
- dependency failure, overload, retry, failover, or recovery ->
  [resilience-and-load.md](references/resilience-and-load.md);
- migration, mixed versions, region, RPO, or RTO ->
  [evolution-and-multi-region.md](references/evolution-and-multi-region.md).
