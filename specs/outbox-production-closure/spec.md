# A committed event reaches an explicit publication outcome and survives uncertain results without silent loss

status: ready
Problem: The PostgreSQL outbox has strong transaction, claim, lease, ordering,
and recovery semantics, but its shipped production path is not closed. A
service initialized with both the PostgreSQL outbox and the repository's NATS
JetStream transport still gets a relay with no registered publisher; the only
worked adapter is test-only, maps every NATS rejection to permanent poison, and
does not carry the stored creation context to the consumer. Separately, the
writer guidance creates an event identifier inside the transaction even though
`ErrCommitUnknown` forbids a blind retry, and an acknowledgement that remains
ambiguous is retried forever. The result is a pack that is correct in its core
but still asks each adopter to invent the failure semantics at its two most
dangerous boundaries.

## Scope and non-goals

In scope: the generated runtime behavior when the outbox and NATS profiles are
selected together, the selected adapter's failure classification and trace
continuity, the terminal state of repeated ambiguous publication outcomes, the
operator actions that recover that state, and the caller contract after an
unknown PostgreSQL commit outcome.

This outcome is completed together with the sibling
[idempotent-consumption specification](../inbox-idempotent-consumption/spec.md):
the outbox remains at-least-once, so closing the production pattern includes a
supported consumer-side transactional deduplication path. That sibling owns its
schema and behavior; this spec does not duplicate them.

Non-goals:

- **Exactly-once delivery or arbitrary external effects.** Delivery can still
  duplicate, and an HTTP call or other effect outside the inbox transaction is
  not made once-only. Reopen only with a concrete downstream protocol that owns
  a stable idempotency key or compensation.
- **Generic end-to-end ordered processing.** The outbox still orders publication
  per key; JetStream and concurrent handlers do not serialize that key. This is
  already an explicit contract, and no domain requirement selects one of the
  materially different consumer-ordering strategies. Reopen for a concrete
  consumer whose business invariant requires it.
- **A universal broker adapter.** Only the repository's selectable NATS
  JetStream profile is composed. An outbox-only service continues to supply its
  own adapter and fails closed until it does.
- **Forwarding caller metadata or source onto NATS.** Their current adapter
  disposition is unchanged. The adapter carries the fields the NATS envelope
  already owns plus the separately outbox-owned creation context; adding new
  public NATS envelope fields requires its own contract.

## Behavior and contract delta

### R1 — The combined outbox and NATS profile is runnable without source edits

Actor: a developer initializing a service with PostgreSQL outbox and NATS
JetStream selected together.

Trigger: the generated outbox relay starts with valid PostgreSQL, outbox, and
messaging configuration.

Rule: startup builds a real NATS publisher and the relay can publish. The
adapter uses the outbox event identifier as both the logical message identity
and publication/deduplication identity, and carries destination, type, schema,
occurrence time, payload, and ordering key without changing their meanings. It
is safe under the relay's configured publication concurrency.

The adapter also puts the event's stored creation context on the broker message
as W3C trace headers. The consumer therefore extracts the context of the
operation that created the event, not merely the later relay attempt. An absent
or invalid stored context remains fail-open: the event still publishes without
that correlation.

Profile outcomes:

| Selected profiles | Outcome |
| --- | --- |
| PostgreSQL outbox + NATS JetStream | Relay starts with the selected adapter and publishes |
| PostgreSQL outbox without a messaging adapter | Relay continues to fail closed before doing work |
| NATS JetStream without the outbox | Existing direct producer and worker behavior is unchanged |

Contract delta: the combined profile changes from "compiles but requires a
manual edit" to a working composition. The outbox-only extension seam remains.

Falsifier: initialize the combined profile, start its dependencies, append one
event, and observe the same identity and envelope values at a worker together
with the producing creation context. Initialize outbox without messaging and
observe the existing missing-adapter startup refusal.

### R2 — Publication failures are classified only by what they prove

Actor: the selected NATS adapter returning a publication result to the relay.

Rule: the adapter classifies by broker-acceptance evidence, not by the broad
transport error family alone. A generic NATS rejection is never sufficient
evidence that retrying the exact event can never succeed.

