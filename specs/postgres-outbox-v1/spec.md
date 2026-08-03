# Optional PostgreSQL transactional outbox

status: ready

This records the V1 build. Claim, publication, and state-observation
mechanics were later reworked for throughput; [PostgreSQL Transactional Outbox](../../docs/postgres-transactional-outbox.md)
owns the current contract.

## Scope and non-goals

### In scope

- Add one initialization choice, `OUTBOX=none|postgres`, defaulting to
  `none`. `OUTBOX=postgres` is valid only with `DATABASE=postgres`.
- Persist a feature-owned domain mutation and its outbound-event intent in one
  PostgreSQL transaction.
- Publish committed intent asynchronously through a separately deployable,
  broker-neutral, bounded relay.
- Provide explicit at-least-once failure, duplicate, retry, poison, ordering,
  redrive, retention, lifecycle, telemetry, rollout, and purity contracts.

### Non-goals

- Broker adapters, broker topology, destination-specific authentication, or a
  production fallback publisher.
- Inbox processing, consumer deduplication, consumer side effects, sagas, or
  domain-specific event definitions.
- Exactly-once publication or consumption, global ordering, event sourcing,
  CDC/Debezium infrastructure, schema auto-migration, or runtime migration.
- Unbounded buffering, indefinite retry, automatic poison discard, automatic
  operator redrive, or partitioning without measured need.

## Behavior and contract delta

### OUT-1 — Profile selection and physical ownership

Initialization accepts exactly `OUTBOX=none` or `OUTBOX=postgres` and validates
the complete profile combination before its first mutation.

- `OUTBOX=postgres` with `DATABASE=none` is rejected without changing a byte.
- `OUTBOX=none` physically removes every outbox-owned source, migration, query,
  generated file, package, relay binary, configuration key/default/example,
  documentation page, test, command, image/build/CI block, and dependency.
- `OUTBOX=postgres` retains those surfaces and removes their generator markers.
- The chosen value is recorded in `template.lock`; repeating the same
  initialization is byte-stable, and a later different choice is rejected
  without mutation.
- Removing the outbox migrations also removes an otherwise-empty
  repository `migrations/` directory so the canonical migration source check
  remains valid.

Runtime configuration also fails closed: an enabled outbox with disabled
PostgreSQL is rejected before dependency construction, transaction opening,
row claim, publication, or cleanup.

### OUT-2 — Event intent contract

One stored event occurrence has these logical fields:

| Field | Contract |
| --- | --- |
| `id` | Required globally unique opaque text, stable for the occurrence and every retry/redrive; at most 256 bytes with no control characters. |
| `type` | Required versioned event-type name, at most 256 bytes with no control characters. |
| `source` | Required stable producer identity, at most 256 bytes with no control characters. |
| `destination` | Required logical broker-neutral route, at most 256 bytes with no control characters. |
| `schema` | Required payload-schema/version identity, at most 256 bytes with no control characters. |
| `occurred_at` | Required finite, non-zero UTC time at which the feature says the occurrence happened. |
| `payload` | Required valid UTF-8 JSON value, stored and published as the exact caller-supplied bytes, no larger than 256 KiB. |
| `metadata` | Valid UTF-8 JSON object, exact caller-supplied bytes, `{}` by default, no larger than 32 KiB; application metadata only, never relay state. Payload/metadata retention and privacy remain feature responsibilities. |
| `ordering_key` | Optional opaque text, at most 256 bytes with no control characters. It is present only together with `ordering_sequence`. |
| `ordering_sequence` | Optional positive signed 64-bit sequence, unique and strictly increasing for one ordering key. |

The stable event `id` is also the publication/deduplication identity handed to
the broker adapter. A retry or redrive of the same occurrence never creates a
new ID or changes the event envelope. Payload and metadata use binary storage;
JSON parsing validates them but never normalizes, compacts, reorders, or
re-encodes them. Their byte limits apply to those exact stored bytes. The sum of
the UTF-8 byte lengths of every text field, payload, and metadata must not exceed
288 KiB; equality is accepted and one byte over is rejected. The relay may add
transport trace context outside the stored payload, but may not change event
meaning.

Validation happens before insert. PostgreSQL constraints independently enforce
the same UTF-8, JSON/object shape, per-field, pair, positivity, and total-byte
boundaries against a caller that bypasses Go validation. An invalid event
aborts the caller-owned transaction.

### OUT-3 — Atomic feature boundary

A successful feature operation that announces an event commits its domain
mutation and event row in the same PostgreSQL transaction. The feature or its
PostgreSQL adapter owns that transaction and passes the transaction-bound query
capability to the outbox append operation.

- Failure before the domain write leaves neither result.
- Failure after the domain write but before append, an append rejection, any
  later statement failure, rollback, or definite commit failure leaves neither
  result.
