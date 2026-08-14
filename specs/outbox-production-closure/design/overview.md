# Design: outbox production closure

status: ready
Realizes `../spec.md` and composes with
`../../outbox-trace-continuity-and-key-lifecycle/design/overview.md`.

## Selected system

Keep the existing transactional outbox, relay state machine, NATS client, and
profile remover. Add only the missing seams:

- the selected NATS profile supplies one concrete `postgresoutbox.Publisher`;
- the outbox row carries one sticky publication-uncertainty fact;
- the existing redrive audit becomes action-aware and durable across event
  cleanup;
- the outbox event ID also keys a compact commit-receipt ledger; and
- writer-primary reconciliation, bounded legacy classification, unknown
  quarantine, confirm-accepted, and redrive-unknown remain concrete
  `postgresoutbox.Store` operations.

No generic broker interface, command-ID subsystem, quarantine table, consumer
ordering mechanism, second attempt limit, or relay-specific NATS client is
introduced. The current `Publisher` interface, `max_attempts`, event identity,
PostgreSQL writer pool, ordering-head locks, and profile markers are sufficient.

The real forks collapse as follows:

| Decision | Selected | Why the smaller alternatives fail |
| --- | --- | --- |
| Durable commit evidence | Compact receipt keyed by `Event.ID` | `outbox_events` alone is not stable because normal cleanup deletes published rows; retaining whole events forever duplicates large payloads |
| Unknown state | Existing parked row plus one sticky boolean | A state enum duplicates published/poison/lease timestamps; a quarantine table creates a second ordering and cleanup authority |
| Operator audit | Extend `outbox_redrives` in place | Separate action tables cannot enforce one global audit identity; renaming the table breaks old tooling without changing behavior |
| Legacy classification carrier | One explicit `outbox-relay --classify-legacy-uncertainty` mode over canonical Store/SQL owners | Claim-only classification cannot close the pre-start zero gate; raw operator SQL duplicates query authority; automatic relay startup can publish before zero readback; another binary duplicates config and PostgreSQL composition |
| Broker adapter | Concrete adapter in `internal/infra/natsjs` | It can reuse the package's private envelope validation; a universal adapter package or new public NATS error type has no second consumer |
| Consumer ordering | None | The accepted authority ends at publication and no domain contract requires generic handler serialization |

Reopen the receipt choice only if an accepted feature-owned domain operation
receipt already provides the same writer-primary presence and immutable-value
conflict check. Reopen the NATS lifecycle choice only if reconnect exhaustion
must terminate the relay rather than remain durable backlog; the current spec
requires durable recovery, not that process-exit policy.

## Durable data authority

`migrations/000001_postgres_outbox.sql` remains the canonical template schema.
The template edits it in place because no template database has applied it;
generated services must port this design through new forward migrations.
`internal/infra/postgres/queries/postgres_outbox.sql` is the statement authority,
and SQLC output remains derived.

### Sticky uncertainty and quarantine

Add `outbox_events.publication_uncertain boolean DEFAULT false`. It is nullable
only so a forward migration can represent unclassified legacy history:

- `false` means no publication ambiguity has been observed;
- `true` means ambiguity has been observed and is never cleared; and
- legacy `NULL` means `true` for an unpublished row with an attempt or expired
  lease, otherwise `false`.

Unknown quarantine is not another stored state. It is the existing terminal
parked row with all of:

```text
published_at IS NULL
poisoned_at IS NOT NULL
publication_uncertain IS TRUE
last_error_class = 'outcome_unknown'
```

Deterministic and attempt-exhausted poison require
`publication_uncertain IS FALSE`. Published rows may retain `true`; a durable
acknowledgement or audited confirmation resolves the outcome without erasing
the historical fact. No cleanup statement can select an unpublished row, so an
unknown row remains durable and continues to block its ordering key.

Claim is the recovery boundary. In the existing claim statement, an expired
lease becomes uncertain because dispatch may already have occurred. Legacy
attempted rows become uncertain before another publish. A sticky row whose
pre-claim cycle count has reached `max_attempts` is quarantined in that same
statement and is not leased or returned to the publisher. This preserves one
claim round trip and prevents a restart from making an extra broker call.