| Proven outcome | Relay classification | Required disposition |
| --- | --- | --- |
| The fixed event envelope is invalid before dispatch and the same bytes can never be accepted | Permanent | Poison on the first occurrence |
| Dispatch did not happen because of capacity, drain, an expired pre-dispatch context, or another transient admission failure | Not accepted | Retry with backoff; poison only at the configured attempt limit |
| The broker explicitly did not accept the publish, but the reason may change with topology or broker state | Not accepted | Retry with backoff; poison only at the configured attempt limit |
| Dispatch may have happened but no durable acknowledgement is available | Ambiguous | Follow R3 |
| The adapter cannot prove one of the first three outcomes | Ambiguous | Follow R3 |

Precedence: permanent requires affirmative evidence about the exact immutable
event. Capacity, draining, context cancellation, missing topology, and an
unrecognized error can never become permanent merely because they share a
wrapper with invalid input.

Contract delta: temporary NATS admission failures stop poisoning an event on
their first occurrence. Invalid envelope data remains immediately operable as a
deterministic poison.

Falsifier: drive invalid input, capacity exhaustion, producer drain, definite
broker refusal, and lost acknowledgement through the selected adapter. Each
lands in the table's class; in particular, capacity and drain never produce the
permanent-publication class.

### R3 — Ambiguous publication stops automatically and remains recoverable

Actor: the relay and an operator responsible for an event whose broker
acknowledgement repeatedly remained unknown.

Trigger: any attempt may have reached the broker without returning a durable
acknowledgement.

Rule: publication uncertainty is a sticky durable fact. Once observed, no later
failure can prove that the earlier attempt was not accepted. A later durable
acknowledgement resolves the event as published; otherwise the fact survives
retry, lease recovery, process restart, and operator redrive.

Automatic disposition follows this precedence:

| History and current result | Disposition |
| --- | --- |
| Durable acknowledgement | Publish, regardless of earlier uncertainty |
| No uncertainty ever observed; current result permanent | Deterministic poison immediately |
| No uncertainty ever observed; current result not accepted below `max_attempts` | Retry |
| No uncertainty ever observed; current result not accepted at `max_attempts` | Attempt-exhausted poison |
| Uncertainty was observed; current result remains retryable below `max_attempts` | Retry with the same identity |
| Uncertainty was observed; current result is permanent, or any failure reaches `max_attempts` | Quarantine as `outcome unknown` |

The unknown quarantine is distinct from deterministic poison and confirmed
publication. The event remains stored, is excluded from cleanup, and blocks
later events for the same ordering key; unrelated keys continue to progress.

Operator outcomes:

| Evidence and action | Result |
| --- | --- |
| Durable acceptance is independently confirmed | Finalize the same event as published without another broker call; record an audit identity |
| Non-acceptance is confirmed | Redrive the same event identity; record an audit identity |
| Outcome is still unknown | Leave quarantined, or explicitly redrive while accepting duplicate risk; never silently mark published or delete |
| The same operator action is repeated with the same audit identity | Return its first result without applying it twice |

Every redrive keeps the original event and publication identity. That reduces
duplicates where the broker still remembers the identity but does not claim
exactly-once across the broker's finite deduplication horizon.
Confirming acceptance performs the same durable finalization as a broker
acknowledgement, including advancing an ordered key and releasing its successor;
it never calls the broker again.

Operator transition rules:

- Confirm-accepted and redrive-unknown are valid only from unknown quarantine.
  Not found returns not found; ready, leased, deterministic poison,
  attempt-exhausted poison, and published states return a state conflict with no
  side effect.
- Each audit identity names exactly one action kind and one event. Repeating
  that same pair returns its first committed result even if the event has since
  moved again. Reusing the identity for another action kind or event is an audit
  conflict with no side effect.
- Actions serialize against the event. The first valid transition that commits
  wins. A concurrent different action observes the resulting non-unknown state
  and returns a state conflict; it never reports success for an action it did
  not apply.
- Redrive returns the event to retryable state and resets its automatic attempt
  budget, but it does not erase the sticky uncertainty fact. A later failure can
  therefore return it to unknown; only a durable acknowledgement or an audited
  confirm resolves the fact as published.

