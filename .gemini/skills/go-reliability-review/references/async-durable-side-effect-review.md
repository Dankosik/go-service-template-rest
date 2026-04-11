# Async Durable Side-Effect Review

## Behavior Change Thesis
When loaded for symptom `async side effect, event publication, message ack, outbox/inbox, relay, DLQ, dedup, or redrive changed`, this file makes the model find dual-write, ack-before-durable-effect, and replay holes instead of likely mistake `assume eventual retry makes async side effects reliable`.

## When To Load
Load when a diff changes asynchronous side effects, event publication, message acking, outbox/inbox logic, webhook enqueueing, background relays, DLQ behavior, redrive, deduplication, consumer idempotency, or any state change that depends on a later async follow-up.

Keep findings local: identify dual-write, ack-before-durable-effect, and replay holes in changed code. Hand off full saga, workflow, outbox architecture, ordering model, or compensation policy to `go-distributed-architect-spec`; hand off transaction mechanics to `go-db-cache-review`.

## Decision Rubric
- Code writes local state and then publishes a message outside the same durable transaction.
- Code publishes a message and then writes local state, so rollback can leave a false event.
- A consumer acknowledges a message before the local effect is durable.
- A retryable publisher or consumer has no idempotency key, dedup table, or natural unique constraint.
- A relay deletes an outbox row before the broker confirms publish.
- A poison message can block all later messages forever.
- DLQ or redrive behavior lacks a bounded retry count and operator signal.
- Event order is assumed but no per-aggregate ordering or sequence is preserved.

## Imitate

Bad finding shape to copy: the local DB write and broker publish do not share a durable success boundary.

```go
func (s *Service) CompleteOrder(ctx context.Context, id string) error {
	if err := s.orders.MarkComplete(ctx, id); err != nil {
		return err
	}
	return s.events.Publish(ctx, OrderCompleted{ID: id})
}
```

```text
[high] [go-reliability-review] internal/orders/service.go:104
Issue: The changed path commits order completion and then publishes the completion event as a separate durable operation.
Impact: A crash or broker outage after the DB commit loses the event, leaving downstream consumers permanently unaware of the completed order.
Suggested fix: Persist the order update and an outbox record in the same local transaction, then let a bounded relay publish and retry with deduplication.
Reference: AWS and Azure transactional outbox guidance.
```

Good correction shape: local transaction owns state plus outbox record; a separate relay owns publication.

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

Bad finding shape to copy: broker ack must follow the durable local effect when the message represents work.

```go
func (c *Consumer) Handle(ctx context.Context, msg Message) error {
	msg.Ack()
	return c.store.Apply(ctx, msg.ID, msg.Payload)
}
```

Copy the review move: ask for ack after `Apply` commits and idempotency for `msg.ID`; escalate if exactly-once semantics across broker and DB are being promised.

## Reject

```go
if err := s.store.Save(ctx, item); err != nil {
	return err
}
return s.bus.Publish(ctx, ItemSaved{ID: item.ID})
```

Reject when losing the publish after the commit leaves downstream systems permanently stale.

```go
if err := s.bus.Publish(ctx, event); err != nil {
	return err
}
return tx.Commit()
```

Reject when publish can escape a transaction that later rolls back.

```go
msg.Ack()
if err := c.apply(ctx, msg); err != nil {
	return err
}
```

Reject because a crash or apply failure after ack loses the work.

## Agent Traps
- Do not say "add retries" as the safe fix for dual writes; retry cannot repair a crash between two durable systems.
- Do not demand a new saga when a local outbox or ack-order fix is the smallest safe correction already supported by the repo.
- Do not assume at-least-once delivery is safe without consumer idempotency or dedup.
- Do not treat a DLQ as reliable unless retry bounds, poison-message behavior, and operator signals are explicit.
- Do not review DB transaction mechanics deeply here; hand off to `go-db-cache-review` when the transaction boundary is the primary uncertainty.

## Validation Shape
- `go test ./... -run 'Test.*(Outbox|Inbox|DualWrite|Publish|Relay)'`
- `go test ./... -run 'Test.*(Ack|Nack|Duplicate|Dedup|Idempot|Replay|Redrive|DLQ)'`
- `go test ./... -run 'Test.*(Rollback|Commit|Transaction)'`
- `go test -race ./...` when relay or consumer state is concurrent.

Useful failure-injection cases: broker publish fails after DB commit, DB commit fails after event construction, process restarts before ack, duplicate message redelivery, poison message redrive.
