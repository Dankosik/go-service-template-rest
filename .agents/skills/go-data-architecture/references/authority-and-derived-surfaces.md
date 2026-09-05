# Authority And Derived Surfaces

## Load When
Load when a change adds downstream delivery, a projection, an export, a search surface, or a retention rule.

## Decide
- `internal/infra/postgresoutbox` already owns transactional delivery, and [postgres-transactional-outbox.md](../../../../docs/postgres-transactional-outbox.md) owns its contract: append through the caller's `pgx.Tx`, at-least-once with explicit duplicate behavior, one variadic call per business transaction. Reuse it — a second delivery path is a second authority.
- Price `OrderingKey` before setting one. `outbox_ordering_heads` is retained authority rather than evidence: `CleanupPublished` deletes from `outbox_events` only, so a key's rejection of a reused sequence outlives its events, and the [outbox contract](../../../../docs/postgres-transactional-outbox.md) adds no head cleanup because proving a key is finished is domain policy. One head row per key, kept forever, is the price of ordering. Prefer no key, or a key whose cardinality is bounded by something other than row count.
- Name the authoritative writer for each fact before naming any read surface. A projection an operator can edit to fix a record has become a second writer.
- Carry each derived copy's freshness bound, rebuild source, repair path, and cost in the [denormalization ledger](../../../../docs/universal-disciplines/postgres-schema-design/references/relational-design.md); it owns that table. A copy that cannot be rebuilt past its source's own retention has quietly become authority for the tail.
- Delivery ordering and replay belong to `go-distributed`; runtime read paths and caches to `go-db-cache`.

## Reject
- Repairing a drifted projection with a direct write when its source can rebuild it. The write hides the defect that caused the drift and leaves it reachable.
- Disk pressure as the retention trigger. Retention is a per-surface lifecycle decision with an owner, and `outbox_ordering_heads` deliberately has none.

## Prove
Name each fact's writer, each derived copy's rebuild source and staleness bound, and the retention action for every table the change touches — including the tables the change decides *not* to write. Outbox changes prove themselves against the package's own integration tests.
