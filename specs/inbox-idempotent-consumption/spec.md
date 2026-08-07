# A redelivered message applies its effect once

status: ready
Problem: The repository closes the producer half of reliable messaging and
leaves the consumer half to every adopting service. `postgresoutbox` makes a
domain event durable in the same transaction as the mutation that caused it, and
`natsjs` delivers it at least once — both correctly, both documented. What
happens when the same message arrives twice is stated as a requirement on the
reader ("handlers must tolerate duplicates") and supported by nothing: no store,
no worked shape, no proof. A redelivery is not exceptional here — it is the
ordinary result of a lost acknowledgement, a forced drain, a lease recovery, or
an operator redrive, all of which the packs already produce on purpose. The
first developer to consume an event therefore has to design the hardest part of
the pattern alone, in the layer where getting it wrong is least visible: a
double-applied effect leaves no error, no failed request, and no alert.

## Scope and non-goals

In scope: what identity a duplicate is recognized by, what "processed once"
guarantees and what it does not, how long that recognition survives, how it
behaves under concurrent delivery of the same message, and the layer an adopting
service's handler shape must be legal in.

Non-goals, each with the reason it stays out and what would reopen it:

- **Effects that do not commit with the claim.** An outbound HTTP call, a file
  write, or a publish to a second broker cannot be rolled back by the
  transaction that records the claim, so this outcome cannot make them
  once-only. The supported answer for those is unchanged and already owned
  elsewhere: carry the same durable business key downstream and let the
  receiver deduplicate, or enqueue the effect through the outbox so it becomes
  a transactional write here and someone else's delivery problem. R5 states the
  boundary rather than hiding it. Reopen only with a repository-owned decision
  to add a two-phase or compensating mechanism, which is a materially larger
  outcome than this one.
- **Exactly-once delivery.** Unchanged. Delivery stays at-least-once in both
  packs; this outcome changes what a second delivery *does*, not whether it
  happens.
- **Ordered processing.** Unchanged, and deliberately not addressed here: the
  composed ordering guarantee already ends at the broker, and a dedup record
  neither restores nor worsens it. Reopen as its own outcome.
- **Automatic dedup inside the transport packages.** `natsjs` stays
  broker-only and must not learn about PostgreSQL; the repository's dependency
  rules already forbid it. The two are joined at the composition root. Reopen
  only if a transport ships its own durable store.
- **A generic idempotency key for inbound HTTP requests.** Same pattern,
  different actor, different lifetime, and a client-supplied rather than
  producer-supplied key. Reopen as its own outcome.

## Behavior and contract delta

### R1 — A message's effect is applied at most once per consumer

Actor: a consuming process running a feature handler.

Trigger: a message is delivered to that handler, whether for the first time or
again.

Precondition: the handler's effect is a write to the same PostgreSQL database
that stores the claim.

Rule: the claim and the effect commit or roll back together. A delivery that
finds no existing claim for its identity applies the effect; a delivery that
finds one applies nothing and reports success to the transport.

Outcomes and transitions:

| Input state | Outcome | Side effect |
| --- | --- | --- |
| No claim for this identity and consumer | Applied | Claim and effect committed together |
| Claim exists, committed by an earlier delivery | Skipped | None; the transport is told the delivery succeeded |
| Claim taken, effect then fails | Not applied | Nothing committed — the claim rolls back with the effect, so redelivery retries it |
| Claim taken, process dies before commit | Not applied | Nothing committed; the transport redelivers |

The skip is an ordinary outcome, not a failure: it must be distinguishable by
the caller from a transport or configuration fault, because a handler routes on
it — a skip is acknowledged, a fault is retried.

Scope of the guarantee: "once" is per consumer identity, not per message. Two
consumers with different identities each apply the same message once, which is
the point of naming the consumer at all — a second consuming service is a
feature, not a duplicate.

Contract delta: new capability. No existing package's behavior changes; a
service that adopts nothing keeps today's semantics.

Falsifier: deliver one message twice to a handler whose effect is a row insert,
and read the effect — exactly one row, and the second delivery is acknowledged
rather than retried. Force the effect to fail on the first delivery and assert
that the second delivery applies it, which proves the claim rolled back rather
than poisoning the message forever.

