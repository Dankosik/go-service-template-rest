# Data and Coordination

## Specify semantics per operation

Avoid describing an entire system as simply “strong,” “eventual,” “CP,” or “AP.” For each critical operation record:

| Question | Required decision |
| --- | --- |
| Authority | Which component and record decide truth? |
| Identity | What makes two requests, entities, or effects the same? |
| Commit | At which durable point may success be reported? |
| Read guarantee | Latest, monotonic, read-your-writes, causal, bounded-stale, or best effort? |
| Write guarantee | Single-key atomicity, multi-record transaction, compare-and-set, or compensatable workflow? |
| Partition behavior | Reject, serve stale, accept conflicting work, or restrict operations? |
| Conflict | Prevent, detect, order, merge, compensate, or reconcile? |
| Recovery | How are missed, duplicate, divergent, or partially applied effects repaired? |

During a network partition, name which operations remain available and what correctness they surrender. During normal operation, separately account for the latency and coordination cost of the selected guarantee.

## Keep one authority

Prefer one authoritative representation for mutable facts. Replicas, caches, indexes, search documents, analytics tables, and read models are derived copies with explicit freshness, rebuild, and reconciliation contracts.

When several components appear to own the same fact, either move the invariant to one transaction boundary or define a protocol that makes the shared decision. Eventual propagation does not enforce a cross-component invariant at decision time.

## Coordination ladder

Climb only as high as the invariant requires:

1. Partition ownership so independent keys do not coordinate.
2. One authority serializes changes.
3. Conditional writes, versions, or fencing reject stale actors.
4. A per-key leader or store-provided transaction coordinates a bounded set.
5. Consensus maintains a replicated decision or log under the stated fault model.
6. A distributed transaction coordinates atomic commit across participants.

Prefer a managed datastore or coordinator with a documented contract over implementing consensus, membership, clocks, or transaction protocols in application code.

## Replication choices

### Leader and followers

Fits one ordered write path with scalable or local reads. Define leader election and fencing, acknowledged replica count, follower-read freshness, failover data-loss window, read-your-writes routing, replication lag, and recovery of a former leader.

### Multiple writers

Fits disconnected or geographically local writes only when conflicts have domain semantics. Define causality or version tracking, merge rules, invariants that cannot be merged, tombstones and deletion, convergence tests, and the user-visible response to unresolved conflicts.

### Quorum or leaderless access

State the exact implementation guarantees; `R + W > N` alone does not prove linearizability under sloppy quorums, concurrent writes, clock uncertainty, repair lag, or failed nodes. Define version comparison, conflict retention, read repair, hinted handoff, membership changes, and availability during partitions.

## Ordering and time

Wall clocks do not provide a free total order across nodes. Use operation identity and domain order first. Where order matters, define whether it is per entity, partition, session, causality chain, or global. Use sequence numbers, versions, logical clocks, generation numbers, leases, or fencing tokens with a named authority. Account for delayed and duplicated work, clock drift, lease expiry, and stale leaders.

## Cross-component state changes

### Transactional outbox

Use to atomically commit local state and the intent to publish. It does not make broker delivery, consumer processing, or the business effect exactly once. Define relay ownership, ordering, duplicates, backlog, retention, and replay; hand implementation to `reliable-messaging`.

### Saga or compensating workflow

Use when a business operation spans independent local commits and can expose intermediate states. Define durable process state, orchestration or choreography owner, idempotent steps, timeouts, retry and terminal states, compensations, irreversible effects, forward recovery, operator intervention, and reconciliation. A compensation is a new business action, not a rollback of history.

### Two-phase commit

Use only when atomicity across participants is worth coordinator dependence, lock duration, blocking or recovery complexity, and every participant supports the required protocol. Define coordinator durability, prepared-state recovery, timeouts, heuristic outcomes, and operational tooling.

### CQRS or materialized view

Use when read and write forces differ enough to pay for another model. Define authority, propagation, freshness, version compatibility, partial update behavior, rebuild source, replay bounds, query behavior during lag, and storage and write amplification.

## Partitioning

Choose a key from access and ownership invariants, not only current cardinality. Account for skew, hot keys, tenant growth, cross-partition transactions and queries, secondary indexes, global constraints, routing metadata, resharding, and migration while requests continue. Keep the partitioning function versioned or mediated by a routing authority when placement changes.

## Proof cases

For every material guarantee, test the counterexample that could falsify it:

- concurrent writes to the same invariant;
- duplicate and delayed operations;
- reordering across partitions or sessions;
- lost response after commit;
- stale replica after failover;
- network partition and rejoin;
- leader or lease-holder pause and return;
- partial workflow completion;
- replay from an older version;
- resharding while reads and writes continue.

## Primary sources

- [Designing Data-Intensive Applications, 2nd Edition](https://www.oreilly.com/library/view/designing-data-intensive-applications/9781098119058/)
- [Patterns of Distributed Systems](https://martinfowler.com/articles/patterns-of-distributed-systems/)
- [Dynamo: Amazon's Highly Available Key-value Store](https://www.amazon.science/publications/dynamo-amazons-highly-available-key-value-store)
- [Spanner: Google's Globally-Distributed Database](https://research.google/pubs/spanner-googles-globally-distributed-database-2/)
- [Raft consensus resources and paper](https://raft.github.io/)
- [Azure Architecture Center: Saga pattern](https://learn.microsoft.com/en-us/azure/architecture/reference-architectures/saga/saga)
