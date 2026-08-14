# Design: idempotent PostgreSQL inbox consumption

status: ready
Realizes `../spec.md` as a sibling capability to the outbox and NATS packs.

## Selected system

Add one PostgreSQL claim table and one stateless claim function. The concrete
feature adapter, not `natsjs` and not the feature use case, joins the claim and
feature effect inside the caller's existing `postgres.Pool.InTx` transaction.

The durable key is `(consumer_identity, logical_message_id)`. For the existing
NATS runtime, the binary-local adapter derives the consumer identity from the
configured source stream and durable consumer as `stream + "/" + consumer`.
Both NATS names already reject `/`, so the encoding is injective; renaming
either component deliberately creates a new consumer identity. The logical ID
is `Message.MessageID()`, never `PublicationID`, broker sequence, subject,
ordering key, or process identity.

Keep the consumer identity opaque to `postgresinbox`. This preserves an
independent `INBOX=postgres` profile and avoids transport columns or a NATS
dependency in the persistence package. A three-column NATS-shaped key loses to
this two-column form because it makes the supposedly independent pack own one
transport's namespace without changing correctness.

No store object, repository interface, handler decorator, generic unit of work,
status machine, payload digest, TTL, cleanup job, or ordering state is added.
The package has no mutable state: one function binding the existing SQLC query
to the caller's `pgx.Tx` is sufficient.

## Data and concurrency authority

Add `migrations/000002_postgres_inbox.sql`. The version is intentionally
distinct from the outbox's `000001`, so both independent packs compose; an
inbox-only generated service may validly begin at version `000002`. The table is:

```text
postgres_inbox_claims
  consumer_identity   text COLLATE "C", 1..1024 UTF-8 bytes, no control bytes
  message_id          text COLLATE "C", 1..256 UTF-8 bytes, no control bytes
  PRIMARY KEY (consumer_identity, message_id)
```

There are no timestamps, payloads, digests, statuses, delivery counters,
foreign keys, secondary indexes, expiry columns, or cleanup statements. The
1,024-byte consumer bound admits the two configured NATS names plus their
separator without turning configuration into an unbounded database key. The
message bound matches the existing wire identity bound. Go validation and SQL
checks mirror these limits and reject rather than truncate.

`internal/infra/postgres/queries/postgres_inbox.sql` owns one statement:

```sql
INSERT ... ON CONFLICT (consumer_identity, message_id) DO NOTHING
```

SQLC returns affected rows. One row is `claimed=true`; zero is the successful
duplicate outcome. PostgreSQL unique-index arbitration is the concurrency
mechanism: a concurrent loser waits for the winner; it skips after winner
commit and inserts/applies after winner rollback. A preselect, advisory lock,
or application mutex is both larger and weaker.

The exact hand-written surface is
`func Claim(context.Context, pgx.Tx, consumerIdentity, messageID string)
(bool, error)`. It returns no exported sentinel: `false, nil` is the only
duplicate result, while invalid/nil input and database faults are contextual
errors. The function starts and commits no transaction.

The concrete feature adapter owns this path:

```text
NATS handler
  -> derive stable consumer identity and read MessageID
  -> Pool.InTx
       -> postgresinbox.Claim(ctx, tx, consumerIdentity, messageID)
       -> false: return nil without invoking the feature effect
       -> true: invoke the feature use case with its tx-bound PostgreSQL repository
  -> commit: handler returns nil and NATS acknowledges
  -> rollback/error: handler returns error and NATS redelivers
  -> ErrCommitUnknown: handler returns error; redelivery observes a landed claim
     and skips, or observes no claim and applies
```

The feature use case remains callable without inbox knowledge. The adapter may
use the existing feature-owned unit-of-work port when the feature already has
one; the template adds no generic port because it has no production feature to
own it. The worked documentation and a compile/lint exemplar show the concrete
placement. Effects outside the same PostgreSQL transaction remain outside the
guarantee.