### R2 — The dedup identity is the message's logical identity

Actor: the handler adapter that supplies the identity.

Rule: recognition is keyed on the identity that survives every path the packs
use to present the same occurrence again — ordinary redelivery, dead-letter
transfer, and operator redrive — together with the consuming identity.

Authority for what that is: `natsjs` already distinguishes the two identifiers
it carries. The publication identifier is the broker's deduplication token and
is deliberately *not* stable: a dead-letter transfer mints a fresh one derived
from the source delivery, and a redrive uses a new one. The logical message
identifier is preserved across both. The stable identity is therefore the
logical one, and an implementation that keys on the broker token would fail to
recognize exactly the redeliveries an operator caused on purpose.

Outcomes:

- Identity present and stable → recognized on every later delivery.
- Identity absent or empty → the claim is rejected before anything is stored.
  An unidentifiable message cannot be deduplicated, and silently applying it
  would make the guarantee depend on a field nobody checked.

Bounds: the identity is size-bounded like every other stored identifier, and a
value exceeding the bound is rejected rather than truncated — a truncated key
would collide two distinct messages into one claim, which is worse than
refusing.

Contract delta: none to the transports. This rule reads identifiers they
already publish.

Falsifier: dead-letter a message and redrive it; the redelivered message's
logical identifier equals the original's while its publication identifier
differs, and a claim keyed on the logical one recognizes it.

### R3 — Concurrent deliveries of one message resolve to a single application

Actor: two handler invocations for the same identity at the same time.

Trigger: possible today without any fault — a lost acknowledgement redelivers a
message while its first delivery is still running, and the worker runs handlers
concurrently.

Rule: the claim is decided atomically against committed state. Of two concurrent
deliveries, exactly one applies the effect. The other observes the first's
outcome once it is decided: skipped if the first committed, applied if the first
rolled back.

Bound: the loser waits for the winner's transaction to resolve, so a slow
handler delays a concurrent duplicate for as long as it runs. That wait is
bounded by the handler timeout the transport already enforces and consumes one
database connection while it lasts — an accepted cost, because the alternative
is applying the effect twice.

Contract delta: none. This states a property the implementation must have, not a
new surface.

Falsifier: start two deliveries of one identity so that both are inside the
claim before either commits; exactly one effect exists afterwards, and neither
delivery reports a fault. Repeat with the winner rolling back and assert the
loser applies the effect.

### R4 — Recognition outlives every redelivery the packs can produce

Actor: an operator sizing retention.

Trigger: claims accumulate, and a store that only grows is not operable.

Rule: claims are retained for a bounded window and then deleted. The window must
exceed the longest interval over which the transports can still present the same
message again by their own mechanisms; deleting a claim earlier reintroduces the
double application this outcome exists to remove.

Authority for the horizon: it is derivable from configuration already validated
at startup — the consumer's attempt budget, its acknowledgement wait, and its
configured retry delays together bound how long ordinary redelivery can
continue. The relationship between that horizon and the retention window is
therefore checkable rather than advisory, and startup is where an unsafe
combination must be refused, for the same reason the relay refuses a lease
shorter than its publish timeout: a window discovered to be too short after the
fact has already lost the guarantee, silently.

Outcomes:

- Retention window above the horizon → accepted.
- Retention window at or below the horizon → refused at startup, naming both
  values, because the operator reading it holds a config file rather than this
  spec.

Explicitly outside the horizon: an operator redrive of a dead-lettered message
after its claim was deleted applies the effect again. Redrive is unbounded in
time by design, so no finite window covers it. This is stated as a known
consequence of redriving old messages rather than defended against; an operator
redriving beyond the retention window is asserting they want the message
processed.

Bounds: deletion is batched and bounded, so retention never takes an unbounded
lock or an unbounded transaction, and a backlog of expired claims drains over
several passes instead of one.

Contract delta: new operator-visible configuration and a new periodic duty in
whichever process owns it.

Falsifier: configure a retention window below the derived horizon and assert
startup refuses it, naming both values. Configure one above it and assert a
claim older than the window is gone while a claim inside it still suppresses a
redelivery.

### R5 — The guarantee's boundary is stated where a handler author reads it