- A successful commit makes both durable and visible together.
- An ambiguous commit is reported as ambiguous; the request path does not
  generate and retry a new business occurrence automatically. A feature that
  permits client/command retries owns command idempotency independently.
- The request process does not synchronously publish, wait for a broker, or
  require broker configuration to commit the transaction.

### OUT-4 — Durable relay state and ownership

The authoritative state of an occurrence is its PostgreSQL outbox row:

```text
pending/retry_wait -> leased -> published
                         |          |
                         |          +-> retained -> cleaned
                         +-> retry_wait
                         +-> poison -> explicitly redriven -> pending
                         +-> lease expires -> recovery_due -> leased
```

- A claim is one short database transaction. It selects eligible work without
  waiting on another relay, increments the attempt, assigns a unique fencing
  token and finite server-clock lease, and commits before publication begins.
- Broker I/O never runs in the claim or feature transaction.
- Only the matching live fencing token may record retry, poison, or published
  progress. A stale owner that publishes after expiry cannot finalize another
  relay's claim.
- Crash after claim leaves recoverable durable work. Expiry makes it eligible
  again; process memory is never the recovery authority.
- Multiple relay replicas may claim concurrently. One occurrence has at most
  one current lease, but lease expiry and acknowledgement ambiguity mean more
  than one publication attempt may occur over time.

### OUT-5 — Publication acknowledgement and duplicates

The relay-owned `Publisher` contract is minimal: publish one immutable event and
return either nil or an error.

- Nil means the selected broker durably acknowledged the same event ID.
- Timeout, disconnect, cancellation, or any missing/ambiguous acknowledgement
  is an error even when the broker may have accepted the event.
- Noop, log-only, drop, in-memory, or fire-and-forget implementations do not
  satisfy `Publisher` and may not be composed into the production relay.
- A missing publisher prevents relay startup; there is no default or fallback.
- Every production adapter requires real-broker conformance proof that the same
  event ID is durably present before nil permits PostgreSQL finalization.

Publication is explicitly at-least-once. Broker success followed by crash or
loss before the published-state commit leaves the row retryable, and restart
may publish it again. A published-state commit whose result is ambiguous is
verified after reconnect or left retryable; it is never assumed successful and
never followed by deletion. Consumers must tolerate duplicate IDs; this pack
does not implement their deduplication.

### OUT-6 — Retry, poison, and redrive

Only the relay retries publication. The default policy is:

- one in-flight publication per relay process; horizontal replicas provide
  additional bounded concurrency;
- a 10-second publication attempt timeout;
- a 30-second lease, which must exceed the publication and progress-commit
  budgets with scheduling margin;
- at most 10 claims per delivery cycle;
- full-jitter exponential retry from a 1-second base to a 5-minute cap;
- caller cancellation stops new claims; graceful drain may finish the current
  attempt inside its remaining process budget; forced shutdown cancels it and
  leaves the lease to expire.

Validation/topology/authentication/authorization or adapter-declared permanent
failures become poison immediately. An adapter-proven non-acceptance retries
until the attempt limit; exhaustion becomes durable poison, which is not
delivery and is never deleted automatically. A failure the adapter cannot prove
was a refusal stays ambiguous and remains retryable past that limit, because
capping attempts on an unknown outcome can drop an event the broker never
received.

Redrive is an explicit operator-owned transition identified by a required
unique audit ID. It preserves the event ID and envelope, records redrive count,
audit ID and time, resets the delivery-cycle attempt budget, and returns one
poison row to pending. The pack supplies the durable operation but no public or
unauthenticated control endpoint; a derived service must place authorization
and approval around any operator surface.

### OUT-7 — Ordering

No global order is promised.

- Events without ordering fields may publish in any claim/retry order.
- An append with ordering fields atomically compares against a separate durable
  per-key high-water row in the same feature transaction. The first positive
  sequence establishes the high-water mark. Each later sequence must be greater
  than the retained value; gaps are allowed, while equal or lower late inserts
  are rejected even after published event rows are cleaned.
- The per-key high-water row is retained independently of event cleanup. V1
  performs no automatic high-water cleanup because proving that a key can never
  be reused is domain policy; a future bounded retirement operation requires an
  explicit feature-owned terminal-key contract.
- Among accepted events with an ordering key, only the earliest unpublished
  sequence for that key is eligible. Different keys may progress concurrently.
- A leased, retry-wait, recovery-due, or poison predecessor blocks later events
  for that key. Skipping poison would violate the promised per-key order.
- Database claim order does not prove broker or consumer order. An adapter and
  downstream broker must preserve the same key before an end-to-end per-key
  ordering claim is made.