The duplicate wait consumes one pool connection and is bounded by the existing
handler context. `postgres.Pool.InTx` owns the shared transaction cancellation
boundary. While it owns an acquired pgx connection, it registers one private
watcher-settlement marker for that connection; the pool's existing pgx
cancel-request watcher marks it immediately before sending PostgreSQL the
cancel request, and `InTx` unregisters it after transaction cleanup. Pgx
settles the in-flight protocol operation before control returns to `InTx`. A
marked operation deterministically attributes a concurrent `57014` to the
handler: its returned transaction error satisfies `errors.Is(err, ctx.Err())`
and also retains the observed PostgreSQL error when there is one. An unmarked
server-originated timeout remains its PostgreSQL error, not a synthetic handler
cancellation. This is the race rule: when both are observed for one in-flight
operation, the watcher mark makes caller cancellation win; a later handler
cancellation after an unmarked server result does not rewrite that result.

After that settlement, `InTx` performs at most one rollback using a
non-cancelled cleanup context and preserves the primary result; an already
closed transaction is normal cleanup, while a distinct cleanup failure is
retained alongside the primary error. It neither retries a query nor retries a
rollback. The inbox claim neither translates errors nor cleans up: adding a
per-inbox path would create a second transaction owner and could displace the
winner's unique-index decision. A database outage remains a handler error under
the worker's current retry/dead-letter policy. No generic worker readiness or
pause mechanism is added: that would change transport behavior beyond the ready
spec. Reopen Specification if database unavailability must pause delivery rather
than spend the configured attempt budget.

Claims never expire automatically. Storage grows by one compact primary-key row
per consumer/message identity. Reopen only on measured capacity breach; any
cleanup mechanism must first replace the accepted permanent-recognition
contract or retain an equivalent identity.

Adoption is forward-only unless the service can seed claims from an
authoritative history of already-applied logical IDs. The template cannot infer
that history. A previously applied message with no claim is new to this pack
and may apply again; rollout must either prove there is no such replay surface,
complete a service-owned seed, or retain the pre-existing domain idempotency for
those messages. This does not weaken recognition after the first claim commits.

## Profile composition

Add independent initialization input `INBOX=none|postgres`, default `none`:

- `INBOX=postgres` requires `DATABASE=postgres` but not `OUTBOX` or
  `MESSAGING`;
- `template.lock` records `inbox = "..."` and repeat initialization must match;
- `profile:inbox-postgres` markers own the migration, query/generated output,
  package, documentation, lint allowance, and inbox-specific proof;
- `INBOX=none` removes those surfaces before SQLC regeneration;
- inbox-only retains no outbox artifact, outbox-only retains no inbox artifact;
  and
- the joined NATS/PostgreSQL proof is retained only when both capabilities are
  selected.

`INBOX=none` explicitly removes
`migrations/000002_postgres_inbox.sql`,
`internal/infra/postgres/queries/postgres_inbox.sql`,
`internal/infra/postgresinbox`, the reference exemplar, and both inbox
integration files. Then, after outbox/inbox removals but before whole-PostgreSQL
removal, the initializer regenerates SQLC so `postgres_inbox.sql.go` and the
inbox model disappear while shared outputs remain. `MESSAGING=none` also removes
the joined NATS/inbox test; `DATABASE=none` removes the remaining PostgreSQL
tree after refusing `INBOX=postgres`.

There is no `APP__INBOX__*` runtime configuration. The pack has no loop,
timeout, cleanup, or policy knob. The adopter's stable consumer identity comes
from its already-owned transport/handler configuration.

## Observability and privacy

`postgresinbox.Claim` emits no new telemetry. Applied versus skipped is its
bounded boolean result, and the caller already owns handler success/failure.
Adding an instrument would create a second account of the same delivery without
an accepted operator use.

