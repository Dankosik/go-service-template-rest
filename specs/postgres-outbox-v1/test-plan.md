# PostgreSQL transactional outbox V1 test design

status: ready

## Proof strategy

Correctness has four complementary boundaries. None substitutes for another:

1. focused Go tests own exact-byte validation, retry classification/backoff,
   cancellation joins, telemetry attributes, and commit-error classification;
2. PostgreSQL 17 integration owns transactions, constraints, server-clock
   leases, row locking, concurrent replicas, fencing, cleanup, and durable state;
3. a deterministic Publisher plus process tests own acknowledgement ambiguity,
   duplicate attempts, poison/redrive, drain, and stuck-adapter cleanup safety;
4. repository migration, generation, image, and template-initialization gates
   own canonical schema and physical profile absence.

The current tree has no outbox implementation, migration, relay, or tests, so
there is no useful behavioral fail-before result. The nearest current evidence
is structural absence plus the existing PostgreSQL/worker/NATS boundaries that
the new scenarios strengthen. Implementation must make every mandatory row
below executable; absence of a named test is never counted as a pass.

## Fixtures and controls

- Real PostgreSQL: existing `internal/infra/postgres/pgtest.DSN`, pinned
  PostgreSQL 17 container, existing bounded `postgres.Pool`, and canonical
  `postgresmigrate` runner. Each test owns schema cleanup and registers
  Testcontainers cleanup immediately.
- Reference feature: an integration-only table and `httptest` handler. The
  handler uses `postgres.Pool.InTx`, writes one feature row, and appends a
  caller-supplied broker-neutral event. It is proof code, not a production
  domain API or event.
- Deterministic Publisher: a test-only channel/state harness recording exact
  `Event` attempts and controlling nil, temporary/permanent error, block,
  context acknowledgement, and release. It never stands in for broker proof.
- Real broker: the existing pinned NATS JetStream integration fixture and
  `natsjs.Producer`. A test-only adapter maps the outbox ID to the accepted NATS
  message/publication IDs and returns only the producer's `PubAck` result.
- Go timers: `testing/synctest` for relay poll/retry/deadline/drain logic. Real
  PostgreSQL lease expiry uses server time and a bounded `pg_sleep` as the
  semantic clock, with an outer test deadline only as a hang diagnostic.
- Commit errors: a deferred PostgreSQL constraint creates a server-confirmed
  commit failure; a focused commit-classifier table uses `PgError`,
  `ErrTxCommitRollback`, an error implementing `SafeToRetry`, and an opaque
  connection error to discriminate definite from `ErrCommitUnknown`. A
  same-package integration test replaces the Pool's unexported commit function
  with one that performs the real commit and then returns the opaque error, so
  `InTx` production control flow and independently read durable rows model a
  lost commit response without a connection-kill race.

## Scenario matrix