Actor: a developer writing the first handler.

Rule: the pack states, in its own documentation and package documentation, that
the guarantee covers effects committing in the claim's transaction and covers
nothing else. An effect already flushed outside that transaction before a
rollback is not undone, and the pack must say so rather than let "idempotent
consumer" be read as a whole-handler property.

Contract delta: guidance only. No runtime behavior.

Falsifier: the boundary appears in the pack's own documentation, not only in
this spec.

### R6 — The documented handler shape compiles where it is shown

Actor: a developer adopting the pack.

Rule: the worked wiring must be legal in the layer it is shown for, and must not
turn deduplication into domain policy. A feature use case must remain callable
without knowing it is deduplicated; the concrete transaction handle and the
claim store appear only where the repository's enforced dependency rules already
permit them.

Authority: the repository's dependency lint denies a feature package both the
concrete infrastructure packages and the driver, and the reference service
already documents the resolving shape for exactly this problem — a feature-owned
unit-of-work port whose adapter binds the driver handle, so that two writes
commit together without the feature naming a transaction. The claim is a third
write with the same requirement. This rule fixes the constraint; which port the
claim joins is a design decision, bounded by that lint and by "not domain
policy".

Contract delta: guidance only.

Falsifier: the documented shape, placed in the layer the document names for it,
passes the repository's dependency lint.

### R7 — The pack is selectable and removable on its own

Actor: a service initializing the template.

Rule: the inbox is an optional pack, selected independently of the outbox and of
any transport, and requiring the PostgreSQL profile because its guarantee is a
property of that database's transaction. Not selecting it removes its schema,
runtime code, configuration, documentation, and tests, leaving no residue —
the disposition every other optional pack here already has.

Precedence: selecting the inbox without PostgreSQL is refused at initialization
rather than accepted and left broken.

Contract delta: one new initialization choice.

Falsifier: initialize without the inbox and assert no inbox schema,
configuration key, or package remains; initialize with the inbox and without
PostgreSQL and assert the combination is refused.

## Invariants and edge cases

Unchanged and load-bearing — this outcome must not weaken any of them:

- Delivery stays at-least-once in both packs, and neither transport learns about
  PostgreSQL. The join happens at a composition root.
- `natsjs` keeps its two identifiers with their current meanings; R2 reads them
  and changes neither.
- A handler that returns an error still retries under the transport's existing
  attempt budget, and still dead-letters when that budget is spent. A claim that
  rolled back leaves no trace that would shorten or extend that budget.
- The transport's drain semantics are unchanged: a handler cancelled by shutdown
  leaves its message unacknowledged for redelivery, and its claim rolls back
  with everything else.
- Metric label vocabularies stay closed. No unbounded attribute — no message
  identifier, no consumer-supplied string — reaches a metric, span, or log
  attribute.