The ready spec forbids message IDs and consumer-supplied strings on metrics,
spans, and logs. The shared NATS telemetry cleanup is owned by the outbox
production-closure design: remove message IDs and consumer names while retaining
bounded outcomes, attempts, destinations, and trace continuity. The inbox table
and PostgreSQL errors never put key values into telemetry.

## Go responsibility map

| Responsibility and paths | Selected owner and exact action | Boundary and proof owner |
| --- | --- | --- |
| Validate and atomically claim one opaque identity | Add `internal/infra/postgresinbox/inbox.go` with exported `Claim` | Stateless concrete function accepts caller `pgx.Tx`; unit validation and real PostgreSQL concurrency proof |
| Settle a canceled transaction and preserve its error identities | Change `internal/infra/postgres/postgres.go` and `internal/infra/postgres/transaction.go` with wrapper-local tests | The watcher marks the one private state registered for the acquired `InTx` connection before cancellation; `Pool.InTx` owns one rollback after it settles; the real PostgreSQL concurrent inbox carrier proves handler-context, SQLSTATE, and cleanup result |
| Canonical claim schema/query | Add `migrations/000002_postgres_inbox.sql` and `internal/infra/postgres/queries/postgres_inbox.sql`; regenerate SQLC | SQL sources win; migration/SQLC drift and populated-schema proof |
| Join claim and effect | Document the concrete adapter in `docs/postgres-idempotent-inbox.md`; add compile/lint exemplar under `examples/reference-service` | Service adapter owns pool/tx-bound feature repository; no template production fake |
| Place and select/remove the independent pack | Change `docs/project-structure-and-module-organization.md`, `docs/repo-architecture.md`, `.golangci.yml`, `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `README.md`, `docs/build-test-and-development-commands.md`, and only required aggregate Make targets | Normative structural model, ownership table, and first-match placement algorithm admit `internal/infra/postgresinbox`; depguard/profile rules enforce that exception and explicit removal/regeneration order |
| Preserve transport privacy | Shared change to `internal/infra/natsjs/telemetry.go` owned by the production-closure design | Existing NATS telemetry tests; no inbox import into NATS |

## Inverse Go file map

| Go file | One present reason to exist or change | Declarations and constraints |
| --- | --- | --- |
| `internal/infra/postgresinbox/inbox.go` (add) | Entire stateless inbox persistence capability | Package documentation, bounds, `Claim`; imports `pgx` and generated SQLC, never NATS/outbox, owns no pool or transaction lifecycle |
| `internal/infra/postgresinbox/inbox_test.go` (add) | Fast boundary validation for the sole package function | No database fake; concurrency belongs to integration |
| `internal/infra/postgres/sqlcgen/models.go` | Derived claim-row schema when the profile is retained | Regenerate only |
| `internal/infra/postgres/sqlcgen/postgres_inbox.sql.go` (add) | Derived binding for the sole inbox statement | Regenerate only |
| `examples/reference-service/postgres_inbox_integration_test.go` (add) | Compile/lint proof of the concrete adapter and transaction placement | Example-only adapter; no production fake or generic port added solely for the example |
| `test/postgres_inbox_integration_test.go` (add) | Real database authority for claim/effect atomicity and concurrent resolution | Uses the existing PostgreSQL integration harness; no transport behavior |
| `test/postgres_inbox_natsjs_integration_test.go` (add) | One joined NATS redelivery/DLQ identity path | Exists only in the combined generated profile; no generic ordering assertions |

The generated `db.go` and unrelated query outputs have no reason to change; a
generator diff that touches them must be mechanical only or reopens the file
map. No `doc.go`, constructor, store type, mock, or new worker bootstrap file is
justified. Exact proof scenarios and commands belong to Test Design.

The placement reopens only if a real feature cannot bind its repository to the
same `pgx.Tx` without violating its existing feature-owned port, or a selected
non-NATS transport cannot supply a stable opaque consumer identity. Either is a
service-specific design input, not a reason to move PostgreSQL into transport.
