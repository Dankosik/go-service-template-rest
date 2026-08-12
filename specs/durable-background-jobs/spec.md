# Durable service-internal work is accepted atomically and completed recoverably

status: ready
Problem: The template has no supported boundary for service-internal work that
must survive a request, process restart, or deployment and normally must be
accepted in the same PostgreSQL transaction as the business mutation that
created it. An adopter would otherwise have to invent job identity,
transactional acceptance, duplicate-effect safety, retry and cancellation
policy, worker lifecycle, compatibility, telemetry, and operator recovery in
feature code, or misuse the messaging pack for private work.

## Scope and non-goals

In scope is an optional, template-init-selected capability, provisionally
`JOBS=postgres`, for one service to durably accept and execute its own bounded
jobs. A selected pack provides an in-process producer boundary and an
independently operable worker boundary. Feature code supplies only typed job
arguments and a business handler; job storage and claiming, attempt state,
retry timing, process lifecycle, readiness, drain, telemetry, operator
controls, migrations, and profile mechanics remain generic infrastructure or
composition concerns.

The normal acceptance path stages the job in the caller-owned PostgreSQL
transaction that stages the causing business mutation. The same service and
PostgreSQL authority own the job; no other service reads private job storage.
Execution is at least once in effect. The pack does not promise exactly-once
execution, synchronous completion, start-time precision, FIFO order, or
automatic progress while no admitted worker is available.

Out of scope, with the condition that reopens each boundary:

- **A selected implementation or edition.** River OSS v0.43.0 is the leading
  Research fit, not an accepted dependency or architecture. Technical Design
  must keep, revise, or reject it after proving the transaction, migration,
  lifecycle, compatibility, dependency, and license gates. This Specification
  does not inherit River defaults or require River Pro behavior.
- **Public or cross-service job APIs.** The pack exposes no external submit,
  status, list, cancel, retry, or redrive contract. A concrete caller or
  operator API requires its own Specification, identity, authorization,
  disclosure, pagination, concurrency, and error semantics before exposure.
- **Business policy values.** A generic pack defines required policy slots and
  safe absence behavior. It does not choose an effect's duplicate tolerance,
  retry budget, calendar meaning, latency SLO, tenant fairness, data retention,
  operator roles, deployment window, or database capacity for an adopter.
- **Technical representation and placement.** Package names, interfaces,
  schemas, SQL, configuration keys, process/binary composition, library
  adapters, retry algorithms, metric names, and generated-source changes belong
  to Technical Design or later phases.
- **Migrations, Test Design, planning, rollout, and implementation.** No
  migration bytes, commands, task ledger, tests, or code are selected here.
- **A generic periodic scheduler or workflow layer.** The baseline pack admits
  durable one-off scheduling only. Periodic occurrence materialization is
  admitted only under B7; durable waits, multi-step state, compensation, and
  checkpointed orchestration remain workflow-engine concerns.
- **Business-effect storage.** The pack tracks job execution, not the domain's
  effect ledger or reconciliation truth. A feature or downstream system owns
  that truth under B5.

## Behavior and contract delta

### Terms and authoritative identities

The identities below are distinct even if a later implementation can derive
one from another. Stored job state is authoritative for acceptance, attempt,
schedule, and terminal job facts. The feature's transactional state, effect
ledger, downstream idempotency contract, or named reconciliation source is
authoritative for the business effect; a job's `succeeded` state alone cannot
prove an external effect happened once.

| Identity | Required meaning and lifetime | Conflict rule |
| --- | --- | --- |
| Logical job ID | One accepted unit of work, stable through every claim, retry, rescue, cancellation request, and authorized manual retry/redrive. | A retry or redrive that creates a new logical job for the same accepted work violates the contract. |
| Producer-deduplication key | One producer-owned acceptance identity within an explicit scope and recognition period. It is reused after an ambiguous enqueue or commit outcome and resolves to the original logical job. | Reuse with different immutable intent is an integrity conflict, never a duplicate success. |
| Attempt ID or generation | One claim of a logical job. Each new claim gets a distinct identity; completion from a stale attempt cannot finalize or overwrite a newer attempt. | A stale finalization is rejected without changing current job state. |
| Schedule-occurrence ID | One intended occurrence, independent of when execution starts. A one-off job has one occurrence; every admitted periodic occurrence has its own stable identity. | Re-materializing the same occurrence resolves to the same logical job or a duplicate acceptance, not another effect. |
| Business-effect key | One domain effect across overlapping attempts, ambiguous completion, and manual recovery. Its scope and lifetime come from the job-kind owner. | Reuse for different business intent is a domain integrity conflict; the job pack cannot reinterpret it. |

No raw payload, tenant value, or one of these unbounded identities is a metric
label. The identities may appear in access-controlled logs or traces when the
adopter's data policy permits it.

### B1 — The pack is optional, independent, and removable

Actor: a developer initializing the template.

Rule: `JOBS` is an independent, default-off selector. `JOBS=postgres` requires
the PostgreSQL database profile but does not require or imply `OUTBOX`, `INBOX`,
or `MESSAGING`. Selecting or omitting any of those sibling profiles does not
merge their schemas, delivery guarantees, identities, retries, readiness, or
operator controls with jobs.

Outcomes:

