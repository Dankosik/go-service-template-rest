# Research synthesis

status: ready

Valid as of: 2026-08-02. Refresh before implementation if PostgreSQL target,
the accepted integration base, or Debezium stable documentation changes.

## Decision-changing questions

| Question | Downstream owner | Leading alternatives / falsifier | Evidence boundary |
| --- | --- | --- | --- |
| Which migration runtime and source are canonical? | System Design | Current Goose path vs an unavailable transition; a merged conflicting owner would reopen. | `origin/main` history, migration runtime, source checks, image rehearsal, profile generation. |
| How can a bounded relay claim work safely across replicas and crashes? | Data and System Design | Transactional row claim plus expiring lease; falsified by skipped head rows, lost ownership, or unbounded locks. | PostgreSQL 17 isolation/locking authority plus real-PostgreSQL proof. |
| Polling relay or logical-decoding CDC/Debezium? | System Design | Complete architectures compared on ownership, deployment fit, recovery, privileges, durability, and proof—not implementation convenience. | PostgreSQL/Debezium primary contracts, repository topology, production evidence where contracts are insufficient. |
| What does publication success mean? | Specification / System Design | Publisher returns success only after broker acknowledgement; crash before delivered marking must replay. | Broker-neutral contract plus deterministic crash-window harness; accepted adapter integration is secondary. |
| What ordering can the pack promise? | Specification / Data Design | No global order; optional ordering key needs a durable per-key head rule or must be declared metadata-only. | Claim SQL, concurrent replicas, duplicate/retry behavior, real PostgreSQL. |
| How do retries, poison rows, retention, cleanup, and replay terminate? | Specification / Reliability / Data Design | Finite async retry and terminal poison state; cleanup cannot erase replay or diagnostic obligations. | Failure classification, operator recovery, growth/index evidence, bounded tests. |
| Which signals let an operator distinguish outage, slow drain, poison, and recovery? | Observability Design | Backlog depth plus oldest age, attempts/outcomes, terminal rows, and relay readiness with bounded labels. | Existing OpenTelemetry/Prometheus conventions and async operator decisions. |
| How is `OUTBOX=none` physically pure? | Go Ownership / Test Design | Existing marker-and-delete initializer path; falsified by any retained file, block, command, dependency, or unstable second run. | `scripts/init-module.sh`, template-init harness, module graph, byte snapshot. |

## Established repository baseline

- `origin/main` is `d8b3ee2`. Commit `240de21` is merged into it and makes
  Goose v3.27.3 the sole migration engine. `cmd/migrate` is the production Up
  owner; `internal/infra/postgresmigrate` owns admission, locking, state, and
  bounded execution; repository `migrations/NNNNNN_*.sql` files are canonical.
- The migration policy rejects `NO TRANSACTION`; sqlc consumes the same
  migration directory; real PostgreSQL 17 and image rehearsal are the current
  proof path. The outbox may add a normal transactional migration and must not
  add another engine or startup migration.
- `internal/infra/postgres.Pool.InTx` is the current composition seam. It uses
  `pgx.BeginTxFunc` and hands the callback a `pgx.Tx`, which satisfies the
  repository `Querier` contract.
- The reference service already demonstrates the required feature shape:
  feature-owned `Atomically.Do` plus a PostgreSQL adapter that binds domain and
  event writes to one transaction. Its current event table is only teaching
  scaffolding and is not an outbox implementation.
- The accepted NATS JetStream pack is optional and explicitly states that it
  provides no outbox/inbox or exactly-once behavior. Its producer returns only
  after a JetStream `PubAck`, so a test-only adapter can prove a real broker
  composition without making NATS an outbox dependency.
- `scripts/init-module.sh` validates profile choices before mutating, removes
  whole owned paths, strips shared profile blocks, records the choices in
  `template.lock`, and treats a repeated matching initialization as a no-op.

The migration transition is therefore not an implementation blocker. Sibling
branches carrying patch-equivalent or pre-squash history are evidence only;
they do not override the `origin/main` tree.