Edge cases each rule must answer, resolved above: a message with no usable
identity (R2), two concurrent deliveries (R3), a redelivery after its claim
expired (R4, and the redrive case named as outside the horizon), an effect that
escaped the transaction before rollback (R5), a handler whose effect targets a
different datastore (non-goals, and R1's precondition).

One further case, resolved by R1's table rather than a rule of its own: a
delivery that skips must still acknowledge. Reporting a skip as a failure would
send an already-applied message around the retry budget and eventually to the
dead-letter stream, turning the protection into a poison generator.

## Decisions, constraints, and authorities

- **D1 — The claim is taken inside the caller's transaction, never in one of its
  own.** A separate transaction reintroduces exactly the dual-write the outbox
  exists to remove: a crash between claim and effect either loses the effect
  permanently (claim committed first) or applies it twice (effect committed
  first). Reopen only for an effect that cannot share the transaction, which the
  non-goals already route elsewhere.
- **D2 — Recognition is keyed on the logical message identity, not the broker's
  deduplication token.** Authority: the transport's own distinction between the
  two, and its documented behavior that a redrive preserves the first and
  replaces the second. Reopen if a selected transport carries no identity stable
  across its own redelivery paths, in which case the producer must supply one.
- **D3 — The claim is scoped by consumer identity.** One message legitimately
  reaches several consumers, and a global key would let the first consumer
  suppress the others. Reopen never on scale grounds; the scope is a correctness
  property, not an index-size trade.
- **D4 — Retention is time-bounded and its floor is derived and enforced, not
  advised.** A documented "make sure it is long enough" is the failure mode this
  spec is trying to remove, since the symptom of getting it wrong is a silent
  double application. Reopen if a transport's horizon stops being derivable from
  validated configuration, in which case the floor becomes an explicit operator
  input with the same startup check.
- **D5 — Deduplication is not domain policy.** The feature decides what the
  effect is; whether this delivery is a repeat is a property of the transport
  and of prior state, and a use case that branches on it has absorbed an
  infrastructure concern it cannot test in isolation. This constrains R6's
  placement without choosing it.
- **D6 — The inbox is a sibling pack, not part of `postgresoutbox`.** Authority:
  that package's recorded "does not own" boundary, the extension seam that
  routes outbox and inbox persistence through separate accepted workflows, and
  the producer-side spec that defers the inbox to its own outcome. Reopen only
  by changing that recorded boundary.
- **Constraint — the composing pieces already exist.** A transaction helper that
  yields a driver handle, a bounded periodic-duty shape with drain, a closed
  telemetry vocabulary pattern, and a per-test database harness are all present
  and in use by the outbox pack. This outcome adds a store and a schema to an
  existing shape rather than introducing a new runtime concern.

## Success criteria and proof expectations

1. A message delivered twice applies its effect once. Scope: one message, one
   handler with a transactional effect, against a real PostgreSQL. Pass: one
   effect, both deliveries acknowledged. Fail: two effects, or a delivery
   reported as a fault.
2. A failed effect leaves no claim. Scope: one message whose handler errors,
   then succeeds on redelivery, against a real PostgreSQL. Pass: the effect
   exists exactly once after the second delivery. This is the criterion that
   separates a correct implementation from one that claims before the effect.
3. Concurrent deliveries of one identity resolve to one application. Scope:
   two overlapping handler invocations forced to be inside the claim
   simultaneously, against a real PostgreSQL, because the resolution is the
   database's and cannot be proven against a stubbed driver. Pass: one effect,
   no fault reported by either.
4. A retention window at or below the derived redelivery horizon is refused at
   startup, naming both values. Scope: configuration validation. Pass: refused
   with both values in the message.
5. A claim older than the window is deleted and a claim inside it still
   suppresses a redelivery. Scope: integration, with retention driven
   explicitly rather than by waiting.
6. Two consumers each apply the same message once. Scope: integration. Pass: two
   effects, one per consumer identity, and a second delivery to either applies
   nothing.
7. Telemetry vocabularies remain closed with the new operations present, and no
   message identifier or consumer-supplied string reaches a metric attribute.
   Scope: the existing bounded-vocabulary proof shape.
8. The documented handler shape passes the repository's dependency lint in the
   layer the document places it in.
9. Every existing messaging proof still passes unchanged, including the
   transport's redelivery, dead-letter, drain, and race suites, and the outbox's
   ordering and lease suites.

Proof expectations name evidence boundaries, not mechanisms: R1, R3, and R4's
deletion behavior have their authority in PostgreSQL and need a real one; R2's
identity claim needs an actual dead-letter and redrive rather than a constructed
message; R6 needs the repository's own lint rather than a reading.

## Risks, assumptions, and reopen conditions

- **Assumption — the logical message identifier is present and stable on every
  delivery path this repository ships.** Affected rules: R1, R2. Safe boundary:
  the transport requires it on publish, preserves it through dead-letter
  transfer, and preserves it through redrive; a message that reaches a handler
  without one cannot have been published by this repository's producer.
  Invalidating evidence: a delivery path that presents a handler with a message
  whose logical identifier differs from the original's. Reopen owner: this spec.
  Reopen condition: if such a path exists, R2's identity must move to a
  producer-supplied envelope field, and the producer contract reopens with it.
- **Assumption — the concurrent-delivery wait in R3 is acceptable at the
  consumer's concurrency.** Affected rule: R3. Safe boundary: the wait occurs
  only for two live deliveries of the *same* identity, which is rare and already
  bounded by the handler timeout. Invalidating evidence: a workload where
  same-identity concurrency is routine — a hot key redelivered continuously —
  such that waiters exhaust the connection pool. Reopen owner: this spec. Reopen
  condition: on that evidence the loser should fail fast and let the transport
  redeliver instead of waiting, which is a different observable outcome and so
  is a spec change rather than a tuning change.
- **Risk — the pack invites the wrong mental model.** "Idempotent consumer"
  reads as a whole-handler property, and a developer who adopts it may stop
  thinking about the external calls it cannot cover. Mitigated by R5 putting the
  boundary in the pack's own documentation rather than only here; not mitigated
  against a handler that ignores it, which is stated as the trade.
- **Risk — retention deletes compete with claims on the same table.** Mitigated
  by R4's bounded batching; residual exposure is a tuning question the same
  shape already answers in the outbox's retention duty.
- **Assumption — no adopting service needs a claim to outlive its retention
  window for audit.** Affected rule: R4. Safe boundary: the claim records that a
  message was processed, which the effect itself also records in domain terms.
  Invalidating evidence: a compliance requirement to prove non-reprocessing over
  a longer horizon than operational safety needs. Reopen owner: this spec.
  Reopen condition: retention gains an audit window distinct from the safety
  floor; the floor itself never moves down.

## Proof gaps

The [Review Independence](../../docs/spec-first-workflow/shared/review-independence.md)
trigger applies: this spec fixes a durable schema, a persisted-data guarantee,
and a concurrency rule whose failure mode is silent. An independent
specification review was **not** run, and the reason is specific rather than
procedural: this spec was authored while a concurrent session held uncommitted
changes to `internal/config`, `migrations/`, the shared sqlc sources, and their
generated output — the same surfaces this outcome extends. The tree did not
compile during authoring, so no reviewer could have validated a claim against
it. The spec carries focused root self-review only, which found and closed three
defects: R1 did not say a skip must be acknowledged (an unacknowledged skip
turns the protection into a poison generator), R4 stated retention as guidance
rather than an enforced floor, and R2 originally keyed on the publication
identifier, which a dead-letter transfer replaces.

Two claims here are read from current source and documentation rather than
executed, because the tree did not build: that a dead-letter transfer preserves
the logical identifier while minting a new publication identifier, and that the
redelivery horizon is derivable from already-validated worker configuration.
Both are cheap to confirm once the tree is green and both are load-bearing for
R2 and R4.

Next useful check, in order of value: confirm those two claims against a
building tree; then an independent reviewer falsifying R3's concurrency rule and
R4's derived floor. Reopen owner: Specification.

## Sequencing

This outcome is specified against the producer contract that
[`specs/outbox-trace-continuity-and-key-lifecycle`](../outbox-trace-continuity-and-key-lifecycle/spec.md)
is currently implementing, which is where that spec's D4 deferred it. Nothing
here depends on that outcome's new envelope field, so the two do not conflict on
behavior — but they do share `migrations/`, the sqlc sources and their generated
output, and `internal/config`. Implementation of this spec should start after
that work merges, or in an isolated worktree whose merge conflict on those four
surfaces is accepted up front.

### Accepted: the append-ownership paragraph stands, and R5 extends it

[PostgreSQL transactional outbox](../../docs/postgres-transactional-outbox.md)
now carries a paragraph naming the append path's owners — the feature owns which
occurrence happened and returns its own type, the PostgreSQL repository adapter
owns the transaction and the translation, the composition root owns building the
store — anchored to the depguard rule that enforces it. That paragraph is
accepted and retained.

That spec's R5 answers the same defect from the other end: it fixes the
feature-owned port the use case reaches the outbox through. The two are the same
shape described from opposite sides, and the reference service already ships it
whole — a feature-declared repository port carrying the feature's own event
type, bound by an adapter that holds the driver handle. R5 therefore adds the
port half rather than replacing the ownership half; an R5 realization that
rewrites this region should preserve the named owners and the lint anchor, and
its falsifier is satisfied by the same lint either way.

Recorded here because this spec is the artifact that observed the overlap.
Reopen condition: R5 lands a port shape whose owners differ from the ones named
above — then the paragraph is wrong rather than merely partial, and this note
does not protect it.
