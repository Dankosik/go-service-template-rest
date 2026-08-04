# Capacity and Topology

## Calculate before distributing

A whiteboard component becomes credible only when it fits a physical envelope. Use ranges and sensitivity rather than false precision.

| Dimension | Useful approximation | Design question |
| --- | --- | --- |
| Peak rate | `average rate x peak multiplier` | Can the serving path sustain the burst duration? |
| Concurrent work | `arrival rate x service time` | Which pool, memory, connection, or queue saturates first? |
| Network | `rate x payload x fan-out` | Where do serialization, egress, or cross-region links dominate? |
| Storage growth | `writes x retained bytes x retention x copies` | When does the current store or partition reach its limit? |
| Recovery rate | `new work + backlog / recovery window` | Can the system catch up without entering overload again? |
| Fan-out cost | `source operations x destinations` | Is the amplification paid on writes, reads, or both? |

Include protocol and metadata overhead, indexes and derived copies, replication, compaction or garbage collection, retries, and headroom. Show the units at every step. Test the largest assumption at a low and high value; an architecture that changes across that range has an unresolved force.

Capacity is defined at the target SLO and failure mode, not at the point where a component merely stays alive. Reserve headroom for a lost instance or failure domain, deployments, rebalancing, repair, noisy neighbors, and forecast error.

## Find the first ceiling

Check, as applicable:

- CPU time per operation and serial sections;
- memory per in-flight request, working set, and garbage collection;
- storage IOPS, bandwidth, latency, compaction, and write amplification;
- network packets, bytes, connections, cross-zone or cross-region egress;
- database connections, locks, hot rows, and transaction duration;
- queue service rate, worker concurrency, backlog age, and drain rate;
- partition-key skew and the hottest tenant or key;
- downstream quota and control-plane rate limits.

An average below capacity does not clear a design with burst or skew above one partition's ceiling.

## Topology ladder

Stop at the first topology that meets the forces:

1. One process and local state for a bounded, disposable workload.
2. One deployable with one authoritative durable store.
3. Stateless serving replicas around the same authority.
4. A derived read model, cache, queue, or worker for a measured path.
5. Partitions or cells when one capacity or blast-radius unit has a proven ceiling.
6. Multiple regions when latency, residency, RTO/RPO, or regional-loss requirements pay for them.

High availability inside a managed primitive does not by itself require another application service. Reuse already-operated platform guarantees when their contract fits.

## Earn a component boundary

A separate component is justified when at least one force requires a different:

- source of truth or transactional boundary;
- scaling dimension or resource profile;
- failure containment or availability target;
- trust, tenant, residency, or compliance boundary;
- deployment lifecycle or compatibility cadence;
- geographic placement or latency budget;
- operational owner with an enforceable interface.

Record the cost introduced by the boundary: remote latency and partial failure, versioning, authentication, observability, deployment ordering, capacity reservation, and on-call ownership.

## Pattern cards

### Partitioning or sharding

Use when one authority has a measured storage, throughput, locality, or blast-radius ceiling and operations can be assigned to stable partition keys.

Specify key and routing owner, expected distribution and hottest-key limit, cross-partition operations, global uniqueness, resharding and migration, failure isolation, rebalancing capacity, and query consequences. Range, hash, directory, and tenant partitioning resolve different forces; name why the chosen form fits.

### Cell or deployment stamp

Use when cohorts of tenants or traffic need repeatable capacity units and bounded blast radius. Define placement and routing metadata, cell capacity and admission, shared global dependencies, spare capacity, cell creation, tenant movement, fleet-wide deployment, and cross-cell operations. A global router or control plane can become the new common failure domain.

### Materialization or fan-out

Move work to writes when read latency and read fan-out dominate; move work to reads when writes or update amplification dominate. Define freshness, rebuild, hot entities, storage amplification, partial update behavior, and the point at which the chosen side saturates.

### Autoscaling

Use for variation within a topology whose scaling signal, startup time, dependency capacity, and maximum safe concurrency are known. Autoscaling reacts after a signal; it is not overload control and cannot create dependency capacity. Keep admission control and degraded modes for bursts and failure.

## Capacity proof

Validate the critical flow with representative data, skew, concurrency, payloads, and dependency behavior. Measure user-visible latency distribution and success together with saturation, queue age, amplification, and cost. Include the intended failure state and recovery catch-up. A component benchmark does not establish end-to-end capacity unless it exercises the design-driving path.

## Primary sources

- [Google SRE Workbook: Non-Abstract Large System Design](https://sre.google/workbook/non-abstract-design/)
- [Designing Data-Intensive Applications, 2nd Edition](https://www.oreilly.com/library/view/designing-data-intensive-applications/9781098119058/)
- [Azure Architecture Center: Sharding pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/sharding)
- [Azure Architecture Center: Deployment Stamps pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/deployment-stamp)
- [AWS Well-Architected: Reducing the Scope of Impact with Cell-Based Architecture](https://docs.aws.amazon.com/wellarchitected/latest/reducing-scope-of-impact-with-cell-based-architecture/welcome.html)
