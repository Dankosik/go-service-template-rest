# Optional PostgreSQL Durable Background Jobs Research

status: ready
Valid as of: 2026-08-11
Boundary: Research only. This note does not define Specification, Technical Design, Test Design, Planning, migrations, or implementation.
Independent review: PASS on 2026-08-11 after repair; no material findings remain.

## Executive conclusion

The candidate space is saturated enough to carry one leading fit into
Specification: **River OSS v0.43.0 is the strongest current library candidate
for a provisional `JOBS=postgres` pack**, because it is the only maintained
candidate found that combines all three of the decisive local constraints:

1. enqueue through the repository's caller-owned `pgx.Tx` without taking over
   commit;
2. typed Go arguments and typed worker registration; and
3. a broad PostgreSQL job lifecycle including durable one-off scheduling,
   retries, uniqueness, priorities, queues, cancellation, maintenance, and a
   separate insert-only/client versus worker-process shape.

That is a Research fit hypothesis, not an architecture selection. River's
defaults are not acceptable as product policy, its v0 compatibility and
MPL-2.0 license require explicit acceptance, its migration model collides with
the repository's canonical Goose-only SQL rules unless a reproducible pinned
integration is proven, and its OSS boundary excludes durable periodic
scheduling, first-class DLQ, global concurrency limits, and workflows.

The smallest honest capability remains narrower than “all background work”:

- one service owns the job and its effects;
- durable acceptance must normally commit with that service's PostgreSQL state;
- the job is bounded, reentrant, and safe under overlapping attempts;
- the worker is a separately deployed process with its own pool, readiness,
  drain, telemetry, and capacity envelope; and
- queue load is allowed to share the service database only after representative
  evidence establishes OLTP headroom, vacuum/WAL behavior, and recovery time.

If those conditions are absent, no job module, the existing NATS/outbox path, a
platform scheduler, or a workflow engine is the smaller and more truthful
owner.

## Accepted outcome and stop condition

Research an optional template-init-selected capability, provisionally
`JOBS=postgres`, that can durably enqueue service-internal jobs in the same
transaction as business state and can run typed business handlers in a
production-owned worker process. Feature code should not own database polling,
claiming, retry scheduling, process lifecycle, readiness, drain, or generic
telemetry.

Stop Research when the repository baseline, decision boundary, candidate
space, leading fit, counter-evidence, owner-controlled semantics, proof
obligations, rejection conditions, and refresh triggers are explicit enough
for a fresh Specification session. Do not decide package/file placement,
schema, migration bytes, command names, configuration keys, or implementation
tasks here.

## Open-item map

| Item | Decision it could change | Evidence and method | Research disposition | Downstream owner |
| --- | --- | --- | --- | --- |
| R1. Does a durable job module need to exist? | Whether the template should carry any new permanent surface | Repository discipline, current capabilities, synchronous/reconciliation alternatives | Conditional. It exists only for accepted work that must outlive a request/process/deploy or needs durable retry, one-off scheduling, throttling, or operator recovery. | Specification states the behavior delta and no-module conditions. |
| R2. Current repository seams | Whether the pack can reuse existing Postgres, lifecycle, telemetry, migration, and profile owners | Current code/docs traced end to end | Established. Reuse platform seams; jobs remain independent of outbox, inbox, and NATS semantics. | Technical Design later fixes composition and placement. |
| R3. Durable acceptance and execution guarantee | Whether direct PostgreSQL enqueue is the right primitive | Caller-owned transaction source, River v0.43.0 source/docs, repository durable-jobs discipline | Same-DB atomic acceptance is a strong fit. Execution remains at-least-once in effect: overlapping attempts and ambiguous completion must be safe. | Specification owns observable acceptance/effect behavior; Design owns mechanism. |
| R4. Job lifecycle semantics | Whether OSS River's features and defaults are sufficient | Release-pinned River source/docs plus operational counter-evidence | Capabilities exist, but retry, poison, cancellation, priority, fairness, scheduling, compatibility, and operator policies are unresolved inputs—not inherited defaults. | Specification fixes behavior; later phases prove it. |
| R5. Candidate space | Whether another maintained implementation dominates River or avoids a dependency | Reuse ladder: no module, repository/stdlib/native, installed dependencies, mature OSS, managed/broker, custom, workflow | Saturated for current Go + PostgreSQL + caller-owned pgx + typed-worker boundary. River leads; no candidate is selected here. | Technical Design rechecks the pinned candidate before selection. |
| R6. PostgreSQL operational fit | Whether queue churn may share the service database | PostgreSQL contract, production articles/talks, current pool/readiness/migration model | Unknown until a workload exists. Shared OLTP is a conditional topology, not a readiness claim. | Business supplies workload/SLO; Design/Performance/Delivery own budgets and proof. |
| R7. Boundaries to messaging, cron, and workflows | Which capability owns each kind of asynchronous work | Current NATS/outbox contracts, Kubernetes/AWS scheduling, Temporal contract and production evidence | Established boundary criteria below. | Specification scopes one capability; architecture reopens if criteria change. |
| R8. Maintenance, license, upgrade, and dependency cost | Whether River is acceptable to a reusable template | Current release/tag/source, license, module graph, upgrade docs | Active and locally compatible at v0.43.0; legal policy, pre-v1 tolerance, exact binary/image cost, and migration integration remain gates. | Legal/dependency policy is external; Design/Delivery measure the rest. |

## Evidence lenses and limits

| Lens | Disposition |
| --- | --- |
| Current repository/runtime authority | Researched. Current `main` was clean at `40e6d212` when the baseline began; CodeGraph was current. Concurrent untracked work outside this feature path was not inspected or changed. |
| Current upstream contract and source | Researched against release-pinned River v0.43.0 plus current primary documentation. Candidate source and release activity were checked as of the valid date. |
| Current alternatives | Researched across maintained Go/PostgreSQL libraries, broker/managed queues, database primitives, and durable workflow engines. |
| Credible production counter-evidence | Researched. Reports are treated as workload-specific evidence, not portable capacity claims. |
| Current adopters/live deployments | Not observable. The template ships no adopter inventory or target job workload. No production-readiness claim is made. |
| Empirical performance/dependency-size proof | Not run because no workload, SLO, selected candidate, or implementation exists. This is a later proof obligation, not a reason to invent a target. |
| Legal approval | Not available. MPL-2.0 text was verified; compatibility for a particular adopter is a legal-policy decision. |

## Capability boundary

