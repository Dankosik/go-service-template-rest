# RabbitMQ delivery decisions

Use this reference only for RabbitMQ-specific choices. Confirm queue type and server/client versions before selecting durability or poison-message behavior.

## Publish acceptance and ambiguity

- Reliable publication combines a durable or replicated queue, persistent messages, and publisher confirms. Each protects a different failure boundary.
- A confirm says the broker accepted responsibility according to every routed queue's type: for persistent messages to durable queues this includes persistence; for quorum queues it includes quorum acceptance. It says nothing about a consumer's business effect.
- An unroutable publish can still be positively confirmed. Use the `mandatory` flag and handle `basic.return`, or an intentionally monitored alternate exchange, when “routed to a queue” is part of success.
- Track publish sequence numbers because confirms can be batched and can arrive in a different order from publishes. Bound outstanding unconfirmed messages.
- Connection loss after broker acceptance but before the confirm is ambiguous. Republish the same logical message identity; consumer idempotency absorbs the possible duplicate.

## Consume, acknowledge, and flow control

- Use manual acknowledgements for durable effects. Acknowledge on the same channel and only after the effect and inbox record commit.
- An unacknowledged delivery is automatically requeued when its channel or connection closes. The `redelivered` flag is evidence of a prior delivery, not a complete attempt counter and not an idempotency key.
- Automatic acknowledgement transfers responsibility when RabbitMQ writes the delivery to the socket and therefore permits loss if the consumer fails before its effect.
- Set per-consumer prefetch from downstream capacity and processing latency. It bounds delivered-but-unacknowledged work; unlimited prefetch moves the queue into consumer and broker memory.
- With concurrent handlers, acknowledge individual tags or maintain an ordered completion frontier before a multiple acknowledgement. A multiple acknowledgement can settle unfinished earlier deliveries.

## Ordering, poison records, and recovery

- Queues enqueue in FIFO order, while priorities, multiple consumers, processing concurrency, and requeue/redelivery can change observed effect order. Use a single active consumer or an application ordering/fencing rule when one queue needs serialized effects.
- Requeue only a failure expected to become healthy. Immediate requeue loops consume capacity; route delayed transient retries through a bounded retry topology or delayed scheduling mechanism.
- Quorum queues support a delivery limit and can dead-letter after repeated delivery. RabbitMQ 4.x applies a default quorum delivery limit, so set and monitor an intentional value rather than inheriting it silently.
- Dead-lettering is another publish boundary. Quorum queues can provide at-least-once dead-lettering when configured for it; retries can still create duplicates at the target.
- Redrive from quarantine with original logical identity, a bounded rate, and a fixed consumer/schema version. Expect redriven work to interleave or reorder unless the recovery design isolates it.
- A quarantine design is not proven until a separate bounded-redrive canary preserves the original identity, leaves an already-applied effect unchanged, applies only the intended repaired item, and stops on the declared business invariant; list this test explicitly rather than folding it into rollout prose.
- Recovery redeclares topology idempotently after connection recovery and resumes only after publishers have restored confirm tracking and consumers have restored acknowledgement behavior.

## Signals and security that change the design

- Observe unroutable returns, confirm latency/nacks/timeouts, ready and unacknowledged counts, oldest age, redelivery rate, consumer capacity, connection churn, delivery-limit events, DLX failures, and quarantine age.
- Use separate least-privilege virtual-host users for publishers, consumers, and operators; restrict configure/write/read permissions, require TLS where appropriate, and keep tenant routing keys from becoming an authorization mechanism by themselves.

## Primary sources

- [RabbitMQ consumer acknowledgements and publisher confirms](https://www.rabbitmq.com/docs/4.2/confirms)
- [RabbitMQ reliability guide](https://www.rabbitmq.com/docs/reliability)
- [RabbitMQ consumer prefetch](https://www.rabbitmq.com/docs/4.2/consumer-prefetch)
- [RabbitMQ quorum queues and poison-message handling](https://www.rabbitmq.com/docs/4.2/quorum-queues)
- [RabbitMQ queue ordering](https://www.rabbitmq.com/docs/4.2/queues)

Sources checked 2026-08-02. Recheck the target version before relying on defaults.
