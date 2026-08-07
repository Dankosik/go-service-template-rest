# A published outbox event is attributable to the operation that produced it, and an ordering key's retained state has a bounded lifetime

status: ready
Problem: Two operator-visible gaps in the PostgreSQL outbox pack. An event that
crosses the outbox boundary loses its trace: the relay installs a tracer
provider and a W3C propagator and exports no span, so nothing joins a broker
publication back to the request that caused it, and an operator diagnosing a
late or failed publication has only metrics and an event id. Separately,
`outbox_ordering_heads` grows one row per ordering key ever used and nothing can
remove one, so a service keyed on a per-aggregate identity accumulates retained
rows for the life of the database with no supported way to retire them.

## Scope and non-goals

In scope: the trace-correlation behavior of `Store.Append` and the relay's
publication, what the `Publisher` adapter receives, the lifecycle of a retained
ordering-key high-water mark, and the adopting-service guidance that tells a
developer where the append composition lives.

Non-goals, each with the reason it stays out and what would reopen it:

- **Consumer-side inbox / idempotent-receiver store.** This is the consumer half
  of the pair and belongs with the consuming transport, not with
  `internal/infra/postgresoutbox`;
  [Repository Architecture](../../docs/repo-architecture.md) already assigns it
  to a separate accepted workflow and names it in this package's "does not own"
  column. This outcome completes the producer half — a stable dedup identity and
  its creation context reach the adapter — so the inbox can be specified against
  a finished contract. Reopen as its own accepted outcome; it is not cancelled.
- **A tenant column on the envelope.** Tenant identity is derivable domain
  policy, and the pack ships into single-tenant services as often as
  multi-tenant ones. A caller encodes tenant in its ordering key, its event type,
  or its metadata, and deployment-level isolation is the usual boundary. Reopen
  when a repository-owned decision requires the outbox itself to enforce tenant
  isolation — for example a shared relay that must refuse cross-tenant claims.
- **Exactly-once delivery.** Unchanged: delivery stays at-least-once and
  consumers deduplicate on event id.
- **Retiring a poisoned event's hold on its ordering key.** Unchanged: poison
  blocks its key until an operator redrive, which is the accepted ordering-over-
  liveness trade.

## Behavior and contract delta

### R1 — The producing operation's trace context is captured at append

Actor: feature code calling `Store.Append` inside the transaction that owns its
domain mutation.

Trigger: an event is stored.

Rule: the outbox captures the W3C trace context active on the calling context
and stores it with the event as outbox-owned envelope state. Caller-supplied
`Metadata` bytes are stored exactly as given and are never merged with, or
overwritten by, that context.

Outcomes:

- Active, valid trace context present → stored with the event, including its
  sampling decision.
- No active trace context, or one that is not valid for propagation → the event
  stores an empty creation context. This is an ordinary outcome, not a
  rejection: an append from a migration, a backfill, or a test still succeeds.
- The stored creation context is immutable for the life of the row. Retry,
  lease recovery, and operator redrive never replace it, so every attempt at one
  event reports the same origin.

Contract delta: the stored envelope gains one outbox-owned field. The
caller-owned `Metadata` contract is deliberately unchanged, including its
"stored and retried as these exact bytes" guarantee.

Bounds: the creation context is size-bounded like every other envelope field. A
context that cannot be captured — too large for its bound, or not encodable —
is stored as absent, and the append succeeds. The degradation is counted so an
operator can see it.

Amended by design. This rule originally rejected such an event under the
invalid-event sentinel. That is wrong in a way worth recording: the creation
context is not supplied by the caller, it is captured from ambient telemetry
configuration, so rejecting the append punishes a caller who cannot fix the
cause and makes a telemetry setting decide whether a business event is stored.
An outbox exists so that infrastructure faults become backlog rather than failed
requests; a field this package added must not become the one fault that fails a
request. The observable outcome changes from "event rejected" to "event stored
with no trace link, and a counter says so".

Falsifier: append inside a sampled trace and read the stored row — it carries
the same trace id; append with no active span — the row stores an empty context
and the call succeeds; append with caller metadata containing a `traceparent`
key — the stored metadata bytes are byte-identical to what was passed.