One current documentation conflict remains for Go Ownership Design:
`postgres.Pool.InTx` is the runtime's bounded transaction seam, while an older
section of `docs/first-production-feature.md` teaches direct `PGX().Begin` plus
`queries.WithTx`. The selected design must reconcile those two existing paths
and must not add a third transaction abstraction.

## Current primary-source facts

- PostgreSQL 17 documents Read Committed as statement-snapshot isolation and
  row-locking statements as waiting for, then re-evaluating, the updated row.
  Source: https://www.postgresql.org/docs/17/transaction-iso.html
- PostgreSQL 17 documents `SKIP LOCKED` as an intentionally inconsistent view
  suitable for multiple consumers of a queue-like table, not general reads;
  deterministic selection still requires `ORDER BY`.
  Source: https://www.postgresql.org/docs/17/sql-select.html
- Debezium's stable PostgreSQL connector requires logical-replication
  privileges/publications and persistent replication slots, stores offsets
  outside PostgreSQL, can duplicate events after task crashes, and can retain
  WAL while a slot lags. PostgreSQL 17 failover slots require additional
  primary/standby configuration.
  Source: https://debezium.io/documentation/reference/stable/connectors/postgresql.html
- Debezium's Outbox Event Router is a Kafka Connect transformation over CDC;
  it is not an embedded broker-neutral Go relay.
  Source: https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html

## PostgreSQL concurrency and storage implications

Established facts:

- PostgreSQL row locks end with the claiming transaction. A persisted lease
  and unique fencing token are therefore required after claim commit;
  `SKIP LOCKED` alone cannot identify the publisher that still owns a row.
- Claiming must lock, set lease owner/token/expiry, and return the row in one
  short transaction. Publication is outside that transaction. Finalization is
  a token-guarded compare-and-set; zero affected rows is lost ownership, never
  permission to mark another relay's work delivered.
- Crash after claim delays recovery until expiry. Crash after broker success
  and before durable finalization necessarily permits a duplicate. An
  ambiguous finalization commit is verified or left retryable; it is never
  converted into assumed success and deletion.
- `now()` is transaction-start time. Claim/lease SQL must use a server timestamp
  whose semantics fit a short claim/finalize statement and must not compare
  application clocks across replicas.
- A partial pending index may exclude terminal rows, but cannot use a volatile
  `now()` predicate. Lease-field updates add index churn and can inhibit HOT
  updates; the pending/claim access path and cleanup path need separate indexes
  tied to actual predicates.
- UPDATE/DELETE churn leaves dead tuples for vacuum. Cleanup deletes only
  terminal published rows older than retention in bounded batches. Poison rows
  remain operator-visible until their own explicit policy is satisfied.
- PostgreSQL provides no universal partition threshold. V1 remains
  unpartitioned. Reopen only on measured relation/index size, dead tuples,
  autovacuum lag, WAL/I/O, claim-plan degradation, or cleanup-budget breach.

Primary sources:
https://www.postgresql.org/docs/17/explicit-locking.html,
https://www.postgresql.org/docs/17/functions-datetime.html,
https://www.postgresql.org/docs/17/indexes-partial.html,
https://www.postgresql.org/docs/17/storage-hot.html,
https://www.postgresql.org/docs/17/routine-vacuuming.html,
https://www.postgresql.org/docs/17/ddl-partitioning.html.

Carried proof gaps:

- Lease and retention durations are reliability/operator policy values; the
  engine cannot supply them. Defaults need bounded rationale and remain
  configurable.
- Real PostgreSQL must prove disjoint claims, expiry reclaim, stale-token
  rejection, deterministic per-key head behavior, ambiguous commit handling,
  cleanup bounds, and representative pending/terminal query plans.
- Production durability assumes logged tables and ordinary durable commit.
  A live rollout must verify `synchronous_commit`/`fsync` posture; the pack must
  never use an UNLOGGED outbox.

## Polling and logical-CDC candidate map

