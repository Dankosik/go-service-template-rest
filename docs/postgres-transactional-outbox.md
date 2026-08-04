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

`Append` is variadic, and a business transaction that emits several events
should pass them in one call:

```go
return outbox.Append(ctx, tx, shipped, invoiced, notified)
```

Each column travels as one array, so the whole call is one statement and one
round trip however many events it carries and whatever mix of ordered and
unordered events those are. The caller holds its own row locks for that much
less time. Nothing is sent unless every event is valid, and one rejected
ordering sequence stores nothing, so a call is never partly applied for a
reason the caller could have seen up front. One call is one `append` operation
in telemetry, because that is what the recorded duration measures; the backlog
gauges report events.

What this buys is not a round trip — a pipeline of per-event statements already
costs one. It is executor setup. PostgreSQL sets up an `INSERT` once per
statement, and setting it up opens every index on this table again, which costs
more than the insert itself. One statement pays that once for the whole call.

Measured against one statement per event, on a CPU-Optimized 4-vCPU Droplet
with the repository's PostgreSQL 17 Testcontainers fixture, 20 samples per side
in alternating batches on one host:

| events per transaction | unordered | ordered |
| --- | --- | --- |
| 1 | within noise (p=0.06) | +11% |
| 4 | −38% | −38% |
| 16 | −69% | −72% |

The gain grows with the event count because the per-statement setup it removes
is per event. The one case that pays is a transaction emitting exactly one
ordered event: the array form costs about ten more microseconds of parameter
decoding and aggregation than a scalar single-row statement, and there is no
second event to amortize it over. Two ordered events already win. A call
carrying no ordering key at all uses a statement that never touches an ordering
head, which is what keeps the single unordered append flat.

Allocations move the same way and for the same reason: a one-event call
allocates about 30% more than the per-event form (78 versus 58 for one
unordered event) because it builds one slice per column, while a sixteen-event
call allocates about 34% less. Against a round trip these are noise; they are
recorded so a later change is measured against something. Re-measure before
assuming any of it survives a different payload size or network path.

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
bytes, through the SQL/JSON `IS JSON` and `IS JSON OBJECT` predicates. Bytes
that are not valid UTF-8 are refused as an encoding error rather than a check
violation, which only a writer bypassing `Append` can reach.
Payloads such as a JSON number outside PostgreSQL `numeric` range and
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

An ordered append advances the high-water mark and stores its events in one
statement, and it advances each key's head once per call rather than once per
event, so a feature transaction takes a given head's row lock a single time.
`outbox_ordering_heads.last_sequence` is the authority that rejects a reused
sequence, and it survives event cleanup, so the event table carries one partial
unique index over unpublished `(ordering_key, ordering_sequence)` instead of
that index plus a full-table one.

A call is accepted only when every sequence it carries for a key clears that
key's retained mark; one rejected key rejects the call. Within a single call the
events of a key may be passed in any order, because they are stored and
published in sequence order either way.

Across calls that rule decides who may write a key concurrently: a key's
sequences must reach PostgreSQL in ascending order, so two unsynchronized
writers sharing one ordering key is not a supported shape — whichever commits
second is rejected rather than stored out of order. Take the sequence from the
aggregate's own revision, under the same lock that serializes the domain
mutation, and each key has one writer at a time by construction. Distinct keys
never contend with each other; they take distinct head rows.

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

A claim costs what its batch costs, not what the table costs. The claim and the
retention delete both pick their rows under `FOR UPDATE` and then reach them by
the physical address that lock pins, rather than by id. Matching on the id
instead leaves PostgreSQL choosing between one index descent per row and a
sequential scan of every retained row, and it takes the scan while the batch is
a large enough share of the table's estimated rows — so a claim would cost what
the retention window costs, over rows that window has nothing to do with. A
`TID` scan reads no index and cannot be planned another way. Measured on a
CPU-Optimized 4-vCPU Droplet with the repository's PostgreSQL 17 fixture,
claiming 100 events took 4.6 ms against 10,000 retained rows and 2.3 ms against
210,000, against 1.6 ms either way once the address is used. End to end, over 10
samples per side on one host, the publish cycle fell 9.5% per event against a
50,000-row retention window and 10.5% against 200,000, and one 1,000-row
retention batch fell 35% against 200,000 retained rows and 37% against 400,000.