Rollout classification uses the same policy before claim is admitted.
`Store.ClassifyLegacyUncertainty(ctx, maxAttempts, batchSize) (int, error)`
executes the canonical `ClassifyLegacyOutboxUncertainty` SQLC statement on the
configured writer pool. The statement selects at most `batchSize` rows whose
sticky fact is `NULL`, ordered by `created_at, id`, and locks them `FOR UPDATE`
without `SKIP LOCKED`. For this statement, attempted means
`total_attempt_count > 0`, leased means `lease_token IS NOT NULL`, and at-limit
means `cycle_attempt_count >= maxAttempts`. It atomically maps every published
row, plus an unpublished row with none of attempted/leased/poisoned, to `false`;
maps every other unpublished row to `true`; and makes a mapped-true row
canonical `outcome_unknown` when it is already poisoned or at-limit. That
terminal transition sets `poisoned_at` if absent, clears both lease fields, and
sets `last_error_class`; a below-limit mapped-true row and every mapped-false
row preserve all other delivery fields. It never creates a receipt, changes an
event identity, attempt count, availability, or ordering head, or clears an
already-classified sticky fact.

The nullable-to-boolean transition is the durable restart cursor, so there is
no second checkpoint table or external tuple cursor. Each successful statement
commits one batch; interruption leaves prior batches classified, and rerunning
continues from the remaining `NULL` predicate. A returned zero is authoritative
only after rollout Gates 1-3 have stopped every old writer, relay, and operator
action; new receipt-writing writers create non-`NULL` rows. Omitting
`SKIP LOCKED` makes an unexpected live locker fail under the configured
statement budget instead of producing a false zero.

The exact carrier is the existing relay binary's one-shot
`--classify-legacy-uncertainty` mode. It loads the ordinary validated outbox and
PostgreSQL config, opens only the writer pool and Store, and loops the method
with `outbox.max_attempts` and the existing bounded maintenance batch
`outbox.cleanup_batch_size` until zero. It does not build a publisher, connect
to NATS, start diagnostics, claim, or report ready; cancellation exits non-zero
after the current statement is canceled. Normal relay startup without that
flag still requires a registered publisher and therefore preserves outbox-only
fail-closed behavior.

`ClaimedEvent` gains `PublicationUncertain bool`. The operator `Record` gains
`PublicationUncertain bool` and `OutcomeUnknown bool`; the latter is a mapped
view of the canonical row predicate, not separately stored state.

For a claimed event, finalization uses this fixed precedence:

| Result and history | Store transition |
| --- | --- |
| Durable acknowledgement | Existing published transition; preserve sticky bit |
| Permanent failure, never uncertain | Existing deterministic poison |
| Not accepted below the limit, never uncertain | Existing retry schedule |
| Not accepted at the limit, never uncertain | Attempt-exhausted poison |
| Ambiguous/default failure below the limit, never uncertain | Retry and set sticky bit |
| Sticky; current failure is non-permanent and below the limit | Retry and preserve sticky bit |
| Sticky plus permanent failure at any attempt | Unknown quarantine |
| Any ambiguous/default failure at the limit, or any failure at the limit after uncertainty | Unknown quarantine |

`ScheduleOutboxRetryBatch` therefore gains an ambiguity input and can only move
the sticky bit toward `true`. The existing lease token and lease-expiry fences
remain on every automatic finalization statement. The relay's detached
finalization context and per-key publication admission remain unchanged.

### Commit receipts

Add `outbox_commit_receipts` with exactly:

```text
event_id               text primary key
fingerprint_version    smallint, value 1
envelope_fingerprint   bytea, exactly 32 bytes
```

The table has no payload, trace context, status, foreign key to the cleanup
target, or automatic retention. Each append statement inserts the receipt and
event together, including the ordered statement, so one `Append` is still one
statement and one round trip. A rejected ordering sequence inserts neither.

The fingerprint uses Go's standard-library SHA-256. Version 1 hashes the exact
bytes below:

1. ASCII `postgresoutbox-receipt-v1` followed by one zero byte.
2. In order: ID, type, source, destination, schema, occurrence time, payload,
   metadata, ordering key, ordering sequence.
3. Each value is preceded by its unsigned 64-bit big-endian byte length.
4. Text and JSON values are their exact stored bytes; absent metadata first
   normalizes to `{}`. Occurrence time is UTC Unix microseconds encoded as a
   signed 64-bit big-endian integer. Ordering sequence uses the same signed
   64-bit encoding. Trace context is excluded because it is outbox-owned rather
   than part of the caller's immutable event.

Golden vector: ID `evt-1`, type `order.created`, source `orders`, destination
`orders.events`, schema `v1`, occurrence `2026-08-08T00:00:00Z` (Unix
microseconds `1786147200000000`), payload `{"id":"order-1"}`, metadata `{}`,
ordering key `order-1`, sequence `7` hashes to
`e5ab0fe21fc3ae1c8f28d7ad5603bacebc8f202a85b0ce4c54424258b4546d9a`.
One shared Go encoder owns append and reconciliation; SQL stores and compares
the result and does not restate the encoding.

`Store.ReconcileCommit(ctx, event)` computes that fingerprint and performs one
read on the store's existing writer pool. The statement returns:

