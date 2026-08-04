# NATS JetStream delivery decisions

Use this reference only for JetStream choices. Distinguish Core NATS publication from JetStream publication and pin the server and client versions.

## Publish acceptance and ambiguity

- Use a JetStream publish API and require its publish acknowledgement when durable storage is part of success. A Core NATS publish to a captured subject has no equivalent application acknowledgement.
- A successful JetStream publish acknowledgement means the stream stored the message according to its storage and replication configuration. It does not cover a consumer effect.
- Attach one stable `Nats-Msg-Id` across publish retries. JetStream deduplicates that ID only inside the stream's configured `DuplicateWindow`; an application restart or replay beyond the window still requires durable business idempotency.
- Loss of the publish acknowledgement is ambiguous. Retry the same logical message and `Nats-Msg-Id`, retain the outbox record until acceptance is known, and reconcile by business identity when the deduplication window may have expired.

## Consumer acknowledgement and worker pools

- Prefer a durable pull consumer with `AckExplicit` for horizontally scaled workers. Consumer state then survives client failure and pull size supplies application-controlled flow.
- Acknowledge only after the durable effect commits. `AckSync` asks the server to confirm receipt of the acknowledgement and removes lost-ack redelivery inside that consumer boundary; the business effect still needs idempotency for concurrent work, prior attempts, and replay.
- `AckWait` controls timeout redelivery unless `BackOff` is configured; `BackOff` then overrides it. `MaxDeliver` bounds attempts. A plain negative acknowledgement redelivers immediately unless the client uses delayed NAK behavior.
- Bound fetched and unacknowledged work with pull batch limits and `MaxAckPending`. An in-progress acknowledgement extends processing ownership; send it only while useful work is alive.
- When `MaxDeliver` is reached, the message remains in the stream and an advisory is emitted. Build an explicit quarantine/recovery path; `MaxDeliver` alone is not a DLQ.

## Retention, ordering, and replay

- `LimitsPolicy` retains replayable messages until age/count/byte limits evict them. `WorkQueuePolicy` removes a message after its consumer acknowledges it and disallows overlapping consumer filters. `InterestPolicy` can delete messages immediately when no matching consumer exists.
- Limits and discard policy still apply to work-queue and interest streams; they can evict unconsumed work. Size retention from outage and recovery needs and alert before the limit becomes a loss boundary.
- A shared pull consumer distributes work without deterministic worker affinity. Concurrent workers, redelivery, and variable processing time can reorder business effects.
- For strict per-key order, use subject-based deterministic partitioning or one in-flight message for the ordered lane, then guard state with version/fencing checks.
- Replay with a separate durable consumer and explicit start sequence/time or delivery policy. Preserve the live consumer, original message identity, rate bounds, and reconciliation evidence.
- `ReplayInstant` runs as fast as the consumer and acknowledgement limits allow; `ReplayOriginal` reproduces original timing. Neither is a safety policy by itself.

## Storage, signals, and security

- Use file storage and an intentional replica count for durable work. Stream and consumer state use separate Raft groups; publish availability requires the relevant stream quorum.
- Observe publish acknowledgement latency/errors, stream bytes/messages and limit headroom, consumer pending and acknowledgement-pending counts, acknowledgement floor, redeliveries, MaxDeliver advisories, oldest age, worker saturation, and reconciliation mismatches.
- Use NATS accounts for tenant namespace isolation where appropriate, distinct publish/subscribe/administration subject permissions, TLS for transport, and JetStream encryption at rest when classification requires it. Treat filter-subject API permissions carefully because multi-filter consumer creation can have broader authorization scope.

## Primary sources

- [NATS JetStream concepts and acknowledgement boundary](https://docs.nats.io/nats-concepts/jetstream)
- [JetStream streams, retention, and deduplication](https://docs.nats.io/nats-concepts/jetstream/streams)
- [JetStream consumers, acknowledgement, redelivery, and replay](https://docs.nats.io/nats-concepts/jetstream/consumers)
- [JetStream message deduplication and double acknowledgements](https://docs.nats.io/using-nats/developer/develop_jetstream/model_deep_dive)
- [NATS subject partitioning and ordering](https://docs.nats.io/nats-concepts/subject_mapping)
- [NATS security and account isolation](https://docs.nats.io/nats-concepts/security)

Sources checked 2026-08-02. Recheck target-version fields and defaults before implementation.