### R2 — A publication is joinable to the operation that produced it

Actor: the relay process.

Trigger: the relay attempts to publish a claimed event.

Rule: each publication attempt is observable as a trace span correlated to that
event's stored creation context, so an operator holding a producing request can
reach its publication and the reverse. Correlation survives every path that
repeats an attempt: retry after a temporary failure, republication after lease
recovery, and operator redrive all correlate to the same creation context.

Outcomes:

- Stored creation context present → the publication is joinable to it.
- Stored creation context empty → the publication is still observable as a span;
  it is uncorrelated rather than suppressed.
- Failed publication → the span records the failure under the same bounded error
  class the publish metric already carries for that failure, so a trace and a
  dashboard name one condition rather than two.

Constraint: the outbox never raises or lowers the producer's sampling decision.
An unsampled producing operation does not become sampled because it went through
the outbox.

Contract delta: the relay emits spans where it previously emitted none. No
metric, log, or error class changes meaning.

Falsifier: publish an event appended inside a sampled trace and assert the
exported spans join on trace id; force a publication failure and assert the
span's recorded error class equals the metric's `error.type` for the same event.

### R3 — The adapter receives the creation context separately from metadata

Actor: a `Publisher` implementation.

Rule: the creation context is readable from the event the adapter is handed,
distinct from `Metadata`, so the adapter can place it on broker headers for the
consumer. The outbox does not choose the broker's header encoding and does not
verify that the adapter forwarded anything — the same posture `Metadata` already
has.

Contract delta: `Event` gains a read path for the creation context. Existing
adapters keep compiling and keep behaving identically; not forwarding the
context is a documented loss of consumer-side trace continuity, not a failure.

Falsifier: a worked adapter reads the creation context off a claimed event and
puts it on a broker header without reaching into `Metadata`.

### R4 — A terminal ordering key's retained state can be retired

Actor: feature code that owns the aggregate the ordering key names.

Trigger: the feature asserts that an ordering key is terminal — that no further
event will ever be appended for it.

Precondition: the key has no unpublished events.

Rule: retirement removes the key's retained high-water mark, inside the
transaction the caller owns, so the assertion commits atomically with whatever
domain write makes the aggregate terminal.

Outcomes and transitions:

| Input state | Outcome | Side effect |
| --- | --- | --- |
| Key exists, no unpublished events | Retired | High-water mark removed; ordering-head count falls |
| Key exists, has unpublished events | Rejected with a distinguishable error | Nothing stored; the caller decides whether to roll its transaction back |
| Key unknown, or already retired | Accepted, no change | None — repeated retirement is idempotent |

The rejection is a distinct, matchable failure — a caller must be able to tell
"this key still has work" apart from a transport or configuration fault, because
the first is an ordinary domain outcome it may choose to absorb.

Concurrency: the precondition and the removal are evaluated as one indivisible
step against the same key state that a concurrent ordered append contends for.
An append for the same key therefore either serializes before the retirement —
in which case that retirement is rejected, because the key now has an
unpublished event — or after it, in which case it establishes a fresh retained
mark from its own sequence. A retirement can never remove a mark that a
concurrently committing append is relying on, and two retirements of one key can
never both report having removed it.

Post-state: the key's sequence space restarts. A later append for a retired key
is accepted at any positive sequence, including one already used before
retirement. This is exactly the protection the caller trades away by asserting
terminality, and it is why the assertion is the feature's rather than the
outbox's: only the feature knows whether the identity can recur.

Bounds: retirement is a caller-driven operation only. No time-based, size-based,
or automatic retirement exists, because idleness does not prove terminality.

Observable: the retained-head count reported by state observation falls, and the
operation appears in the relay's bounded operation vocabulary like every other
statement.

Falsifier: retire a key holding a pending event — rejected, and the head is
still readable; retire a fully published key — the head is gone and the observed
head count falls by one; retire the same key twice — the second call succeeds
and changes nothing; append at sequence 1 after retiring a key that reached
sequence 9 — accepted.

### R5 — The documented append shape compiles where it is shown — SCOPE EXIT