- `Applied` when the receipt exists with version 1 and the same fingerprint;
- `NotApplied` only when the receipt is absent and the same statement proves
  `pg_is_in_recovery() = false` and `transaction_read_only = off`;
- `ErrReceiptConflict` when the ID exists with different immutable evidence;
  and
- `StillUnknown` with the read error when PostgreSQL, the authority check, or
  the version is unavailable or inconclusive.

The repository already rejects multi-host/fallback DSNs. The remaining bounded
assumption is that its single configured endpoint is the deployment's current
writer. A writable fork or split brain reopens platform/distributed design;
PostgreSQL cannot prove global topology from a local statement.

The exact Go surface is:

```text
type CommitOutcome uint8
const (
    CommitStillUnknown CommitOutcome = iota
    CommitApplied
    CommitNotApplied
)
func (s *Store) ReconcileCommit(context.Context, Event) (CommitOutcome, error)
var ErrReceiptConflict error
```

The zero outcome is `CommitStillUnknown`, so an uninitialized result cannot
authorize a mutation retry. Invalid events fail before the read. An unavailable
authority returns `CommitStillUnknown` plus its contextual error; conflict
returns `CommitStillUnknown` wrapping `ErrReceiptConflict`.

The event ID must be created before the caller enters `Pool.InTx`, and the same
`Event` must be retained across reconciliation and any not-applied retry. No
caller may retry the mutation on `StillUnknown` or mint a replacement ID.
`ReconcileCommit` is valid only for an append attempted by the receipt-writing
version; pre-cutover transactions have no live commit-reconciliation caller and
are not assigned synthetic receipts. This is a forward guarantee, not a claim
that already-cleaned historical IDs can be reconstructed.

The concrete PostgreSQL repository adapter owns the caller loop, not `Store`.
It builds the stable `Event` before `Pool.InTx`, invokes the domain mutation and
`Append` once, and only when `InTx` returns `postgres.ErrCommitUnknown` calls
`ReconcileCommit`: `Applied` returns the original success, `NotApplied` retries
the same mutation with the same event, receipt conflict fails permanently, and
`StillUnknown` retries reconciliation only. The template's worked guidance and
reference-service integration adapter carry this orchestration because the
template has no production feature repository to own it. No generic retry or
unit-of-work abstraction is added.

One compact receipt is retained per append and fingerprinting is one linear
pass over the already-bounded envelope. There is no evidence for another index,
cache, or cleanup path. Reopen performance only if measured append latency or
receipt-table growth breaches a service-owned budget; cleanup also reopens the
stable-receipt lifetime in Specification.

### Audited recovery

Keep the physical table name `outbox_redrives`, but add `action_kind` with the
closed values `redrive_poison`, `redrive_unknown`, and `confirm_accepted`.
`cycle_number` is nullable only for confirmation. Remove the cascading event
foreign key so an audit identity survives event cleanup; `audit_id` remains the
single global primary key.

Keep the existing `Store.Redrive` for deterministic and attempt-exhausted
poison. Add concrete `Store.RedriveUnknown` and `Store.ConfirmAccepted`
operations rather than one public action enum. All three use the same
transactional sequence:

1. Read the audit identity first so a replay can succeed after event cleanup.
2. If no committed audit exists, lock the event row `FOR UPDATE`.
3. Insert/reserve the audit identity; on conflict, re-read it.
4. Return success only for the same action and event, audit conflict for any
   other use, and otherwise validate and perform the state transition.

The same event lock serializes competing actions. `RedriveUnknown` clears the
parked state, lease, last attempt class, and cycle attempt count, increments the
existing redrive count, and preserves `publication_uncertain=true` and the same
event ID. `ConfirmAccepted` calls no broker and performs the same unordered or
ordered finalization as a durable acknowledgement, including locking the head,
advancing its high-water mark, and releasing the successor. Its audit fence is
different from the relay's lease fence, so the two SQL statements remain
separate and share parity proof instead of a synthetic lease.

Existing redrive error compatibility remains through aliases to the generalized
operator state-conflict and audit-conflict sentinels. No public generic action
type is added.

> Superseded: the aliases were removed. They held the same two error values as
> the generalized sentinels, so `errors.Is` could not separate them and every
> caller had to be warned not to switch on both. This is a template, so the
> compatibility surface had no adopter to keep: nothing outside the repository
> consumed the old names. `ErrOperatorStateConflict` and
> `ErrOperatorAuditConflict` are the only names now.
> Reopen if a released, externally consumed build ever ships the old names.

