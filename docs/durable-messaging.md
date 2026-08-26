# Durable messaging with NATS JetStream

Select the pack only for a service that uses JetStream:

```bash
make template-init \
  MODULE=github.com/acme/orders \
  CODEOWNER=@acme/backend \
  MESSAGING=nats-jetstream
```

`MESSAGING=none` removes the adapter, worker, configuration, tests,
documentation, commands, and NATS dependencies.

## Business API

Business code declares an event kind, creates a typed event, or registers a
typed handler. It never sees subjects, streams, consumers, headers, publication
attempts, delivery metadata, acknowledgements, retries, or dead letters.

```go
var OrderUpdated = domainevent.Define[order.UpdatedV1]("order.updated", 1)

event, err := OrderUpdated.New(eventID, occurredAt, order.UpdatedV1{
    OrderID: orderID,
    Revision: revision,
})
if err != nil {
    return err
}
if err := publisher.Publish(ctx, event); err != nil {
    return err
}
```

Composition owns the route:

```go
registry, err := natsjs.NewRegistry(natsjs.Route{
    Type: OrderUpdated.Type, Version: OrderUpdated.Version, Subject: "events.orders",
})
```

The worker builder registers the typed handler on that registry:

```go
err = registry.Handle(OrderUpdated,
    func(ctx context.Context, event domainevent.Typed[order.UpdatedV1]) error {
        return orders.Apply(ctx, event.Payload)
    },
)
```

Return `natsjs.Permanent(err)` only when retrying the same event bytes
cannot succeed.

## Operator topology and configuration

The operator creates the source and dead-letter streams and owns their subject
coverage, storage, replicas, retention, capacity, discard policy, and duplicate
window. The application does not reconcile streams. It creates or updates only
the durable consumer fields coupled to settlement: explicit acknowledgements,
unlimited broker delivery until local DLQ transfer, concurrency, filter, and
handler acknowledgement budget.

Runtime configuration is limited to:

- `APP__MESSAGING__URLS`, credentials, optional root CA, and explicit local
  plaintext/unauthenticated escape hatches;
- source stream, durable consumer, filter subject, and dead-letter subject;
- maximum payload bytes and worker concurrency.

Empty URLs disable messaging in the API process. Worker and outbox binaries
require URLs. Delivery bytes, the 30-second handler timeout, retry sequence
`1s,5s,30s,2m`, 30-second DLQ retry delay, and drain behavior are safe code
defaults rather than operator knobs. Shutdown spends the existing process grace
deadline.

## Guarantees

Publishing succeeds only after a synchronous JetStream acknowledgement.
Validation or a definite broker refusal returns `natsjs.ErrRejected`; a lost or
inconclusive response returns `natsjs.ErrAmbiguous`. Retry the same logical event
with the same ID. Broker deduplication applies only inside the configured stream
window.

Delivery is at-least-once. The worker runs one serial native JetStream consume
context per concurrency slot. The NATS client owns pulling, buffering,
reconnect, and callback drain. A successful handler is confirmed with
`DoubleAck`. A crash after a business effect and before that acknowledgement
redelivers the event, so the effect must be durably idempotent by event ID for
the complete retention, redrive, and replay horizon.

Retryable failures request delayed redelivery. Malformed, permanent, or
exhausted work is published to the dead-letter stream before the source is
acknowledged. Lost publication or source acknowledgements can duplicate a DLQ
record, especially beyond the duplicate window.

The outbox preserves the same boundary: River completes a job only after a
conclusive broker acknowledgement, and every retry uses the stored event ID.

## Lifecycle and telemetry

Disconnect clears readiness. The cached readiness probe verifies the connection,
source stream, and durable consumer; no broker I/O occurs in HTTP handlers or
metric callbacks. Shutdown drains every native consume context, waits for their
`Closed` signals, then drains the connection. Deadline expiry cancels handlers,
stops the contexts, and leaves unacknowledged work for redelivery.

The adapter emits only:

- `messaging.publish.operations` and `.duration`;
- `messaging.handler.operations` and `.duration`;
- `messaging.dlq.transfers`;
- publish and process spans plus failure-only logs.

Use NATS Surveyor or the deployment's equivalent for connection, stream,
consumer, backlog, redelivery, retention, and capacity metrics.

## Redrive

`natsjs.RestoreDeadLetter` reconstructs the original logical event and gives a
redrive a deterministic new publication ID. The logical event ID is preserved;
the publication ID changes so the source stream accepts the redrive. Redrive is
still at-least-once and the consumer's durable idempotency remains authoritative.

## Proof

```bash
go test -vet=off ./internal/domainevent ./internal/messagingconfig ./internal/infra/natsjs ./cmd/worker/...
go test -vet=off -tags=integration ./internal/infra/natsjs
REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration-messaging
make test-messaging-race
```
