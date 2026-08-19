# PostgreSQL Transactional Outbox

Select the complete pack with:

```bash
DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream make template-init \
  MODULE=github.com/acme/orders CODEOWNER=@acme/backend
```

`OUTBOX=postgres` requires both PostgreSQL and NATS JetStream. The generator
rejects an incomplete selection instead of producing a relay with no publisher.
`OUTBOX=none`, the default, removes the event contract, River appender and
worker, migration, command, tests, and this document.

## Contract

The pack guarantees at-least-once publication:

1. The feature's PostgreSQL adapter mutates business state and calls
   `Appender.Append` through the same caller-owned `pgx.Tx`.
2. River inserts one `publish_domain_event` job in that transaction. A rollback
   removes both writes; a commit exposes both.
3. `cmd/outbox-relay` works the River job through the concrete NATS producer.
4. A JetStream acknowledgement followed by process loss may run the job again.
   Every attempt uses the same logical event ID for both `Message-Id` and
   `Publication-Id`.

Consumers must make non-idempotent effects duplicate-safe. The pack does not
provide exactly-once delivery or generic ordering.

River owns job state, concurrency, retry scheduling, crash rescue, cleanup,
operator retry, and maintenance. The template owns only typed event encoding,
atomic insertion, subject routing, NATS mapping, and process composition.

## Append a typed event

Create the immutable event once, before any transaction callback that may be
retried:

```go
event, err := domainevent.New(
    eventID,
    "order.updated",
    1,
    occurredAt,
    order.UpdatedV1{
        OrderID:  orderID,
        Revision: revision,
    },
)
if err != nil {
    return err
}
```

Build the appender once in the service composition root. This is where the
service owns its event-to-subject mapping; feature code never receives the
subject:

```go
outbox, err := natsjs.NewOutboxAppender(
    cfg.Messaging.MaxPayloadBytes,
    natsjs.OutboxRoute{
        Type: "order.updated", Version: 1, Subject: "events.orders",
    },
)
if err != nil {
    return err
}
```

The PostgreSQL adapter may declare only the method it consumes:

```go
type outboxAppender interface {
    Append(context.Context, pgx.Tx, domainevent.Event) error
}
```

Its transaction contains the exact business call:

```go
return postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
    if err := orders.Update(ctx, tx, change); err != nil {
        return err
    }
    return outbox.Append(ctx, tx, event)
})
```

The template does not construct this appender in `cmd/service`: it ships no
business event or repository that could consume it.

## Event shape

`domainevent.Event` carries only:

- one stable logical ID;
- event type and positive schema version;
- UTC occurrence time;
- JSON encoded from the typed payload.

It carries no source, broker subject, arbitrary metadata, ordering key,
publication-attempt identity, retry policy, or trace carrier. The appender
resolves the subject, enforces the configured NATS payload bound, and stores the
payload bytes unchanged inside River's JSON job args.

Reusing an event ID with the same immutable job is idempotent while River retains
that job. Reusing it with different bytes returns
`postgresoutbox.ErrEventIDConflict`.

## Trace continuity

The River OpenTelemetry plugin stores W3C `traceparent` and `tracestate` in
job metadata. River's work span links to the producing operation; the NATS
outbox worker additionally restores that original context before calling the
producer, so the broker message retains the producing trace rather than the
relay process's local trace. Missing or malformed trace metadata never blocks
publication.

## Runtime and operations

`cmd/outbox-relay` runs one River queue named `outbox`. It uses at most 16
workers and never exceeds `messaging.max_pending_publishes`. River polling is
used instead of `LISTEN` because this repository applies a finite
`statement_timeout` to every pooled PostgreSQL connection; a long-lived
`LISTEN` session would be cancelled by that safety bound.

There are no `APP__OUTBOX__*` settings. River owns its retry and rescue
defaults. Cancelled and discarded jobs are retained indefinitely so unpublished
intent cannot disappear through cleanup; completed jobs use River's normal
retention. The existing process grace, PostgreSQL, messaging, and diagnostics
settings own lifecycle and capacity.

Use River's job list/retry and queue pause APIs from an authenticated,
service-owned operator tool when needed. The template ships no generic admin
endpoint and performs no automatic discard.

## Schema and upgrades

`migrations/000008_river.sql` installs the shared River v0.44.0 PostgreSQL
schema after the retained legacy capability migrations. Its
`river_migration` ledger records main versions 1 through 7, so River's migrator
starts from the same baseline. Upgrade River modules together and append the
matching upstream migration delta; never edit an applied generated-service
migration.

The former `outbox_events`, `outbox_ordering_heads`, receipt, and redrive tables
remain for rollback but are not read by the River worker. A service that
deployed the older pack must drain or explicitly bridge its remaining rows
before switching workers. Dropping the legacy tables or pending rows is a
separate authorized production action.

## Proof

```bash
go test -vet=off ./internal/domainevent ./internal/infra/postgresoutbox \
  ./internal/infra/natsjs ./cmd/outbox-relay/...
go test -vet=off -tags=integration ./test -run '^TestPostgresOutbox' -count=1
make sqlc-check migration-check template-init-check
```