| Need | Correct owner | Why it is not `JOBS=postgres` by default |
| --- | --- | --- |
| Service-internal durable work | PostgreSQL job module when the same service/database owns acceptance and effect | This is the target boundary: one logical job, typed args, one business handler, bounded retries, and no cross-service delivery contract. |
| Cross-service event/command delivery, fan-out, replay, heterogeneous consumers | Existing NATS JetStream messaging, with PostgreSQL outbox/inbox when transactionality/idempotency is required | Subjects, schemas, consumer identities, acknowledgement, retention, replay, and independent consumer scaling are integration contracts. A service must not expose or let another service poll its private job tables. |
| Simple periodic trigger | Platform scheduler such as Kubernetes CronJob or a managed scheduler | A scheduler creates occurrences; it does not make business completion exactly once. It is smaller when the trigger is idempotent and does not need same-transaction acceptance. |
| Long-lived multi-step process | Workflow engine such as Temporal | Durable waits, signals/humans, compensations, child processes, checkpoints across many releases, execution history, and workflow versioning are orchestration—not a single retryable handler. |
| Short work whose result the caller needs and which fits the request budget | Synchronous feature code | Backgrounding would hide an error and add a worker/schema/operations surface with no durability benefit. |
| Derived or repairable work with a reliable source-of-truth scan | Reconciliation or bounded platform batch | A durable per-item job may be redundant when a deterministic scan can cheaply reconstruct missing work. |

Kubernetes explicitly documents that a CronJob may create two Jobs or no Job
for an occurrence and requires idempotent work. Temporal persists workflow Event
History and replays deterministic coordination while external I/O runs in
retryable Activities. Both facts reinforce rather than blur the boundaries.

## Current repository baseline

### PostgreSQL and transaction ownership

