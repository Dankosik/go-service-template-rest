# Capacity And Topology

Load when capacity, decomposition, partitioning, cells, or topology can change
the mechanism.

Keep arrival rate, service time, payload, fan-out, retention, copy count, retry
amplification, and recovery backlog independent until evidence relates them.
Useful approximations are:

- concurrency = arrival rate x service time;
- network = rate x payload x fan-out;
- storage = writes x retained bytes x retention x copies;
- recovery demand = new work + backlog/recovery window.

Include protocol overhead, indexes/derived copies, replication, compaction,
skew, and headroom for failure, deploy, repair, and forecast error. Capacity is
the sustainable SLO under the required failure state, not the point where a
component remains alive. Find the first CPU, memory, I/O, network, connection,
lock, queue, partition, quota, or hot-tenant ceiling.

Stop at the first topology that holds: one process/local disposable state; one
deployable plus authority; stateless replicas; one measured derived
cache/queue/read model; partitions/cells for a proven capacity or blast-radius
ceiling; multiple regions for quantified latency, residency, RPO/RTO, or
regional-loss needs.

A component boundary requires a distinct authority/transaction, scaling or
resource profile, failure/trust/residency boundary, deployment cadence,
geographic placement, or accountable operator. Record its added remote failure,
versioning, auth, observability, deployment, capacity, and on-call cost.

Proof exercises the critical end-to-end flow with representative skew, payload,
concurrency, dependency behavior, failure loss, and recovery catch-up; a
component benchmark alone does not establish system capacity.