The exact additions are
`RedriveUnknown(context.Context, string, string) error` and
`ConfirmAccepted(context.Context, string, string) error` on `*Store`, plus
`ErrOperatorStateConflict` and `ErrOperatorAuditConflict`.

## NATS publication and trace flow

The adapter lives in `internal/infra/natsjs`, converts the existing fields, and
uses `Event.ID` for both `MessageID` and `PublicationID`. It pre-validates with
the package's existing private NATS envelope validator:

- pre-dispatch immutable validation failure becomes
  `postgresoutbox.ErrPermanentPublication`;
- any later `natsjs.ErrRejected` becomes
  `postgresoutbox.ErrPublicationNotAccepted`;
- `natsjs.ErrAmbiguous` and unknown errors pass through and are ambiguous by
  the relay's default rule; and
- only the JetStream `PubAck` returns `nil`.

Before calling `Producer.Publish`, the adapter removes the relay-attempt span
from the context while preserving cancellation and deadline, then extracts the
stored W3C creation carrier. The NATS producer span is therefore a child of the
original operation and injects that continuity for the worker. Empty or invalid
stored context creates an unrelated producer root and never rejects delivery.
The broker-neutral relay attempt remains its own root linked to the creation
context as fixed by the sibling trace design.

The adapter type stays unexported. Its only cross-package surface is
`NewOutboxPublisher(*Producer) postgresoutbox.Publisher`; a nil producer returns
a nil interface so the existing `postgresoutbox.ValidatePublisher` fails closed.
The unexported `outboxPublisher.Publish(context.Context,
postgresoutbox.Event) error` is the sole mapping implementation. Source and
metadata remain intentionally absent because the NATS event contract does not
carry them.

`cmd/outbox-relay/internal/bootstrap.BuildNATSPublisher` reuses
`natsjs.Connect` with `RoleProducer` and returns one bootstrap-local runtime
carrier containing the concrete publisher plus `Client.Run`, `Client.Ready`,
and `Client.Shutdown`. Reconnect reprobes and `ErrTerminal` are part of the
existing NATS contract, so the relay process supervises them rather than
reducing the client to synchronous publishes.

The exact bootstrap shape is:

```text
type PublisherRuntime struct {
    publisher postgresoutbox.Publisher
    run       func(context.Context) error
    ready     func() bool
    shutdown  func(context.Context) error
}
func NewPublisherRuntime(
    postgresoutbox.Publisher,
    func(context.Context) error,
    func() bool,
    func(context.Context) error,
) (PublisherRuntime, error)
type PublisherBuilder func(context.Context, config.Config, *slog.Logger) (PublisherRuntime, error)
func BuildNATSPublisher(context.Context, config.Config, *slog.Logger) (PublisherRuntime, error)
```

All four components are required after a successful build. Fields stay private;
`NewPublisherRuntime` validates the publisher and functions and is the only way
the NATS builder or a derived service's custom outbox-only adapter constructs
the value; invalid input wraps the existing `postgresoutbox.ErrConfig` rather
than adding a bootstrap sentinel. The exported concrete carrier preserves that
registration seam but is lifecycle data, not a second publisher abstraction;
`postgresoutbox.Publisher` remains the only publication interface.

`runRelayLifecycle` starts the runtime's `run` beside the relay. Diagnostics
readiness is `relay.Ready() && runtime.ready()`: a reconnect makes the
process unready until `Client.Run` reprobes the stream, while durable outbox
retry continues. A publisher supervisor return without process cancellation is
a process fault; `natsjs.ErrTerminal` drains/stops the relay and exits under the
new bounded class `messaging_terminal` instead of consuming attempts forever
against a closed client.

Shutdown order is fixed: stop relay claims, let the current batch finish within
the existing drain budget, cancel and join the NATS supervisor, stop
diagnostics, then call the error-returning `Client.Shutdown` before closing the
PostgreSQL pool. A supervisor that does not join marks cleanup unsafe just like
an unjoined publisher call; shutdown errors join the process result rather than
being discarded. Startup connection/topology admission and the current forced
join/cleanup budgets remain authoritative.

The bootstrap must repeat the existing typed-config-to-`natsjs.Config` mapping
because composition packages cannot import each other. `internal/config` owns
the policy and its existing cross-package validation corpus gains this builder
as another forced representation. Add the table-only shared corpus at
`internal/config/configtest/messaging.go`; both the existing worker parity test
and the new relay-local parity test load those cases, perform their own mapping,
and call the real `natsjs.ValidateConfig`. The helper imports no runtime adapter;
the bootstrap file owns no new defaults or validation rule.

`cmd/outbox-relay/main.go` defaults its builder to nil and assigns the NATS
builder only inside the existing messaging profile marker. Removing messaging
therefore preserves outbox-only fail-closed startup. Removing outbox deletes the
NATS adapter and relay bootstrap additions, so the NATS-only producer and worker
retain their functional behavior.