### OUT-8 — Backlog, retention, and cleanup

Broker outage does not make the API dual-write or block on broker I/O. New
domain operations continue while PostgreSQL can commit both state and intent.
If the outbox insert cannot be durably stored, the domain mutation rolls back.

PostgreSQL is a finite outage buffer, not an unlimited availability promise.
Operators observe unpublished count, bytes, oldest time, retry/poison state and
drain rate and own service-specific admission/capacity policy. V1 adds no
automatic API rejection based on backlog because no accepted capacity threshold
exists.

Published rows are retained for seven days by default. Cleanup runs separately
from publication in transactions of at most 1,000 published rows using
published time. An incomplete batch returns to the one-minute cadence; a full
batch schedules another bounded batch no later than the poll interval. Cleanup
never deletes pending, leased,
retry-wait, recovery-due, or poison rows. Partitioning is absent; measured
relation/index size, vacuum lag, claim-plan degradation, or cleanup-budget
failure reopens data/performance design.

### OUT-9 — Relay startup, readiness, shutdown, and failure

The relay is a separate process with diagnostics and no API routes.

- Startup admits only valid config, expected schema, reachable PostgreSQL, a
  production publisher successfully constructed under its adapter contract, a
  running relay loop, and one successful fresh state observation. An adapter
  whose construction requires an initial broker connection fails startup when
  that connection cannot be established.
- Liveness reports only local process/diagnostics progress. Database, broker,
  backlog age, poison rows, and telemetry export do not trigger restart loops.
- After admission, transient broker rejection, timeout, disconnect, or ambiguous
  acknowledgement does not by itself change relay readiness: the ready relay is
  still correctly making durable retry/poison progress and the outage is exposed
  by operation/backlog signals. Fatal adapter/relay-loop failure, PostgreSQL or
  schema loss, a state observation older than two configured observation
  intervals, or drain makes readiness false. Backlog
  age or poison alone does not make a capable relay unready.
- Shutdown marks readiness false, stops new claims, gives the current attempt a
  20-second drain window, cancels it on expiry, closes diagnostics and database
  resources, and flushes telemetry inside the repository's one process grace
  period. Publisher cleanup has its own five-second supervision bound; timeout
  or panic is returned as a process error instead of hanging shutdown. A forced
  stop leaves durable lease recovery rather than marking loss or success.
- Broker outage accumulates visible retry/backlog state. It never terminates or
  degrades the API process merely because publication is unavailable.

### OUT-10 — Observability and privacy

The relay exposes broker-neutral low-cardinality signals for:

- mutually exclusive counts and oldest timestamps for eligible, in-progress,
  retry-wait, recovery-due, ordering-blocked, poison, and published-retained
  states;
- retained ordering high-water row count plus event, ordering-head, and redrive
  table/index byte observations, so key-cardinality or audit-ledger growth cannot
  hide behind event-row cleanup;
- last successful state observation and last durable publication progress;
- claim, publish, progress-commit, retry, poison, recovery, redrive, cleanup,
  observation, and drain operation outcomes/durations;
- current in-flight work and readiness.

Counts and oldest timestamps come from authoritative row state. A stale
observation is distinguishable from an empty backlog. Database-global gauges
from multiple replicas are freshness-filtered and aggregated with `max`, not
summed; process-local counter rates are summed.

Metric attributes are bounded state, operation, outcome, and error-class enums.
Event, ordering, tenant, destination, payload, SQL, error text, and other
unbounded values are never metric labels. Payload, metadata, credentials, DSN,
and raw broker/SQL errors are never logged. Event ID appears only in
entity-level terminal/recovery/redrive logs or sampled traces needed for
forensics, never ordinary per-event success logs.

### OUT-11 — Migration, compatibility, rollout, and rollback

The canonical Goose runtime remains the only schema owner. Outbox migrations
are normal transactional, logged PostgreSQL migrations with canonical Up and
disposable-proof Down sections. Published versions remain append-only.
Application startup and relay startup never migrate.

Rollout order is migration, writer, then relay. The pack ships its whole schema
as one migration, so a service has no intermediate version to stop at and no
mixed-version transition to fence. The template authors that schema in place
because nothing has applied it; in a service generated from the template, every
migration is append-only from the moment it exists.

The API may run before its relay; committed rows form backlog. The relay may run
before any writer and observes an empty queue.

Rollback stops new writer behavior or the relay but leaves the table and rows
intact. It does not retract published events, run production Down, or remove
data. A replacement relay must read every stored envelope version until backlog
and the seven-day published-retention window close. Envelope changes are
additive or versioned; incompatible contraction requires drain, verification,
and a separately proved release operation.

## Invariants and edge cases

- There is no code path that commits a required domain mutation while silently
  omitting its required event intent.