| Selection | Observable outcome |
| --- | --- |
| Jobs not selected | No jobs runtime, worker surface, schema, dependency, configuration, documentation, or test residue remains. Existing service behavior is unchanged. |
| `JOBS=postgres` with PostgreSQL | The producer and independently operable worker capability are retained, subject to the admission rules below. |
| `JOBS=postgres` without PostgreSQL or with an unknown value | Initialization is rejected before mutating the destination. |
| Jobs combined with any retained OUTBOX/INBOX/MESSAGING profile | Each pack keeps its independent behavior and can be built and operated without source edits. |

Falsifier: initialize the omitted, jobs-only, each relevant pair, and
all-retained profiles twice. An invalid selection changes no destination; a
valid selection has no removed-profile references or cross-profile behavioral
dependency.

### B2 — Durable acceptance is part of the caller-owned transaction

Actor: a use case that has decided a job is required for its accepted business
mutation.

Preconditions: before opening the transaction, the caller has stable logical
job, producer-deduplication, and business-effect identities and immutable typed
arguments. The concrete job kind has passed B4 admission.

Rule: the producer stages the job through the existing caller-owned `pgx.Tx`.
It neither begins, commits, rolls back, nor retries that transaction. A staging
failure is returned to the caller and the required business mutation must not
commit. A rollback or definite commit failure accepts neither mutation nor job.
Only a known successful transaction commit makes a newly staged job durably
accepted.

The staging result distinguishes:

| Result | Meaning and allowed caller action |
| --- | --- |
| Staged new | This transaction contains a new logical job; final acceptance still follows the caller's commit outcome. |
| Existing identical acceptance | The producer key already identifies the same immutable intent and logical job; expose that identity and do not create another job. The caller must not treat this as permission to commit a new business mutation unless its own stable operation receipt proves that mutation is the same already-applied operation. |
| Producer-key conflict | The key identifies different immutable intent; reject the transaction without changing the existing job. |
| Rejected | Arguments, kind, policy, schedule, or storage admission failed; return the cause and do not treat the mutation as accepted. |

`postgres.ErrCommitUnknown` preserves an ambiguous joint outcome. The caller
must not rerun the mutation or enqueue with fresh identities. It reads the
current PostgreSQL writer by the stable producer key and resolves:

| Authoritative read | Outcome | Allowed next action |
| --- | --- | --- |
| Matching acceptance exists | Accepted | Return or reconcile using the original logical job; do not enqueue again. |
| No acceptance exists on a successful current-writer read | Not accepted | Retry the business operation only under its own stable operation identity, reusing every job identity. |
| The key exists with different immutable intent | Integrity conflict | Fail without mutation or a second job. |
| Read is unavailable, stale, or inconclusive | Still unknown | Preserve the identities and retry readback; do not retry the mutation. |

A cache, replica, timeout, or failed read cannot prove absence. The pack does
not add an external status API; this is an in-process acceptance/readback
contract.

Caller cancellation or deadline before a known commit stops the acceptance
attempt subject to the same definite-versus-unknown commit rule. After durable
acceptance, the originating request context no longer owns the job lifetime;
request cancellation cannot cancel the accepted job. Only the durable
cancellation transition in B9 or the worker's bounded attempt/shutdown context
can interrupt execution.

Falsifier: with a real PostgreSQL transaction, commit and roll back a business
row plus enqueue; lose the commit response after both possible database
outcomes; prove the table above and that no outcome creates a second logical
job or silently commits a required mutation without its job.

### B3 — Producer deduplication suppresses acceptance, not effects

Actor: a producer repeating an acceptance request.

Rule: every admitted same-transaction producer supplies a deduplication key,
scope, immutable-intent comparison rule, and recognition lifetime. No library
uniqueness default or serialization of typed arguments defines business
identity. A matching key returns the original logical job and an explicit
duplicate result that telemetry records; callers may not ignore the result.

The recognition lifetime covers every interval in which the producer may
retry or reconcile the same operation. No automatic expiry is enabled for a
job kind until its owner has supplied a safe lifetime and deletion rule. A
queue, kind, or payload rename cannot change an already accepted producer
identity.

Producer deduplication does not make execution or the business effect once-only.
It cannot replace B5.

Falsifier: submit identical and conflicting intent under one key, including
after an unknown commit outcome and across a compatible deployment. Exactly
one logical job exists; the identical result is observable, and conflicting
intent is rejected.

### B4 — A concrete job kind is fail-closed until its policy is complete

Actor: the worker process admitting a typed handler, and the producer trying to
accept that kind.

Rule: a kind is admitted only when its external owner has supplied all
behavior-changing values required for that kind:

- stable kind and argument compatibility rules;
- producer-key scope, immutable intent, and recognition lifetime;
- business-effect key plus the B5 duplicate/late-effect policy;
- retryable, permanent, poison, cancellation, and operator-actionable failure
  classifications;
- per-attempt maximum duration/cost, maximum attempts, maximum elapsed age,
  backoff cap and jitter rule, downstream retry-hint precedence, and
  recovery-wave admission rule;
- manual-recovery eligibility by terminal class, required remediation or
  duplicate-risk evidence, and whether a recovery cycle resets attempt and
  elapsed-age budgets;
- one-off schedule policy when scheduling is used;
- maximum useful duration, partial-effect behavior, and fit within the
  deployment termination envelope;
- payload classification, redaction, retention/deletion policy, and required
  operator roles; and
- explicit work class. Additional priority, queue, or tenant-isolation policy
  is required only if that feature is enabled.

Missing or invalid policy prevents producer acceptance for that kind and keeps
a worker configured to serve it not ready. The pack does not substitute a
library default. Before any kind exists, the selected generic pack may build
and migrate but has no production-ready handler surface to claim.