Both families can preserve state-plus-intent atomicity and both remain
at-least-once. Their post-commit progress authority differs:

| Driver | Bounded PostgreSQL polling | Logical CDC / Debezium |
| --- | --- | --- |
| Progress owner | Outbox row state and lease token. | WAL, one replication slot, and a separately durable connector offset. |
| Replicas | Active/active row claimers. | One active receiver per slot; replicas are active/passive. Multiple slots duplicate the stream. |
| Privileges/topology | Ordinary table DML and the existing Go/PostgreSQL deployment. | Logical WAL, replication identity/login, publication/slot privileges and capacity, durable offsets, provider/failover configuration, and normally another runtime. |
| Duplicate window | Broker ack to delivered-row finalization. | Broker ack to offset/confirmed-LSN finalization; a batch or crash rewind can replay multiple records. |
| Poison isolation | Per-row availability, attempts, terminal state, and unrelated-row progress. | A linear offset needs durable parking/DLQ semantics before it can advance past poison without loss. |
| Outage storage pressure | Table/index growth, vacuum, and bounded cleanup. | Replication-slot WAL retention; a finite WAL cap can invalidate continuity. |
| Schema/rollout | Migration and Go reader/writer compatibility. | Adds publication/filter, connector transform, snapshot, offset, sink, and failover-slot compatibility. Logical replication does not carry DDL. |
| `OUTBOX=none` | Remove repository-owned pack surfaces. | Also requires external publication/slot/offset/runtime cleanup, which initialization cannot own. |

Polling fits the current repository because it reuses PostgreSQL 17, pgx, the
separate Go worker lifecycle pattern, active/active replicas, broker-neutral
`Publisher`, and the literal lease/crash proof required by the accepted
outcome. This is local-fit evidence, not a convenience argument.

CDC remains viable only if a later accepted environment supplies logical-WAL
and failover-slot capability, a connector/offset/runtime owner, active/passive
replica semantics, durable poison parking, and provider drills. Debezium Kafka
Connect is Kafka-coupled; Debezium Engine/Server adds a JVM and durable offset
store; custom Go `pgoutput` would make this template own protocol parsing,
keepalives, LSN durability, reconnect/failover, and compatibility.

Candidate-space stop: no third family changes the live ownership level.
Database triggers, listen/notify, or direct broker dual writes may complement a
poller but do not replace durable progress/recovery; custom CDC is the same
logical-decoding family with more local ownership.

Primary sources:
https://www.postgresql.org/docs/17/logicaldecoding-explanation.html,
https://www.postgresql.org/docs/17/logical-replication-security.html,
https://www.postgresql.org/docs/17/logical-replication-failover.html,
https://www.postgresql.org/docs/17/view-pg-replication-slots.html,
https://debezium.io/documentation/reference/3.6/development/engine.html,
https://debezium.io/documentation/reference/3.6/configuration/storage.html,
https://debezium.io/documentation/reference/3.6/connectors/postgresql.html.

## Failure and recovery implications

The supported claim is conditional at-least-once durable publication, not
unconditional eventual delivery: PostgreSQL and the broker must recover, a
relay must keep running, retention must preserve unfinished work, and poison
must receive explicit operator disposition.

