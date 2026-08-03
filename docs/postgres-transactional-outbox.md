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
production noop fallback. The adapter is outside this pack, must be safe for
concurrent `Publish` calls, and must return nil only after the broker durably
acknowledges the same event ID.

## Envelope and ordering

Each immutable row carries event ID, type, source, destination, schema,
occurrence time, exact JSON payload bytes, exact JSON-object metadata bytes,
and an optional ordering key plus positive sequence. Text fields and the
ordering key are limited to 256 bytes, payload to 256 KiB, metadata to 32 KiB,
and the complete stored envelope to 288 KiB.

The database validates the same JSON language as Go while retaining the exact
bytes. Payloads such as a JSON number outside PostgreSQL `numeric` range and
escaped `\u0000` strings are valid; the outbox never normalizes them through
`jsonb`.

For an ordering key, retained PostgreSQL high-water state rejects equal or
lower sequences even after event cleanup. Gaps are allowed. Only the earliest
unpublished sequence for a key is claimable; retry, lease, and poison state
block later rows for that key. This is an outbox claim-order guarantee, not an
end-to-end broker ordering claim. The selected adapter and consumer must also
preserve the key.

PostgreSQL materializes that earliest row as `ordering_ready` and serializes
append/finalization through `outbox_ordering_heads.current_sequence`. Claim and
observation therefore do not scan every predecessor for a hot ordering key.

An ordered append advances the high-water mark and stores the event in one
statement, so a feature transaction holds the head row lock for a single round
trip rather than two. `outbox_ordering_heads.last_sequence` is the authority
that rejects a reused sequence, and it survives event cleanup, so the event
table carries one partial unique index over unpublished `(ordering_key,
ordering_sequence)` instead of that index plus a full-table one.

## Claim, acknowledgement, and recovery

Relays claim a batch of up to `batch_size` rows in one statement with
`FOR UPDATE SKIP LOCKED`, a single random lease token, and a lease expiry.
Token-and-expiry compare-and-set updates fence stale owners, and the whole batch
is fenced as a unit. Multiple relay replicas can claim unrelated rows safely.

The batch is published through at most `publish_concurrency` concurrent
`Publish` calls, so an adapter must be safe for concurrent use. Concurrency
never reorders an ordering key: the partial unique index on ready ordered rows
means at most one event per key is claimable, so a batch can never hold two
events of the same key.

Every disposition finalizes in one statement for the whole batch. Unordered
acknowledgements take one, ordered acknowledgements take another — that one also
advances each event's key head and unblocks each key's successor, which the
statement can do for the whole lease because a lease never holds two events of
the same key. Retries and poison transitions take one statement each. A backlog
therefore costs roughly two database round trips per batch rather than two per
event, ordered work included. Only an event the statement reports as
unfinalized, which means a lost lease, falls back to a per-event resolution
against durable state.

Every publication attempt is bounded by both `publish_timeout` and the lease the
batch was claimed under, measured on the relay's own clock from before the
claim. Whatever the batch does not finish inside that window is released for
retry rather than abandoned until the lease expires, and finalization is
detached from process cancellation so a shutdown still records acknowledged
work instead of creating duplicates.

Idle relays wait on a PostgreSQL `LISTEN`/`NOTIFY` channel that a statement-level
insert trigger signals, so a committed append is normally picked up within a
round trip instead of within `poll_interval`. The listener owns one connection
outside the pool and adds no round trip to the appending transaction. It is a
latency optimization only: a lost notification, a dropped listener connection,
or a retry becoming due again falls back to the poll timer.

After broker acknowledgement, the relay marks the row published. A crash after
the acknowledgement but before that PostgreSQL update leaves the row leased;
after expiry another relay publishes the same ID and bytes again. Consumers
must therefore deduplicate when their business effect is not naturally
idempotent. A crash before acknowledgement, cancellation, or forced shutdown
also leaves durable work for lease recovery rather than inventing success.

Temporary failures use full-jitter exponential retry from one second to five
minutes by default. A permanent adapter rejection poisons immediately. The tenth
adapter-proven `ErrPublicationNotAccepted` failure moves the row to poison
state. Poison rows are never discarded and block later work for the same
ordering key. Operators redrive one with a unique audit ID; the operation is
idempotent for that audit ID and retains the original event ID and bytes.

The attempt threshold is evaluated only when the adapter proves the broker did
not durably accept the event. An unclassified error, timeout, disconnect, or
panic remains retryable even at that threshold: the relay never trades possible
event loss for a strict attempt-count cap. An adapter that cannot distinguish
refusal from an ambiguous outcome therefore retries indefinitely, and the
oldest-pending-age signal is what surfaces such a row to operators.

## Runtime and operations

The default relay settings are:

| Setting | Default |
| --- | --- |
| Poll (notification fallback) / observation | 500 ms / 5 s |
| Batch size / publish concurrency | 100 / 16 |
| Publish timeout / lease | 10 s / 30 s |
| Attempts / retry | 10 / 1 s to 5 min full jitter |
| Cleanup / retention | 1,000 rows/transaction; one-minute normal cadence, poll-cadence catch-up / 7 days |
| Graceful drain | 20 s inside the process grace budget |

