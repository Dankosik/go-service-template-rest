# Durable Background Jobs — Technical Design

status: ready

Authority: [`spec.md`](../spec.md) owns B1-B13 and caller/operator-visible
meaning. [`research/synthesis.md`](../research/synthesis.md) owns the candidate
evidence and adopter-input checkpoints. This design selects mechanism and
runtime ownership only. It does not admit a concrete job kind or a production
topology.

## Decision summary

Select a small repository-owned PostgreSQL engine built on the existing `pgx`,
SQLC, Goose, lifecycle, health, and OpenTelemetry owners. Reject River OSS
v0.43.0. River's public execution path does not fence completion or rescue by
the attempt generation, and its durable pause is not serialized with claim SQL.
Adapting either invariant requires replacing River's claim/finalize/rescue core,
not wrapping its supported API. A River fork would retain River's schema,
upgrades, and operational surface while adding the same private engine this
repository would still have to own.

The selected runtime is one additional `/jobs-worker` process in the existing
runtime image. The API process only stages admitted jobs in its existing
caller-owned transaction and performs a schema admission check when runtime
jobs are enabled. `/migrate` remains the only schema writer. PostgreSQL remains
the only durable authority. No broker, periodic scheduler, public job API,
cleanup loop, priority subsystem, or workflow state machine is added.

Specification does not reopen: the selected design preserves every accepted
identity, result, failure class, cancellation result, readiness meaning,
compatibility rule, and operator absence rule. Evidence that the PostgreSQL
engine cannot implement a fenced transition or commit-effective pause with the
transactions below would reopen System Design; any need to weaken B1-B13 would
reopen Specification instead.

## Drivers and affected deployment graph

### Design drivers

| Driver | Admission or rejection consequence |
| --- | --- |
| Caller-owned atomic acceptance and `ErrCommitUnknown` | Staging must accept `pgx.Tx`, perform no transaction lifecycle action, and support current-writer readback under the same identities. An engine whose native identity is not the producer identity needs extra authority and loses if that authority cannot stay atomic. |
| Attempt/effect separation | Each claim needs a monotonically fenced attempt generation. Job success is not proof of exactly-once business effect. No stale attempt may mutate the current job. |
| Independent lifecycle | The worker needs its own process, pool use, readiness, claim admission, drain acknowledgement, and hard shutdown bound. Reusing the NATS worker, outbox relay, or API supervisor is rejected. |
| Explicit kind policy | Definition validation admits only executable retry, recovery, ambiguous-effect, duration, payload, and revision policy. Effect idempotency, privacy/retention, operator, topology and capacity remain evidence-backed production checkpoints; placeholder strings cannot close them. |
| Canonical schema authority | Only six-digit transactional Goose migrations may create or evolve job schema. Runtime migration code is forbidden. |
| N/N-1 with live jobs | Kind, args, and policy versions are immutable stored facts; workers use exact registered decoders. Unknown versions remain retained and visible without being executed. |
| Profile independence and removability | `JOBS=postgres` requires only `DATABASE=postgres`. All JOBS/OUTBOX/INBOX/MESSAGING combinations must generate independently, and `JOBS=none` must leave no job residue. |
| Unknown workload and policy values | The design makes no shared-OLTP capacity, SLO, retention, fairness, or alert-threshold claim. Production registration remains closed at the named adopter checkpoints. |
| Smallest maintained surface | Prefer PostgreSQL constraints/row locks, existing pgx/SQLC/Goose, and the standard library. Add no production module and no edition/license obligation. |

### Target graph

| Node or edge | Current -> target | Authority and boundary |
| --- | --- | --- |
| Runtime image | `/service`, optional `/worker`, optional `/outbox-relay`, `/migrate` -> the same image plus optional `/jobs-worker` | `build/docker/Dockerfile`; each binary is an independent entrypoint. |
| API process | No job producer -> optional in-process producer adapter | Feature use case owns the required business mutation and pre-transaction identities; `postgresjobs` owns staging mechanics only. |
| Jobs worker | Absent -> independently deployed `/jobs-worker` | Its bootstrap owns config, registry construction, PostgreSQL connection, diagnostics, readiness, claim lifecycle, drain, and cleanup. |
| Migration process | Existing canonical Goose runner -> same runner with one append-only jobs migration | `/migrate` is the only schema writer. API and worker only verify compatible schema. |
| PostgreSQL | Existing service database -> three active job-owned relations in the same writer database | Job rows own acceptance/current state; attempt rows own attempt history; claim-scope row owns durable pause. Business data/effect truth remains outside them. |
| Operator edge | No generic transport -> still no generic transport | A later authenticated adapter may invoke the fixed control semantics only after security/data policy admission. Manual SQL is not the control plane. |
| Deployment owner | One API service definition -> an adopter adds a separately scalable worker service using the same image and `/jobs-worker` entrypoint | Platform-specific service creation, region, secrets, and resource sizing remain adopter delivery/SRE inputs. |

