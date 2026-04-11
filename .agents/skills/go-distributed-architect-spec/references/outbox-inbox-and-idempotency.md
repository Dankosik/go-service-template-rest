# Outbox, Inbox, And Idempotency

## When To Load
Load this when a spec needs to avoid dual writes, define outbox relay behavior, choose consumer dedup or inbox semantics, set ACK or offset-commit order, or define idempotency keys and retention.

## Option Comparisons
- Dual write: reject as the correctness mechanism because database commit and broker publish can diverge.
- Transactional outbox: write business state and message intent in one owner transaction, then publish asynchronously. Choose this for reliable producer-side emission.
- Dapr outbox: use when the service already relies on Dapr state transactions and pub/sub; model the internal intent, transaction marker, verification, and at-least-once publish behavior.
- Inline idempotent consumer: handle the message inside one transaction that also records a durable processed-message marker.
- Inbox pattern: store incoming messages durably, ACK reception, and process them separately. Choose when batching, custom retry, or independent scaling matters.
- Natural idempotency: acceptable only when the domain operation is truly equivalent on repeat and protected by durable uniqueness or version checks.

## Good Flow Examples
- Producer transaction writes `order.status=PENDING` and `outbox(OrderCreated, message_id, aggregate_id, version)`. Relay publishes later and may publish twice.
- Consumer transaction inserts `(subscriber_id, message_id)` into a dedup table, applies the business side effect, writes any outgoing outbox messages, commits, then ACKs.
- Inbox receiver stores `message_id`, `type`, `payload`, and `received_at` with `ON CONFLICT DO NOTHING`; a processor claims unprocessed rows and marks terminal outcomes.
- Idempotency key policy uses CloudEvents `source + id`; for non-CloudEvents, use `producer_service + message_id` or a domain operation key.

## Bad Flow Examples
- Publish event to broker, then commit database state.
- Commit state, then publish directly to broker without a durable outbox or equivalent.
- ACK the broker before durable side effects and dedup markers are committed.
- Keep processed message IDs only in process memory.
- Treat an outbox as eliminating the need for consumer idempotency.

## Failure-Mode Examples
- Relay publishes then crashes before marking delivered: consumers see a duplicate and must dedup or handle idempotently.
- Consumer commits side effect but crashes before ACK: broker redelivers; dedup marker prevents repeat side effects.
- Dedup retention expires too early: redrive or replay can reapply an old side effect; retention must cover plausible replay windows.
- Inbox table grows forever: the spec defines retention, partitioning, or cleanup after the redrive risk window.
- Framework claims "exactly once": interpret it as a scoped infrastructure behavior and still document broker redelivery, handler idempotency, and storage boundaries.

## Exa Source Links
- [Dapr transactional outbox docs](https://docs.dapr.io/developing-applications/building-blocks/state-management/howto-outbox/)
- [Microservices.io Transactional Outbox pattern](https://microservices.io/patterns/data/transactional-outbox.html)
- [Microservices.io Idempotent Consumer pattern](https://microservices.io/patterns/communication-style/idempotent-consumer.html)
- [Eventuate Tram getting started](https://eventuate.io/docs/manual/eventuate-tram/latest/getting-started-eventuate-tram-spring-boot.html)
- [MassTransit transactional outbox configuration](https://masstransit.io/advanced/transactional-outbox.html)
- [MassTransit outbox pattern docs](https://masstransit-project.com/documentation/patterns/transactional-outbox)
- [NServiceBus Outbox docs](https://docs.particular.net/nservicebus/outbox/?version=core_10)
