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

This pack ships the relay side composed and the append side unwired, because the
template has no feature that emits events — a `Store` built for nobody would be
a dependency with no consumer. Wiring it is the first thing an adopting service
does.

Build one `*postgresoutbox.Store` over the pool that
`cmd/service/internal/bootstrap/startup_dependencies.go` already opens
(`runtimeDependencies.postgres`), beside the other profile-guarded dependencies
in `cmd/service/internal/bootstrap/run.go`, and pass it into whatever implements
`openapi.StrictServerInterface` — the `Handlers.API` seam. Its meter comes from
the same `telemetry.Metrics` the rest of that composition uses. Both arguments to
`NewTelemetry` may be nil, and so may the store's telemetry. The event is then
appended through the same `pgx.Tx` that owns the domain mutation.

The adapter declares the narrow interface it consumes instead of depending on
the concrete `*postgresoutbox.Store`. Interfaces belong to their consumers, so
the template does not export one for a feature that does not exist yet:

```go
type outboxAppender interface {
	Append(context.Context, pgx.Tx, ...postgresoutbox.Event) error
}
```

`*postgresoutbox.Store` satisfies that interface implicitly. Keeping the
interface in the adapter prevents the request path from reaching claim,
finalization, and operator methods without adding a provider-owned abstraction.

```go
// Composition root, once.
outboxTelemetry, err := postgresoutbox.NewTelemetry(meter, log)
if err != nil {
	return err
}
outbox, err := postgresoutbox.NewStore(pool, outboxTelemetry)
if err != nil {
	return err
}
// newOrdersRepository is the service's own constructor, not a template symbol:
// this pack ships the append side unwired, so the repository below is the one
// an adopting service writes. It takes the local outboxAppender interface.
adapter := newOrdersRepository(pool, outbox)

// PostgreSQL repository adapter, before entering the business transaction.
event := postgresoutbox.Event{
	ID:               postgresoutbox.NewID(),
	Type:             "order.updated",
	Source:           "orders",
	Destination:      "orders.events",
	Schema:           "v1",
	OccurredAt:       time.Now().UTC(),
	Payload:          payload,
	Metadata:         metadata,
	OrderingKey:      orderID,
	OrderingSequence: revision,
}
err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
	if err := repository.Update(ctx, tx, change); err != nil {
		return err
	}
	return outbox.Append(ctx, tx, event)
})
```

`Store` serves three audiences: the write path calls only `Append`, the relay
process owns claim, finalization, cleanup, and observation, and `Get`,
`Redrive`, `RedriveUnknown`, and `ConfirmAccepted` are operator tooling.
`ClassifyLegacyUncertainty` belongs only to the pre-relay upgrade command. The relay reads `Get` too, to resolve a
finalization its batch statement did not report.
`cmd/outbox-relay/internal/bootstrap/run.go` is the worked composition for the
relay side.

That write path is not `internal/<feature>`, and the block above says
`repository adapter` rather than `feature` for a reason a linter enforces:
depguard's `feature_packages_no_adapters` rule denies a feature package both
`internal/infra` and `github.com/jackc/pgx/v5`, so a feature package can neither
name `postgresoutbox.Event` nor hold a `pgx.Tx`. The split that leaves is the
one the rest of this repository already uses. The feature owns which occurrence
happened and what its payload means, and returns that as its own type. The
PostgreSQL repository adapter under `internal/infra/postgres` owns the
transaction, translates the feature's result into a `postgresoutbox.Event`, and
makes both calls inside one `InTx`. The composition root owns building the
`Store` and handing it to that adapter.

These operator actions are Go methods, not a shipped operator interface. Deciding
who may act is an authorization question this pack does not answer, so a
service exposes them the way it exposes anything else — an admin endpoint, a
maintenance command, a support tool — before an operator can use the redrive
procedure below.

A rejected event returns `ErrInvalidEvent` before any statement is sent. A
sequence at or below its key's retained high-water mark returns
`ErrOrderingSequence`, which only PostgreSQL can decide: the append statement
goes out and reports the offending key back, having stored nothing.