Rollout rule: current storage cannot prove whether an earlier failed attempt was
ambiguous because every retry overwrote the previous error class. A pre-rollout
unpublished row with any recorded attempt is therefore treated as having
observed uncertainty, including a row currently labelled permanent or poisoned:
the legacy NATS adapter used that label for capacity, drain, and broker refusal
as well as invalid immutable input. A sticky row already poisoned or at or above
`max_attempts` enters unknown quarantine before another publish; one below the
limit may continue with the same identity under the table above. Unattempted and
published rows keep their current meanings. This deliberately prefers one-time
operator review over guessing that overwritten history proved non-acceptance.

Contract delta: `max_attempts` bounds ambiguous automatic retries as well as
definite non-acceptance, but sticky uncertainty takes precedence at exhaustion
because its recovery evidence is different. No second attempt-limit setting is
added.

Falsifier: force only ambiguous results through the limit. The event becomes
unknown rather than published, deleted, or deterministic poison; a successor on
its key stays blocked, an unrelated event publishes, and each operator action
obeys the table including idempotent repetition. Repeat with an ambiguous result
followed by permanent or definite non-acceptance and assert the earlier
uncertainty cannot be overwritten. Race confirm against redrive and assert one
transition wins while the other reports conflict.

### R4 — An unknown transaction commit is resolved by a stable receipt

Actor: feature code executing a domain mutation and outbox append in one
PostgreSQL transaction.

Precondition: before beginning the transactional attempt, the caller establishes
a stable operation receipt. The outbox event identifier may be that receipt
when its row commits in the same transaction; a domain operation identifier may
serve instead. Every retry of the same operation reuses it.

Trigger: the transaction returns `ErrCommitUnknown` because the commit response
was lost.

Rule: the caller does not rerun the mutation with a new identity. It reads the
writer-primary authority by the stable receipt and resolves one of these states:

| Authoritative read | Outcome | Allowed next action |
| --- | --- | --- |
| Matching receipt is present | Applied | Report success with the original identity; do not append again |
| Receipt is absent on a successful current writer-primary read | Not applied | Retry the operation with the same receipt |
| Receipt exists but conflicts with the attempted immutable values | Integrity conflict | Fail; do not mutate or append again |
| The authoritative read itself is unavailable or inconclusive | Still unknown | Preserve the identity and retry reconciliation, not the mutation |

Authority and finality: only a successful read from the current PostgreSQL
writer is decisive. A cache, asynchronous replica, failed read, or timeout
cannot prove absence. When the outbox event identifier is the receipt, presence
of that row proves the domain mutation committed because both writes share the
same transaction.

Contract delta: the worked append guidance no longer creates the event identity
inside a callback that a caller might blindly repeat. `ErrCommitUnknown` becomes
an actionable three-way result instead of an error with no safe caller path.

Falsifier: lose the commit response after PostgreSQL commits and resolve
`Applied` by the pre-existing receipt; force rollback before commit and resolve
`Not applied`; make the read unavailable and observe that no second mutation is
attempted.

## Invariants and edge cases

- The domain mutation and outbox append still commit or roll back in one
  caller-owned PostgreSQL transaction. The outbox never starts or commits it.
- Delivery remains at-least-once. The same event identity is reused for every
  relay retry and operator redrive; a consumer still needs the sibling inbox or
  an equivalent domain-owned idempotency mechanism.
- Confirmed publication, deterministic poison, attempt exhaustion, sticky
  uncertainty, and unknown quarantine remain distinguishable operator facts.
  No transition deletes an unfinished event.
- Poison or unknown outcome for an ordered event continues to block that key.
  The bounded-retry change must not trade ordering for liveness silently.
- The selected adapter does not widen metric, trace, or log cardinality. Event
  identifiers, ordering keys, payload, metadata, broker text, and credentials
  remain absent from bounded telemetry attributes.
- The NATS worker's concurrency and per-key ordering contract is deliberately
  unchanged. Preserving an ordering key as data does not claim ordered handler
  execution.

## Decisions, constraints, and authorities

- **D1 — Compose only selected capabilities.** The combined profile has enough
  information to choose NATS and should not require an adopter to copy test
  code. The outbox-only profile has no such authority and retains its fail-closed
  builder seam.
- **D2 — Reuse `max_attempts`.** It is already the operator's bound on automatic
  attempts for an unfinished event. A second ambiguous-attempt knob would add
  policy without a distinct operational need; the terminal state, not another
  number, expresses the semantic difference.