**Left to another owner, 2026-08-07.** A concurrent work stream in this checkout
is landing a resolution to this same defect, with a different and equally valid
answer: it places the append in a PostgreSQL repository adapter rather than
behind a feature-owned port, and has already rewritten `doc.go`, `store.go`,
`docs/postgres-transactional-outbox.md`, `docs/repo-architecture.md`, and
`docs/project-structure-and-module-organization.md` to match. Two answers to one
ownership question is worse than either answer, so this outcome yields rather
than competing.

Consequence for the rest of this spec: nothing. R1-R4 are independent of where
the append is composed. The rule below is retained as the statement of the
defect and its acceptance test; the owner is the other work stream.

Reopen condition: that work stream lands without a shape that passes the
repository's dependency lint in the layer its documentation places it in.

Actor: a developer adopting the pack.

Rule: the guidance that tells an adopting service how to append must be legal in
the layer it is shown for. A feature-owned use case reaches the outbox through a
feature-owned port that the composition root binds; the concrete transaction
handle and the outbox store appear only where the repository's enforced
dependency rules already permit them.

Current defect this replaces: the worked snippet in
[PostgreSQL transactional outbox](../../docs/postgres-transactional-outbox.md)
composes `pgx.Tx` and the outbox store directly, which the repository's own
dependency rules reject inside `internal/<feature>`, while the reference service
documents the opposite and correct shape for the same problem. Two canonical
documents currently disagree about where the append lives.

Contract delta: guidance only. No runtime behavior changes under this rule.

Falsifier: the documented shape, placed in the layer the document names for it,
passes the repository's dependency lint.

## Invariants and edge cases

Unchanged and load-bearing — this outcome must not weaken any of them:

- Append neither begins nor commits a transaction; returning its error rolls the
  domain mutation and the event back together.
- One append call remains one statement and one round trip whatever mix of
  ordered and unordered events it carries.
- At most one event per ordering key is claimable, so concurrent publication
  cannot reorder a key.
- Finalization stays detached from process cancellation.
- The retained high-water mark still rejects a replayed sequence for every key
  that has not been explicitly retired, and cleanup of published events still
  never removes a head on its own.
- Metric label vocabularies stay closed; no new unbounded attribute appears on
  any metric, span, or log. Payload, metadata, credentials, DSN, ordering keys,
  broker error text, and SQL text remain absent from every telemetry surface.