`Append` neither begins nor commits a transaction. Returning an error rolls
back both the domain mutation and outbox row. The API process never calls a
broker and can keep committing while the broker is unavailable, subject to
PostgreSQL capacity. An outage therefore appears as observable backlog instead
of request-path dual-write failure.

### Reconciling a lost commit response

Each append statement stores the event and a compact immutable commit receipt
atomically. The receipt is keyed by `Event.ID` and holds only version 1 plus a
SHA-256 fingerprint of the caller-owned envelope. It excludes the outbox-owned
trace context, has no foreign key to `outbox_events`, and is never removed by
published-event cleanup. That lifetime is what keeps reconciliation possible
after the full event has expired.

The repository adapter must build and retain the stable `Event` before entering
`postgres.InTx`, as in the example above. When `InTx` wraps
`postgres.ErrCommitUnknown`, call `Store.ReconcileCommit` with that same event
on the same configured writer pool:

| Result | Repository action |
| --- | --- |
| `CommitApplied` | Return the original operation as successful; do not rerun the mutation |
| `CommitNotApplied` | Retry the mutation with the same event; this result is possible only after the same read proves the receipt absent on a writable current primary |
| `ErrReceiptConflict` | Fail permanently; the event ID already names different immutable evidence |
| `CommitStillUnknown` plus error | Retry reconciliation only within the caller's budget; never rerun the mutation or mint another event ID |

`examples/reference-service/postgres_outbox_reconciliation_integration_test.go`
shows the concrete adapter loop and its real-PostgreSQL lost-response proof. It
is intentionally repository-owned rather than a generic transaction retry
helper: only that adapter knows which mutation and original success result it
is reconciling. Do not use this path for pre-cutover writes that could not have
created a receipt, and do not synthesize receipts for historical events.

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

`cmd/outbox-relay` is a separately deployable process. Selecting both the
PostgreSQL outbox and NATS profiles registers `bootstrap.BuildNATSPublisher` in
`main.go`; selecting outbox without messaging leaves the builder nil and fails
before telemetry, PostgreSQL, or a claim is started. There is no noop fallback.

The builder returns a validated `bootstrap.PublisherRuntime`: the one
`postgresoutbox.Publisher` plus its `Run`, `Ready`, and error-returning
`Shutdown` lifecycle. The relay reports ready only while both the relay and
publisher are ready, supervises a terminal NATS client return, drains claims,
joins the supervisor, stops diagnostics, and then shuts the client down before
closing PostgreSQL. A custom outbox-only adapter uses the same constructor:

```go
func main() {
	if err := bootstrap.Run(os.Args[1:], buildPublisher); err != nil {
		reportFailure(os.Stderr, err)
		os.Exit(1)
	}
}

func buildPublisher(
	ctx context.Context,
	cfg config.Config,
	log *slog.Logger,
) (bootstrap.PublisherRuntime, error) {
	client, err := broker.Dial(ctx, cfg /* ... */)
	if err != nil {
		return bootstrap.PublisherRuntime{}, fmt.Errorf("dial broker: %w", err)
	}
	return bootstrap.NewPublisherRuntime(
		brokerPublisher{client: client}, client.Run, client.Ready, client.Shutdown,
	)
}
```

### Classifying a publication failure

What the adapter returns decides whether an event is retried, counted against
`max_attempts`, or parked for an operator. Only the adapter can tell these
apart, because only it knows what its broker's errors prove:

| The adapter can prove | Return | Relay behavior |
| --- | --- | --- |
| Retrying these exact bytes can never succeed — the broker refused the payload, subject, or size | `ErrPermanentPublication` | Poisoned on the first occurrence; blocks its ordering key until an operator redrive |
| The broker definitely did not accept it, but the same bytes could succeed later — no stream, no responder, a definite API rejection | `ErrPublicationNotAccepted` | Retried with backoff, and poisoned once `max_attempts` is reached |
| Nothing — a timeout, cancellation, disconnect, or a lost acknowledgement | any other error | Marks publication uncertainty sticky; retries below `max_attempts`, then parks as `outcome_unknown` |