Falsifier: omit each required policy slot in turn. Producer admission is
rejected, the worker does not claim that kind, and diagnostics name the missing
slot without exposing sensitive values.

### B5 — Attempts are at least once; each kind owns duplicate-effect safety

Actor: a typed business handler executing a logical job.

Rule: the handler may run again or overlap after worker death, lease rescue,
lost completion, timeout, cancellation race, retry, or authorized redrive. Each
attempt receives the same logical job, schedule-occurrence, and business-effect
identities and a new attempt identity. A stale attempt cannot finalize a newer
attempt.

Before the kind is approved, its domain owner selects and proves one explicit
effect policy:

- a same-database conditional write or effect ledger keyed by the business
  effect identity;
- a downstream idempotency contract that durably recognizes that identity for
  the required duplicate and late-result window;
- behavior whose accepted invariant makes repetition and overlap harmless; or
- a named authoritative reconciliation path that detects and repairs the
  possible duplicate or missing effect within its accepted objective.

The policy states duplicate tolerance, late-result precedence, partial-effect
behavior, retention, and authoritative readback. An external effect committed
before job completion may already exist even when the attempt later fails.
Absent affirmative effect evidence, the pack reports the outcome as retryable
or operator-actionable according to the kind policy; it never promotes job
state to proof of exactly-once effect.

Falsifier: crash before the effect, after the effect but before completion, and
while two attempts overlap. The selected effect authority satisfies its stated
duplicate and late-result invariant, while attempt fencing prevents stale job
finalization.

### B6 — Retry, poison, recovery, and manual redrive preserve truth

Actor: the worker after a handler or infrastructure outcome.

Rules:

- A retryable outcome creates another eligible attempt only within the kind's
  maximum-attempt and maximum-elapsed-age bounds. Timing follows the kind's
  explicit backoff, cap, jitter, retry-hint, and recovery-wave rules.
- A permanent outcome performs no automatic retry. A deterministic poison
  payload or compatibility failure never invokes business behavior. Exhausted,
  permanent, poison, and outcome-unknown facts remain distinguishable.
- A timeout or lost worker can be rescued as a new attempt; the earlier attempt
  is stale and cannot finalize it. Rescue does not erase possible-effect
  ambiguity.
- A terminal record and the evidence needed for its approved audit window are
  retained until the adopter's explicit retention/deletion policy permits
  removal. No library cleanup default or ad hoc SQL silently discards it.
- An authorized retry/redrive creates a new recovery generation for the same
  logical job, schedule occurrence, producer identity, and business-effect key.
  It never creates fresh business intent. Succeeded and non-terminal jobs are
  never eligible. A cancelled, exhausted, permanent, poison, or outcome-unknown
  job is eligible only when that kind's admitted manual-recovery policy names
  the source class and its required remediation or duplicate-risk evidence is
  present. The same policy fixes whether automatic attempt and elapsed-age
  budgets restart for that recovery generation. Without the policy or evidence,
  redrive fails with a state/policy conflict and changes nothing. A valid action
  is idempotent under its audit identity and visible in the audit trail.
- A recovery wave obeys explicit database and downstream admission bounds; it
  may delay retries rather than turn an outage into unbounded claims or retry
  amplification.

First-class DLQ, batch-redrive UI, or global concurrency is not implied. If an
adopter requires those as mandatory controls and the selected OSS mechanism
cannot provide them without a new private subsystem, Technical Design must
select another edition or owner.

Falsifier: exercise each classification, exhaust both attempt and elapsed-age
bounds, recover a dead claim, repeat a manual action with the same audit
identity, and restart the worker. State, identities, and retention follow the
rules above without an inherited default changing the outcome.

### B7 — One-off scheduling is durable; periodic behavior needs a separate admission

Actor: a producer accepting work for a future time.

One-off rule: the accepted job stores an explicit not-before instant and one
schedule-occurrence identity in the same durable acceptance. It is not eligible
before that instant; it may run later because of polling, outage, backpressure,
or capacity. No start-time accuracy or maximum lateness is promised without an
owner-supplied SLO. Cancellation before claim prevents the first attempt.

Periodic rule: the baseline pack has no process-memory cron registration and
does not claim miss-free periodic behavior. A concrete periodic use case is
admitted only after its schedule owner fixes timezone, DST gap and fold,
overlap, misfire, catch-up, late-delivery, jitter, start/end, cancellation, and
occurrence-retention semantics. Every intended occurrence is then a separate
durable acceptance with a stable occurrence identity. Technical Design must
choose the smaller correct owner:

- platform scheduling when invoking one bounded idempotent action is enough and
  its overlap/misfire semantics meet the accepted policy;
- a durable scheduling capability when every occurrence and catch-up decision
  must survive scheduler failure; or
- a workflow engine when timers are part of long-lived process state.

Falsifier: a one-off job is never claimed early and survives worker downtime.
With no complete periodic policy, registration is rejected and restart cannot
silently skip or duplicate an alleged durable occurrence.

### B8 — Priority and isolation are explicit policy, not an accidental queue name

Actor: an adopter assigning work classes and worker capacity.

Rule: the baseline has one explicit neutral work class and promises no relative
priority, FIFO ordering, tenant fairness, or latency target. Multiple priorities
or queues are disabled until the product/SRE/tenancy owner supplies latency
classes, starvation tolerance, noisy-neighbor limits, global or tenant bounds,
and the proof workload. A queue name is routing and compatibility state, not a
fairness guarantee.

