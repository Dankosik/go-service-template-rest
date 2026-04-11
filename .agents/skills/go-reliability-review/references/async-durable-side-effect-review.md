# Async Durable Side-Effect Review

## When To Load
Load this reference when a diff changes asynchronous side effects, event publication, message acking, outbox/inbox logic, webhook enqueueing, background relays, DLQ behavior, redrive, deduplication, consumer idempotency, or any state change that depends on a later async follow-up.

Keep findings local: identify dual-write, ack-before-durable-effect, and replay holes in changed code. Hand off full saga, workflow, outbox architecture, ordering model, or compensation policy to `go-distributed-architect-spec`; hand off transaction mechanics to `go-db-cache-review`.

## Review Smells
- Code writes local state and then publishes a message outside the same durable transaction.
- Code publishes a message and then writes local state, so rollback can leave a false event.
- A consumer acknowledges a message before the local effect is durable.
- A retryable publisher or consumer has no idempotency key, dedup table, or natural unique constraint.
- A relay deletes an outbox row before the broker confirms publish.
- A poison message can block all later messages forever.
- DLQ or redrive behavior lacks a bounded retry count and operator signal.
- Event order is assumed but no per-aggregate ordering or sequence is preserved.

## Failure Modes
- State commits but the event is lost, so downstream services never learn about the change.
- Event publishes but state rolls back, so downstream services process a fact that never became true.
- Message processing succeeds locally, but a crash before ack causes duplicate processing.
- A malformed event retries forever and blocks the relay.
- Redrive replays duplicate or stale events and corrupts downstream state.

## Review Examples

Bad: dual write between DB and broker.

```go
func (s *Service) CompleteOrder(ctx context.Context, id string) error {
	if err := s.orders.MarkComplete(ctx, id); err != nil {
		return err
	}
	return s.events.Publish(ctx, OrderCompleted{ID: id})
}
```

Review finding shape:

```text
[high] [go-reliability-review] internal/orders/service.go:104
Issue: The changed path commits order completion and then publishes the completion event as a separate durable operation.
Impact: A crash or broker outage after the DB commit loses the event, leaving downstream consumers permanently unaware of the completed order.
Suggested fix: Persist the order update and an outbox record in the same local transaction, then let a bounded relay publish and retry with deduplication.
Reference: AWS and Azure transactional outbox guidance.
```

Good: local transaction owns state plus outbox record; a separate relay owns publication.

```go
func (s *Service) CompleteOrder(ctx context.Context, id string) error {
	return s.db.InTx(ctx, func(tx *sql.Tx) error {
		if err := s.orders.MarkCompleteTx(ctx, tx, id); err != nil {
			return err
		}
		return s.outbox.InsertTx(ctx, tx, OutboxEvent{
			ID:          newEventID(),
			AggregateID: id,
			Type:        "order.completed",
		})
	})
}
```

Bad: ack before the effect is durable.

```go
func (c *Consumer) Handle(ctx context.Context, msg Message) error {
	msg.Ack()
	return c.store.Apply(ctx, msg.ID, msg.Payload)
}
```

Smallest safe local fix: move ack after `Apply` commits and make `Apply` idempotent for `msg.ID`. If the consumer needs exactly-once semantics across broker and DB, escalate.

## Smallest Safe Fix
- Persist state and an outbox/event record in one local transaction where the repo already has transaction support.
- Ack or delete messages only after the local side effect is durable.
- Add idempotency/dedup on message ID, aggregate sequence, or operation key.
- Keep retry counts, DLQ, and redrive explicit for poison messages.
- Preserve per-aggregate order when the consumer relies on order.
- Add an operator-visible signal for stuck relay, DLQ growth, or retry exhaustion.
- Escalate when the right fix requires a new saga, compensation, global ordering, or cross-service consistency model.

## Validation Commands
- `go test ./... -run 'Test.*(Outbox|Inbox|DualWrite|Publish|Relay)'`
- `go test ./... -run 'Test.*(Ack|Nack|Duplicate|Dedup|Idempot|Replay|Redrive|DLQ)'`
- `go test ./... -run 'Test.*(Rollback|Commit|Transaction)'`
- `go test -race ./...` when relay or consumer state is concurrent.

Useful failure-injection cases: broker publish fails after DB commit, DB commit fails after event construction, process restarts before ack, duplicate message redelivery, poison message redrive.

## Exa Source Links
- AWS transactional outbox pattern: https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html
- Azure transactional outbox with Cosmos DB, change feed, and Service Bus: https://learn.microsoft.com/en-us/azure/architecture/databases/guide/transactional-outbox-cosmos
- Azure Compensating Transaction pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/compensating-transaction
- Azure Design for self-healing, async decoupling and checkpoints: https://learn.microsoft.com/en-us/azure/architecture/guide/design-principles/self-healing
- Azure Queue-Based Load Leveling pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/queue-based-load-leveling