Deletion is explicit, not inferred from block stripping. `OUTBOX=none` removes
`internal/infra/natsjs/outbox_publisher.go` and its test, the reference
reconciliation exemplar, every outbox integration including the joined NATS
test, and the existing relay/outbox trees and migrations. `MESSAGING=none`
removes the whole NATS package, the joined test, and
`cmd/outbox-relay/internal/bootstrap/natsjs_publisher.go` plus its test, while
also removing `internal/config/configtest/messaging.go` and stripping only the
builder assignment so nil remains. All profile removals run
before SQLC regeneration; the combined profile retains every one of these
paths. `scripts/ci/template-init-check.sh` proves each exact path and repeat
zero-drift.

### Reopened relay-stop and legacy-cancellation boundary

The shared PostgreSQL context watcher is an operation boundary, not a
connection-template default. `Pool.InTx` owns its watcher mark, one
non-cancelled rollback, and the resulting caller-cancellation/error identity
precedence. Its cancel-request handler is correct for a transaction statement,
but a relay append listener is an idle `WaitForNotification`, not a statement:
a PostgreSQL `CancelRequest` has no statement to cancel. Copying that handler
onto the listener therefore leaves the relay blocked until its later socket
deadline, so `Relay.Run` cannot complete the one listener join after
`StartDrain`.

`Store.listenerConfig` must return a copied, dedicated listener configuration
whose `BuildContextWatcherHandler` is reset to pgx's ordinary immediate
deadline watcher. It retains the configured DSN, TLS, connect settings, and
server statement limits, but does not inherit the Pool's operation watcher or
its transaction attribution map. The existing listener-owned cancellation and
bounded close remain its sole lifecycle controls: `StartDrain` cancels and
joins exactly that listener before `Relay.Run` returns; no extra relay
goroutine, poll loop, timeout, or NATS lifecycle path is added.

`Store.ClassifyLegacyUncertainty` instead runs its one canonical SQLC batch
inside the existing `Pool.InTx` callback. The Store still owns the one bounded
writer transition and the bootstrap still owns the DB-only loop-to-zero; the
transaction only gives this cancellation-sensitive write the already-accepted
watcher attribution and one cleanup rollback. A caller-cancelled lock wait
therefore returns an error matching both `context.Canceled` and the actual
PostgreSQL `57014`; an unrelated server SQLSTATE remains discoverable and is
not relabelled as caller cancellation. The method does not retry a cancelled
or failed batch. Previously committed batches remain the nullable sticky-fact
cursor, so rerun monotonicity, no synthetic receipt/order-head mutation, and
the publisher-free carrier stay unchanged.

This is the smallest boundary: changing the global watcher would reopen the
accepted `Pool.InTx` contracts; wrapping the listener in a transaction or
adding a relay-specific watcher would mix notification lifecycle with delivery
or duplicate the PostgreSQL owner. Reopen System Design only if a copied
listener config cannot preserve the validated connection policy while restoring
immediate listener cancellation.

## Observability and privacy

Observation distinguishes `poison` from `outcome_unknown`, including count and
oldest timestamp, and includes the receipt ledger's table/index bytes. The
existing bounded operation and error vocabularies gain only the three audited
operator operations, rollout-only `classify_legacy`, and `outcome_unknown`; no
event or audit value becomes a label.
Operators alert on a non-zero/aging unknown backlog and recover through the
audited store operations.

`StateObservation` therefore gains `OutcomeUnknownCount`,
`OutcomeUnknownOldestAt`, `ReceiptsBytes`, and `ReceiptsIndexBytes`; existing
poison fields exclude unknown rows.

Aggregate telemetry deliberately cannot reveal the event ID required by an
operator action. `docs/postgres-transactional-outbox.md` therefore owns one
authorized, non-telemetry writer readback using the canonical unknown predicate,
ordered by `poisoned_at, id`, with a required finite limit and tuple cursor. It
returns at most 100 rows with `id`, `destination`, `event_type`, `schema_name`,
cycle/total attempt counts, last-attempt/poison timestamps, last error class,
and the sticky flag; payload, metadata, trace context, credentials, and ordering
key stay out. A service may wrap that readback in its authorized admin surface.
Rollout must prove one such route before enabling recovery; absence of an
authorized route reopens this design rather than adding IDs to telemetry.

The ready inbox and production-closure specs impose a stricter privacy boundary
than the earlier trace design and current runtime. Remove event/message IDs,
consumer names, and ordering keys from outbox and NATS span/log attributes.
Keep trace IDs, destination, attempt numbers, bounded outcome/error classes,
and aggregate state metrics. This preserves trace continuity without making a
durable identity an unbounded telemetry attribute. The sibling trace design's
earlier `messaging.message.id` decision is superseded accordingly.