If multiple classes are admitted, each has an explicit capacity and starvation
policy. Replicas multiply worker concurrency and database connections; all
PostgreSQL-backed classes still share the accepted pool, WAL, vacuum, disk, and
failover envelope unless Technical Design selects proven isolation.

Falsifier: absent a multi-class policy, configuration cannot enable a second
priority/queue or tenant-sensitive routing. With one admitted, sustained higher
class and tenant-skewed load meets the owner-supplied starvation and isolation
bounds rather than merely demonstrating that jobs eventually ran in a sample.

### B9 — Cancellation and shutdown never claim an effect was undone

Actor: a caller or authorized operator cancelling work, and the worker process
receiving shutdown.

Cancellation outcomes:

| State when cancellation wins | Required outcome |
| --- | --- |
| Accepted but not claimed | No handler starts; the job becomes terminally cancelled. |
| Running and handler cooperates before any effect | The handler context is cancelled; no automatic retry occurs; the job becomes cancelled. |
| Running and the effect is affirmatively completed | Success wins; cancellation is recorded as not applied to the completed outcome. |
| Running with partial or ambiguous effect | The job becomes operator-actionable outcome-unknown unless the kind's accepted B5 policy proves a safe terminal or retry transition. It is never labelled effect-free cancellation by guess. |
| Already terminal, already cancel-requested under a different action identity, or stale expected generation | Return a state conflict naming the current state; change nothing. A not-found target returns not found. |

Handlers must propagate their attempt context to blocking I/O and return an
interrupted result; Go code cannot forcibly stop a non-cooperative handler.
The originating API/request context is never reused as that attempt context
after the job has been durably accepted.

On process shutdown, worker readiness is withdrawn, new claims stop, and live
handlers receive a bounded soft-drain opportunity. At the configured boundary,
their contexts are cancelled; the process then reaches its hard termination
deadline and cleans up. Unfinished work remains durable and recoverable by a
later worker. An uncooperative handler cannot keep the process alive forever
and produces a diagnostic drain outcome.

Every admitted kind's maximum attempt duration and cancellation/partial-effect
policy must fit the deployment termination envelope. Work that cannot fit must
be split into replay-safe bounded chunks under an explicit progress contract or
routed to a workflow engine.

Falsifier: cancel before and during work, race cancellation with success, then
send shutdown with cooperative and uncooperative handlers. No new claim begins
after drain, the process respects the owner-supplied hard bound, and every
unfinished or ambiguous effect remains recoverable or operator-actionable.

### B10 — Compatibility covers every live accepted job

Actor: producers and workers during rolling deploy, rollback, pause, retry,
schedule delay, or redrive.

Rule: kind, typed argument envelope, queue/work class, job schema, and selected
engine modules remain mutually compatible for the maximum live-job and rollback
window. N and N-1 processes can decode and safely handle every live kind during
that window. An incompatible argument change uses an explicit version or an
expand/contract interval; missing JSON fields may not silently become meaningful
zero values. New consumers are admitted before producers switch kind or queue,
and old registrations are removed only after authoritative state proves no live
job requires them and the delivery owner reaches the deletion checkpoint.

A worker registry that is incomplete or incompatible at startup is not ready
and makes no claims. If an individual accepted record is nevertheless unknown
or undecodable, no business handler runs; the record becomes retained,
observable compatibility poison for authorized recovery rather than looping or
being discarded.

Falsifier: roll N and N-1 producers/workers forward and back with ready,
scheduled, retrying, paused, and terminal-redriveable jobs. Every live job is
handled by a compatible worker or retained visibly; no rename strands work and
no decoded zero value changes business meaning.

### B11 — API and worker readiness are separate truths

Actor: the platform probing the API process and the independently operable job
worker.

Rules:

- API liveness remains process-only. When the jobs producer is selected, API
  readiness includes its usable PostgreSQL and compatible producer-schema path,
  but never worker liveness, queue depth, downstream handler health, or worker
  backlog.
- Worker liveness remains process-only. Worker readiness means this process has
  compatible schema and registered policies/handlers, a usable PostgreSQL path,
  a live claim loop, and is not draining. It does not mean the queue is empty,
  every handler dependency is healthy, or backlog age is within SLO.
- If workers are unavailable while PostgreSQL acceptance remains available,
  callers can still durably accept work and observe the acceptance outcome.
  Backlog depth and oldest age expose the degradation. An adopter's supplied
  backlog/recovery policy decides when the service, alerting, or rollout must
  degrade further; absent that policy, no shared-OLTP production-readiness or
  SLO claim is permitted.
- PostgreSQL pool saturation is capacity evidence, not a reason to create a
  readiness restart loop. Job claim capacity and backlog signals remain
  distinct from the repository's dependency reachability probe.

Falsifier: stop every worker while leaving PostgreSQL available, then restore
them; API readiness never follows worker state, accepted work remains durable,
worker readiness follows its own admission/claim/drain state, and recovery is
visible without a false queue-empty readiness rule.

### B12 — Telemetry and operator controls expose bounded, authorized facts

The generic capability emits enough evidence to distinguish:

- enqueue accepted, rejected, and producer-duplicate outcomes;
- ready, scheduled, retry, running, and terminal depth plus oldest age;
- attempt start, queue delay, duration, success, retry, exhaustion, permanent
  failure, poison, cancellation, rescue, outcome unknown, and manual recovery;
