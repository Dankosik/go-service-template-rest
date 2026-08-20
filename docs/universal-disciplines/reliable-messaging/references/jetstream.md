# NATS JetStream

Load for JetStream publish acknowledgement, consumer acknowledgement/redelivery,
retention, ordering, or replay; distinguish Core NATS and pin versions.

Use JetStream publish acknowledgement when durable stream acceptance is part of
success. Stable `Nats-Msg-Id` deduplicates only within the stream's configured
window; lost acknowledgement remains ambiguous and application business
idempotency survives beyond that window.

Durable pull consumers with explicit acknowledgement fit scaled workers.
Acknowledge only after durable effect. Synchronous acknowledgement narrows
lost-ack redelivery but does not replace effect identity. `AckWait`/`BackOff`,
`MaxDeliver`, delayed NAK, batch size, and `MaxAckPending` define ownership and
pressure. In-progress acknowledgement extends only useful live work. Reaching
`MaxDeliver` needs an explicit quarantine/recovery path; an advisory alone is
not a DLQ.

Retention policy changes semantics: Limits retains until limits evict;
WorkQueue removes on acknowledgement and restricts overlapping consumers;
Interest can remove when no consumer exists. All remain subject to limits and
can evict unconsumed work. Shared pull consumers do not preserve effect order;
strict per-key order needs deterministic subject partition or one in-flight lane
plus version/fencing.

Replay uses a separate durable consumer and explicit start, preserves identity,
and remains bounded against live traffic. Observe publish ack, stream limit
headroom/quorum, consumer pending/ack floor/redelivery/oldest age, MaxDeliver,
worker saturation, and reconciliation mismatch.