| Window | Required durable result |
| --- | --- |
| Domain/outbox transaction fails | Neither mutation nor intent exists. An ambiguous commit is not retried as a new business occurrence without command idempotency. |
| Broker definitely rejects | Mark durable terminal poison; retain identity, payload, attempt and bounded failure class; never mark delivered. |
| Broker ack is lost/ambiguous | The broker may own the event. Retry later with the same logical event ID; do not generate a new occurrence. |
| Broker succeeds; relay crashes before finalization | Lease expiry causes replay and a duplicate. This is the mandatory crash-window proof. |
| Delivered-mark commit is ambiguous | Verify state after reconnect or leave retryable. Never assume success and delete. |
| Repeated transient outage | One relay-owned retry layer uses a bounded publish attempt and capped exponential backoff with jitter. Attempts are finite. |
| Exhaustion | Durable poison is visible and not delivery. For an ordered key it blocks later sequence values; unrelated keys continue. |
| Redrive/replay | Preserve logical event ID and envelope version; bound/rate-limit and audit the operator action. Broker dedupe windows are not replay correctness. |
| Cleanup | Delete only delivered terminal rows by delivered time, never pending, leased, or poison rows; preserve the accepted investigation/redrive window. |
| Rollback | Stored rows outlive binaries. Schema/envelope changes remain additive/version-compatible until backlog plus retention closes; rollback does not drop a non-empty outbox or retract published events. |
| Broker outage | API success means domain plus intent committed, not broker delivery. PostgreSQL is a finite outage buffer; backlog count/bytes/oldest age and drain rate govern capacity. If intent cannot be stored, domain mutation rolls back. |

No global order is promised. When an event supplies both ordering key and
monotonic sequence, the database claim path preserves head-of-line order for
that key; poison therefore blocks that key until explicit repair. The selected
broker adapter must preserve the same key semantics before an end-to-end
ordering claim is made. Events without that pair are independently claimable.

Primary and credible operational sources:
https://www.rabbitmq.com/docs/reliability,
https://docs.nats.io/learn/jetstream/publishing,
https://microservices.io/patterns/data/transactional-outbox.html,
https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md,
https://zapier.com/blog/lessons-from-using-outbox-pattern-at-scale/.

## Production Go implementation evidence

No surveyed dependency is a clean drop-in. The materially distinct maintained
families are SQL-offset forwarding (Watermill), committed leased polling
(Dataddo PGQ), durable job-state rescue (River), and WAL CDC (Debezium).

- Watermill SQL v4.1.5 can insert through native `pgx.Tx`, but its subscriber
  holds a database transaction/lock while the broker handler runs and ACKs,
  serializes broadly, and does not supply this contract's finite durable poison
  state. Reuse the transaction-bound insert and explicit publish-success
  boundary; reject the long transaction and unbounded NACK behavior.
- Dataddo PGQ main `ef03827` proves the short `SKIP LOCKED` claim, committed
  `locked_until`, outside-transaction handling, and later ACK/NACK pattern. Its
  publisher cannot join the repository's pgx transaction, its lock has no
  fencing token, and exhausted rows can simply become unclaimable.
- River v0.42.0 proves native pgx `InsertTx`, ordered `SKIP LOCKED` claims,
  explicit retry/rescue, retention cleanup, and finite lifecycle. It is a full
  job runtime with its own schema/migration/plugin surface; its timeout rescue
  explicitly permits duplicates and is not fencing.
- Debezium 3.6 is the distinct insert-only WAL family already dispositioned
  above; it moves progress and failure ownership into a slot/offset/connector
  topology.

Reuse only the common mechanics: caller-owned transactional insert, stable
logical ID, short committed claim, lease plus fencing, broker success before
progress commit, finite attempt/poison state, separate bounded cleanup,
graceful stop, and backlog/age/outcome telemetry. A repository-native package
is smaller than importing a second migration/runtime owner and still requires
less new correctness code than adapting any surveyed library's conflicting
state machine.

Sources and pinned revisions:
https://github.com/ThreeDotsLabs/watermill-sql/releases/tag/v4.1.5,
https://github.com/ThreeDotsLabs/watermill/releases/tag/v1.5.2,
https://github.com/dataddo/pgq/commit/ef03827ef6679fb1545e3b93bb758cd9276964d7,
https://github.com/riverqueue/river/releases/tag/v0.42.0,
https://debezium.io/releases/3.6/.

## Operator evidence contract

Authoritative outbox state—not estimated PostgreSQL statistics—answers backlog
questions. The relay exports low-cardinality state snapshots and stage
outcomes; entity IDs stay out of metrics.