| ID | Claim/risk and plausible wrong result | Controlled trigger and independent oracle | Boundary and command | Owner / reopen |
| --- | --- | --- | --- | --- |
| TD-01 | Exact immutable envelope; a validator accepts malformed UTF-8/JSON/control text, normalizes bytes, rejects Go-valid JSON, admits zero/infinite occurrence time, or crosses a bound | Table rows at every exact field/288 KiB boundary and one byte over; query stored `bytea` and compare byte-for-byte; require exact round trips for a number beyond PostgreSQL `numeric` and escaped `\u0000`; bypass Go with invalid JSON, Go-zero `occurred_at`, and PostgreSQL `±infinity`, and assert SQL rejection | Unit + PostgreSQL integration; `go test -vet=off -count=1 ./internal/infra/postgresoutbox` and `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxEnvelope$'` | runtime/store tests; reopen Specification for different bounds |
| TD-02 | Failure before domain write leaves an outbox orphan | Return a sentinel before either statement; independent queries require zero feature and event rows | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxAtomicity/failure_before_domain_write$'` | integration test; reopen transaction design if orphan possible |
| TD-03 | Failure after domain write but before append commits feature state | Insert feature row, return sentinel before append; both tables remain empty | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxAtomicity/failure_after_domain_write$'` | integration test; reopen transaction owner |
| TD-04 | Outbox insert failure still commits the domain mutation | Write feature row then append an SQL-invalid envelope; assert returned transaction error and both tables empty | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxAtomicity/outbox_insert_failure$'` | integration test; reopen append/transaction design |
| TD-05 | Failure after append or explicit rollback leaves an event/domain split | Write both, then return sentinel/cancel callback; independent new connection sees neither row | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxAtomicity/(failure_after_append|cancellation)$'` | integration test; reopen transaction design |
| TD-06 | A commit-time failure is misreported or leaves one side | Same-package real-PG subtests drive `Pool.InTx`: a deferred constraint fails only at commit and must return `ErrTransaction` without `ErrCommitUnknown` and persist no rows; the unexported hook performs a real commit then returns an opaque error and must return `ErrCommitUnknown` while another connection sees the committed rows. Classifier table independently covers SQLSTATE `08007`/`40003` as unknown, other PgError, rollback, definitely-unsent, and opaque error categories. The later outbox atomicity row repeats the definite commit oracle with real feature+event statements | Focused + PostgreSQL integration; `go test -vet=off -count=1 ./internal/infra/postgres -run 'TestClassifyCommit|TestInTx'`, `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./internal/infra/postgres -run '^TestInTxCommitOutcomes$'`, and `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxAtomicity/commit_failure$'` | PostgreSQL owner for classification, outbox integration owner for repeated feature/event proof; reopen Go ownership if stage cannot be known |
| TD-07 | Concurrent ordered append/finalize loses the next head, admits duplicate/lower sequence, trusts mutated claim identity, or cleanup forgets history | Race append of sequence N+1 against finalization of N across repeated keys; require N+1 is next claim every time. Mutate the returned ordering identity and require token-fenced rejection. Concurrent increasing/equal/lower appends keep one high-water authority; publish+cleanup then lower insert remains rejected | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutbox(OrderingAuthority|OrderingHandoffRace|OrderedMarkFencesClaimIdentity)$'` | store/schema; reopen Specification for overtaking/retirement |
| TD-08 | Replicas claim the same current lease or block unrelated work | Seed ordered and unordered rows. Connection A begins a transaction, selects the first eligible row `FOR UPDATE`, and deliberately leaves that lock open. Concurrent claims must return other-key/unordered rows inside the query deadline; then synchronize N further claims and require disjoint IDs/tokens | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxConcurrentClaims$'` | claim query; reopen concurrency design if serialized |
| TD-09 | Same-key later event overtakes an unfinished/poison predecessor | Seed sequences 1..3; lease/retry/poison sequence 1 in turn; other replicas may claim other keys but never same-key 2/3; after sequence 1 publish, sequence 2 becomes eligible | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxOrderingClaims$'` | claim query; reopen ordering contract |
| TD-10 | Lease expiry loses work or stale owner finalizes a new claim | Claim token A, advance PostgreSQL server time past lease, claim token B; A's mark/retry/poison affects zero and returns `ErrLeaseLost`, B can finalize | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxLeaseExpiryAndFence$'` | store; reopen lease mechanism |
| TD-11 | Relay crash after claim strands a row | Claim and deliberately abandon token; after server-clock expiry a new Store/Relay claims same ID and the row remains unfinished until a valid result | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxCrashAfterClaim$'` | store/relay; reopen recovery design |
| TD-12 | Broker failure before acknowledgement is treated as success | Publisher records ID then returns temporary/timeout/ambiguous error; token row becomes retry-wait, never published, attempt/error metrics change | Component + PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxPublishFailure$'` | relay; reopen acknowledgement semantics |
| TD-13 | Broker ack followed by crash before marking silently loses or invents exactly-once | Manually claim, Publisher returns nil and records ID, omit `MarkPublished` to model process death; after lease expiry/restart publish again; oracle requires two identical ID/envelope attempts and one final published row | Deterministic Publisher + PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxAckCrashDuplicate$'` | relay/store; reopen at-least-once design if no duplicate |
| TD-14 | Multiple full relay replicas race or stop progress | Start several Relay instances on one database with synchronized Publisher; seed rows; require every ID eventually published, no ID concurrently owned twice before expiry, and all loops stop | Component + PostgreSQL integration under race; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxRelayReplicas$'` and `go test -vet=off -race -count=1 ./internal/infra/postgresoutbox -run '^TestRelayReplicas$'` | relay/store; reopen concurrency/capacity design |
| TD-15 | Retry policy retries permanent errors, exceeds budget, or silently discards exhaustion | Controlled temporary failures then success; permanent first failure; ten temporary claims; assert attempt counters/delays bounds, success only after nil, immediate/exhausted poison durable and never cleanup-eligible | Unit + PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxRetryAndPoison$'` | relay/store; reopen retry policy |
| TD-16 | Redrive is non-idempotent, changes envelope/ID, or accepts published work | Poison an event; same audit ID twice produces one ledger transition, reused ID for another event fails, new ID starts next cycle; exact envelope/ID unchanged; pending/published redrive rejected | PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxRedrive$'` | store; reopen redrive/finality contract |
| TD-17 | Cancellation reports delivery or leaks work | Publisher waits on context and returns; cancel an attempt under `synctest`; assert no published mark, retry only when not process-drain cancellation, inflight returns zero and goroutines join | Component; `go test -vet=off -count=1 ./internal/infra/postgresoutbox -run '^TestRelayCancellation$'` | relay; reopen cancellation policy |
| TD-18 | Graceful drain claims new work or abandons a completed ack | Block current Publisher, begin shutdown, assert readiness false/no new claim, release before 20s, require fenced finalization, joined loop and `cleanupSafe=true` | Bootstrap component with controlled channels/synctest; `go test -vet=off -count=1 ./cmd/outbox-relay/internal/bootstrap -run '^TestOutboxRelayGracefulDrain$'` | bootstrap/relay; reopen lifecycle design |
| TD-19 | Forced shutdown marks success, hangs forever, or closes dependencies under a stuck Publisher | Publisher ignores cancellation until test release; advance attempt/drain plus 1s join; assert fatal `ErrPublisherStuck`, `cleanupSafe=false`, no progress transition/no further attempt, Publisher/PG cleanup skipped; release only for test leak cleanup. Separately make the admitted Publisher cleanup callback ignore context or panic and require a bounded, redacted process error | Bootstrap component under race/synctest; `go test -vet=off -race -count=1 ./cmd/outbox-relay/internal/bootstrap -run '^TestOutboxRelay(ForcedShutdown|PublisherCleanupIsBounded)$'` | bootstrap/relay; reopen adapter termination design |
| TD-20 | Broker outage causes request-path dual write or API failure | `httptest` POST handler commits reference feature+append while Publisher returns broker errors; send multiple requests and require success plus equal feature/outbox backlog counts, zero direct publish calls from handler, growing retry/backlog state | HTTP component + PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxRequestContinuesDuringBrokerOutage$'` | reference integration; reopen async boundary |
| TD-21 | Backlog metrics are stale/high-cardinality, incomplete, or lie about state | Move controlled rows through every durable state; collect OTel manual-reader data and compare counts/oldest/high-water/event/head/redrive bytes to direct SQL. Drive success/failure for claim, publish, progress-commit, retry, poison, recovery, redrive, cleanup, observe, and drain and require operation counter plus duration series. Assert last-progress changes only after durable `MarkPublished`; inflight/readiness transitions match controlled joins. Fail periodic observation and prove the timestamp stays stale while the loop retries; remove the redrive ledger before startup and require fail-closed admission. Run two telemetry instances against the same DB and assert each database-global sample equals SQL rather than a divided/summed value, process counters remain instance-local, and the documented freshness-filtered `max without(service.instance.id)` yields the one database value. Inspect all attributes to exclude IDs/keys/destination/error text | Unit + PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutbox(Observability|StartupRequiresRedriveLedger)$'` and `go test -vet=off -count=1 ./internal/infra/postgresoutbox -run '^TestTelemetry'` | telemetry/store; reopen observability design |
| TD-22 | Cleanup deletes unfinished/poison/high-water data, exceeds batch, races replicas, or cannot catch up | Seed >batch old published plus recent published/pending/leased/retry/poison/ordering heads; concurrent cleanup deletes at most batch each, eventually only old published, cascades retained redrive ledger with event, preserves ordering head and late-sequence rejection. A full batch schedules the next bounded pass at poll cadence while an incomplete batch returns to normal cadence | PostgreSQL integration plus deterministic relay time; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxCleanup$'` and `go test -vet=off -count=1 ./internal/infra/postgresoutbox -run '^TestRelayMaintenance'` | store/schema/relay; reopen retention design |
| TD-23 | Empty queue, broker outage, schema loss, fatal loop, or Publisher panic produce wrong health/finality | Fresh empty observation ready; transient Publisher error stays ready; missing schema/stale observation/fatal loop/drain unready; liveness remains process-only. A panicking Publisher is recovered only at supervision boundary and must return fatal, set readiness false, leave the row unfinished, start no further attempt, and complete cleanup-safe bounded shutdown because the panicking goroutine terminated | Bootstrap component + integration; `go test -vet=off -count=1 ./cmd/outbox-relay/internal/bootstrap -run '^TestOutboxRelay(Readiness|Liveness|PublisherPanic)$'` | bootstrap/relay; reopen readiness or panic design |
| TD-24 | A noop/missing Publisher silently marks rows | Nil builder is rejected before config/pool mutation; nil Publisher is rejected after builder; no fallback path executes. Static compile is insufficient: test side-effect counters for builder/config/dependency order | Bootstrap component; `go test -vet=off -count=1 ./cmd/outbox-relay/internal/bootstrap -run '^TestOutboxRelayComposition'` | bootstrap; reopen publisher admission |
| TD-25 | Accepted NATS adapter returns nil without durable same-ID presence | Real JetStream stream/consumer; test adapter publishes outbox Event; after nil, consume durable message and assert accepted NATS message/publication IDs equal outbox ID and payload bytes match. Reject/no-ack path never finalizes outbox | Real broker + PostgreSQL integration; `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxNATSConformance$'` | integration only; missing adapter narrows readiness, never local correctness |
| TD-26 | Migration forks authority, is non-transactional, drifts from sqlc, rewrites inactive heads, admits a mixed-version rollout, or cannot rehearse | Existing canonical source/history checks plus real PostgreSQL Up/no-op/Down/Up and built-image `/migrate`/schema contents; populated backfill proves inactive historical heads are unchanged. Insert Go-valid/jsonb-incompatible bytes, require Down to fail with the explicit compatibility fence, and require the transaction to preserve schema and bytes. Rollout docs must quiesce every version-1 writer/relay and require a zero-row head invariant readback before version-2 admission; generated diff must be empty | Migration/generation/image plus real PostgreSQL integration; `make migration-validate`, `make sqlc-check`, `make migration-check`, and `REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutboxOrderingReadyMigrationBackfill$'` | migration/sqlc/image/rollout owners; reopen migration design |
| TD-27 | `OUTBOX=none` leaves owned bytes/commands/dependencies, retained profile is incomplete, or invalid combination mutates | Extend canonical init harness: default/explicit none with both DB choices, `DATABASE=postgres OUTBOX=postgres`, and invalid `DATABASE=none OUTBOX=postgres`. None assertions require total path/help/config/Go-list/image absence; invalid input uses pre/post tree hashes. Positive retained assertions require migration/query/generated/runtime/relay/config/doc/test paths, Make targets, `/outbox-relay` image binary, dependency/import graph, `template.lock outbox="postgres"`, and zero unresolved profile markers | Template structural/behavioral harness; `make template-init-check` | initializer; reopen profile ownership |
| TD-28 | Repeated initialization drifts or choice can change | Initialize each valid matrix once/twice and compare full byte tree/template.lock; different second choice must fail with unchanged tree | Template harness; `make template-init-check` | initializer; reopen lock contract |
| TD-29 | Config admits invalid DATABASE/outbox/lease/drain combinations after mutation begins | Config table covers disabled/valid and each cross-budget failure; bootstrap spies prove rejection before builder/pool/relay construction | Focused config/bootstrap; `go test -vet=off -count=1 ./internal/config ./cmd/outbox-relay/internal/bootstrap` | config/bootstrap; reopen configuration design |
| TD-30 | Shared transaction or relay state races, deadlocks, or outlives test/process | Execute concurrent claim, replica, cancellation, graceful and forced lifecycle scenarios under race; every test owns joins and outer deadlines and reports stuck state | Focused race/liveness; `go test -vet=off -race -count=1 ./internal/infra/postgres ./internal/infra/postgresoutbox ./cmd/outbox-relay/internal/bootstrap` | changed Go packages; reopen concurrency design on race/hang |
| TD-31 | Materialized ordering improves one microcase but still collapses with backlog, its migration rewrites the hot backlog, or ordered finalization cost dominates the accepted local envelope | On the same PostgreSQL 17 fixture and host, repeat candidate and fixed-tree `EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)` and workload cases for empty/1K/100K/1M unordered, high-cardinality, one hot key, poison head, claim, observation, ordered mark, cleanup, and the populated `000001` to `000002` migration. Retain medians/dispersion, migration time/WAL/dead tuples, and environment identity; never extrapolate local capacity | Real PostgreSQL benchmark artifacts outside the repository; exact command and result manifest retained with the audit evidence | data/performance owner; reopen materialized-head design if claim/observation remain nonlinear, migration work scales with blocked hot-key rows, or ordered-mark regression breaches the declared envelope |

