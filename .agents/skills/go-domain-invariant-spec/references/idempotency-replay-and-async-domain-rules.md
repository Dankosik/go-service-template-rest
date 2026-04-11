# Idempotency, Replay, And Async Domain Rules

## When To Load
Load this when a spec touches retries, duplicate requests, replay, out-of-order events, async commands, side effects, eventual consistency, process managers, or reconciliation.

Use task-local product/spec artifacts first. External idempotency and replay sources calibrate the questions to ask, not the final business policy.

## Domain Rule Pattern
Before choosing transport headers, tables, queues, or workflow engines, answer:

```text
Logical operation:
Sameness rule:
Idempotency key or business identity, at the domain level:
Allowed duplicate behavior:
Different-payload reuse behavior:
In-progress behavior:
Expired-window behavior:
Out-of-order behavior:
Replay mode behavior:
Side-effect policy:
Reconciliation or manual-intervention path:
Traceability and tests:
```

## Example Invariant Statements
- `OneDomainEffectPerIntent`: the same logical operation repeated with the same intent must produce one domain effect, not duplicate side effects.
- `DifferentIntentConflict`: the same idempotency identity reused for a different domain intent must reject or conflict rather than silently reuse a prior outcome.
- `ReplayIsolation`: replay of historical events must not trigger irreversible production side effects unless an explicit domain replay policy allows it.
- `ConsumerIdempotency`: every async consumer that performs state-changing domain work must define duplicate handling in domain terms, not only in broker mechanics.
- `OutOfOrderStateGuard`: an event that arrives after the aggregate has moved past the relevant state must no-op, reject, compensate, or route to reconciliation explicitly.

## Good And Bad State Transition Specs
Good idempotency transition spec:

```text
Lifecycle: command_idempotency
States: new, in_progress, completed, conflict, expired, ambiguous_commit, manual_review
Initial state: new
Allowed transitions:
- new -> in_progress when the service accepts the first logical command
- in_progress -> completed when the domain effect commits
- in_progress -> ambiguous_commit when the dependency outcome is unknown
- completed -> completed on same key + same intent, returning equivalent outcome
- completed -> conflict on same key + different intent
- ambiguous_commit -> completed after reconciliation proves the effect committed
- ambiguous_commit -> manual_review when automated proof is unavailable
Forbidden transitions:
- completed -> in_progress for the same logical command
- conflict -> completed without a new domain decision
Violation outcome: reject conflicting intent; do not execute duplicate side effects
```

Bad idempotency transition spec:

```text
Use an idempotency key and dedupe messages.
```

Why it fails: it chooses mechanics without defining sameness, conflict, in-progress, expiration, replay, or side-effect behavior.

Good replay transition spec:

```text
State: live_processing
Trigger: operator requests event replay
Allowed transition: live_processing -> replay_sandbox
Preconditions:
- replay purpose is projection rebuild, decision-model validation, or migration
- side effects are classified as recomputable, sandbox-only, forbidden, or policy-controlled
- reconciliation target is defined
Forbidden transition: live_processing -> direct_production_replay for events with irreversible side effects
Violation outcome: reject direct replay and require replay isolation or accepted manual process
```

## Edge-Case Prompts
- What counts as the same logical command: caller key, business key, payload hash, event ID, or aggregate checkpoint?
- What happens when two duplicates arrive concurrently?
- What response/outcome is replayed after the first attempt committed but the response was lost?
- What if the key is reused with a different tenant, actor, payload, or amount?
- How long does duplicate suppression need to match business risk and retry windows?
- What if an event is redelivered after the aggregate passed the original target state?
- What side effects are forbidden during replay?
- Does reconciliation create corrective domain actions, or does it patch state invisibly?

## Downstream Handoff Notes
- API handoff: after domain sameness and conflict behavior are stable, define external idempotency contract and response replay semantics.
- Data handoff: choose durable uniqueness, checkpoint, inbox/outbox, or state storage only after the domain identity and effect boundary are clear.
- Reliability handoff: retry policy must preserve domain idempotency; retries without duplicate rules can amplify damage.
- Distributed handoff: cross-service process invariants need idempotent consumers, durable deduplication, and reconciliation.
- QA handoff: include tests for same key/same intent, same key/different intent, concurrent duplicates, expired replay window, in-progress duplicate, out-of-order event, and replay side-effect suppression.

## Exa Source Links
- [EventSourcingDB: Building Event Handlers](https://docs.eventsourcingdb.io/best-practices/building-event-handlers/) for handler progress tracking, duplicate delivery, idempotency, and replay responsibility.
- [NILUS: Command Idempotency Keys in Microservices](https://www.nilus.be/blog/command_idempotency_keys_in_microservices/) for bounded-context sameness and response replay concerns.
- [NILUS: Message Deduplication Patterns](https://www.nilus.be/blog/message_deduplication_patterns_in_event-driven_systems/) for business-key dedupe, aggregate-level idempotency, and reconciliation patterns.
- [NILUS: Event Replay Isolation](https://www.nilus.be/blog/event_replay_isolation_in_event-sourced_systems/) for replay sandboxing, side-effect policy, semantic translation, and reconciliation.
- [Spec Coding: Designing Idempotent Workflows with Specs](https://spec-coding.dev/blog/designing-idempotent-workflows-with-specs) for testable retry-safe acceptance criteria.
