# PostgreSQL Transactional Outbox

Select this optional pack with:

```bash
DATABASE=postgres OUTBOX=postgres make template-init \
  MODULE=github.com/acme/orders CODEOWNER=@acme/backend
```

`OUTBOX=postgres` requires the PostgreSQL profile. `OUTBOX=none`, including the
default, removes the outbox schema, generated queries, runtime package, relay
command, configuration, tests, documentation, image binary, and Make targets.

## Delivery contract

This pack provides at-least-once durable publication with explicit duplicate
behavior. It does not provide exactly-once delivery.

Feature code chooses the event and appends it through the same `pgx.Tx` that
owns the domain mutation:

```go
err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
	if err := repository.Update(ctx, tx, change); err != nil {
		return err
	}
	return outbox.Append(ctx, tx, postgresoutbox.Event{
		ID:          postgresoutbox.NewID(),
		Type:        "order.updated",
		Source:      "orders",
		Destination: "orders.events",
		Schema:      "v1",
		OccurredAt:  time.Now().UTC(),
		Payload:     payload,
		Metadata:    metadata,
		OrderingKey: orderID,
		OrderingSequence: revision,
	})
})
```

`Append` neither begins nor commits a transaction. Returning an error rolls
back both the domain mutation and outbox row. The API process never calls a
broker and can keep committing while the broker is unavailable, subject to
PostgreSQL capacity. An outage therefore appears as observable backlog instead
of request-path dual-write failure.

`cmd/outbox-relay` is a separately deployable process. The template deliberately
registers no publisher: an initialized service must replace the `nil` builder
in `cmd/outbox-relay/main.go` with its selected messaging adapter. There is no
production noop fallback. The adapter is outside this pack and must return nil
only after the broker durably acknowledges the same event ID.

## Envelope and ordering

Each immutable row carries event ID, type, source, destination, schema,
occurrence time, exact JSON payload bytes, exact JSON-object metadata bytes,
and an optional ordering key plus positive sequence. Text fields and the
ordering key are limited to 256 bytes, payload to 256 KiB, metadata to 32 KiB,
and the complete stored envelope to 288 KiB.

For an ordering key, retained PostgreSQL high-water state rejects equal or
lower sequences even after event cleanup. Gaps are allowed. Only the earliest
unpublished sequence for a key is claimable; retry, lease, and poison state
block later rows for that key. This is an outbox claim-order guarantee, not an
end-to-end broker ordering claim. The selected adapter and consumer must also
preserve the key.

## Claim, acknowledgement, and recovery

Relays claim one row at a time with `FOR UPDATE SKIP LOCKED`, a random lease
token, and a lease expiry. Token-and-expiry compare-and-set updates fence stale
owners. Multiple relay replicas can claim unrelated rows safely.

After broker acknowledgement, the relay marks the row published. A crash after
the acknowledgement but before that PostgreSQL update leaves the row leased;
after expiry another relay publishes the same ID and bytes again. Consumers
must therefore deduplicate when their business effect is not naturally
idempotent. A crash before acknowledgement, cancellation, or forced shutdown
also leaves durable work for lease recovery rather than inventing success.

Temporary failures use full-jitter exponential retry from one second to five
minutes by default. A permanent adapter rejection or the tenth unsuccessful
attempt moves the row to poison state. Poison rows are never discarded and
block later work for the same ordering key. Operators redrive one with a unique
audit ID; the operation is idempotent for that audit ID and retains the original
event ID and bytes.

## Runtime and operations

The default relay settings are:

| Setting | Default |
| --- | --- |
| Poll / observation | 500 ms / 5 s |
| Publish timeout / lease | 10 s / 30 s |
| Attempts / retry | 10 / 1 s to 5 min full jitter |
| Cleanup / retention | 1,000 rows each minute / 7 days |
| Graceful drain | 20 s inside the process grace budget |

The lease must exceed the publish timeout, the fixed one-second publisher join
bound, and PostgreSQL acquire and statement budgets. Configuration validation
rejects an unsafe combination before publisher or database mutation.

Readiness requires valid configuration, a real publisher, reachable expected
schema, a running relay loop, and a fresh PostgreSQL state observation. A stale
observation, database/schema loss, fatal publisher failure, or drain makes it
false. A transient broker outage does not: the relay remains capable of durable
retry/poison progress. Liveness remains process-only.

On shutdown the process flips readiness false, stops new claims, and lets the
current attempt finish within the drain window. Expiry cancels the attempt. A
publisher that ignores cancellation beyond the one-second join bound makes
cleanup unsafe, so the process does not close dependencies still reachable by
that goroutine. The process grace period also reserves the relay's two-second
outer join plus bounded diagnostics, publisher, and telemetry cleanup after the
drain window.

Metrics expose mutually exclusive counts and oldest timestamps for eligible,
in-progress, retry-wait, recovery-due, ordering-blocked, poison, and retained
published rows; table/index bytes; ordering-head count; observation freshness;
last durable progress; operations; in-flight work; and readiness. Attributes
are fixed enums. Payload, metadata, credentials, DSN, ordering keys, broker
errors, and SQL text are never metric labels or logs.

Published rows are retained for seven days and deleted in bounded concurrent
batches. Pending, leased, retry, recovery, poison, and ordering-high-water rows
are not deleted. PostgreSQL is a finite outage buffer: alert on unpublished
count/oldest age, poison, retry errors, state-observation freshness, drain rate,
and relation/index growth. Add partitioning only after measured table/vacuum or
claim-plan evidence shows the bounded cleanup design no longer holds.

## Rollout, replay, and rollback

Apply the canonical Goose migration before starting relay replicas. Deploy the
API append path before or with the relay; a stopped relay safely accumulates
rows. Scale replicas only after one replica is healthy and backlog signals are
visible. Roll back the relay by stopping it; leases expire and rows remain.
Do not roll back the schema while any producer or relay binary can use it.

Replay is explicit poison redrive, or a separately reviewed bounded operator
procedure for already published rows. Never reset published state casually:
doing so intentionally creates duplicates.

This pack uses polling rather than CDC. Polling keeps transaction, claim,
recovery, and rollout authority inside the repository's existing PostgreSQL,
Go binary, and deployment model. Debezium-style CDC becomes preferable only
when the platform already owns connector availability, replication slots,
WAL retention, schema-history storage, offset recovery, broker routing, and
their rollout/monitoring burden. CDC is a replacement architecture, not an
extra relay mode to turn on beside this one.