- There is no production publisher path that reports success without durable
  broker acknowledgement.
- There is no state transition that treats timeout, cancellation, ambiguous
  acknowledgement, lost fencing, or ambiguous database commit as delivery.
- Duplicate publication never changes the event ID or envelope.
- Published-row cleanup cannot permit a duplicate or lower sequence to be
  appended for an ordering key; the independent high-water authority survives.
- A stale relay never updates a row after another lease is issued.
- Retry exhaustion and poison remain visible and block their ordering key.
- Cleanup cannot select unfinished or poison work.
- An empty queue is a valid ready state when its observation is fresh.
- A publisher panic or relay-loop failure is supervised, makes readiness false,
  and leads to bounded process shutdown without marking delivery.
- A payload, metadata value, ordering key, and total envelope at each exact size
  limit succeeds; one byte over any boundary is rejected inside the feature
  transaction.

## Decisions, constraints, and authorities

- PostgreSQL row state is authoritative for intent, ownership, attempt,
  terminal, redrive, and retention state. Metrics/logs are derived evidence.
- A broker's durable acknowledgement is authoritative only for the selected
  publication attempt; PostgreSQL finalization remains the relay progress
  authority and can lag that fact, producing duplicates.
- The feature owns domain meaning, transaction orchestration, event content,
  privacy classification, and whether an event is required.
- The outbox pack owns envelope validation, append semantics, relay state,
  retry/poison/redrive policy, cleanup, lifecycle, and broker-neutral telemetry.
- A separately selected adapter owns mapping destination/envelope to the broker,
  durable acknowledgement, connection security, broker telemetry, and
  conformance proof.
- Downstream consumers own duplicate-safe side effects and any inbox/dedupe.
- The merged Goose/sqlc/PostgreSQL 17/profile/lifecycle/telemetry authorities on
  `origin/main` remain canonical.

## Success criteria and proof expectations

1. Real PostgreSQL proves atomic feature mutation plus append, every requested
   rollback/failure boundary, and commit failure with no orphaned side.
2. Real PostgreSQL proves disjoint concurrent claims, lease expiry, stale-token
   fencing, crash recovery, multiple replicas, and per-key head blocking.
3. A deterministic publisher harness proves failure before acknowledgement,
   acknowledgement followed by skipped finalization, duplicate publication on
   restart, retry/poison/redrive, cancellation, drain, and forced stop.
4. A real accepted broker adapter proves event-ID conservation and durable
   acknowledgement before finalization; its absence narrows the claim to local
   relay correctness rather than production adapter readiness.
5. Backlog signals prove empty, outage, retry, recovery, poison, cleanup, stale
   observation, retained ordering-authority growth, and resumed progress without
   high-cardinality labels. Cleanup proof also demonstrates that deleting a
   published event cannot delete or weaken its ordering high-water row.
6. Canonical migration Up/no-op/Down/Up, sqlc drift, image packaging, and
   service/relay process startup prove one ordered migration path.
7. Initialization proves invalid combination rejection before mutation,
   `OUTBOX=none` physical purity, retained-profile completeness, alternate-choice
   refusal, and repeated byte stability.
8. Focused unit, real PostgreSQL, race/liveness, process lifecycle, and the
   smallest matching repository aggregate pass on the final fixed candidate.
9. Documentation states at-least-once, duplicate behavior, operator recovery,
   capacity limits, rollout/rollback, adapter boundary, and non-goals without an
   exactly-once claim.

## Risks, assumptions, and reopen conditions

- `[assumption]` PostgreSQL uses logged tables and ordinary durable commit.
  Risk: weakened `synchronous_commit`/`fsync` can acknowledge data later lost
  on crash. A live rollout must verify the database durability posture; reopen
  delivery readiness if it is intentionally weaker.
- `[decided]` One in-flight event per relay keeps local ownership and shutdown
  minimal; scale through safe replicas. Reopen performance design only when a
  representative workload misses accepted pickup/drain budgets.
- `[decided]` Per-key poison blocks later events to preserve the explicit order
  contract. Reopen Specification if a domain instead accepts overtaking.
- `[decided]` Ordering high-water rows survive event cleanup and may grow with
  distinct keys. Data/observability proof measures row and relation/index-byte
  growth; reopen terminal-key retirement when that retained surface violates an
  accepted storage or maintenance budget and the feature can prove key finality.
- `[decided]` Polling remains selected while no accepted CDC owner/provider and
  no polling capacity failure exist. Reopen System Design under the CDC
  decision-flip conditions in `research/synthesis.md`.
- `[decided]` Seven-day published retention is the template default, not a legal
  or domain retention promise. A derived service must reopen data/privacy policy
  when its investigation, replay, or deletion obligations differ.