- [`internal/infra/postgres/transaction.go`](../../../internal/infra/postgres/transaction.go#L13)
  owns begin/commit/rollback and hands the use case a concrete `pgx.Tx`.
  Unknown commit outcome is preserved as `ErrCommitUnknown`; retries remain
  caller-owned because an outbound effect may already have occurred.
- [`internal/infra/postgres/postgres.go`](../../../internal/infra/postgres/postgres.go#L235)
  exposes the already configured `*pgxpool.Pool` at composition and supplies
  bounded acquisition, statement/idle-transaction timeouts, health, tracing,
  metrics, and close.
- Atomic enqueue must therefore accept the existing `pgx.Tx` and must not begin,
  commit, retry, or hide transaction state. A lost commit response is an
  ambiguous acceptance: retry uses the same producer-deduplication/effect
  identity and durable readback, not a fresh logical job.

### Background lifecycle and executable ownership

- [`internal/background/background.go`](../../../internal/background/background.go#L1)
  says explicitly that it is a supervisor, not a job framework. It owns
  cancellation, panic containment, join, and failure reporting; schedules,
  retries, and locks stay with the task.
- [`cmd/service/internal/bootstrap/run.go`](../../../cmd/service/internal/bootstrap/run.go#L22)
  keeps supervised API work alive through HTTP drain, then bounds diagnostics,
  background join, dependency close, and telemetry flush. Durable job execution
  must not be hidden inside this API supervisor.
- [`cmd/worker/internal/bootstrap/run.go`](../../../cmd/worker/internal/bootstrap/run.go#L21)
  and [`lifecycle.go`](../../../cmd/worker/internal/bootstrap/lifecycle.go#L45)
  establish the reusable process shape: binary-local business adapter, startup
  admission, diagnostics, readiness, panic containment, stop-claims-first
  drain, bounded handler join, safe cleanup, and final telemetry flush. Their
  NATS types and ACK/DLQ semantics are not reusable job policy.
- The intended outcome already requires a production-owned worker process, and
  current repository evidence makes an independently deployed lifecycle the
  leading implication. Research does not decide whether that is a new
  executable, an existing worker executable with another adapter, or another
  composition; Specification owns the observable separation and Technical
  Design owns the mechanism, name, and package layout.

### Readiness and drain

- API readiness already reports PostgreSQL dependency health. It must not become
  coupled to a separately scaled worker's liveness; an accepted durable backlog
  can survive a worker outage and should be surfaced through age/SLO alerts.
- Worker readiness should mean “this process can safely claim supported jobs”:
  compatible schema, registered handlers, usable PostgreSQL path, live worker
  loop, and not draining. It should not mean queue empty, downstream healthy, or
  backlog below an alert threshold.
- The repository's Postgres probe deliberately treats pool saturation as
  reachable/healthy, so job capacity needs separate saturation telemetry rather
  than readiness-triggered restart loops.
- The current NATS worker and outbox relay provide decision evidence for a later
  job-worker contract: readiness is withdrawn before admission stops, bounded
  work drains before the process deadline, and unfinished durable work remains
  recoverable. Research does not select that exact order or its budgets for the
  job capability. Specification must make the externally observable drain and
  recovery result falsifiable; Technical Design must choose a mechanism that
  cannot be held open forever by a non-cooperative handler.

### Messaging and outbox

- [`docs/durable-messaging.md`](../../../docs/durable-messaging.md) owns direct
  NATS publication/consumption, stream and consumer identity, ACK/redelivery,
  broker retry/DLQ, ordering, and cross-service contracts.
- [`docs/postgres-transactional-outbox.md`](../../../docs/postgres-transactional-outbox.md)
  owns same-transaction event append and a relay to an external broker. Its
  immutable event, publication uncertainty, retention, and reconciliation state
  is not a job execution state machine.
- `JOBS=postgres` must be independent of `MESSAGING`, `OUTBOX`, and `INBOX`.
  Reusing their lifecycle patterns does not merge their semantics or schemas.

### Migrations and profiles

- Canonical schema authority is transactional SQL in six-digit Goose files.
  Application startup does not run migrations; the existing migrator, history
  gates, image rehearsal, and shared SQLC fan-in own delivery.
- River v0.43.0 ships seven ordered migration versions. River documents that
  its complete history cannot run inside one transaction, recommends explicit
  target versions with Goose, and can export versioned SQL. The repository
  rejects Go migrations and `NO TRANSACTION` files. A later design must prove a
  reproducible pinned SQL/Goose integration that preserves both River's order
  and the repository's canonical source/history policy; calling River's
  migrator at application startup is not viable here.
- [`scripts/init-module.sh`](../../../scripts/init-module.sh#L429) currently has
  no `JOBS` selector. A later pack would need an independent default-off choice,
  `DATABASE=postgres` dependency, permanent `template.lock` record, profile
  stripping, shared SQLC/migration fan-in, jobs-only and mixed-profile proof,
  conditional binary/image surface, and idempotent initialization. Research
  creates none of those artifacts.

### Telemetry ownership

The repository already supplies logger/resource/exporter installation,
diagnostics, tracing, and metric helpers. A job adapter needs its own bounded
vocabulary rather than outbox or NATS labels:

- counters for accepted, rejected, and uniqueness-skipped enqueue;
- ready/scheduled/retry depth and oldest age;
- attempt start, duration, queue delay, success, retry, terminal failure,
  cancellation, rescue, and manual recovery;
- claim/worker capacity, PostgreSQL pool/WAL/vacuum signals, and drain outcome;
- job, attempt, producer, schedule occurrence, effect, request, and trace
  identities in logs/traces, never unbounded IDs or tenant values as metric
  labels; and
- cached DB observations for asynchronous gauges; metric callbacks must not do
  database I/O.

River's optional `otelriver` plugin can provide insert/work spans and execution
metrics, but it is another dependency and does not replace cluster-wide depth,
oldest-age, reconciliation, database, readiness, or drain evidence.

## Required behavior findings

### Durable acceptance and identity

The repository's durable-jobs discipline is the governing model:

`identity -> durable acceptance -> claim/lease -> execute -> durable effect -> complete -> retry/recover -> prove`

Keep these identities distinct even when River supplies only some of them:

| Identity | Required meaning |
| --- | --- |
| Logical job ID | Stable across every attempt and operator retry/redrive. |
| Producer-deduplication key | One logical acceptance with explicit scope and retention; also resolves ambiguous enqueue/commit readback. |
| Attempt ID/generation | One execution claim; stale attempts cannot finalize a newer attempt. |
| Schedule occurrence ID | One intended civil/instant occurrence, independent of actual start time. |
| Business-effect key | One durable effect across overlapping attempts and ambiguous completion. |

Atomic enqueue prevents the “business row committed but job was not accepted”
gap. It does not provide exactly-once execution. A worker can commit an external
effect and crash before completion is recorded; retry/rescue may overlap the
original attempt. The effect therefore needs an idempotency key, conditional
write/ledger, or reconciliation. Uniqueness is producer suppression, not an
effect guarantee.

### Scheduled and periodic work

- River's `ScheduledAt` persists a one-off future time in the job row. The
  scheduler normally exposes it within roughly one poll interval and is not a
  seconds-precision timer.
- River OSS periodic/cron registrations are held in leader memory and reset
  across restart/election. Miss-free persisted periodic scheduling is a Pro
  feature. This is not sufficient for a business requirement that every
  occurrence be durably represented.
- Every civil schedule needs timezone, DST gap/fold, overlap, misfire, catch-up,
  late-delivery, jitter, start/end, and cancellation semantics plus a stable
  occurrence identity. No library default can infer those values.
- Prefer platform cron for a simple bounded idempotent trigger without
  same-transaction acceptance. Prefer a durable scheduler/workflow engine when
  occurrence history and catch-up are intrinsic. Do not grow a private cron
  subsystem merely to make OSS River periodic registration look durable.

### Uniqueness and idempotency

River can make an insert unique by kind, encoded args, queue, period, and states,
and returns whether an insert was skipped. Callers must inspect and emit that
result: a 2026 Basedash incident dropped 92% of a cohort because a singleton
policy rejected enqueue without the result being observed.

Uniqueness scope and retention must match the producer business identity. It
cannot replace handler reentrancy, external idempotency, an effect ledger, or
reconciliation. Queue/name/JSON changes can also change a derived uniqueness
key and accidentally admit a second logical job.

### Retries, backoff, and poison work

River defaults to 25 attempts with `attempt^4` delay and jitter, leaving the last
retry about three weeks after the first. This is an upstream default, not a
template policy. Each job kind needs:

- retryable, permanent, poison, and operator-actionable failure classes;
- per-attempt deadline/cost, maximum attempts, maximum elapsed age, capped
  backoff, jitter, and downstream retry hints;
- a recovery-wave admission rule so an outage does not become a retry storm;
- terminal retention and a documented audit/redrive authority; and
- the same business-effect key on manual retry.

OSS River retains exhausted jobs as `discarded` and allows terminal
cancellation/retry operations, but first-class DLQ is Pro. If audited quarantine
and batch redrive are mandatory, the chosen edition or a different owner must
provide them; ad hoc SQL is not a production control plane.

### Cancellation and graceful shutdown

Queued jobs can be atomically cancelled. A running cancellation is cooperative:
River notifies the client and cancels the handler context, but Go cannot stop
the goroutine, success may win the race, and a handler that returns `nil` after
cancellation is recorded as successful. Handlers must thread context through
I/O, return an error on interrupted work, and protect partial effects
independently.

Long, non-interruptible work that cannot fit the deployment termination budget
is not a normal job. Chunk/checkpoint it under an explicit replay-safe contract
or move it to a workflow engine.

### Priority, queue isolation, and concurrency

- River priorities are strict; sustained high-priority traffic can starve lower
  priorities. Business latency classes and starvation tolerance are required
  before setting them.
- Separate queues give independent process-local worker pools and are useful for
  interactive versus batch capacity. All queues still share the same River
  table, PostgreSQL pool/WAL/vacuum/disk/failover envelope.
- Per-queue concurrency is OSS. Global or tenant-keyed concurrency is Pro or a
  separate application/platform contract. A queue name is not tenant fairness.
- Concurrency must derive from downstream and database capacity; replicas
  multiply both workers and pool connections. Backpressure should reduce claims
  rather than create retry amplification.

### Payload, schema, and deployment compatibility

River persists the job kind, queue string, and JSON args. Renaming a JSON field
can silently decode an old payload into a zero value. Renaming a kind or queue
can strand work during a rolling deploy. Compatibility therefore spans the
maximum of retry, schedule, pause, redrive, and rollback windows:

- keep N and N-1 workers able to decode and run all live kinds;
- version kinds/envelopes for incompatible changes or retain old fields during
  an expand/contract window;
- add new queue/kind consumers before producers switch, drain old rows, then
  remove old registrations in a later deploy;
- prove database schema remains compatible with both worker versions; and
- update River's interrelated modules together, as its own upgrade guide
  requires.

## Candidate space

Links inside this table are the claim-level, release-pinned source locators for
material exclusions. A statement that no first-class API was found is scoped to
the pinned exported source/package surface; it is not a claim about every fork
or future release.

| Candidate | Current evidence | Fit for this repository | Disposition / reopen condition |
| --- | --- | --- | --- |
| No module: synchronous work or reconciliation | No new dependency, schema, binary, or operations surface | Best when the caller needs the result, work fits the request, or source-of-truth scanning reconstructs it | Prefer by default. Reopen only for a concrete durable acceptance/retry/scheduling need. |
| Existing background supervisor | Already installed and production-owned | Good for process-lifetime refreshers/loops; explicitly not durable jobs | Reject as queue. It has no durable acceptance, schedule, retry, claim, or recovery. |
| River OSS v0.43.0 | Released 2026-08-05, MPL-2.0, active source; native `pgx.Tx`, typed args/workers, broad lifecycle | Strongest direct fit; matches repo Go 1.26.5 and exact `pgx/v5` v5.10.0 | Carry as leading candidate. Reopen on license/pre-v1 rejection, migration incompatibility, unmeasured DB impact, or required Pro-only semantics. |
| Gue v6.0.0 | MIT, current source activity; delays, priority, queues, retry/discard, worker pools. [v6 source change](https://github.com/vgarvardt/gue/blob/c09342b3979315ae75137721658eb489f21d716c/CHANGELOG.md#L3-L10), [license](https://github.com/vgarvardt/gue/blob/c09342b3979315ae75137721658eb489f21d716c/LICENSE) | v6 removed adapters; enqueue takes [`*sql.Tx`](https://github.com/vgarvardt/gue/blob/c09342b3979315ae75137721658eb489f21d716c/client.go#L94-L104), payloads are [bytes](https://github.com/vgarvardt/gue/blob/c09342b3979315ae75137721658eb489f21d716c/job.go#L28-L57), and the [handler runs before the claim transaction is released](https://github.com/vgarvardt/gue/blob/c09342b3979315ae75137721658eb489f21d716c/worker.go#L179-L269). No first-class uniqueness, cron, client-cancel, or retained-DLQ API was found in the pinned exported surface. | Reject current fit. Reopen if the repository moves to `database/sql` and required behavior shrinks. |
| Neoq v0.72.0 | MIT, released 2026-08-10; PostgreSQL backend, delayed/cron, retries, uniqueness, dead jobs. [tagged contract](https://github.com/acaloiaro/neoq/tree/v0.72.0) | Public API exposes only [`Enqueue(ctx, *jobs.Job)`](https://github.com/acaloiaro/neoq/blob/40c5345af864034616c687cf14f03ea775cebf24/neoq.go#L65-L87); PostgreSQL enqueue [begins/commits its own transaction](https://github.com/acaloiaro/neoq/blob/40c5345af864034616c687cf14f03ea775cebf24/backends/postgres/postgres_backend.go#L448-L509), args are [`map[string]any`](https://github.com/acaloiaro/neoq/blob/40c5345af864034616c687cf14f03ea775cebf24/jobs/jobs.go#L35-L48), initialization [auto-runs migrations](https://github.com/acaloiaro/neoq/blob/40c5345af864034616c687cf14f03ea775cebf24/backends/postgres/postgres_backend.go#L390-L445), durability [defaults `synchronous_commit` off](https://github.com/acaloiaro/neoq/blob/40c5345af864034616c687cf14f03ea775cebf24/backends/postgres/postgres_backend.go#L175-L192), cron is [process-local](https://github.com/acaloiaro/neoq/blob/40c5345af864034616c687cf14f03ea775cebf24/backends/postgres/postgres_backend.go#L533-L585), and the observed shutdown path [cancels/closes/stops cron without a visible handler join](https://github.com/acaloiaro/neoq/blob/40c5345af864034616c687cf14f03ea775cebf24/backends/postgres/postgres_backend.go#L603-L621). | Reject current contract. Reopen if caller-owned pgx enqueue, repository-owned migrations, durable defaults, and bounded handler join appear upstream. |
| goqite v0.4.0 | MIT, active, small SQL queue with caller [`*sql.Tx`, delay, priority, and byte payloads](https://github.com/maragudk/goqite/blob/471f9d49ce356737fc756a287007d7b4c54c61e1/goqite.go#L88-L143) | Handler and transactional creation are [`[]byte`/`*sql.Tx`](https://github.com/maragudk/goqite/blob/471f9d49ce356737fc756a287007d7b4c54c61e1/jobs/runner.go#L201-L226). No first-class typed registry, cron, uniqueness, client cancellation, or retained-DLQ API was found; the PostgreSQL [receive query has no row lock/`SKIP LOCKED`](https://github.com/maragudk/goqite/blob/471f9d49ce356737fc756a287007d7b4c54c61e1/goqite.go#L189-L211), so its concurrent-claim guarantee is unproven here. | Reject current fit; watch only if the capability becomes intentionally much smaller and concurrency is proven. |
| dataddo/pgq, txix-open/bgjob | Apache-2.0/MIT lower-level PostgreSQL queues. pgq's publisher [owns a `database/sql` transaction](https://github.com/dataddo/pgq/blob/ef03827ef6679fb1545e3b93bb758cd9276964d7/publisher.go#L23-L114) and carries [`json.RawMessage`](https://github.com/dataddo/pgq/blob/ef03827ef6679fb1545e3b93bb758cd9276964d7/message.go#L64-L88); bgjob enqueue requires a [`sql.Result`-returning executor](https://github.com/txix-open/bgjob/blob/1b4172b44de7de76886f8be83a2d2f4441894e2c/enqueue.go#L11-L25) | bgjob is byte-oriented and its PostgreSQL store [holds the row-lock transaction through handler completion](https://github.com/txix-open/bgjob/blob/1b4172b44de7de76886f8be83a2d2f4441894e2c/pg_store.go#L31-L90). Neither exposes a comparable native caller-owned pgx plus typed-worker surface. | Reject current fit. Reopen on transaction-authority change and materially smaller behavior. |
| Goncordia v0.14.0 | MIT, 2026 project with [generic pgx transactional enqueue](https://github.com/kirimatt/goncordia/blob/a1237e9f5f503f90a91754e85776458d5d884e53/client.go#L50-L80), priorities, uniqueness, and cron | The pinned claim path changes rows to [`running`](https://github.com/kirimatt/goncordia/blob/a1237e9f5f503f90a91754e85776458d5d884e53/driver/pgxv5/executor.go#L251-L282), [discards finalization errors](https://github.com/kirimatt/goncordia/blob/a1237e9f5f503f90a91754e85776458d5d884e53/worker.go#L262-L289), and keeps cron cursor/enqueue errors [in process memory](https://github.com/kirimatt/goncordia/blob/a1237e9f5f503f90a91754e85776458d5d884e53/cron.go#L40-L96). No lease/heartbeat/rescue path was located in v0.14.0; this is an unresolved guarantee, not proof that future versions cannot recover. | Reject until upstream crash-recovery and finalization guarantees are proven. |
| simple-durable-jobs v4.10.0 | MIT, broad 2026 feature set | Typed transactional enqueue requires [`*gorm.DB`](https://github.com/jdziat/simple-durable-jobs/blob/d55fc0fdea0f72c397c3867daa9006251d17f1f6/pkg/typed/typed.go#L120-L139), and direct dependencies include [ConnectRPC, Prometheus, OTel, GORM, and three database drivers](https://github.com/jdziat/simple-durable-jobs/blob/d55fc0fdea0f72c397c3867daa9006251d17f1f6/go.mod#L7-L28). | Reject dependency/boundary cost. Reopen only if GORM and the larger workflow surface become accepted owners. |
| Custom PostgreSQL queue | Full local control; existing outbox demonstrates some lease/query patterns | Would reimplement identity, claim, retry, schedule, poison, cancellation, compatibility, controls, and observability | Reject while River fits. Reopen only for a much smaller accepted contract or measured River failure that custom code can materially avoid. |
| Existing NATS JetStream + outbox | Already installed optional profile; mature cross-service ACK/replay/DLQ path | Correct for distributed integration; same-DB atomicity needs the outbox and adds a relay hop | Conditional substitute only when work is broker-distributed or database isolation/throughput dominates. Do not wrap it as local jobs. |
| Redis/asynq v0.26.0 | MIT Redis-backed framework with retries, crash recovery, weighted/strict queues, and archived exhausted tasks. [Pinned contract](https://github.com/hibiken/asynq/blob/d704b68a426d1d3a6c707f9661d29296e1350775/README.md#L3-L40) | Client/enqueue is [Redis-owned with no caller database transaction](https://github.com/hibiken/asynq/blob/d704b68a426d1d3a6c707f9661d29296e1350775/client.go#L20-L48); an outbox/reconciliation remains necessary. Task headers exist, but the repository would still own its OTel middleware and conventions. | Reject baseline. Reopen when Redis is already authoritative and transactionality is not required. |
| Managed SQS/Cloud Tasks | Managed isolation, retry, DLQ/rate controls through external [`SendMessage`](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_SendMessage.html) or [`tasks.create`](https://cloud.google.com/tasks/docs/reference/rest/v2/projects.locations.queues.tasks/create) APIs | No shared PostgreSQL transaction surface exists; atomic domain-write plus enqueue therefore requires an outbox or reconciliation. Provider limits, cost, identity, egress, and data custody become owners. | Reopen when managed operations and database isolation are accepted outcomes. |
| PostgreSQL primitives PGMQ/PgQue | [PGMQ](https://pgmq.github.io/pgmq/) is SQL objects with no background worker; [PgQue](https://pgque.dev/docs/concepts/) is an event/message queue optimized around sustained load | They provide SQL message/event primitives but no Go typed handler registry or repository-owned production worker lifecycle. | Not a better job framework. Reopen for a platform-standard primitive or measured high-throughput streaming need. |
| Temporal/Hatchet workflow platforms | Durable histories, waits, signals, checkpoints, and orchestration/versioning. [Temporal determinism](https://docs.temporal.io/workflow-definition#deterministic-constraints), [Hatchet architecture](https://hatchet.run/) | A service/SDK/operator boundary for workflow semantics, not a smaller implementation of one retryable internal handler. | Reject for ordinary jobs. Reopen only under workflow conditions below. |
| DBOS Go | Can couple an application transaction and workflow enqueue with a pgx-backed datasource. [Transaction contract](https://docs.dbos.dev/golang/tutorials/transaction-tutorial), [integration model](https://docs.dbos.dev/golang/integrating-dbos) | This does not establish compatibility with the repository's existing caller-owned `pgx.Tx`; it also introduces durable workflow/history/system-schema ownership beyond an ordinary job. | Carry as a workflow-boundary candidate, not a direct River-equivalent fit. Reopen if durable workflow semantics are required or its exact transaction seam later proves compatible and smaller. |

### River dependency, maintenance, license, and upgrade facts

- v0.43.0 was released from signed commit `60435dc2` on 2026-08-05; the
  repository was active after the tag. It remains a v0 API, so an adopter must
  accept pre-v1 compatibility risk.
- The license is MPL-2.0. This Research does not give legal advice or import
  another organization's approval. Each adopter's legal policy owns acceptance
  and notice/source obligations.
- The release's top module requires `riverdriver`, `riverpgxv5`, `rivershared`,
  `rivertype`, `robfig/cron`, `tidwall` JSON helpers, `pgerrcode`, `pgx`,
  `puddle`, and `x/sync`. The current repository already has the exact `pgx`,
  `puddle`, and `x/sync` versions; it would add the River modules and remaining
  production graph. v0.43.0 specifically fixed accidental linking of several
  test-only packages into production binaries.
- River instructs consumers to upgrade its interrelated modules together to
  avoid incompatible resolution. A pinned upgrade must prove source migrations,
  N/N-1 worker/schema behavior, dependency diff, binary/image size, build time,
  vulnerability/license scan, and rollback—not merely `go test` one package.
- No representative binary size, cold-start, memory, CPU, connection, WAL, or
  vacuum cost was measured in Research. “Moderate” or “fast” would be an
  unsupported claim.

## Production counter-evidence

- **Basedash, 2026-06-16:** PostgreSQL-backed pg-boss still fit their system,
  but a singleton policy silently rejected 92% of a cohort, one shared queue let
  batch work starve interactive requests, a rolling queue rename orphaned jobs,
  and a 200-job cron burst exhausted a five-connection pool. Implications:
  observe enqueue results, centralize routing, overlap old/new queue consumers,
  split latency classes, bulk insert fan-out, and size against the burstiest
  second—not an average. Limit: pg-boss/Node, not River.
- **GitLab Artifact Registry ADR, current 2026:** selected a hybrid—River for
  consistency-critical transactional jobs and asynq for high-throughput
  eventual work. It records roughly 2.4 dead tuples per River job in its test,
  limits River to lower-volume critical work, and relies on a dedicated service
  database plus autovacuum/retention tuning. Limit: planned/current ADR and
  GitLab-specific infrastructure/legal approval, not proof for this template.
- **RudderStack, 2026-05-26:** six years of a bespoke high-scale PostgreSQL
  queue required compaction, vacuum/index tuning, COPY batching, cache layers,
  retry-storm controls, and continuous workload-specific refinement. Limit:
  event-pipeline architecture, not River/shared OLTP.
- **DBOS, 2026-06-02:** reported 30k workflow starts/s only after changing flow
  control and selective indexes to reduce serialization, CPU, and autovacuum
  pressure. Limit: vendor self-report and specialized durable execution.
- **Hypothesis talk/article, 2026-01:** a PostgreSQL job table closed the
  Postgres/RabbitMQ dual-write outage, but reliability came from reconciliation,
  stable tagging/expiry/priority, bounded batches, and alerts for both producer
  and consumer low-watermarks. Limit: custom queue and a narrow indexing use
  case.
- **Loop, 2026-03-19:** retained Kafka/SQS/pollers for their proper boundaries
  and adopted Temporal when multi-step state, retries, and progress across
  restarts became the problem. At scale it required its own queues, versioning,
  migrations, failure/redrive UI, and substantial operator tooling. Limit:
  mature workflow platform, evidence for the boundary and cost—not a default.

These reports contradict both extremes. PostgreSQL queues are neither
automatically unsafe nor operationally free. The decision is driven by
transactional value and a measured shared-database envelope.

## Downstream input closure and decision ownership

The generic template cannot infer adopter-specific business or deployment
values. Their absence does not block a vendor-neutral Specification that defines
the required policy slots, validation, and safe absence behavior; it does block
approval of a concrete job kind or production adoption at the checkpoint named
below. Specification must not manufacture these values or inherit them from a
library default.

| Required input | Named owner and authoritative source | Required shape | Availability now | Earliest required checkpoint | Technical decision derived later |
| --- | --- | --- | --- | --- | --- |
| Durable acceptance visible to the caller | Capability sponsor's intended outcome plus [`postgres.InTx`](../../../internal/infra/postgres/transaction.go#L13) and `ErrCommitUnknown` | Whether failure rejects the business mutation; stable acceptance identity/readback; whether status/cancel is exposed outside feature code | Partly closed: same-transaction acceptance and ambiguous commit are authoritative; external status exposure has no use case | Generic Specification must preserve the closed transaction semantics; any external API exposure is required before that feature's Specification approval | API/use-case acceptance and reconciliation behavior |
| Duplicate/late effect tolerance and the identity of one effect | Adopter domain/product owner; accepted business invariant or external idempotency contract | Stable effect key, duplicate window, late-result rule, retention/reconciliation authority | Unavailable because no concrete job/effect exists | Before the first job kind's Specification is approved | Effect ledger, conditional write, idempotency, or reconciliation mechanism |
| Completion/backlog-age SLO and outage recovery objective | Adopter product/SRE owner; service SLO and recovery policy | Target percentile/maximum age, recovery deadline, allowed degradation | Unavailable because no adopter workload or SLO exists | Before production topology/capacity acceptance; not needed to specify the optional pack's policy surface | Queue split, concurrency, backpressure, alerts, and topology |
| Cost and retry limits for each effect/dependency | Job-kind domain owner and downstream service owner; accepted dependency/error contract | Per-attempt deadline/cost, maximum attempts/elapsed age, retryable/permanent classes, retry hints | Unavailable without a job kind | Before the first job kind is registered for production | Attempt budgets, backoff, jitter, and permanent classification |
| Schedule occurrence semantics | Schedule/business owner; accepted calendar/occurrence policy | Timezone, DST, overlap, skip/duplicate/late tolerance, misfire/catch-up, start/end | Unavailable; no periodic use case exists | Before periodic scheduling is admitted for a concrete use case | Platform cron, durable periodic feature, or workflow selection |
| Maximum useful duration and durable progress requirement | Job-kind domain owner; accepted process invariant and deployment termination envelope | Maximum duration, cancellation result, checkpoint/resume need, partial-effect rule | Unavailable without a job kind; current process budgets are evidence, not the value | Before classifying the first long-running candidate as a job | Timeout, chunk/checkpoint contract, drain budget, or workflow engine |
| Interactive/batch and tenant fairness | Adopter product/SRE/tenancy owner; latency and isolation policy | Latency classes, starvation tolerance, tenant/global concurrency, noisy-neighbor budget | Unavailable; template has no workload or tenancy policy | Before enabling multiple priorities/queues or tenant-sensitive work in production | Queue/priority layout, concurrency, and isolation mechanism |
| Payload sensitivity, retention/compliance, and operator roles | Adopter security/data owner; data-classification, retention, authorization, and audit policies | Allowed fields/secrets, encryption/redaction, retention/deletion, inspect/cancel/retry/redrive roles and audit | Unavailable because no payload or adopter policy exists | Before the first job kind persists arguments or operator access is enabled | Stored envelope, redaction, retention, authorization, audit, and control plane |
| Deployment cadence and rollback window | Adopter delivery owner; release/rollback policy and maximum live-job age | Minimum N/N-1 coexistence and deletion checkpoint across retry/schedule/pause/redrive windows | Repository requires compatibility proof; adopter window unavailable | Before removing any old args/kind/queue/schema compatibility and before production rollout | Compatibility window and expand/contract sequence |
| Target workload and PostgreSQL topology/headroom | Adopter SRE/data owner; representative workload, topology, pool and database capacity evidence | Arrival/burst/running distributions, duration, payload size, replicas, DB budget, failover and recovery envelope | Unavailable; no adopter inventory or workload exists | Before claiming shared-OLTP production readiness | Pool/worker limits, indexes/vacuum/retention, separate database, or non-Postgres owner |
| Dependency license and pre-v1 tolerance | Template/adopter legal and dependency-policy owner; MPL-2.0 text, security policy, and compatibility policy | Approval or rejection of River edition/version plus notice/source and upgrade obligations | MPL-2.0 and v0 status are known; organizational approval is unavailable | Before Technical Design selects River for implementation | Candidate/edition selection or rejection |

Technology choices—package boundaries, defaults that do not invent those
values, library/edition, schema, retry algorithm, metric names, process
composition, and proof level—remain agent-owned in later phases.

## Proof obligations for later phases

No item below was executed; each is a required evidence class if the capability
is selected.

### Contract and transaction

- Real PostgreSQL and at least two worker processes.
- Business mutation plus enqueue commit/rollback atomically.
- Lost commit/enqueue response, same producer key, durable readback, and no new
  logical effect.
- Crash before effect, effect-before-completion, lost completion, worker death,
  rescue, overlapping attempts, and one durable effect.
- Uniqueness-skip result observed and correctly surfaced.

### Lifecycle and recovery

- Startup rejects missing/incompatible schema or handler registry.
- Worker loop/panic/DB loss withdraws worker readiness without making API
  liveness or an already durable backlog dishonest.
- SIGTERM: readiness false first, no new claim, bounded soft drain, context
  cancellation, process hard deadline, safe cleanup, restart rescue, and an
  uncooperative handler diagnostic.
- Pause, cancel, terminal failure, retained poison evidence, audited retry/redrive,
  and stale operator action.

### Retry, scheduling, fairness, and compatibility

- Dependency outage and recovery wave stay within attempt, elapsed, connection,
  and downstream budgets.
- Continuous high-priority/batch/tenant load proves accepted latency and no
  forbidden starvation.
- One-off schedule delay; periodic overlap, downtime, catch-up, clock skew, DST
  fold/gap, duplicate occurrence, and late delivery for every selected schedule
  owner.
- N/N-1 args, kinds, queues, River modules, database schema, rolling deploy,
  rollback, paused/retrying jobs, and criterion for deleting old compatibility.

### Database, dependency, delivery, and profiles

- Representative OLTP plus queue load measures connections, CPU, WAL, dead
  tuples, indexes, autovacuum lag, disk, customer latency, oldest-job age,
  recovery drain time, and failover behavior.
- Exact candidate dependency graph, license/vulnerability scan, compile/build
  time, binary/image size, idle/loaded memory, and startup/shutdown cost.
- Pinned River migration export/integration, append-only history, up/down/up
  rehearsal, image migrator, mixed-version schema compatibility, and recovery.
- `JOBS=none`, jobs-only, jobs+outbox, jobs+inbox, jobs+messaging, and all-retained
  template initialization; invalid combinations are non-mutating and generated
  checkouts build/test/migrate with no removed dependency leaks.

### Observability and operator evidence

- Attempt versus logical-job correlation, enqueue accept/skip/reject, depth and
  oldest age, runtime/delay, retries/exhaustion, cancellation/rescue, drain,
  reconciliation drift, DB saturation, and bounded labels.
- Producer and consumer low-watermark alerts, backlog-age/SLO alerts, terminal
  discovery, recovery-wave visibility, and runbooks for authorized controls.
- Metric collection causes no dependency I/O or unbounded cardinality.

## When another owner is preferable

### Prefer no job module when

- no concrete accepted use case requires work to outlive a request/process;
- the caller needs the result synchronously and work fits the request budget;
- a deterministic source-of-truth reconciliation scan is simpler than one job
  per change;
- simple platform cron can run one bounded idempotent repair/maintenance action;
- the organization will not own another worker, schema, migration, capacity,
  alerts, and recovery surface; or
- shared PostgreSQL impact cannot be justified and no alternative topology is
  part of the intended outcome.

### Prefer integration messaging when

- another service owns the handler/effect;
- independent consumers, fan-out, replay, retained event history, heterogeneous
  languages, or broker-level routing/backpressure are required; or
- database isolation/high throughput is more important than direct same-DB
  enqueue. Use the outbox for atomic publication; do not let consumers poll
  private job tables.

### Prefer platform scheduling when

- the requirement is only “invoke this bounded idempotent action on a schedule”;
- same-transaction occurrence creation is not required; and
- platform overlap/misfire/catch-up semantics can express the business policy.

### Prefer a workflow engine when

- durable intermediate state or several dependent steps is intrinsic;
- the process waits for signals, humans, timers, or another service;
- compensation/saga behavior, child processes, durable checkpoints, or queryable
  execution history is required;
- work spans many deployments and must be pinned/versioned/replayed safely; or
- a single job row would start accumulating a private workflow state machine.

Temporal is not exactly-once either: an Activity can commit an effect and crash
before completion is recorded. It changes the orchestration and recovery model;
it does not remove the effect-idempotency obligation.

## Leading-fit falsifiers and refresh triggers

Reopen the River hypothesis when any of these becomes true:

- authoritative business state and enqueue cannot share one PostgreSQL
  transaction;
- mandatory first-class DLQ, durable periodic scheduling, global/tenant
  concurrency, workflow/resumability, or checkpointing is outside the accepted
  River edition;
- MPL-2.0, v0 compatibility, split-module upgrades, or exported migration
  integration is rejected;
- job duration/cancellation cannot fit the rollout termination budget;
- required fairness cannot be expressed without starvation or unbounded custom
  policy;
- queue churn/failure must be isolated from OLTP and measured headroom is absent;
- cross-service consumers, fan-out, replay, or heterogeneous contracts appear;
- production-like evidence shows connection, WAL, vacuum, disk, latency, or
  recovery-age budgets fail; or
- another maintained candidate proves native caller `pgx.Tx`, typed workers,
  crash recovery, bounded join, versioned migrations, compatible license, and a
  smaller dependency/operations surface.

Refresh dynamic evidence when River or its selected edition/version changes,
PostgreSQL major/topology or PgBouncer/failover enters scope, the first actual
job/workload/SLO is known, template profile or migration authority changes, a
cross-service consumer or long-lived process appears, or production reports
starvation, retry storms, bloat, vacuum lag, queue-age breach, or OLTP headroom
erosion.

## Evidence sources

### Repository authority

- [Durable background jobs discipline](../../../docs/universal-disciplines/durable-background-jobs/SKILL.md)
- [Database-backed queue mechanics](../../../docs/universal-disciplines/durable-background-jobs/references/database-backed.md)
- [Operational branches](../../../docs/universal-disciplines/durable-background-jobs/references/operations.md)
- [Durable engine mechanics](../../../docs/universal-disciplines/durable-background-jobs/references/durable-engine.md)
- [PostgreSQL transaction owner](../../../internal/infra/postgres/transaction.go)
- [Background supervisor](../../../internal/background/background.go)
- [NATS worker lifecycle](../../../cmd/worker/internal/bootstrap/lifecycle.go)
- [Migration policy](../../../docs/build-test-and-development-commands.md#migrations-and-containers)
- [Template initialization](../../../scripts/init-module.sh)

### Primary upstream contracts and source

- [River v0.43.0 release](https://github.com/riverqueue/river/releases/tag/v0.43.0)
- [River v0.43.0 client source](https://github.com/riverqueue/river/blob/60435dc2c58e3d3dfbde6a00f08bc98a3f13c37e/client.go)
- [River pgx v5 driver source](https://github.com/riverqueue/river/blob/60435dc2c58e3d3dfbde6a00f08bc98a3f13c37e/riverdriver/riverpgxv5/river_pgx_v5_driver.go)
- [River module graph](https://github.com/riverqueue/river/blob/60435dc2c58e3d3dfbde6a00f08bc98a3f13c37e/go.mod)
- [River MPL-2.0 license](https://github.com/riverqueue/river/blob/v0.43.0/LICENSE)
- [Transactional enqueue](https://riverqueue.com/docs/transactional-enqueueing)
- [Insert-only clients](https://riverqueue.com/docs/insert-only-clients)
- [Retries](https://riverqueue.com/docs/job-retries)
- [Unique jobs](https://riverqueue.com/docs/unique-jobs)
- [Scheduled jobs](https://riverqueue.com/docs/scheduled-jobs)
- [Periodic jobs](https://riverqueue.com/docs/periodic-jobs)
- [Cancellation](https://riverqueue.com/docs/cancelling-jobs)
- [Graceful shutdown](https://riverqueue.com/docs/graceful-shutdown)
- [Multiple queues](https://riverqueue.com/docs/multiple-queues)
- [Migrations](https://riverqueue.com/docs/migrations)
- [Changing job args](https://riverqueue.com/docs/changing-job-args)
- [Updating River](https://riverqueue.com/docs/updating-river)
- [OpenTelemetry](https://riverqueue.com/docs/open-telemetry)
- [PostgreSQL `SKIP LOCKED`](https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE)
- [Kubernetes CronJob creation semantics](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#job-creation)
- [Temporal workflow replay/determinism](https://docs.temporal.io/workflow-definition#deterministic-constraints)
- [Temporal Go workflow versioning](https://docs.temporal.io/develop/go/workflows/versioning)

### Alternative and counter-evidence locators

Release-pinned claim-level source locators for material candidate exclusions are
embedded directly in the candidate table above. These links provide discovery
and operational counter-evidence context.

- [Gue](https://github.com/vgarvardt/gue)
- [Neoq v0.72.0](https://github.com/acaloiaro/neoq/releases/tag/v0.72.0)
- [goqite](https://pkg.go.dev/maragu.dev/goqite/jobs)
- [dataddo/pgq](https://github.com/dataddo/pgq)
- [Goncordia v0.14.0](https://github.com/kirimatt/goncordia/releases/tag/v0.14.0)
- [simple-durable-jobs v4.10.0](https://github.com/jdziat/simple-durable-jobs/releases/tag/v4.10.0)
- [Asynq v0.26.0](https://github.com/hibiken/asynq/releases/tag/v0.26.0)
- [PGMQ 1.11.1 contract](https://pgmq.github.io/pgmq/)
- [PgQue concepts](https://pgque.dev/docs/concepts/)
- [DBOS Go transaction contract](https://docs.dbos.dev/golang/tutorials/transaction-tutorial)
- [NATS JetStream consumers](https://docs.nats.io/nats-concepts/jetstream/consumers)
- [SQS visibility queue contract](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
- [Basedash production lessons, 2026-06-16](https://www.basedash.com/blog/what-we-learned-running-background-jobs-on-postgres)
- [GitLab Artifact Registry background-jobs ADR, accessed 2026-08-11](https://handbook.gitlab.com/handbook/engineering/architecture/design-documents/artifact_registry/decisions/006_technology_stack/#1-background-job-processing-hybrid-river--asynq)
- [RudderStack PostgreSQL queue lessons, 2026-05-26](https://www.rudderstack.com/blog/scaling-postgres-queue/)
- [DBOS PostgreSQL queue scaling, 2026-06-02](https://www.dbos.dev/blog/making-postgres-queues-scale)
- [Hypothesis transactional job queue talk/article, 2026-01](https://www.seanh.cc/2026/01/29/transactional-job-queues/)
- [Loop Temporal production evolution, 2026-03-19](https://www.loop.com/engineering-blog/temporal-at-loop---evolution-and-scaling)

## Research closure

All named repository surfaces and evidence lenses have a disposition. Search
covered current Go/PostgreSQL libraries, database primitives, installed broker
reuse, Redis/managed queues, custom code, platform scheduling, and workflow
engines. Within the decisive caller-owned `pgx.Tx` + typed handlers + broad job
lifecycle boundary, another materially distinct mature candidate is unlikely
without a new upstream release. Adopter-specific behavior, workload, legal,
capacity, and production-proof inputs are closed to named owners and
checkpoints; migration integration and candidate selection belong to later
design. None is missing candidate research or permission for Specification to
invent business meaning.

Research therefore stops here. Specification is the next macro phase; no
Specification content is written in this note.

## Standalone prompt for Specification

```text
Create the Specification for an optional durable background-job capability in go-service-template-rest, using specs/durable-background-jobs/research/synthesis.md as the Research authority and the current structured spec-first workflow.

Define the falsifiable behavior of a template-init-selected pack, provisionally JOBS=postgres, for service-internal jobs whose durable acceptance normally commits in the caller-owned PostgreSQL transaction. Keep feature code limited to typed job arguments and business handlers; keep job storage/claiming, retry timing, process lifecycle, readiness, drain, telemetry, operator controls, migrations, and profile mechanics outside feature behavior.

Define the generic capability contract for the caller-visible acceptance boundary; logical job, producer-deduplication, attempt, schedule-occurrence, and business-effect identities; duplicate-effect/idempotency policy slots; retry/backoff/poison/cancellation semantics; one-off versus periodic schedule behavior; priority/fairness/queue-isolation policy slots; maximum duration and shutdown outcome; payload/kind/queue compatibility; retention/privacy/operator authorization inputs; readiness and degraded behavior; telemetry; and the conditions that require no module, integration messaging/platform cron, or a workflow engine instead. Use the Research input-closure table: preserve closed repository and sponsor semantics, keep adopter-owned values explicitly unresolved until their named checkpoint, and do not invent business meaning or inherit library defaults. If a value is required to make the generic pack itself falsifiable and is unavailable from its named owner, stop Specification as blocked rather than guessing.

Treat River OSS v0.43.0 only as the leading Research fit, not as an already selected architecture. Do not inherit its defaults or Pro-only features as requirements. Preserve the repository's caller-owned pgx.Tx and ErrCommitUnknown semantics, separate API and worker readiness, canonical Goose migration authority, independent JOBS/OUTBOX/INBOX/MESSAGING profiles, and claim-scoped proof boundary. Record every business/deployment-owned input, bounded assumption, acceptance scenario, out-of-scope condition, and reopen trigger.

Stop at the Specification macro-phase boundary. Do not write Technical Design, Test Design, tasks, migrations, or code. Finish with the standalone prompt for Technical Design.
```
