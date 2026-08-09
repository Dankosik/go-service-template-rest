# PostgreSQL Idempotent Inbox

Select `DATABASE=postgres INBOX=postgres` when a consumer must apply one
same-database effect per consumer and logical message. The inbox is independent
of the outbox and NATS profiles.

The concrete consumer adapter derives a stable consumer identity and calls
`postgresinbox.Claim` with the logical message ID inside the same caller-owned
`pgx.Tx` as the feature repository:

```text
Pool.InTx
  Claim(ctx, tx, consumerIdentity, messageID)
  claimed=false -> skip the feature effect and return success
  claimed=true  -> apply one effect through a tx-bound repository
commit          -> acknowledge the delivery
error/rollback  -> return an error so the transport redelivers
```

For NATS JetStream, use `Message.MessageID()` and the configured
`stream/consumer` pair. `PublicationID`, broker sequence, ordering key, and
process identity are not durable inbox keys. Renaming the stream or durable
consumer deliberately creates a new claim scope.

Claims are permanent and scoped per consumer. There is no TTL, cleanup job,
status machine, runtime configuration, telemetry, or generic per-key ordering.
The guarantee covers only the claim and effect committed in the same PostgreSQL
transaction. HTTP calls, broker publishes, and other external effects need
their own downstream idempotency or a transactional outbox.

The worked adapter in
`examples/reference-service/postgres_inbox_integration_test.go` keeps `pgx`,
the inbox claim, and the tx-bound repository in the infrastructure layer; the
feature use case keeps its existing ports and has no duplicate branch.

Initialization records `inbox = "postgres"` in `template.lock`. Adoption is
forward-only unless the service can seed claims from authoritative history:
pre-cutover effects without a claim may be applied again. Before enabling a
consumer, prove there are no replayable applied logical IDs lacking a seeded
claim or retained domain idempotency.
