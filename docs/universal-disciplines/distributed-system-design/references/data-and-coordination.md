# Data And Coordination

Load when state crosses a process, replica, partition, store, or region, or when
ordering and distributed commit can change the design.

For each critical operation name authority, logical identity, durable success
point, read guarantee, write atomicity, partition behavior, conflict
disposition, and repair. Avoid system-wide labels such as simply strong,
eventual, CP, or AP. Keep one authority for each mutable fact; caches, indexes,
analytics, replicas, and read models remain derived with explicit freshness,
rebuild, and reconciliation.

Climb only as high as the invariant requires: partition independent keys; one
authority serializes; conditional versions/fencing reject stale actors; a
per-key leader or store transaction coordinates a bounded set; consensus
replicates a decision under a fault model; distributed transaction coordinates
atomic commit across participants. Prefer a documented managed primitive to
application-built consensus, clocks, or membership.

Define order by entity, partition, session, causality chain, or global scope.
Wall clocks do not give a free total order; use sequence/version/generation or
logical-clock authority. For leader/follower replication name acknowledgement,
follower freshness, failover loss, read-your-writes, fencing, and former-leader
recovery. Multiple writers require domain merge semantics, tombstones,
unmergeable invariants, and a visible conflict path.

Outbox commits local state plus publish intent but does not make downstream
effects exactly once. A saga exposes intermediate states and needs durable
process state, idempotent steps, timeouts, terminal states, compensation or
forward repair, and reconciliation. CQRS/materialization pays another model's
freshness, rebuild, compatibility, and amplification cost.

Falsify with concurrent writes, duplicates, reorder, lost response after
commit, stale failover reads, partition/rejoin, stale leader, partial workflow,
old-version replay, and live resharding as applicable.