| Operator question | Bounded signal | Completion/action |
| --- | --- | --- |
| Falling behind? | Fresh count and oldest timestamp for eligible, in-progress, retry-wait, poison, and published-retained states. | Compare accepted backlog/age policy, then split by relay stage. |
| Alive but not progressing? | Unpublished rows, last durable-progress timestamp, and claim/publish/finalize outcome rates. | Restore the failing stage; broker-call return alone is not progress. |
| Poison/retry exhaustion? | Poison count/oldest timestamp plus one entity-level terminal log. | Correct cause, then use an explicit auditable redrive; never delete silently. |
| Crash recovery working? | Recovery-due state, recovery outcomes, falling recovered count, and resumed finalization. | Treat recovered ambiguous publishes as duplicate-risk evidence. |
| Cleanup/storage pressure? | Published-retained count/age, cleanup outcomes, table/index size, dead tuples, and vacuum state. | Use only bounded data-owner maintenance. |
| Telemetry stale? | Last successful observation timestamp and observation errors. | Repair observation before trusting a frozen zero/backlog value. |
| Shutdown complete? | Readiness transition, stop-claim, in-flight drain, forced-cancel outcome, telemetry flush. | Process exit alone is insufficient; later recovery proves stranded work. |

Stable V1 instruments use an `outbox.relay.*` scope for state count, oldest
timestamp, observation timestamp, operations, operation duration, in-flight,
last durable progress, and readiness. Attributes are bounded state, operation,
outcome, and error-class enums. Database-global gauges exported by every relay
are aggregated with `max` after freshness filtering, not summed; process-local
counter rates are summed.

Lifecycle policy: liveness is process-local. Readiness fails until
configuration/schema, PostgreSQL, a non-noop Publisher, the relay loop, and the
first state observation are admitted; it becomes false before drain or when a
required work dependency fails. Backlog age, poison count, or exporter failure
alone does not cause restart/readiness churn when the replica can still work.

Sources:
https://prometheus.io/docs/practices/instrumentation/,
https://prometheus.io/docs/practices/naming/,
https://prometheus.io/docs/practices/alerting/,
https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/,
https://opentelemetry.io/docs/specs/semconv/messaging/messaging-metrics/,
https://www.postgresql.org/docs/17/monitoring-stats.html.

## Downstream disposition

- System Design receives bounded polling as the selected candidate unless the
  independent challenge exposes an uncovered owner/correctness failure.
- Specification receives atomic state-plus-intent, conditional at-least-once,
  stable duplicate identity, no global order, optional strict per-key head
  semantics, finite visible poison, non-destructive retention, and API/broker
  decoupling.
- Go Ownership receives the existing `Pool.InTx`/sqlc/documentation conflict,
  the separate-process lifecycle pattern, and the no-dependency conclusion.
- Test Design receives every real-PostgreSQL, crash-window, ambiguity,
  concurrency, ordering, poison, lifecycle, observability, migration, and
  profile-purity obligation named above.

## Independent challenge disposition

Verdict: `CONCERNS`, dispositioned 2026-08-02.

The reviewer correctly found that database state and relay metrics cannot
detect a publisher that returns success without durable broker acceptance.
That concern is carried as a hard cross-phase closure obligation:

- Specification owns the rule that `Publisher.Publish` may return nil only
  after the selected broker has durably acknowledged the same event identity.
  Noop, drop, log-only, in-memory, or fire-and-forget implementations are not
  publishers under this contract.
- System Design owns fail-closed process composition: a missing publisher
  prevents startup, and no generic fallback may return success.
- Test Design owns adapter conformance and event-ID conservation: for every
  production adapter accepted in an integration base, a real broker must show
  that the same event ID is durably present before relay finalization is
  allowed. The deterministic fake is valid only for local relay failure-window
  proof and may not establish production adapter conformance.
- Reopen Specification/System Design and block publication-safety completion if
  any executable can finalize through an adapter without that proof.

With this owner, observable, and reopen condition recorded, the concern does
not leave research-owned evidence unresolved. Specification may proceed.