- worker and claim capacity, PostgreSQL pool/WAL/vacuum pressure, and shutdown
  drain outcome; and
- logical job, attempt, producer, schedule occurrence, business effect,
  request, and trace correlation in access-controlled logs/traces where policy
  permits.

Metric labels use a closed vocabulary such as outcome, registered kind, and
bounded work class. They never contain raw arguments, arbitrary error text,
tenant values, or identity values. Asynchronous gauges read cached observations;
metric collection performs no database I/O. Alert thresholds and backlog SLOs
remain adopter-owned inputs.

When an operator control is admitted, its semantic result is fixed before its
transport is chosen:

| Control | Observable result |
| --- | --- |
| Inspect | Return the current stored job/attempt state and permitted audit facts with payload and identities redacted by policy; change nothing. |
| Pause claims for a scope | After the pause transition commits, start no new claim in that scope. Durable acceptance remains available unless a separately accepted backpressure policy says otherwise; in-flight attempts follow their existing policy. |
| Resume claims for a scope | Make eligible retained work claimable again without changing its identities, attempt history, or schedule meaning. |
| Cancel | From accepted, scheduled, or retry-ready state with the expected generation, commit terminal cancellation. From running state with the expected attempt, commit `cancel requested` and signal that attempt. Apply B9 to the eventual outcome. Every other existing state is a state conflict with no mutation; not found is distinct. Never report that an already or ambiguously committed effect was undone. |
| Retry/redrive | Apply B6 only from a terminal class admitted by the kind's manual-recovery policy and only with its required evidence. Succeeded and non-terminal states are always conflicts. The valid transition preserves logical, producer, occurrence, and business-effect identities and opens a new recovery/attempt generation. |
| Delete or terminally dispose | Reject until retention, legal-hold, deduplication, effect-evidence, and audit policy all permit the exact transition. |

No operator mutation is enabled until the security/data owner supplies allowed
inspect, pause/resume claiming, cancel, retry/redrive, and terminal-disposition
roles and scopes. Every enabled mutation requires authenticated actor identity,
authorization for the action and target scope, a stable audit/action identity,
an expected current state or generation, and an audit reason. Repetition of the
same action identity returns its first result; reuse for different action or
target is a conflict. A stale or unauthorized action changes nothing. Payload
inspection is separately authorized and redacted. A credential, secret, or
other sensitive field is rejected unless the data owner has explicitly
admitted that field and fixed its minimization, encryption, redaction,
retention, and access rules.

No automatic terminal deletion runs until retention, legal hold, audit, and
producer-deduplication lifetimes are jointly closed. Manual SQL is recovery
evidence at most, not the production control plane.

Falsifier: collect metrics while PostgreSQL is unavailable and exercise
authorized, repeated, stale, conflicting, and unauthorized actions. Collection
does no dependency I/O, labels stay bounded, and only one authorized current
transition commits with its audit evidence.

### B13 — Repository migration and runtime authorities remain unchanged

Actor: an operator migrating or starting an initialized service.

Rule: six-digit transactional Goose SQL under the canonical migration authority
is the only source of job schema truth. Application and worker startup never run
an upstream or Go migrator. Schema history remains append-only and participates
in the existing migrator, history, image rehearsal, rollback/recovery, and
shared SQL-generation gates.

The selected jobs worker owns its own startup, readiness, claim admission,
drain, cleanup, and telemetry flush. Durable execution is not hidden inside the
API process's background supervisor. Technical Design may choose a new or
existing worker executable only if the result remains independently deployable,
scalable, probeable, and drainable from the API.

Falsifier: application binaries start against missing or incompatible schema
without mutating it and fail the relevant admission; the canonical migrator is
the only path that changes schema; stopping or scaling the worker does not stop
or scale the API process.

## Capability owner boundary

| Condition | Required owner | Observable disposition |
| --- | --- | --- |
| No concrete work must outlive the request/process, the caller needs the result, a source-of-truth scan cheaply reconstructs it, or the organization will not own the extra worker/schema/operations surface | No job module; synchronous work or reconciliation | Leave `JOBS` off. Do not add durable machinery for hypothetical use. |
| Another service owns the handler/effect; fan-out, heterogeneous consumers, broker replay/history, or integration routing/backpressure is required | Existing integration messaging, with outbox/inbox when their transactional guarantees are needed | Do not expose or poll private job tables across services. |
| Only a bounded idempotent action must be invoked periodically and same-transaction occurrence acceptance is unnecessary | Platform scheduler | Use the platform's explicit overlap, misfire, and catch-up contract; do not add a private cron engine. |
| Durable waits, signals/humans, compensation, dependent steps, checkpoints, child processes, queryable history, or execution across many deployments is intrinsic | Workflow engine | Do not grow a logical job into a private workflow state machine. |
| Same-database atomic acceptance has insufficient measured OLTP headroom, or required isolation/throughput dominates it | Reopen System Design for an isolated database, messaging/managed queue, or another owner | Make no shared-OLTP production-readiness claim and do not hide the capacity failure with retries. |
| Required first-class DLQ, durable periodic history, global/tenant concurrency, or resumability is unavailable in the accepted edition | Select another edition/product or another capability owner | Do not emulate the missing platform as ad hoc feature behavior or operator SQL. |

## Decisions, constraints, and authorities

- **D1 — Generic Specification is ready without adopter values.** Research
  explicitly closes unavailable adopter values to policy slots, safe absence,
  and named checkpoints. Each missing value rejects or disables only the
  concrete behavior it governs; no `TBD` or upstream default can become policy.
