# Kafka

Load for Kafka-specific acceptance, offsets/groups, partition order, retention,
transactions, or replay; verify broker/client versions and settings.

A produce response proves broker acceptance under the chosen acknowledgement
policy, not consumer effect. Durable messages need intentional replication,
`acks=all`, and `min.insync.replicas`. Producer idempotence suppresses protocol
retry duplicates and preserves partition order only inside its contract; it
does not recover an application identity lost or recreated later. A lost
produce response is ambiguous, so retry the same outbox/message identity.
Kafka transactions can fence producers and atomically include Kafka records and
offsets, but cannot join an external database transaction.

Committed offset is recovery position. Disable automatic commit when a durable
external effect is owned by processing, and commit only after all preceding
effects in that partition are durable. Parallel handlers need a per-partition
completion frontier. Rebalance stops new work for revoked partitions and commits
only completed frontiers; failed commit remains ambiguous. External effects stay
idempotent because crash after effect and before offset causes redelivery.

Ordering is per partition under a stable business key. Retention can remove
needed replay regardless of consumer progress; compaction is latest-per-key,
not immutable history. Replay uses a separate group or reviewed offset reset,
preserves original identity, verifies the range is retained, and rate-limits
against live work.

Observe publish ambiguity/latency, replication health, lag/oldest retained
position, rebalance and poll deadline, completion frontier, and inbox conflicts.