That address is valid only inside the statement holding the lock, and only while
events live in a single table — partitioning `outbox_events` would have to
return to the id.

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
| Batch size / publish concurrency | 500 / 16 |
| Publish timeout / lease | 10 s / 30 s |
| Attempts / retry | 10 / 1 s to 5 min full jitter |
| Cleanup / retention | 1,000 rows/transaction; one-minute normal cadence, poll-cadence catch-up / 7 days |
| Graceful drain | 20 s inside the process grace budget |

The lease must exceed the publish timeout, the fixed one-second publisher join
bound, and PostgreSQL acquire and statement budgets. Configuration validation
rejects an unsafe combination before publisher or database mutation.

Raise `batch_size` for backlog drain rate and lower it to cap peak relay memory,
which is that many stored envelopes; the envelope limit makes 500 worth up to
about 144 MiB and a typical payload far less — a 4 KiB event is about 2 MiB.

The default is the measured knee rather than a round number. Per event, one
claim-and-publish cycle cost 36.1 µs at a batch of 100, 30.1 µs at 250, 28.2 µs
at 500, and 40.4 µs at 1,000: batching amortizes two round trips and two commits
until the statements themselves start to dominate, and the validated maximum of
1,000 is slower than 100. Lower it to 250 to halve worst-case memory for about
two thirds of the gain. Raise `publish_concurrency` when
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

That cost is roughly half a microsecond per unpublished row: 4.8 ms at a 5,000
backlog and 50.8 ms at 100,000. Against the five-second default that is a 1%
duty cycle on one connection at 100,000 pending and around 10% approaching a
million — and a backlog is largest exactly when the broker is down and the
signal matters most. Lengthen `observation_interval` before the backlog a given
deployment must survive makes the observation itself a load problem.

Identifiers are stored `COLLATE "C"`. Event ids, ordering keys, and audit ids
are opaque tokens compared for equality and sorted only for determinism, but a
locale collation makes every B-tree descent call `strcoll` instead of `memcmp`,
and claim, batch finalization, and the ordered head lookup each descend one of
those indexes per event. It is worth 11% of a 100-event claim or unordered
publication and 17% of the ordered one. The control-character constraints are
unaffected, because PostgreSQL derives their regex classes from the database
encoding rather than the collation.

Payload and metadata use lz4 rather than the `pglz` default. Compression engages
only past the 2 KiB TOAST target, so small events are untouched; above it lz4
costs much less to compress at a slightly worse ratio. Appending four 4 KiB
events fell 19% and four 64 KiB events 58% — paid on the request path, inside
the caller's transaction — while re-reading a 64 KiB payload on claim rose 44%,
83 µs against the 1.37 ms the append saved. This needs a PostgreSQL built with
lz4 (14+, including the pinned image and the major managed providers); drop the
clause for a service that stores mostly very large payloads and drains far more
often than it appends.

Identifiers are stored `COLLATE "C"`. Event ids, ordering keys, and audit ids
are opaque tokens compared for equality and sorted only for determinism, but a
locale collation makes every B-tree descent call `strcoll` instead of `memcmp`,
and claim, batch finalization, and the ordered head lookup each descend one of
those indexes per event. It is worth 11% of a 100-event claim, 11% of the
unordered publication, and 17% of the ordered one. No check weakens: PostgreSQL
derives the control-character regex classes from the database encoding rather
than the collation, so the same bytes are rejected, C1 controls included.

### Optional: lz4 for large payloads

Payload and metadata use PostgreSQL's default compression. A service whose
events are routinely larger than the 2 KiB TOAST target can declare `lz4`
instead, which costs far less to compress at a slightly worse ratio:

```sql
ALTER TABLE outbox_events ALTER COLUMN payload SET COMPRESSION lz4;
ALTER TABLE outbox_events ALTER COLUMN metadata SET COMPRESSION lz4;
```

This is deliberately not the default. Compression engages only past the TOAST
target, so it does nothing for the many services whose events are smaller, while
it makes the schema require a PostgreSQL built with lz4 (14+, including the
pinned image and the major managed providers) — where that is missing the
migration fails outright rather than degrading. The trade is also two-sided.
Measured on a CPU-Optimized 4-vCPU Droplet:

| payload | append (request path) | claim re-read (relay) |
| --- | --- | --- |
| 256 B | unchanged | unchanged |
| 4 KiB | −19% | −1% |
| 64 KiB | −58% | +44% |

At 64 KiB the append saves 1.37 ms per event and the relay pays back 83 µs, so
it is still a net win — but a service that drains far more often than it appends
should measure its own ratio first. `SET COMPRESSION` applies to newly written
values; existing rows keep their current compression until rewritten.

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