Edge cases each rule must answer, resolved above: absent trace context at append
(R1), absent stored context at publish (R2), an adapter that ignores the context
(R3), retirement racing an in-flight append for the same key (R4 precondition,
enforced inside the caller's transaction).

## Decisions, constraints, and authorities

- **D1 — The creation context is outbox-owned envelope state, not merged into
  caller metadata.** Two independent current sources converge on a dedicated
  store for it rather than the message body. Beyond that, merging would break
  the caller's byte-exactness guarantee on `Metadata` and would collide with a
  caller that already carries its own `traceparent` key. Reopen if a selected
  broker cannot carry a header separate from the message body.
- **D2 — The outbox never changes a sampling decision.** Forcing sampled spans
  for outbox publications would make the relay's trace volume a function of
  backlog size, which is largest exactly during an incident.
- **D3 — Retirement is explicitly asserted by the feature, never inferred from
  time or idleness.** The existing accepted decision in the pack's operator
  document already names "an explicit feature-owned terminal-key contract" as
  its reopen condition, and this rule is that contract. A TTL would silently
  weaken a stated safety property for keys that merely went quiet. Reopen only
  with domain evidence that a class of keys is provably non-recurring.
- **D4 — The inbox is a separate accepted outcome, not part of this one.**
  Authority: this package's recorded "does not own" boundary and the extension
  seam that routes outbox and inbox persistence through separate accepted
  workflows.
- **D5 — Tenant identity stays caller-encoded.** See non-goals.
- **Constraint — the relay already has what R2 needs.** The relay bootstrap
  installs a tracer provider, and the W3C trace-context propagator is set
  globally by the telemetry package. R2 adds spans to an existing, configured
  pipeline rather than introducing tracing to the process.

## Success criteria and proof expectations

1. An event appended inside a sampled operation and later published produces
   telemetry in which the publication carries a resolvable reference to the
   producing operation's trace identity. Scope: one event, end to end, against a
   real PostgreSQL and an exported span recorder. Pass: the reference resolves to
   the producing trace. Fail: the publication carries no reference, or one to a
   different trace.

   Amended by design. The original wording required the publication and the
   producing operation to *share* one trace identity, which fixes a parent-child
   topology. Design chose a link instead, because an outbox publication can occur
   long after the append — after a backlog drain or an operator redrive — and
   parenting would hold the producing request's trace open for that whole
   horizon. Reference rather than shared identity is the weaker claim that both
   topologies could satisfy; it is stated here so the narrowing is visible rather
   than silently absorbed by design.
2. Correlation survives repetition. Scope: one event forced through a temporary
   failure and a lease recovery. Pass: every attempt reports the same creation
   context.
3. Appending without an active trace context, and publishing such an event,
   both succeed. Scope: unit level. Pass: no error and no dropped span.
4. Caller metadata is unaffected. Scope: a stored row's metadata bytes compared
   against what was passed, including a payload that itself contains a
   `traceparent` key. Pass: byte equality.
5. Retirement satisfies its state table, including the rejected case and the
   idempotent repeat, and a retired key accepts a restarted sequence. Scope:
   integration level against real PostgreSQL, because the precondition and the
   caller's transaction cannot be proven against a stubbed driver.
6. The retained-head count observable falls when a key is retired. Scope: the
   existing state observation.
7. Telemetry vocabularies remain closed with the new operation present. Scope:
   the existing bounded-vocabulary proof.
8. The documented append shape passes the repository's dependency lint in the
   layer the document places it in.
9. Every existing outbox proof still passes unchanged, including the ordering,
   lease-fencing, reconciliation, and drain suites.

Proof expectations name evidence boundaries, not mechanisms: contract-level
rules (R1, R4) need proof against real PostgreSQL because their authority is in
the database; correlation rules (R2, R3) need exported telemetry rather than
internal state; R5 needs the repository's own lint rather than a reading.

## Risks, assumptions, and reopen conditions

- **Assumption — the envelope size budget absorbs the new field.** Affected
  rule: R1. Safe boundary: a creation context is small and bounded relative to
  the existing whole-envelope budget. Invalidating evidence: a service already
  appending events at the envelope limit would see previously accepted events
  rejected. Reopen owner: this spec. Reopen condition: the budget must be
  restated if the new field is charged against the existing total rather than
  given its own allowance — design owns which, and must not let the choice
  change whether a previously valid event is now rejected without saying so.
- **Risk — span volume.** A full batch publishes up to the configured batch size
  per cycle, so a naive span per attempt makes trace volume track backlog.
  Mitigated by D2; residual exposure belongs to the deployment's sampler.
- **Risk — R4 hands a footgun to the caller.** Retiring a key that later recurs
  silently reopens sequence replay for it. Mitigated by making the assertion
  explicit, transactional, and precondition-checked; not mitigated against a
  caller that asserts terminality wrongly, which is stated as the trade rather
  than defended against.
- **Assumption — no derived service depends on the absence of relay spans.**
  Affected rule: R2. Safe boundary: the relay already builds a tracer provider
  and a sampler from the same configuration the service uses, so a deployment
  that wants fewer spans already owns the control. Invalidating evidence: a
  deployment whose trace budget is sized for zero relay spans and cannot express
  that through its sampler. Reopen owner: this spec. Reopen condition: the spans
  gain a configuration gate only on that evidence; adding one up front would
  ship a switch with no demonstrated user.

## Proof gaps

The [Review Independence](../../docs/spec-first-workflow/shared/review-independence.md)
trigger applies to this spec: it fixes a public envelope contract, a durable
schema change, and — in R4 — a deliberate, hard-to-reverse trade of a stated
safety property. An independent specification review was **not** run, because
subagent dispatch is disabled in the authoring session. The spec carries focused
root self-review only, which found and closed two defects: R4 had no concurrency
rule against a racing append for the same key, and the rejection outcome did not
say it must be distinguishable from a fault.

Next useful check, in order of value: an independent reviewer falsifying R4's
concurrency rule and its post-retirement sequence-replay trade, then R1's
envelope-budget assumption. Reopen owner: Specification.