The third row is the default on purpose. It never asserts that the event was
unpublished: the sticky bit survives retry, lease recovery, restart, and
redrive. Once sticky, a permanent failure or any failure at `max_attempts`
parks the row as `outcome_unknown`; an acknowledgement still wins and marks it
published. Fresh permanent and exhausted not-accepted results remain ordinary
deterministic poison with the sticky bit false.

One outcome in the table has no adapter behind it. `publish_timeout` budgets the
whole claimed batch rather than one event, so a broker slow enough — or a batch
large enough — leaves the tail of a batch with no budget at all. Those events
are never handed to the adapter, and they are not classified by the third row:
they are released for immediate retry as `publisher_not_attempted`, they add no
uncertainty, and the attempt the claim charged is given back, so the attempt cap
keeps counting attempts actually made. Without that, a slow broker would walk
events nobody tried to publish to `max_attempts` and quarantine them as
`outcome_unknown`, turning a throughput problem into an operator action per
event. Alert on
`outbox.relay.operations{operation="publish",outcome="skipped"}`: a sustained
rate means the relay is claiming more per batch than its budget can publish, and
the fix is a smaller `batch_size`, a higher `publish_concurrency`, or a longer
`publish_timeout`.

<!-- profile:messaging-nats-jetstream:start -->
`natsjs.NewOutboxPublisher` is the selected adapter. It pre-validates the
immutable NATS envelope as permanent, maps later definite NATS rejection onto
`ErrPublicationNotAccepted`, leaves ambiguous/unknown errors ambiguous, and
returns nil only for a JetStream `PubAck`. Event ID is both logical and
publication identity; source and caller metadata do not widen the NATS wire
contract.

<!-- profile:messaging-nats-jetstream:end -->
## Trace continuity

An outbox breaks a trace by construction: the request that produced an event has
long returned by the time the relay publishes it. This pack keeps the join.

`Append` captures the W3C trace context active on its own context and stores it
with the event, in its own column rather than in `Metadata` — metadata is the
caller's bytes, stored and retried exactly as given, and merging into them would
break that and collide with a caller carrying its own `traceparent`. A caller
cannot set or forge the stored context; it comes from the ambient context only.

The relay emits one span per publication attempt, `publish {destination}`,
**linked** to that stored context rather than descending from it. The link is
deliberate: a publication can happen minutes after the append, or days after an
operator redrive, and a child span would hold the producing request's trace open
for that whole horizon — past the assembly window of every backend that has one.
The link carries the same join without that lifetime coupling, which is also
what the OpenTelemetry messaging convention prescribes for a send with a
separate creation context.

The selected NATS adapter removes the relay-attempt span while preserving the
batch cancellation/deadline, extracts `Event.CreationContext()`, and lets the
NATS producer span descend from that stored origin before injecting W3C headers.
The worker therefore sees the producing trace. Empty or invalid stored context
starts an unrelated producer root and never rejects publication.

Nothing about the trace context can fail a delivery. A context too large for its
1 KiB bound, or one the propagator cannot encode, is stored as absent and
counted on `outbox.relay.operations` as `operation=trace_capture`,
`outcome=rejected` — the append still succeeds. A stored context that cannot be
decoded reads as absent and the event still publishes. The reasoning is the
point of the pattern: an outbox exists so that infrastructure faults become
backlog instead of failed requests, so a field this pack added must never be the
one fault that fails a request.

`messaging.system` is deliberately absent from the span — the relay is
broker-neutral and only the adapter knows the system. Event IDs and ordering
keys never reach relay spans or logs. W3C trace links remain the correlation
mechanism without exposing stored identities.

## Envelope and ordering

Each immutable row carries event ID, type, source, destination, schema,
occurrence time, exact JSON payload bytes, exact JSON-object metadata bytes,
the captured trace context, and an optional ordering key plus positive
sequence. Text fields and the
ordering key are limited to 256 bytes, payload to 256 KiB, metadata to 32 KiB,
and the complete stored envelope to 288 KiB. The trace context is bounded
separately at 1 KiB and is deliberately *not* charged against that 288 KiB: it
is the outbox's field rather than the caller's, and charging it would start
rejecting events a service appends successfully today. The stored row therefore
exceeds the caller-facing budget by at most that allowance.

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