- **D3 — Uncertainty is monotonic until publication is resolved.** Last-error
  text describes one attempt and cannot overwrite the fact that an earlier
  attempt may have published. A process restart, lease recovery, later rejection,
  redrive, or cleanup cycle must not erase that evidence.
- **D4 — A stable outbox event ID is sufficient as the minimal commit receipt.**
  It commits atomically with the domain mutation and already has a direct read
  path. A separate generic command-id subsystem is not required; a feature may
  use its own operation identity when its business contract needs one.
- **D5 — The existing publication-order boundary stands.** No current domain
  authority requires the generic worker to serialize by ordering key, so this
  outcome fixes lost failure semantics and context propagation without
  inventing a consumer scheduler.

## Success criteria and proof expectations

1. A generated combined outbox/NATS service starts a real relay and publishes
   one event without a source edit. Scope: initialized checkout with real
   PostgreSQL and JetStream. Pass: the worker observes the preserved envelope
   and producing creation context. The outbox-only generated service still
   refuses a missing adapter before work begins.
2. The selected adapter satisfies R2 for invalid envelope, capacity, drain,
   definite broker refusal, ambiguous acknowledgement, and unknown error.
   Scope: deterministic adapter tests plus real-broker proof where acceptance
   evidence belongs to JetStream.
3. Ambiguous attempts stop at `max_attempts` in the distinct unknown state.
   Scope: real PostgreSQL state transitions. Pass: no cleanup or implicit
   publication, same-key blocking, unrelated-key progress, and idempotent
   audited confirm/redrive actions. Earlier ambiguity survives a later rejection;
   confirm/redrive races have one winner; invalid source states and audit reuse
   fail without side effects; pre-rollout unpublished rows, including poisoned
   rows with overwritten history, receive the conservative disposition in R3.
4. `ErrCommitUnknown` resolves applied, not-applied, conflict, and still-unknown
   without a fresh receipt. Scope: real PostgreSQL with a lost commit response
   and an unavailable reconciliation read.
5. The sibling inbox specification's success criteria pass before the complete
   transactional outbox pattern is called production-closed. Its guarantee is
   limited to effects in the claim transaction exactly as that spec states.
6. Time-sensitive relay telemetry proof uses owned synchronization and does not
   depend on metric collection completing inside the publication timeout.
   Scope: the outbox race target. Pass: the previously timing-sensitive
   transition test is repeatable under `-race` without widening production
   timeouts.
7. Existing outbox and messaging proofs remain green, including transaction
   atomicity, lease fencing, ordering, redelivery, dead-letter, drain, profile
   generation, and bounded telemetry vocabularies.

## Risks, assumptions, and reopen conditions

- **Trade-off — quarantine bounds duplication pressure by giving up automatic
  liveness for one event.** The event is retained and visible, and unrelated
  keys progress; an ordered successor remains blocked until explicit recovery.
  Reopen owner: this spec. Reopen condition: a selected broker supplies a
  durable query that resolves publication identity automatically, in which case
  reconciliation may replace operator action without changing the outcomes.
- **Risk — explicit redrive of an unknown event can duplicate it.** Retaining
  the same publication identity helps only within the broker's deduplication
  horizon. The sibling inbox is the supported protection for transactional
  consumer effects; external effects still require their own stable key.
- **Assumption — reconciliation reads the writer.** A replica can report false
  absence under lag and would make R4 unsafe. Reopen owner: Technical Design.
  Reopen condition: if the repository's available read path cannot guarantee
  writer-primary currentness, design must add a writer-owned path or return
  `Still unknown`; it may not weaken absence semantics.
- **Risk — profile composition touches generated-source ownership.** The
  combined and outbox-only initialized checkouts must both compile and repeat
  initialization with zero drift; one profile may not retain an import of a
  package the other removes.

## Review result

Independent whole-bundle review initially returned `FAIL` on two R3 omissions:
uncertainty could be overwritten by a later rejection, and concurrent operator
actions had no closed state/audit precedence. Focused repairs made uncertainty
monotonic, defined the complete operator transition rules, and conservatively
mapped every attempted unpublished legacy row because its overwritten history
cannot prove non-acceptance.

Focused fresh review returned `PASS` for both repairs and the final simplified
rollout rule. No Specification-owned decision remains open. Runtime, migration,
profile, race, and real-dependency evidence remains downstream proof under the
success criteria above; it is not claimed by this artifact.