Network paths are API-to-PostgreSQL for staging/readback and
jobs-worker-to-PostgreSQL for claims, leases, transitions, observations, and any
same-database feature effect. A handler may use an admitted external dependency,
but that dependency and its effect authority belong to the concrete kind. There
is no JOBS-to-NATS, OUTBOX, or INBOX edge.

### B1-B13 closure map

| Rule | Realizing decision |
| --- | --- |
| B1 | Independent removable JOBS profile and `/jobs-worker`; no sibling-profile edge. |
| B2 | Feature-owned transaction, `Store.Stage(pgx.Tx)`, and writer-only unknown-commit readback. |
| B3 | Unique producer authority plus immutable-intent comparison; the retained job row is the recognition receipt, never effect idempotency. |
| B4 | Versioned typed definition/registry rejects incomplete executable policy and registers no concrete template kind; adopter-owned effect/data/operator obligations remain closed gates outside the generic Definition. |
| B5 | Distinct attempt generation, fenced transitions, lease-safe bounded control operations, and feature/downstream-owned effect policy; success never proves exactly-once effect. |
| B6 | Fenced finalize/rescue, explicit terminal classes/budgets, distinct recovery generation, retained history, and no default redrive. |
| B7 | One persisted `available_at` for a one-off occurrence; no periodic scheduler. |
| B8 | One literal neutral work class, capacity bound only; no priority/fairness mechanism or claim. |
| B9 | Durable cancel request, lease-safe timeout cancellation, attempt-scoped context, join-aware drain, and a hard process bound. |
| B10 | Immutable revisions, strict decode, authoritative execution-required revision coverage at observation/claim, retained visible terminal history, and exact-revision admission for any future redrive. |
| B11 | A runtime producer writer/schema probe and separate worker progress/readiness; backlog/SLO remains degradation evidence. |
| B12 | Cached bounded telemetry, scope-lock pause barrier, expected-generation controls, atomic idempotent action receipt, and no transport without policy. |
| B13 | One canonical transactional Goose source, SQLC output only, runtime compatibility reads only, and independent profile/image proof. |

## Engine selection

### River v0.43.0 disposition