The one adapter this repository ships worked does not. `natsOutboxPublisher` in
`test/postgres_outbox_natsjs_integration_test.go` forwards the ordering key onto
the JetStream envelope as data, but JetStream assigns its own stream sequence
and a worker above `MAX_CONCURRENCY=1` runs one key's handlers concurrently —
see [Ordering does not compose with the
outbox](durable-messaging.md#ordering-does-not-compose-with-the-outbox) for the
two shapes that close it. Read the claim here as ordered publication, and decide
ordered processing at the consumer.

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
event, ordered work included.

The ordered statement has one built-in second attempt. A key whose successor
was committed but not yet visible to that statement's snapshot comes back marked
as a conflict, and resending only those keys takes a fresh snapshot, which
resolves it. `orderedPublishSnapshots` in the package is that count.

An acknowledgement neither statement reported falls back to a per-event
resolution against durable state. That normally means a lost lease; for an
ordered event it can instead mean the snapshot conflict outlasted the retry, and
the two are indistinguishable from the outside — which is why the fallback
resolves the row rather than assuming either. A retry or poison the statement
did not report is treated differently: it means this relay held the batch past
its own lease, so another relay already owns the event and will deliver it.
Nothing is at risk and nothing is reconciled, but the overrun is a lease or
replica misconfiguration, so the relay stops instead of continuing quietly.

That fallback is the one part of finalization whose cost is not per batch. Each
leftover event is resolved over two passes, and for an ordered event every pass
is itself worth the ordered statement's two snapshots — so a batch that comes
back short costs statements per leftover event on top of its two round trips.
Nothing sizes `lease_duration` for that worst case, deliberately: a
finalization that runs out of lease stops the relay with `progress_unknown`
rather than claiming a publication it cannot prove. Size the lease for the
ordinary batch and treat `progress_unknown` as the signal, not as something to
outrun. `ErrProgressUnknown` in the package owns the derivation.

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

A batch gets one publication deadline, not one per event: the earlier of
`publish_timeout` from the start of the batch and the lease it was claimed
under, less the one-second publisher join bound. The lease is measured on the
relay's own clock from before the claim, so it is the conservative side under
any skew against PostgreSQL. Subtracting the join bound is what leaves a stuck
adapter time to stop while the lease still covers finalization, so the budget an
adapter actually sees is a second shorter than `lease_duration`. Every `Publish`
call in that batch shares it, so an adapter sees the batch's remaining budget
rather than a fresh timeout. Whatever the batch does not finish inside that
window is released for retry rather than abandoned until the lease expires, and
finalization is detached from process cancellation so a shutdown still records
acknowledged work instead of creating duplicates.

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
minutes by default. A fresh permanent adapter rejection poisons immediately,
and the tenth adapter-proven `ErrPublicationNotAccepted` failure becomes
deterministic poison. An ambiguous result sets sticky publication uncertainty;
the claim statement quarantines a sticky row already at the attempt limit
without leasing or publishing it again. Both poison classes block later work
for the same ordering key and are retained for operator action.

### Legacy classification and outcome-unknown recovery

After the schema expansion and before any new relay starts, classify every
legacy NULL through the DB-only bootstrap route:

```sh
outbox-relay --classify-legacy-uncertainty
```

It loads normal outbox/PostgreSQL configuration, opens only the writer pool and
store, processes `cleanup_batch_size` rows per transaction using
`max_attempts`, and exits only after a batch returns zero. It does not build a
publisher, start diagnostics, claim, or report readiness. Normal relay startup
still fails closed when no publisher is registered.

Authorized discovery uses a writer-primary read, a limit no larger than 100,
and the `(poisoned_at, id)` cursor. Do not add payload, metadata, trace context,
credentials, or ordering keys:

```sql
SELECT id, destination, event_type, schema_name, cycle_attempt_count,
       total_attempt_count, last_attempt_at, poisoned_at, last_error_class,
       publication_uncertain
FROM outbox_events
WHERE published_at IS NULL
  AND poisoned_at IS NOT NULL
  AND publication_uncertain IS TRUE
  AND last_error_class = 'outcome_unknown'
  AND (poisoned_at, id) > ($1, $2)
ORDER BY poisoned_at, id
LIMIT 100;
```

After broker-side evidence is checked, call `RedriveUnknown` to retry the same
event identity while preserving stickiness, or `ConfirmAccepted` to record
broker acceptance without publishing again. Ordered confirmation advances the
existing outbox head and unblocks its successor. Both require a unique audit
ID; replaying the same action/event/audit succeeds, while reuse for another
action or event returns `ErrOperatorAuditConflict`. A wrong source state returns
`ErrOperatorStateConflict`, and an id naming no stored event returns
`ErrNotFound`. All three are `outcome=rejected` with `error.type=validation` on
`outbox.relay.operations`, not database failures: a mistyped id is a refused
call and must not read as an outage. Audit rows have no event foreign key and
survive event cleanup. Ordinary `Redrive` remains valid only for deterministic
poison.

Two adapter behaviors are fatal to the process rather than to one event. A panic
releases its own event for retry, finalizes the rest of the batch, and then
stops the relay with `publisher_panic`, because a broken adapter is a deployment
fault rather than a transient one. The one exception is a panic in a batch that
was already being cancelled: shutdown is checked first, so the relay reports an
ordinary drain and the panic surfaces in the publish metric rather than in the
exit class. Finalization still runs either way. Returning nil after the batch deadline has
passed is treated as unproven and retried: a publisher that stopped waiting
cannot have observed the broker's acknowledgement.

### What stops the relay

The adapter is not the only source. `failureClass` in
`cmd/outbox-relay/main.go` is the complete set, because it is what names the
process exit line, and a new stop reason belongs in that switch:

| Exit class | Cause |
| --- | --- |
| `config` | Invalid configuration, an unregistered or unusable publisher, or a store or relay that could not be built |
| `postgres_unavailable` | The pool could not connect, health-check, or acquire |
| `publisher_stuck` | An adapter goroutine outlived cancellation past the join bound; cleanup is unsafe |
| `publisher_panic` | An adapter panicked |
| `progress_unknown` | A publication could be neither recorded nor disproven |
<!-- profile:messaging-nats-jetstream:start -->
| `messaging_terminal` | The supervised NATS client stopped after reconnect exhaustion or another terminal lifecycle fault |
<!-- profile:messaging-nats-jetstream:end -->
| `lost_lease` | A retry or poison statement finalized fewer rows than it was given — the batch outlived its lease |
| `runtime` | Anything else: a failed claim, retention delete, or startup observation; a diagnostics address that could not be bound, or a diagnostics server that stopped or would not join; a publisher builder that failed; and publisher cleanup that timed out or panicked |

The classes are matched on sentinel errors, so a builder decides its own. `config`
covers a builder that returns `postgresoutbox.ErrConfig` — the right answer for a
missing or malformed adapter setting. Every other builder failure, including the
`dial broker` example above, is `runtime`: wrap `ErrConfig` when the operator's
fix is in configuration, and leave it unwrapped when it is not.

Two of these classes are reached only after they repeat. `lost_lease` and
`progress_unknown` end the current cycle, but neither is a problem the stop
prevents: a lost lease means another relay already owns those events, and
unknown progress means lease recovery will republish one — a duplicate the
delivery contract above already permits. Stopping on the first occurrence would
spend a whole process restart on one ambiguous event, and a lease that is
persistently too short would halt delivery instead of slowing it. So the relay
absorbs up to two consecutive faults, waits one poll interval, and counts each on
`outbox.relay.operations{operation="finalize",outcome="tolerated"}`; the third in
a row exits with the class above. Any cycle that finalizes cleanly resets the
count, so the exit means consecutive faults rather than a total reached over an
uptime. Alert on the tolerated counter — it is the signal that arrives before the
exit, and a steady rate of it is a lease or replica misconfiguration whether or
not three ever land in a row.

A transient broker outage is in none of these: it produces retries and bounded
`outcome_unknown` quarantine, which are durable progress rather than a stop.

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
about 141 MiB and a typical payload far less — a 4 KiB event is about 2 MiB.

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
retry/quarantine progress. Liveness remains process-only.

A sample is fresh for two configured observation intervals. A failed periodic
observation retains the last sample and retries; readiness turns false only when
that sample crosses the freshness bound. Startup still fails closed if any of
the events, ordering-head, audit, or commit-receipt relations is unavailable.

On shutdown the process flips readiness false, stops new claims, and lets the
current attempt finish within the drain window. Expiry cancels the attempt. A
publisher that ignores cancellation beyond the one-second join bound makes
cleanup unsafe, so the process does not close dependencies still reachable by
that goroutine. The outer budget is `http.grace_period` — the relay has no HTTP
server of its own beyond diagnostics, but it shares the process-wide shutdown
setting rather than defining a second one. It must cover `outbox.drain_timeout`
plus the relay's two-second outer join and the bounded diagnostics, publisher,
and telemetry cleanup that follow the drain window; startup rejects a
combination that does not, naming `http.grace_period`. A separate Publisher cleanup callback is supervised for five
seconds; timeout or panic is reported instead of hanging process termination.

Metrics expose mutually exclusive counts and oldest timestamps for eligible,
in-progress, retry-wait, recovery-due, ordering-blocked, deterministic poison,
`outcome_unknown`, and retained published rows; table/index bytes;
ordering-head count; observation freshness; audit-ledger and commit-receipt
bytes; last durable progress; an operation counter and a separate
operation-duration histogram; the size of the claimed
batch; and readiness. The two operation instruments do not carry the same set:
an operation with no span of its own — `recovery`, `drain`, `finalize`, the
`skipped` outcome of `publish`, and the
`reconciled` outcome of `mark_published` — reaches the counter only, so the
histogram stays a latency signal instead of absorbing placeholder durations. `outbox.relay.inflight` is that batch, not the events
inside `Publish` right now — it reports how much durable work one lease holds,
which is what a crash would redeliver; `publish_concurrency` is what bounds the
adapter. Attributes are fixed enums. Payload, metadata, credentials, DSN,
event IDs, ordering keys, broker
errors, and SQL text are never metric labels or logs. That holds for driver
error text too, which is why the listener-retry log records only which stage
failed — `connect`, `subscribe`, or `wait`: pgx formats the DSN's user,
database, and host into its connect error.

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
batches. Their compact commit receipts and operator audit rows remain
indefinitely. Pending, leased, retry, recovery, deterministic poison,
`outcome_unknown`, and ordering-high-water rows are not deleted.
High-water rows carry no *automatic* cleanup because proving
that an ordering key can never be reused is domain policy rather than an outbox
decision. `Store.RetireOrderingKeys` is that terminal-key contract, and it is
the only way a high-water row is removed:

```go
// In the same transaction that closes the aggregate.
return outbox.RetireOrderingKeys(ctx, tx, orderID)
```

It runs in the caller's transaction, so the assertion commits atomically with
whatever domain write makes the key terminal. A key that still owns unpublished
events is refused with `ErrOrderingKeyActive` and nothing is retired — not that
key and not the rest of the call. A key that is unknown or already retired
succeeds and changes nothing, so a repeated call is idempotent. Retirement and a
concurrent append for the same key take the same head lock in the same order, so
the append is either visible as pending work that refuses the retirement, or
lands after it and establishes a fresh mark.

After retirement the key's sequence space restarts: a later append is accepted
at any positive sequence, including one already used. That is the protection the
caller trades away by asserting terminality, and it is why the outbox never
infers it. Nothing is time-based — an idle key is not a terminal key, and a
quiet aggregate that wakes up must still have replayed sequences rejected. Watch
`outbox.relay.ordering_heads` to decide whether a service needs the contract at
all: a bounded key space never does.

PostgreSQL is a finite
outage buffer: alert on unpublished
count/oldest age, deterministic poison, `outcome_unknown`, retry errors,
state-observation freshness, drain rate, and relation/index growth. Add partitioning only after measured table/vacuum or
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