## Go responsibility map

| Responsibility and paths | Selected owner and exact action | Boundary and proof owner |
| --- | --- | --- |
| Receipt encoding, append evidence, writer reconciliation | Add `internal/infra/postgresoutbox/store_receipt.go`; change `store_append.go` | Concrete `Store`, standard library only; unit golden vector plus real PostgreSQL lost-commit/cleanup proof |
| Domain-mutation commit reconciliation | Change `docs/postgres-transactional-outbox.md`; add `examples/reference-service/postgres_outbox_reconciliation_integration_test.go` | Concrete PostgreSQL repository adapter owns stable event and the four-way caller route; worked lost-response proof, no feature or Store retry owner |
| Sticky row mapping and claim recovery | Change `store_claim.go`, `store_rows.go`, `store_operator.go` | `postgresoutbox` owns durable state; real PostgreSQL legacy/lease/operator proof |
| Pre-start legacy classification | Add `store_legacy_classification.go`; change `store.go` and `doc.go`; add `cmd/outbox-relay/internal/bootstrap/legacy_classification.go` and `legacy_classification_test.go`; change `run.go` | Concrete Store method uses the existing `Pool.InTx` watcher/cleanup boundary for one canonical batch; the explicit DB-only relay mode owns config, loop-to-zero, exit, and cleanup |
| Automatic retry/poison/unknown precedence | Change `relay.go`, `relay_finalize.go`, `store_finalize.go` | Existing relay/store boundary and lease fencing; focused relay table proof plus PostgreSQL transitions |
| Observation and bounded telemetry | Change `store_maintenance.go`, `telemetry.go`, `vocabulary.go` | Existing snapshot/instrument owner; bounded-vocabulary and state-query proof |
| Operator API and compatibility sentinels | Change `store_operator.go`, `errors.go` | Concrete methods, no public action enum; concurrent audit-action integration proof |
| NATS conversion, validation mapping, stored-context forwarding | Add `internal/infra/natsjs/outbox_publisher.go`; update the adapter pointer in `internal/infra/postgresoutbox/publisher.go` | `natsjs` may import `postgresoutbox`; reverse import is forbidden; adapter unit and joined trace proof |
| Remove unbounded NATS telemetry attributes | Change `internal/infra/natsjs/telemetry.go` | Transport telemetry only; snapshot/bounded-attribute proof |
| Relay process composition, readiness, supervision, cleanup, and one-shot routing | Add `cmd/outbox-relay/internal/bootstrap/natsjs_publisher.go` and `natsjs_publisher_test.go`; add `internal/config/configtest/messaging.go`; change `cmd/outbox-relay/internal/bootstrap/run.go`, `run_test.go`, `cmd/worker/internal/bootstrap/messaging_config_parity_test.go`, and `cmd/outbox-relay/main.go` | Concrete `PublisherRuntime` plus validating constructor preserves custom registration while reusing Client Run/Ready/Shutdown; both command-local mappings consume one table-only corpus; `run.go` routes the explicit classification flag before publisher admission; main owns bounded terminal exit class |
| Canonical SQL and generated access | Change `migrations/000001_postgres_outbox.sql` and `internal/infra/postgres/queries/postgres_outbox.sql`; regenerate `internal/infra/postgres/sqlcgen/models.go` and `postgres_outbox.sql.go` | Migration/query source wins; migration and SQLC drift checks |
| Package authority and profile retention/removal | Change `internal/infra/postgresoutbox/doc.go`, `internal/infra/natsjs/doc.go`, `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `.golangci.yml`, `Makefile`, `README.md`, `docs/repo-architecture.md`, and `docs/postgres-transactional-outbox.md` only where the new files/operations require it | Package docs name the selected state/operator/adapter owners; profile script owns explicit removal and generated combined/outbox-only/NATS-only process proof |

Two representations are forced by different boundaries. Ordered finalization
is lease-fenced for broker acknowledgement and audit-fenced for operator
confirmation; both remain in canonical SQL and one integration corpus proves
identical head advancement and successor release. Legacy sticky/quarantine
policy appears in the online claim statement, which must lease/return eligible
work with `SKIP LOCKED`, and in the pre-start classification statement, which
must update every legacy row without claiming or skipping locks. Canonical SQL
owns both statements, and one shared legacy-state integration corpus runs the
overlapping attempted-below-limit, attempted-at-limit, and expired-lease cases
through each path and compares the common sticky, terminal, error-class, and
at-limit lease-clearing outcomes. Below the limit, the same corpus separately
asserts the forced difference: claim renews the lease and returns the row,
while pre-start classification preserves delivery fields and returns only its
batch count.

The current test-only NATS outbox adapter is removed once the production adapter
owns the joined integration path; retaining both would leave a stale failure
classification.

## Inverse Go file map

| Go file | One present reason to change or exist | Declarations and constraints |
| --- | --- | --- |
| `internal/infra/postgresoutbox/store_receipt.go` (add) | Own one receipt encoding and reconciliation responsibility | `CommitOutcome`, `ReconcileCommit`, private v1 encoder; imports no transport and owns no transaction commit |
| `internal/infra/postgresoutbox/store.go` | Keep the shared Store seam and its audience/file map true after adding rollout classification | The Store comment changes from three to four audiences and names `store_legacy_classification.go`; shared constructor/validity/telemetry logic is unchanged |
| `internal/infra/postgresoutbox/store_append.go` | Extend the existing one-statement append columns | Supplies fingerprints to both canonical insert queries; no reconciliation policy |
| `internal/infra/postgresoutbox/store_claim.go` | Claim/recovery maps legacy and expired leases into sticky uncertainty | Canonical SQL owns the online sticky/quarantine plus lease/return transition; no Go disposition policy |
| `internal/infra/postgresoutbox/store_rows.go` | Restore the new durable fact into `ClaimedEvent` and `Record` | Mapping only |
| `internal/infra/postgresoutbox/store_finalize.go` | Execute retry, poison, unknown, and ordered/unordered persistence directives | Keeps lease fencing; no classification policy |
| `internal/infra/postgresoutbox/relay_finalize.go` | Own the fixed automatic-disposition precedence | No SQL and no broker-specific errors beyond outbox sentinels |
| `internal/infra/postgresoutbox/relay.go` | Keep the relay-store seam aligned with required transitions | Existing test seam only; no new public interface |
| `internal/infra/postgresoutbox/store_operator.go` | Own all audited operator transactions and `Record` operator state | Concrete actions, row/audit locks, and compatibility behavior stay together |
| `internal/infra/postgresoutbox/errors.go` | Own generalized receipt/operator sentinels and legacy aliases | No error-text routing |
| `internal/infra/postgresoutbox/store_legacy_classification.go` (add) | Own the separate pre-start rollout classification path | Adds `ClassifyLegacyUncertainty`; validates existing max-attempt and maintenance-batch bounds, executes one canonical statement, and owns no rollout loop or classification SQL |
| `internal/infra/postgresoutbox/store.go` | Supplies the separate relay-listener connection configuration | Its copied config resets only the pool operation watcher, retaining validated connection policy; it owns neither listener loop nor transaction semantics |
| `internal/infra/postgresoutbox/store_maintenance.go` | Map the expanded aggregate observation | Periodic relay observation/cleanup only; no rollout transition or delivery decision |
| `internal/infra/postgresoutbox/telemetry.go` | Export unknown state and remove unbounded event IDs | Bounded attributes only |
| `internal/infra/postgresoutbox/vocabulary.go` | Close new operation/state values | No dynamic values |
| `internal/infra/postgresoutbox/doc.go` | Keep the package's nearest authority true after receipt, unknown, operator, and rollout additions | Names receipt lifetime, three audited operator actions, the separate pre-start classification audience, and combined-profile registration; no implementation detail |
| `internal/infra/postgresoutbox/publisher.go` | Replace the stale test-adapter documentation with the selected production owner | Contract text only; no new publisher surface |
| `internal/infra/natsjs/outbox_publisher.go` (add) | One NATS-specific realization of the existing publisher seam | Unexported `outboxPublisher`; exported constructor returns the consumer-owned `postgresoutbox.Publisher`; no relay state/lifecycle |
| `internal/infra/natsjs/doc.go` | Record the concrete outbox adapter as a composition-root surface | No new feature audience and no broker abstraction |
| `internal/infra/natsjs/telemetry.go` | Enforce the accepted no-ID/no-consumer attribute boundary | Transport spans/logs retain bounded outcomes and destination only |
| `cmd/outbox-relay/internal/bootstrap/natsjs_publisher.go` (add) | Build and clean up the selected NATS publisher | Concrete builder; no goroutine or retry policy |
| `cmd/outbox-relay/internal/bootstrap/run.go` | Route the explicit one-shot mode or construct, supervise, and join the selected publisher runtime with relay drain/readiness | Parses `--classify-legacy-uncertainty` and dispatches it before publisher admission; normal mode owns `PublisherRuntime`, combined readiness, terminal trigger, and error-returning cleanup; no classification SQL or NATS-specific policy |
| `cmd/outbox-relay/internal/bootstrap/legacy_classification.go` (add) | One DB-only rollout carrier for the fixed pre-start classification gate | Opens the writer pool and Store, loops to authoritative zero using configured `max_attempts` and `cleanup_batch_size`, honors cancellation, logs bounded counts only, and never builds a publisher, claims, or reports readiness |
| `cmd/outbox-relay/main.go` | Select the builder and classify supervised NATS termination under the messaging marker | Nil remains the outbox-only default; `natsjs.ErrTerminal` maps to `messaging_terminal` without raw broker text |
| `internal/config/configtest/messaging.go` (add) | One shared input corpus for forced command-local NATS config mappings | Test data only; imports neither `natsjs` nor bootstrap packages and is removed with messaging |
| `internal/infra/postgres/sqlcgen/models.go`, `internal/infra/postgres/sqlcgen/postgres_outbox.sql.go` | Derived representation of the changed schema/query source | Regenerate only; no hand edits |
| `examples/reference-service/postgres_outbox_reconciliation_integration_test.go` (add) | Prove the repository-adapter caller orchestration that no infrastructure Store can own | Integration-only concrete adapter; stable `Event` precedes `InTx`, and only `NotApplied` may rerun the mutation |

`cmd/outbox-relay/internal/bootstrap` deliberately retains no `doc.go`: its
single process responsibility and public builder contract remain readable from
`run.go` and the selected `natsjs_publisher.go`; another package audience would
reopen that decision.

### Exact test-file map

| Test file | Changed proof responsibility |
| --- | --- |
| `internal/infra/postgresoutbox/store_receipt_test.go` (add) | Version-1 golden vector, validation, outcome zero value |
| `internal/infra/postgresoutbox/store_append_test.go` | Fingerprint arrays remain in the one append statement |
| `internal/infra/postgresoutbox/store_rows_test.go` | Sticky/unknown row mapping |
| `internal/infra/postgresoutbox/store_finalize_test.go` | Retry, poison, and unknown directive binding/fencing |
| `internal/infra/postgresoutbox/relay_finalize_test.go` | Exact automatic-disposition precedence and attempt boundary |
| `internal/infra/postgresoutbox/store_test.go` | New Store method validation and legacy sentinel aliases |
| `internal/infra/postgresoutbox/telemetry_test.go` | Unknown vocabulary/storage gauges and absence of event IDs |
| `internal/infra/postgresoutbox/relay_publish_span_test.go` | Preserve root/link/kind/error semantics and prove the relay span has no event ID or ordering key |
| `internal/infra/postgresoutbox/notify_test.go` | Dedicated listener config retains connection policy but uses immediate cancellation, and one drain joins `Relay.Run` |
| `internal/infra/postgresoutbox/store_legacy_classification_test.go` (add or existing nearest Store test) | Cancellation/error identity through the one `Pool.InTx` batch, without retry or false zero |
| `internal/infra/natsjs/outbox_publisher_test.go` (add) | Conversion, private prevalidation mapping, ambiguity, and creation-context forwarding |
| `internal/infra/natsjs/telemetry_test.go` | No message/consumer attributes after privacy repair |
| `cmd/outbox-relay/internal/bootstrap/natsjs_publisher_test.go` (add) | Typed config mapping parity, connect failure, and complete runtime carrier |
| `cmd/outbox-relay/internal/bootstrap/run_test.go` | Runtime-constructor validation, custom-builder usability, combined readiness, terminal supervisor drain/exit, join safety, and shutdown-error propagation |
| `cmd/outbox-relay/internal/bootstrap/legacy_classification_test.go` (add) | Flag exclusivity, no-publisher branch, loop-to-zero, cancellation/non-zero exit, bounded logging, and pool cleanup |
| `cmd/worker/internal/bootstrap/messaging_config_parity_test.go` | Consume the same table-only mapping corpus as the new relay test |
| `test/postgres_outbox_*_integration_test.go` | Schema/state/audit/receipt/cleanup/ordering transitions against real PostgreSQL |
| `test/postgres_outbox_natsjs_integration_test.go` | Delete local `natsOutboxPublisher`; construct the production adapter and prove joined identity/trace/failure semantics |
| `examples/reference-service/postgres_outbox_reconciliation_integration_test.go` (add) | Repository-adapter lost-commit caller route |

Generated-profile/process behavior remains in
`scripts/ci/template-init-check.sh` and the existing process-integration
surface. Exact scenarios and commands belong to Test Design, not this artifact.

The file map reopens if implementation evidence shows receipt encoding and
reconciliation change independently, the operator file contains a fourth
unrelated lifecycle, or the adapter cannot reuse NATS validation without an
import cycle. None is true in the current graph.
