# Idempotency, Replay, And Async Domain Rules

## Behavior Change Thesis
When loaded for symptom "the spec touches retries, duplicates, replay, async commands, or out-of-order events", this file makes the model define domain sameness, effect boundaries, and replay policy before mechanics instead of likely mistake "say use an idempotency key, queue dedupe, or retries without saying what counts as the same operation."

## When To Load
Load this when a spec touches retries, duplicate requests, replay, out-of-order events, async commands, side effects, eventual consistency, process managers, consumer idempotency, or reconciliation.

## Decision Rubric
- Start with the logical operation and one-domain-effect boundary, not the header, table, queue, or workflow engine.
- Define the sameness rule: caller key, business key, request fingerprint or payload hash, event ID, aggregate version/checkpoint, tenant/actor scope, or a deliberate combination.
- Do not collapse HTTP request equality into domain sameness; define which fields carry business intent and which are incidental metadata.
- Specify same identity plus same intent, same identity plus different intent, in-progress duplicate, expired-window duplicate, concurrent duplicate, and unknown-commit duplicate behavior when relevant.
- Define out-of-order behavior in domain terms: no-op, reject, compensate, forward-recover, or reconcile.
- Classify replay side effects as recomputable, sandbox-only, forbidden, or policy-controlled before allowing replay.
- State whether reconciliation creates corrective domain actions or only repairs derived state; do not patch source-of-truth state invisibly.

## Imitate
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

Copy the shape: sameness, in-progress, conflict, ambiguous commit, and duplicate side-effect prevention.

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

Copy the side-effect policy: replay is not just "rerun handlers."

## Reject
```text
Use an idempotency key and dedupe messages.
```

Failure: chooses mechanics without defining sameness, conflict, in-progress behavior, expiration, replay, or side-effect policy.

```text
The consumer is idempotent because it checks whether the message ID was seen.
```

Failure: message-ID dedupe may not protect the business effect if the same logical operation arrives under another message or tenant/actor scope.

## Agent Traps
- Do not let transport idempotency keys define domain identity by accident; scope and sameness are domain decisions.
- Do not treat concurrent duplicates as the same as completed duplicates when an in-progress effect can still be ambiguous.
- Do not replay production events through irreversible side-effect handlers unless the domain policy explicitly allows it.
- Do not assume out-of-order means "ignore"; it may require conflict, compensation, or reconciliation.
- Do not design retry policy before duplicate and unknown-commit outcomes are explicit.

## Validation Shape
Proof should cover same identity plus same intent, same identity plus different intent, concurrent or in-progress duplicate, ambiguous commit or lost response, out-of-order event, replay side-effect suppression, and the chosen reconciliation path when those cases are in scope.