- **D2 — Transaction ownership does not move.** The existing `pgx.Tx` caller
  owns begin, commit, rollback, and retry. `ErrCommitUnknown` remains ambiguous
  until current-writer readback. Reopen only if business state and job
  acceptance cannot share that authority.
- **D3 — At-least-once is the honest execution guarantee.** Attempt fencing and
  producer deduplication protect job state and acceptance; B5 protects the
  business effect. Reopen only if a concrete effect authority proves a stronger
  result without hiding ambiguity.
- **D4 — One neutral class and one-off scheduling are the minimal baseline.**
  Multiple classes and periodic semantics remain disabled until their named
  owners supply behavior and proof inputs. This avoids inheriting strict
  priority, cron, or queue defaults.
- **D5 — No public control plane is implied.** Operator behavior is specified
  as an authorization and audit contract, but its transport and exposure remain
  Design decisions and are disabled without adopter policy.
- **D6 — Goose and profile independence are canonical repository authority.**
  A selected library must adapt to them; it cannot run migrations at startup or
  merge JOBS with OUTBOX, INBOX, or MESSAGING.
- **D7 — River OSS v0.43.0 remains only a hypothesis.** Its MPL-2.0 and pre-v1
  status, module graph, OSS/Pro boundary, exported migrations, and runtime cost
  are Technical Design gates. A River default has no normative force here.

Research authority: [research/synthesis.md](research/synthesis.md), valid as of
2026-08-11. Repository authorities remain `postgres.InTx` and
`ErrCommitUnknown`, the architecture baseline, canonical Goose migration
policy, template-init profile ownership, and the existing API and worker
lifecycle boundaries named by that synthesis.

## Adopter and deployment input closure

Unavailable values do not block this vendor-neutral contract because each has
a named owner, checkpoint, and fail-closed absence rule. They do block the
concrete approval named below.

| Input and owner | Required before | Safe absence behavior and downstream reopen |
| --- | --- | --- |
| Duplicate/late-effect tolerance and effect identity — job-kind domain/product owner | First concrete job kind's Specification approval | Do not admit the kind. Reopen B5 when an accepted invariant or downstream idempotency/reconciliation authority exists. |
| Completion/backlog-age SLO and recovery objective — adopter product/SRE owner | Production topology/capacity acceptance | Publish no SLO or shared-OLTP readiness claim. Reopen readiness, queue split, capacity, backpressure, and alerts when representative values exist. |
| Per-effect/dependency cost and retry limits — job-kind and downstream owners | Production registration of the first kind | Do not admit the kind or inherit retry defaults. Reopen B4/B6 with classifications, deadlines, attempts/age, hints, and recovery bounds. |
| Schedule occurrence semantics — schedule/business owner | Admission of periodic scheduling | Periodic registration stays disabled. Reopen B7 and owner selection with timezone, DST, overlap, misfire/catch-up, lateness, jitter, and boundaries. |
| Maximum useful duration and durable progress requirement — job-kind domain and delivery owners | Classifying a long-running candidate as a job | Do not admit work that cannot fit the termination envelope; chunk under an accepted replay contract or route to a workflow engine. |
| Interactive/batch and tenant fairness — product/SRE/tenancy owner | Enabling multiple priorities/queues or tenant-sensitive work in production | Keep one neutral class and make no fairness claim. Reopen B8 and capacity/isolation design with accepted latency and starvation bounds. |
| Payload sensitivity, retention/compliance, and operator roles — security/data owner | First kind persists arguments or any operator access is enabled | Persist no concrete job args and enable no operator control. Reopen B4/B12 with classification, redaction, retention/deletion, roles, scopes, and audit policy. |
| Manual recovery by terminal class — job-kind domain owner plus security/SRE operator-policy owners | Enabling retry/redrive for a concrete kind | Permit no manual recovery. Reopen B4/B6/B12 with eligible terminal classes, remediation or duplicate-risk evidence, recovery-budget behavior, roles, scopes, and audit policy. |
| Deployment cadence and rollback window — delivery owner | Removing old args/kind/queue/schema compatibility or production rollout | Retain old compatibility and delete nothing. Reopen B10 compatibility and expand/contract design at the authoritative deletion checkpoint. |
| Target workload and PostgreSQL topology/headroom — adopter SRE/data owner | Shared-OLTP production-readiness claim | Make no capacity claim. Reopen topology and limits with arrival/burst/running distributions, duration/payload, replicas, DB budget, failover, and recovery evidence. |
| Dependency license and pre-v1 tolerance — template/adopter legal and dependency-policy owner | Technical Design selects River | Keep River unselected. Reopen the candidate decision on approval or rejection of the exact edition/version and obligations. |
| External submit/status/cancel visibility — concrete feature/API owner | That feature's Specification approval | Expose no external API. Reopen the caller contract only for a named use case. |

## Invariants and edge cases

- A required business mutation never knowingly commits without its staged job;
  an unknown commit outcome remains unknown until authoritative readback.
- A durable accepted job is never lost because a worker, API process, or
  deployment stops. Lack of worker progress is visible backlog, not rewritten
  acceptance.
- Success, retry, exhaustion, permanent failure, poison, cancellation, and
  outcome unknown remain distinguishable. No terminal label claims more about a
  business effect than its authority proves.
- Every automatic or manual attempt preserves logical, producer, occurrence,
  and business-effect identity while changing attempt identity.