The lease must exceed the publish timeout, the fixed one-second publisher join
bound, and PostgreSQL acquire and statement budgets. Configuration validation
rejects an unsafe combination before publisher or database mutation.

Raise `batch_size` for backlog drain rate and lower it to cap peak relay memory,
which is that many stored envelopes; the envelope limit makes 100 worth up to
about 29 MiB and a typical payload far less. Raise `publish_concurrency` when
broker acknowledgement latency, not the database, bounds throughput. A crash
mid-batch can redeliver up to `batch_size` events, which at-least-once delivery
already permits; size the batch against consumer deduplication cost as well as
memory.

Readiness requires valid configuration, a real publisher, reachable expected
schema, a running relay loop, and a fresh PostgreSQL state observation. A stale
observation, database/schema loss, fatal publisher failure, or drain makes it
false. A transient broker outage does not: the relay remains capable of durable
retry/poison progress. Liveness remains process-only.

A sample is fresh for two configured observation intervals. A failed periodic
observation retains the last sample and retries; readiness turns false only when
that sample crosses the freshness bound. Startup still fails closed if any of
the events, ordering-head, or redrive relations is unavailable.

On shutdown the process flips readiness false, stops new claims, and lets the
current attempt finish within the drain window. Expiry cancels the attempt. A
publisher that ignores cancellation beyond the one-second join bound makes
cleanup unsafe, so the process does not close dependencies still reachable by
that goroutine. The process grace period also reserves the relay's two-second
outer join plus bounded diagnostics, publisher, and telemetry cleanup after the
drain window. A separate Publisher cleanup callback is supervised for five
seconds; timeout or panic is reported instead of hanging process termination.

Metrics expose mutually exclusive counts and oldest timestamps for eligible,
in-progress, retry-wait, recovery-due, ordering-blocked, poison, and retained
published rows; table/index bytes; ordering-head count; observation freshness;
redrive-ledger bytes; last durable progress; operations; in-flight work; and
readiness. Attributes are fixed enums. Payload, metadata, credentials, DSN, ordering keys, broker
errors, and SQL text are never metric labels or logs.

One statement produces the whole observation, and it reads only unpublished rows
through the `outbox_events_pending_idx` partial index, so its cost tracks
backlog rather than retention volume. The `published_retained` count is the one
exception: counting it exactly would scan the entire retention window on every
observation, so it is reported as the planner's own row estimate for
`outbox_events` minus the exact pending count. Alert on
`published_retained` oldest age, which stays exact, rather than on that count;
the estimate also needs `autovacuum`/`ANALYZE` to have run to be meaningful.

Both queue tables are stored at `fillfactor = 45` with churn-proportional
autovacuum and analyze thresholds instead of PostgreSQL's size-proportional
defaults. A batch claim updates most of one heap page at a time and touches no
indexed column, so reserving just over half of each page lets every claim stay a
heap-only update: no index entry is written and the previous version is
reclaimed by ordinary page pruning rather than an index vacuum pass. The reserve
pays for itself in space as well, because the version sprawl a packed page
causes is larger than the reserve. On the repository's PostgreSQL 17 fixture,
inserting, claiming, and publishing 2,000 events moved the cycle's WAL from
2.12 MB to 0.97 MB with a 30-byte payload and from 6.28 MB to 1.00 MB with a
1 KiB payload, while the event heap shrank 22% and 33%. Publication itself
changes indexed columns and remains a normal indexed update.

Both tables carry these parameters from `CREATE TABLE`, so a service gets the
shape from its first migration. Reopen the setting if the relay stops claiming
consecutive rows in bulk, since the reserve is sized for whole-page updates; a
later change to it applies to newly written pages, reaching existing rows as
they turn over or immediately through a `VACUUM FULL` or `pg_repack` window.

Published rows are retained for seven days and deleted in bounded concurrent
batches. Pending, leased, retry, recovery, poison, and ordering-high-water rows
are not deleted. PostgreSQL is a finite outage buffer: alert on unpublished
count/oldest age, poison, retry errors, state-observation freshness, drain rate,
and relation/index growth. Add partitioning only after measured table/vacuum or
claim-plan evidence shows the bounded cleanup design no longer holds.

Budget PostgreSQL connections across the complete deployment, not one process:
`sum(API replicas * API max_open_conns) + sum(relay replicas * (relay
max_open_conns + 1)) + migration/admin reserve` must stay below the database
connection budget. The relay runs one database statement at a time, so
`APP__POSTGRES__MAX_OPEN_CONNS=2` is the validated minimum; the extra `+ 1` is
the notification listener, which owns its own connection outside the pool so a
blocked listener can never starve claim or finalization. Raise the pool only
when measured acquire wait or maintenance overlap proves two connections
insufficient. Keep API sizing tied to its own request workload rather than
copying the relay value.

## Rollout, replay, and rollback

Apply the canonical Goose migration before starting relay replicas. The pack
ships one migration holding the whole schema, so a new service has no
compatibility transition to plan and no intermediate version to stop at.

A stopped relay safely accumulates rows. Scale replicas only after one replica
is healthy and backlog signals are visible. Roll back the relay by stopping it;
leases expire and rows remain. Production rollout is forward-only by default:
the Down section drops the outbox tables, so it is a development affordance
rather than a production rollback path.

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