## Commands and gate composition

During implementation, run the narrow commands in the matrix. Final candidate
evidence is serialized in this order so Docker and broad Go gates do not
overlap:

```bash
go test -vet=off -count=1 ./internal/infra/postgres ./internal/infra/postgresoutbox ./internal/config ./cmd/outbox-relay/internal/bootstrap
go test -vet=off -race -count=1 ./internal/infra/postgres ./internal/infra/postgresoutbox ./cmd/outbox-relay/internal/bootstrap
REQUIRE_DOCKER=1 go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^TestPostgresOutbox'
make sqlc-check
make migration-check
make migration-validate
make template-init-check
make check-full
git diff --check
```

`make check-full` is publication-scale local evidence only after the focused
oracles pass; it does not replace them or prove deployment. If the accepted NATS
fixture is unavailable, TD-25 is a named adapter-readiness blocker and the claim
remains broker-neutral local relay correctness. The user requires it on this
base, so completion cannot silently skip it.

## Bidirectional closure

- Spec OUT-1 maps to TD-27..29.
- OUT-2 maps to TD-01 and TD-25.
- OUT-3 maps to TD-02..07 and TD-20.
- OUT-4/5 map to TD-08..14, TD-19, TD-24, and TD-25.
- OUT-6/7 map to TD-07, TD-09, and TD-15..19.
- OUT-8 maps to TD-20..22.
- OUT-9 maps to TD-17..19 and TD-23..24, including the explicit Publisher
  panic row in TD-23.
- OUT-10 maps to TD-21 and TD-23.
- OUT-11 maps to TD-26..28.
- Ordering high-water growth/preservation maps to TD-07, TD-21, and TD-22.
- Commit ambiguity, stuck-adapter, ordering-query, and telemetry completeness
  audit repairs map to TD-06, TD-19, TD-31, and TD-21 respectively;
  retained-profile positive proof maps to TD-27.

Every matrix row maps back to one of these accepted claims, risks, or repaired
design boundaries. There is no security/tenant proof because the pack exposes
no inbound operator endpoint and defines no tenant field; any derived redrive
surface reopens security/API design. TD-31 supports only a comparative local
envelope and query-shape claim; it is not production capacity proof. Production
throughput still requires production SLOs, traffic shape, and runtime telemetry.