- A duplicate enqueue result is never silently treated as new, and a
  uniqueness collision never suppresses different immutable intent.
- A cancelled or timed-out handler may still have produced an effect; the
  cancellation record cannot erase that fact.
- Old payloads, kinds, queues, and schema remain readable for every live-job and
  rollback window. Unknown data is retained and visible rather than decoded
  permissively, retried forever, or discarded.
- API and worker health remain separate; capacity, backlog, and dependency
  reachability remain separate signals.
- Profile composition reuses lifecycle patterns only. It does not inherit NATS
  ACK/DLQ semantics, outbox publication semantics, or inbox effect semantics.
- Unclassified payloads and unapproved credentials or secrets are rejected.
  Metrics contain no raw payload, arbitrary error, identity, or tenant value;
  collection performs no database I/O.
- Application startup never changes schema, and feature code never owns job
  polling, claims, retry timing, readiness, drain, telemetry, operator control,
  migration, or profile selection.

## Acceptance scenarios and proof expectations

These scenarios define observable acceptance. Test Design later selects the
smallest deterministic level and command for each; this Specification does not
claim that any has been executed.

| ID | Scenario and pass/fail boundary |
| --- | --- |
| AC-01 | Select no jobs pack. Pass: the initialized checkout contains no jobs residue and existing behavior is unchanged. Select an invalid database combination. Pass: initialization is non-mutating. |
| AC-02 | In a real PostgreSQL transaction, commit and roll back a business mutation plus enqueue. Pass: they become visible together or neither does; the pack never owns transaction completion. |
| AC-03 | Lose the commit response after commit and after rollback. Pass: the same producer identity resolves Accepted, Not accepted, Conflict, or Still unknown from current-writer readback without a new logical job or blind mutation retry. |
| AC-04 | Repeat identical and conflicting producer intent. Pass: identical intent returns the original logical job with an observed duplicate outcome; conflicting intent changes nothing and reports integrity conflict. |
| AC-05 | Crash before effect, after effect before completion, and during overlapping attempts with at least two workers. Pass: attempt fencing protects job state and the kind's B5 authority satisfies its duplicate/late-effect rule. |
| AC-06 | Drive retryable, permanent, poison, timeout, exhaustion, rescue, and recovery-wave outcomes. Pass: explicit per-kind bounds and classifications determine transitions; no library default or restart changes them. |
| AC-07 | Cancel queued and running jobs, race cancel with success, and interrupt partial/ambiguous work. Pass: the B9 table holds and no cancelled label falsely proves absence of effect. |
| AC-08 | Schedule a one-off job, stop all workers across its instant, then recover. Pass: it is never claimed early, remains durable, and later execution makes no unsupported precision claim. Periodic registration without complete calendar policy is rejected. |
| AC-09 | Attempt to enable a second priority/queue or tenant-sensitive routing without its policy, then exercise an admitted multi-class workload. Pass: absence fails closed; admitted behavior meets owner-supplied starvation/isolation bounds. |
| AC-10 | Roll N/N-1 producer and worker versions forward and back with live ready, scheduled, retrying, paused, and redriveable jobs. Pass: every job stays compatible or visibly retained; no rename or permissive decode changes its meaning. |
| AC-11 | Stop workers, break their database path, drain them, and restore them while the API producer path remains healthy. Pass: API readiness is independent, worker readiness follows B11, accepted backlog survives, and oldest-age/capacity evidence exposes degradation. |
| AC-12 | Send SIGTERM to cooperative and non-cooperative handlers. Pass: readiness withdraws, claims stop, soft drain and cancellation occur inside the deployment envelope, the process reaches its hard bound, and unfinished work is recoverable. |
| AC-13 | Exercise enqueue, attempt, terminal, rescue, drain, and manual-control telemetry with high-cardinality identities and PostgreSQL failure. Pass: required distinctions are observable, metric vocabulary stays bounded, and collection performs no dependency I/O. |
| AC-14 | Repeat, conflict, race, stale, and deny operator actions across every non-terminal and terminal class. Pass: cancel returns the exact B12 result; redrive is impossible from succeeded/non-terminal or any class absent from the kind policy; one eligible current action commits, same-action repetition returns its first result, and conflicting or stale actions mutate nothing. |
| AC-15 | Rehearse canonical migration history and every independent JOBS/OUTBOX/INBOX/MESSAGING profile combination in generated checkouts. Pass: only Goose changes schema, application startup does not, combinations build and migrate without cross-profile residue, and repeated initialization has zero drift. |

Proof is claim-scoped:

- Transaction, locking/fencing, crash recovery, migration, and queue-state claims
  require a real PostgreSQL and at least two worker processes where overlap is
  material; a stub does not prove them.
- Process lifecycle claims require real signal/drain boundaries and a forced
  non-cooperative handler; API/worker independence requires separate process
  observations.
- Profile claims require initialized generated checkouts, including omitted and
  mixed profiles, not only the template repository.
- Compatibility claims cover the accepted N/N-1 and maximum live-job window;
  they do not become indefinite compatibility claims.
- Telemetry proof covers the named bounded vocabulary, identities, and no-I/O
  collection rule; it does not prove adopter alert thresholds or SLOs.
- Local/template evidence can prove reusable pack behavior only. Production
  readiness additionally requires the adopter-owned workload, topology,
  retention, security, SLO, recovery, and rollout inputs above on the actual
  deployment path. Unsupported production surfaces remain explicitly
  unverified rather than narrowing this contract or substituting a mock.