The evaluation pins tag `v0.43.0` at commit
[`60435dc2`](https://github.com/riverqueue/river/tree/60435dc2c58e3d3dfbde6a00f08bc98a3f13c37e).

| Gate | Current evidence | Disposition |
| --- | --- | --- |
| Caller-owned transaction | `InsertTx` unwraps caller `pgx.Tx` and does not own begin/commit/rollback. | Fits. |
| Producer identity and unknown-commit readback | River uniqueness hashes River-selected fields and its readback is by River's generated integer ID, not the accepted producer identity/immutable intent. | Adaptable only with a repository acceptance authority. |
| Attempt fencing | Claim increments `attempt`, but [`JobSetStateIfRunningMany`](https://github.com/riverqueue/river/blob/60435dc2c58e3d3dfbde6a00f08bc98a3f13c37e/riverdriver/riverpgxv5/internal/dbsqlc/river_job.sql#L620-L699) predicates on row ID and `running`, not expected attempt. Rescue also updates by ID without an attempt fence. A rescued stale attempt can finalize a newer running attempt. | Rejects B5/B6. |
| Pause | [`QueuePauseTx`](https://github.com/riverqueue/river/blob/60435dc2c58e3d3dfbde6a00f08bc98a3f13c37e/client.go#L2721-L2767) persists pause, but claim SQL does not lock or check the queue row; workers learn pause asynchronously. | Rejects B12 commit-effective pause. |
| Worker readiness/drain | Startup proves database contact, while claim/maintenance loops log and retry without a supported live-loop readiness contract; fetch cancellation can finish after shutdown admission closes. | A process wrapper helps the outer bound but cannot close claim quiescence/readiness. |
| Operator semantics | Native cancel/retry lacks expected generation, stable action identity, authorization/audit CAS, and the accepted terminal classes. | Would require a private control subsystem and still not repair fencing. |
| Goose authority | River can export migration SQL. Its upstream versions must remain separate because later SQL depends on earlier committed enum changes. | Feasible, but adds several canonical files and an upstream translation/upgrade owner. |
| Compatibility | River has no general N/N-1 mixed-process guarantee. Plain JSON decode permits absent fields to become zero values; unknown kinds follow River failure handling rather than the accepted compatibility condition. | Requires a separate strict versioned envelope and rollout contract. |
| OSS/Pro | No identified Pro feature repairs attempt fencing or claim-time pause. | Pro is unnecessary and does not change the rejection. |
| License/pre-v1 | MPL-2.0 and pre-v1 acceptance is unavailable from the adopter owner. | Because River is rejected, this gate is not crossed and no legal decision is guessed. |

Measured with Go 1.26.5 on darwin/arm64 and a temporary exact-module
`pgxpool`-only versus River client probe, the stripped baseline binary was
8,109,874 bytes and
the River binary 8,970,946 bytes: +861,072 bytes (about 0.82 MiB, 10.6%). Seven
no-database launches had median maximum RSS 11,780,096 versus 11,976,704 bytes:
+196,608 bytes; elapsed-time resolution was too coarse to support a startup
claim. River added ten external module paths relative to the current module
requirements and upgraded `pgerrcode`; a shared fresh build cache measured
about 2.48 seconds of incremental compile work after the pgx baseline. These
measurements exclude a database, handlers, workers, and telemetry and make no
loaded-runtime or capacity claim.

### Same-level alternatives

| Alternative | Decision |
| --- | --- |
| No module | Rejected for this accepted capability; adopters that do not need it select `JOBS=none`. |
| River through supported adapters | Rejected by unfenced completion/rescue and asynchronous pause. |
| River fork or replacement driver/pilot | Rejected: it owns River core across multiple modules plus River schema/upgrades while still requiring repository acceptance, compatibility, operator, readiness, and telemetry subsystems. |
| Repository PostgreSQL engine | Selected: native row locks and conditional updates express the accepted transactions exactly, current dependencies already own all database/generation machinery, and only the required state machine is shipped. |
| Integration messaging, platform scheduler, workflow engine | Not same-boundary substitutes. Reopen to the owner table in `spec.md` only when another service owns the effect, periodic calendar policy is admitted, or durable orchestration becomes intrinsic. |

The selected custom code is narrower than a River fork, not a new general queue
framework. It supports one PostgreSQL database, one neutral work class, one-off
availability times, typed registered handlers, and B1-B13 transitions only.

## Durable authorities and state model

The planned canonical migration is `migrations/000003_postgres_jobs.sql` in the
current tree. If another accepted change claims version 000003 before
implementation, the deterministic rule is the next unclaimed six-digit version
with the same `_postgres_jobs` stem. This design creates no migration.

PostgreSQL server time is authoritative for availability, leases, and
transition timestamps. Names below are active logical schema ownership; the
migration set owns exact SQL and constraints.

| Relation | Authority and minimum facts |
| --- | --- |
| `postgres_jobs` | One row per logical job. Stores bounded opaque logical, producer-scope/key, occurrence-scope/ID, and effect-scope/key identities; immutable-intent fingerprint; kind, args version, policy version, immutable validated JSON bytes, neutral work class; current state; `available_at`; recovery generation; monotonically increasing attempt generation; current recovery attempt count and budget-start timestamp; current worker/lease; and created/updated/terminal timestamps. Unique constraints prevent a second logical job, producer acceptance, occurrence, or effect authority from being created under conflicting identities. The row is the producer readback authority and is never automatically deleted. |
| `postgres_job_attempts` | One row per `(logical_job_id, attempt_generation)`, including recovery generation, worker identity, start/lease/final timestamps, bounded outcome/failure code, and possible-effect classification. It is attempt history; it never becomes business-effect truth. |
| `postgres_job_claim_scopes` | The single `neutral` work-class row with paused flag and scope generation. Claim transactions take a shared row lock; pause/resume takes an exclusive row lock and changes the generation. This serializes a committed pause with all claim transactions without a broker or notification subsystem. |

Migration `000004` also created `postgres_job_actions` before any operator
capability was admitted. It is now a deprecated compatibility relation, absent
from schema admission, runtime privileges, and proof. A later append-only
contract migration removes it only after the N-1 worker that still checks the
original exact schema has left the rollback window; it grants no current
operator authority.

Identity and bounded audit values are opaque application strings of 1-256
bytes in C-collated text columns; database comparisons are exact, not
locale-folded. Each definition fixes a positive payload limit no greater than
256 KiB; absence or excess rejects admission. Payload
is stored as `bytea` with immutable `kind`, `args_version`, and
`policy_version`; SQL never interprets or queries argument fields. A job
definition owns the deterministic SHA-256 immutable-intent bytes and golden
vector; the engine compares the 32-byte fingerprint and never treats serialized
JSON order as business identity. The definition also owns strict decode and
validation: unknown fields, trailing values, absent required fields, invalid
zero values, unknown versions, and malformed data never invoke the handler.

No generic retention clock exists. The job row is also the deduplication
receipt, so removing it would remove acceptance/readback authority. There is no
separate receipt table, DLQ table, schedule table, priority table, or schema
metadata table.

### State and transition authority

Current states are `ready`, `scheduled`, `retry_wait`, `running`,
`cancel_requested`, `succeeded`, `cancelled`, `exhausted`, `permanent`,
`poison`, and `outcome_unknown`. Due `scheduled` and `retry_wait` rows may be
claimed directly; a maintenance transition to `ready` is unnecessary.

Every mutation of a running job predicates on logical job ID, expected attempt
generation, expected recovery generation, current state, and current worker
where applicable. Zero affected rows is a stale/state conflict and changes no
job or attempt fact. A claim increments the generation and inserts its attempt
row in the same transaction. Rescue finalizes the expired attempt before it
opens eligibility for a later generation. Manual redrive, when admitted, keeps
all logical/producer/occurrence/effect identities, increments recovery
generation, and follows the kind's explicit budget-reset rule.

Finalize and rescue linearize on the current job row. Finalize may still win
after the lease timestamp passes while its generation remains current; rescue
may act only after it locks that expired generation. Once rescue commits the
state/generation transition, the old finalizer affects zero rows and cannot
overwrite the recovery decision.

The engine never infers effect absence from timeout, cancellation, panic,
connection loss, or worker death. Each definition maps those infrastructure
outcomes through its admitted B5/B6 policy to a safe retry, terminal class, or
`outcome_unknown`. Handler success wins a race with cancellation only through
the same fenced transaction. A cooperative cancellation may become
`cancelled` only when the definition supplies affirmative no-effect evidence;
partial or ambiguous work becomes `outcome_unknown` unless the admitted effect
authority proves another transition.

That mapping is one pure evaluator owned by the immutable definition revision.
The engine captures bounded handler/panic/timeout/cancel/lost-attempt facts and
invokes it with persisted budget facts and PostgreSQL observation time. Store
stages persist only the resulting transition, delay, and budget mutation; they
never reclassify or supply a default.

## Material flows

### 1. Kind admission and startup

A feature owns one typed definition per immutable `(kind, args_version,
policy_version)`. The definition contains every B4 slot; it has no default
retry, effect, schedule, cancellation, retention, operator, duration, or work
class. Construction fails with bounded diagnostics naming missing fields, not
their values. The producer can prepare only a complete definition. The worker
registry rejects duplicate keys and requires one typed handler for every
selected definition.

When the first concrete jobs producer is composed, the API runs the same exact
canonical schema verification as the worker, including every relation, column,
check, and producer-identity uniqueness constraint/index required by Stage and
readback. It then checks writer state and required producer privileges. This
bounded read-only producer-path sequence replaces the generic PostgreSQL probe
for each later readiness evaluation; it never migrates or mutates data. Pool
saturation preserves the repository's existing capacity-only readiness
meaning, while every completed probe detects a read-only target, missing
producer authority, or incompatible schema. A concrete feature's composition
must construct every required producer definition before serving; otherwise
its own startup fails. The generic template has no concrete producer, so it
adds no jobs producer probe.

`/jobs-worker` rejects a missing registry builder, disabled jobs, disabled
PostgreSQL, incomplete definitions, missing handlers, incompatible schema,
registry coverage missing any retained job revision, or unusable diagnostics
before opening claim admission. Policy/config failures exit; transient
compatibility loss after startup makes the process unready and closes claim
admission. An unusable reserved control Session is terminal for that process;
an acquired pgx connection never reconnects in place.

### 2. Atomic acceptance and ambiguous commit

Before the transaction, the feature definition validates typed arguments and
produces immutable identities, strict payload bytes, intent fingerprint, and
one-off `available_at`. The feature use case owns the transaction through
`postgres.Pool.InTx`; its narrow feature-owned transaction port keeps `pgx.Tx`
out of core feature behavior.

Inside that transaction, `postgresjobs.Store.Stage` uses the caller's `pgx.Tx`
and performs no begin, commit, rollback, or retry. It inserts exactly one job or
reads the conflicting unique authority. It returns staged-new, identical
existing acceptance with the original logical ID, producer conflict, or
rejection. Any logical/occurrence/effect uniqueness collision with different
immutable identity is an integrity conflict. The feature operation receipt,
not the job duplicate result, decides whether a repeated business mutation may
commit.

After `postgres.ErrCommitUnknown`, the caller preserves every identity and the
prepared immutable-intent fingerprint, then runs `ResolveAcceptance` against
the configured writer. The read requires a writable primary through both
`pg_is_in_recovery()` and `transaction_read_only` and never treats a replica,
read-only target, timeout, cache, or failed read as absence. Matching identity and fingerprint means accepted; a
successful writer read with no row means not accepted; differing identity or
fingerprint means integrity conflict;
anything else remains unknown. The pack does not retry the business mutation.

### 3. Claim, attempt, effect, and finalization

One worker coordinator acquires one connection from the existing pgx pool for
the engine lifetime. It serializes claim, batched lease renewal/cancellation
polling, finalization, rescue, and state observation on that connection; handler
business I/O uses ordinary feature dependencies. Reserving the connection
prevents saturated handlers from starving ownership renewal. Configuration
requires at least one additional pool connection and production headroom is not
claimed without adopter workload evidence.

Every reserved-session operation runs in a transaction with a child context
bounded by explicit `StoreOperationTimeout`. The transaction gives local
`statement_timeout` and `lock_timeout` a deterministic server-outcome margin:
they use the smaller effective Store/PostgreSQL budget less than 10% (capped at
10 ms), while the child context retains the full Store budget. Jobs admission
rejects a Store operation budget below 100 ms, so PostgreSQL's millisecond
parameter rounding cannot erase that ordering. Client cancellation still wins;
otherwise PostgreSQL reports `55P03` or `57014` before the client deadline, and
both remain inspectable beneath the typed operation timeout. This bounds server
query and lock lifetime without changing unrelated pool traffic. Read-only
observation uses the same transaction wrapper. Config admission requires
`LeaseDuration >= 6 * StoreOperationTimeout`; enabling jobs rejects absence or
overflow of that relationship.

Renewal becomes due one-third of a lease after its last successful claim or
renewal commit and has priority over claim, observation, rescue, and
finalization work. Once due, the coordinator starts no later operation until
renewal succeeds or ownership is declared unsafe. A non-renewal operation that
starts just before renewal is due ends within one operation budget; renewal
then ends within a second, no later than two-thirds of the lease. Any operation
or renewal timeout/transport error closes claim admission and readiness,
signals every owned attempt context before lease expiry, and returns a typed
terminal engine failure. Lifecycle withdraws readiness, invokes the engine's
drain exactly once, applies the existing hard bound, and propagates its result;
the engine drain alone quiesces, cancels, and joins attempts. `run` then releases
the broken Session so pgxpool can destroy it and returns the error; `main`
classifies it as a nonzero exit. There is no in-process Session replacement or
blind control-operation replay. A replacement process acquires a fresh connection,
re-runs schema/registry admission, and uses fenced rescue for durable running
state. An uncooperative effect remains governed by its admitted duplicate/
late-effect policy; cancellation is never treated as proof that no effect
occurred.

On each coordinator poll, due renewal runs first. The coordinator then examines
at most `MaxConcurrency` expired attempts, one bounded Store operation at a
time, and invokes the exact stored revision's lost-attempt evaluator before a
fenced rescue write. It rechecks renewal priority between candidates. Claim
runs next only when handler capacity is free; due observation runs last. This
is one serial coordinator cycle, not a rescue goroutine or daemon, and it makes
recovery progress without permitting a rescue batch to delay lease safety.

When handler capacity is free, one claim statement first compares the worker's
sorted exact registry keys with the authoritative distinct revision inventory
of execution-required states in the same PostgreSQL snapshot. Unknown terminal
history remains visible but does not block unrelated claims; any future redrive
must recheck its exact revision before changing state. If any execution-required
key is absent, claim returns only the scope row and a compatibility fault closes
this process's admission and readiness. Otherwise the short transaction takes a
shared lock on the neutral claim-scope row, verifies it is not paused, and
selects up to the free capacity using `available_at, logical_job_id FOR UPDATE
SKIP LOCKED`. For each selected row it increments attempt generation, records
worker/lease, moves to `running`, and inserts attempt history. An acceptance
that commits after the statement linearizes after that claim; the next claim
snapshot must cover it. Only a known commit hands the attempt to a handler. An
unknown claim commit is resolved on the writer by the expected job/generation/
worker tuple before execution. If the control Session cannot perform that
readback, no handler starts and the process follows the terminal failure path;
the durable attempt is recovered after its lease.

The coordinator registers every known-committed claim in the in-flight join
before it can observe or acknowledge drain. It never leaves a committed claim
in an untracked handoff queue. If registration itself cannot complete, the
attempt remains durably visible but receives no effect; the coordinator stops
renewing it and lease recovery applies the kind's lost-attempt policy.

The handler receives typed arguments plus logical job, attempt generation,
recovery generation, occurrence ID, and business-effect key. Its context is
bounded by the definition's maximum duration and by worker cancellation. The
feature or downstream authority enforces the admitted duplicate/late-effect
policy. The handler returns a bounded outcome and safe code; arbitrary error
text stays in controlled logs, never state or metric labels.

Completion is a fenced transaction that updates the current job and attempt
row together. A retry schedule is calculated once and persisted. Backoff,
retry-hint precedence, cap, and jitter come from the versioned definition;
deterministic jitter is derived with standard-library SHA-256 from stable job
and attempt facts so replay cannot choose another time. A repeated completion
returns the recorded attempt outcome; a newer generation rejects the stale
completion.

### 4. Lost ownership, retry, and recovery

The coordinator renews active leases in batches under the priority and timeout
rule above and simultaneously observes durable cancellation. Losing the
control connection, exceeding an operation budget, or failing renewal closes
readiness and new claims, signals active handler contexts within the remaining
lease margin, and terminates this worker through bootstrap. A later process is
the only restoration owner. No process reports ownership it cannot renew;
fencing still decides which process may persist a terminal result.

The coordinator's bounded rescue stage reads an expired current attempt and
classifies the lost outcome using the exact stored definition. The fenced Store
write locks and rechecks that generation before recording it as expired with
possible-effect ambiguity. The definition decides whether the job becomes
retry-wait, exhausted, permanent, poison, or outcome-unknown. A retry remains
within explicit attempt/elapsed-age/recovery-wave budgets. The next claim
receives a new attempt generation; the old attempt cannot finalize it.

If finalization commit is unknown, the coordinator first reads the job and
attempt on the writer. It returns the stored result when already committed,
repeats the same fenced finalization only when the old attempt is still current,
and otherwise reports stale/unknown. This readback/repeat is allowed only while
the reserved Session remains usable; loss of that Session takes the terminal
process path and leaves later fenced recovery to observe durable truth. It
never creates a new logical job or effect key.

### 5. Cancellation, pause, and shutdown

An admitted queued cancellation is a job-row compare-and-set on expected state
and generation. A running cancellation records `cancel_requested` against the
expected attempt; the coordinator observes it during renewal and cancels that
attempt context. Final outcome follows B9 and the definition's effect evidence.

Pause/resume is a transaction over the neutral scope and action audit. Claim
transactions hold a shared lock until their claim commit; pause takes the
conflicting exclusive lock. Therefore a successful pause commit waits out all
claims that observed the prior state, and later claim transactions see paused.
Acceptance remains open. Resume changes only scope state/generation.

On process shutdown, jobs-worker first turns readiness off and closes its local
claim admission. It waits for the sole coordinator to acknowledge that no claim
transaction is open or can start and that every committed claim is registered
in the handler join. Already committed attempts are in flight.
The worker then spends the configured soft-drain bound, cancels remaining
attempt contexts, and records the one drain outcome. A terminal control-Session
fault has already closed admission and signalled every owned attempt before this
drain begins; it is not retried, reconnected, or reclassified by shutdown.

If that drain is unsafe because a handler did not join, the worker returns its
terminal error immediately after the one drain result. It does not spend the
remaining process grace on diagnostics, telemetry, pool, Session, or registry
cleanup, and it does not run a second drain. Those resources remain owned by
the exiting process, so OS process exit—not an in-process replacement or
cleanup retry—ends them. This leaves the durable running attempt untouched for
lease expiry and a distinct newly admitted process to rescue through the
existing generation fence. Diagnostics shutdown is best-effort only after a
safe drain; it cannot delay an unsafe terminal exit or change its nonzero
classification.

The rejected alternative is to continue joining diagnostics or dependencies
after the unsafe drain. It consumes the same hard shutdown budget without
making the live handler safe, can delay the fresh fenced recovery past the
process carrier's recovery window, and creates a second lifecycle wait with no
new authority. The selected boundary retains the existing engine-to-lifecycle
terminal chain and `run`'s safe-versus-unsafe cleanup ownership. Test Design
must extend TD-JOBS-017's real-process oracle to prove the unsafe path exits
after one drain without a diagnostics/dependency join deadline, while retaining
pre-expiry cancellation, nonzero exit, distinct-PID fresh admission, and
fenced-only recovery. Reopen System Design if that immediate unsafe exit cannot
leave restoration solely to a fresh process; reopen Go Ownership only if the
existing engine/lifecycle/run ownership cannot express this boundary.

### 6. Operator controls

The template exposes no REST, gRPC, CLI, or diagnostics mutation endpoint. A
concrete authenticated adapter is admitted only after security/data owners
close roles, scopes, redaction, audit, and retention. It passes verified actor
and authorization context into the internal controller, which revalidates the
kind policy, expected state/generation, reason, target, and stable action ID.

Until that checkpoint, no Controller is exported or constructed and no
operator query or adapter is shipped; every inspect/mutation attempt is absent,
not implicitly authorized. The immutable definition revision owns the admitted
pure minimization/redaction evaluator. Once a present adapter justifies the
surface, the controller applies that evaluator and returns only the permitted
internal view; the adapter owns authentication, authorization evidence, and
transport rendering, not redaction logic.

Inspect is read-only and redacted by the admitted policy. Pause/resume, cancel,
and redrive use the transactions above and append the first action result.
Succeeded/nonterminal redrive, unauthorized/stale actions, action-ID reuse for
different intent, and every delete without complete retention/legal-hold/
effect/audit authority are no-op conflicts. There is no default manual recovery
and no manual SQL production path.

An operator mutation and its first action result commit in one controller-owned
transaction through the existing PostgreSQL transaction seam. An unknown
commit outcome is resolved on
the current writer by stable action identity and request fingerprint: the
matching stored result is returned, a conclusive absence permits the same
action to be retried, a different fingerprint conflicts, and an unavailable or
non-writer read remains unknown. A blind action retry is forbidden.

### 7. Telemetry and readiness

`postgresjobs.Telemetry` owns job instruments and closed vocabularies. The
coordinator periodically queries counts/oldest timestamps by closed state,
registered kind, and neutral work class, then replaces an in-memory snapshot.
OpenTelemetry callbacks read only that snapshot. They perform no database I/O
and expose observation freshness when PostgreSQL is unavailable.

Counters/histograms cover acceptance result, bounded Store-operation
duration/outcome, claim, attempt outcome, retry, rescue, cancellation, terminal
engine failure, and drain. Gauges cover state depth/oldest age,
in-flight/capacity, observation freshness, and readiness.
Raw payloads, identities, tenant values, and arbitrary error strings are never
metric labels. Access-controlled logs/traces may carry identities only after
the job's data policy admits them. Existing pgx telemetry remains the owner of
pool signals; PostgreSQL operational evidence supplies WAL/vacuum pressure.

API readiness is its existing service readiness plus, when a concrete jobs
producer is present, the bounded runtime producer-path sequence described above.
Concrete feature composition must have admitted every job it requires before
serving. Backlog and worker failure do not automatically make an API that can
still durably accept work unready.

Worker readiness is true only while schema, registry coverage of every
execution-required revision, reserved control connection, claim loop, and a
locally age-bounded telemetry observation are current and the process is not
draining. The observation deadline is derived from its interval plus one poll
and one Store-operation budget, and never compares a PostgreSQL timestamp with
the worker clock. Periodic observation refreshes coverage; a claim-time coverage
fault closes admission before another claim.
Backlog size, SLO thresholds, downstream business dependency health, and
shared-OLTP capacity are separate degradation signals, not invented readiness
policy. Liveness remains process-local.

The jobs-worker bootstrap lifecycle is the single owner that aggregates and
publishes that final predicate. Engine and telemetry expose component facts;
telemetry's readiness instrument reports only the engine component predicate.
Neither component publishes a competing final diagnostics answer.

## Compatibility, migration, profiles, and release closure

### Compatibility

Stored `kind`, `args_version`, and `policy_version` are immutable. The distinct
keys of rows in `ready`, `scheduled`, `retry_wait`, `running`, and
`cancel_requested` are the authoritative execution-required inventory.
Periodic observation compares that inventory with the exact registry. Claim
repeats the comparison in its own statement snapshot and claims no row on any
gap. Known versions use strict typed decode; malformed known payload becomes
fenced `poison` without business execution. An unknown execution-required
version is never claimed or rewritten by that process; it remains retained and
observable while the process is unready. Terminal history remains stored but is
not a global readiness dependency. A future authenticated redrive path must
check its target's exact revision before changing state and remains absent now.

Every new revision uses a mandatory expand/enable/contract sequence. The prior
expand worker release, designated N-1, first ships the new definition/handler
alongside every execution-required revision while producers continue emitting only old
revisions. Its exact image and registry are proven as the rollback authority.
Release N may roll and its producer may emit the new revision only after both
the active N fleet and rollback N-1 image cover every execution-required plus candidate
revision. Thus N and N-1 can execute every live job throughout the live-job and
rollback window. If the new handler cannot be forward-shipped in N-1, emission
remains blocked and Specification must reopen; draining an incompatible N-1 is
not an alternative.

Existing names remain aliases only as their original stored keys; a rename
never rewrites a live job. The code registry remains authoritative for every
execution-required policy revision, not merely the two newest releases;
behavior-changing scalar policy and handler logic are never looked up under a
reused revision. Producer and worker both fail closed when the exact stored
revision is absent. Old definitions, schema, and rows are deleted only in a
later contract release after the delivery owner closes the maximum live-job/
rollback window and the data owner closes retention.

### Migration/profile authority

The jobs schema is one transactional Goose migration because no upstream River
history remains. It contains an `Up` and disposable-environment `Down`; a
production rollback does not run destructive `Down`. SQLC source is
`internal/infra/postgres/queries/postgres_jobs.sql`; generated Go is output
only. Runtime startup performs compatibility reads, never migration.

`scripts/init-module.sh` gains `JOBS=none|postgres`, default `none`, with
fail-before-mutation validation that `postgres` requires `DATABASE=postgres`.
The `jobs-postgres` marker owns all job code, config, docs, image, command,
query/generated output, migration, and proof surfaces. Removing JOBS discovers
`[0-9]*_postgres_jobs*.sql`, removes job query/generated files, then regenerates
the surviving shared SQLC package once. It neither tests nor changes OUTBOX,
INBOX, or MESSAGING selections.

Template proof covers JOBS alone and with every sibling profile, default and
explicit none, invalid/empty selections, no-destination-change failure,
double-initialization stability for all retained profiles, template-lock
recording, no removed-profile residue, generated drift, and runtime image
presence/fail-closed execution of `/jobs-worker`.

The operational sequence is recorded in [`../rollout.md`](../rollout.md).
Migration precedes compatible workers; workers precede any producer that emits
a new definition. No schema or old definition is contracted in the initial
release. Platform region, worker service definition, resource limits, secrets,
and replica count remain deployment-owner inputs.

## Performance and cost boundary

This is a `constraint_only` decision. No adopter workload, backlog SLO, recovery
objective, or database headroom is available, so this design makes no latency,
throughput, connection, CPU, WAL, vacuum, storage, or recovery-duration claim.

Structurally, one accepted job adds one insert in the caller transaction; one
attempt adds one short claim transaction, periodic batched renewal/cancel
observation, and one fenced finalization; rescue adds one fenced transaction.
Each claim also compares the sorted registry with the composite-indexed distinct
execution-required revision keys. That conservative scan and the single coordinator/
reserved connection are deliberate first ceilings: they avoid a second durable
revision catalog and private pool scheduler. Claim work remains bounded by free
handler capacity and is indexed by neutral class, state, availability, and
stable ID. Store-operation and lease ratios are safety bounds, not latency
claims. Handler memory is bounded by explicit concurrency and admitted payloads.
Reopen only when representative measurement shows revision coverage or
serialized transitions miss an accepted throughput/recovery budget; do not
weaken the lease margin to recover throughput.

Before a shared-OLTP production-readiness claim, the adopter supplies arrival
and burst distributions, running duration/payload, retry/recovery wave,
retention, topology/failover, connection budget, and SLO. Required proof then
measures enqueue latency, claim/finalize throughput, renewal headroom,
backlog-recovery amplification, index/storage growth, WAL/vacuum pressure,
worker RSS/CPU, image/binary delta, and N/N-1 rolling behavior on the target
shape. CPU or memory limits follow representative peaks; none are guessed here.

## Adopter checkpoints and fail-closed behavior

The input-closure table in `spec.md` is unchanged:

- no concrete production emission without effect identity/tolerance,
  retry/dependency budgets, maximum duration, payload classification,
  retention, and target operational evidence; executable fields are validated
  by Definition while external obligations require owner receipts;
- no periodic schedule, multiple class/priority/fairness, operator transport,
  manual recovery, deletion, SLO/alert, capacity, or production topology claim
  without its named owner;
- no external submit/status/cancel API without a feature Specification;
- no removal of N-1 definitions/schema without delivery-window authority;
- no River license/pre-v1 decision because River is not selected.

The first implementation may build the generic pack, canonical schema, worker,
and fail-closed registry surface. It cannot claim a production-ready handler or
deployment until adopter-owned inputs close at their named checkpoints.

## Required downstream proof and reopen conditions

Technical Design fixes proof ownership but does not enter Test Design. Later
proof must cover atomic commit/rollback/unknown readback, all identity
collisions, stale completion/rescue, pause/claim serialization, cancellation
races, a lock/query blocking every serialized operation stage, renewal
preemption and timeout cancellation before lease expiry, uncooperative
shutdown, classification/budget exhaustion, strict decode, N/N-1
rolling/rollback, no-DB telemetry collection, profile independence, canonical
migration history, image entrypoints, and representative cost when a production
claim is requested.

Reopen System Design when measured transition throughput requires more than one
coordinator, the job database is no longer the caller transaction database,
another service owns the effect, or a required operator capability needs a
separate durable owner. Reopen Specification for any proposed change to
acceptance results, identities, effect guarantee, retry/cancellation/schedule
meaning, readiness, compatibility, or operator results. Route periodic calendar
work, multi-step durable progress, and cross-service delivery to their existing
owner boundaries instead of extending this engine.

## Technical Design review receipt

Independent Technical Design review returned `PASS` on the fixed candidate
after reconstructing every material B1-B13 flow and consuming the compatible
Go ownership panel receipt. The final review specifically falsified River
selection, transactional acceptance/readback, all-claim revision coverage,
mandatory prior-expand N/N-1 rollback, producer readiness, lease/rescue
safety, terminal control-Session replacement, operator absence, migration, and
profile independence. No downstream owner must invent behavior, mechanism,
authority, or ownership. The verdict is design-only; executable proof and
adopter production readiness remain downstream.