Success means every applicable acceptance scenario passes at its named boundary,
every concrete job kind has passed B4, and no completion statement exceeds the
environment and profile combinations actually exercised.

## Risks, assumptions, and reopen conditions

- **Assumption — same service, same PostgreSQL transaction.** Affected rules:
  B2-B6. Safe boundary: the service owns both the causing mutation and private
  job record in one caller-owned `pgx.Tx`. Invalidating evidence: enqueue cannot
  share that transaction, another service owns the effect, or job tables become
  an integration contract. Reopen owner: Specification/System Design. Route to
  outbox plus messaging, reconciliation, or another durable owner.
- **Assumption — authoritative producer readback is available on the current
  writer.** Affected rule: B2. Safe boundary: absence is decisive only on a
  successful current-writer read. Invalidating evidence: the adopter cannot
  provide that path. Reopen owner: Technical Design; it must preserve Still
  unknown rather than weaken finality.
- **Assumption — a bounded handler can cooperate with cancellation.** Affected
  rules: B4, B9. Safe boundary: each admitted kind fits the deployment envelope
  and propagates context through blocking I/O. Invalidating evidence: required
  work is non-interruptible, needs durable progress, or outlives deployments.
  Reopen owner: the job-kind Specification; chunk it or select a workflow engine.
- **Assumption — one neutral work class is sufficient for the empty template.**
  Affected rule: B8. Safe boundary: no concrete workload or tenancy policy
  exists. Invalidating evidence: a second latency class, starvation incident,
  tenant isolation requirement, or measured capacity conflict. Reopen owner:
  adopter product/SRE/tenancy policy, then Specification and System Design.
- **Assumption — one-off scheduling needs a not-before instant, not precise
  start.** Affected rule: B7. Invalidating evidence: a maximum lateness, every
  civil occurrence, catch-up, DST, or schedule-history requirement. Reopen
  owner: schedule/business owner, then the capability boundary.
- **Risk — shared PostgreSQL queue churn can harm OLTP.** No capacity or
  readiness claim is accepted without representative arrival, burst, duration,
  payload, replica, connection, WAL, vacuum, disk, failover, and recovery-age
  evidence. A failed budget reopens topology or capability selection; it does
  not authorize hidden throttling or weaker durability.
- **Risk — terminal retention is intentionally fail-closed when policy is
  absent.** This protects audit and dedup semantics but can grow storage.
  Measured growth plus an accepted data/legal policy reopens compaction; cleanup
  must retain the required identity/effect evidence or explicitly narrow the
  guarantee in Specification.
- **Risk — operator recovery can duplicate an ambiguous effect.** An authorized
  redrive keeps the business-effect key and records the accepted duplicate risk;
  it does not claim exactly-once. A need to eliminate that risk reopens the
  effect authority or workflow boundary.
- **Reopen the River leading hypothesis** on license/pre-v1 rejection,
  incompatible canonical migration integration, required OSS-missing behavior,
  unbounded shutdown, failed fairness/capacity proof, cross-service ownership,
  measured OLTP harm, or a maintained candidate with a smaller proven surface.
- **Refresh Research** when the selected engine/version/edition, PostgreSQL
  topology or major version, template profile/migration authority, actual
  workload/SLO, legal policy, or failure evidence changes materially.

No unavailable owner input currently blocks this vendor-neutral Specification.
The first concrete kind or production adoption becomes blocked at the exact
checkpoint in the input-closure table if its required value is still absent.

## Review result

Independent whole-artifact review initially returned `FAIL` because manual
cancellation and redrive did not close eligible source states, stale/repeated
results, or recovery-budget ownership. B4, B6, B9, B12, the input-closure table,
and AC-14 now fix those semantics and fail closed when the named policy is
absent. Focused fresh review of that repair returned `PASS`; no Specification-
owned decision remains open.

## Standalone prompt for Technical Design

```text
Continue with Technical Design only. Specification is ready with independent review PASS: specs/durable-background-jobs/spec.md defines the vendor-neutral JOBS=postgres behavior, and specs/durable-background-jobs/research/synthesis.md is its Research authority. Start from those files plus docs/repo-architecture.md and the current workflow's System / Integration Design and Go Code / Ownership Design owners.

Use River OSS v0.43.0 as the leading hypothesis because Research found native caller-owned pgx.Tx enqueue, typed arguments/handlers, and the broadest matching PostgreSQL lifecycle. Test it first against the canonical transactional Goose migration authority, ErrCommitUnknown readback, independently operable worker lifecycle/readiness/drain, explicit no-default policy admission, OSS/Pro boundaries, N/N-1 compatibility, independent JOBS/OUTBOX/INBOX/MESSAGING profiles, MPL-2.0/pre-v1 acceptance, and measured dependency/runtime cost. Keep, revise, or reject River from current evidence; do not treat it as selected.

Design the smallest coherent runtime, data, failure/recovery, operator, telemetry, migration/profile, compatibility, and Go ownership model that realizes every B1-B13 rule without moving job mechanics into feature code. Preserve every adopter-owned input at its named checkpoint and fail-closed absence rule. First, reconstruct the affected deployment graph and transaction/attempt/effect authority, then decide the engine and process composition. Stop or reopen Specification if any design choice would change caller-visible acceptance, identity, duplicate-effect, retry, scheduling, cancellation, readiness, compatibility, or operator meaning. Do not enter Test Design, Planning, migrations, or implementation in this session.
```
