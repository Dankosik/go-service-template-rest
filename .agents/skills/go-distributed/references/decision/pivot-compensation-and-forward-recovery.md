# Pivot, Compensation, And Forward Recovery

## Behavior Change Thesis

When loaded for rollback, cancellation, timeout, or irreversible-step symptoms,
this file makes the model name the pivot and bound every compensation by the
lifetime of the thing it undoes, instead of promising generic rollback or
scheduling a compensation whose retry budget outlives the hold it releases.

## When To Load

Load when a flow must undo work another owner already performed: pivot
identification, compensating actions, cancellation policy, or a step whose
outcome is neither proven applied nor proven unapplied. Saga process state,
orchestration ownership, and reconciliation belong to
[`distributed-system-design`](../../../../../docs/universal-disciplines/distributed-system-design/references/data-and-coordination.md);
delivery, acknowledgement, and replay belong to
[`reliable-messaging`](../../../../../docs/universal-disciplines/reliable-messaging/SKILL.md).
This file owns only the undo decision.

## Decision Rubric

- Name the pivot: the step after which the flow can no longer roll back and
  every later failure resolves forward. Steps before it may compensate; steps
  after it need a forward path with an owner and a terminal state.
- Derive each compensation's deadline from the lifetime of what it undoes, and
  state the disposition when that deadline passes. A hold, authorization, seat,
  or lease that expires on its own schedule turns a later compensation into
  either a vacuous success — the flow records the invariant as restored when it
  was not — or a double release against a counter the expiry already credited.
- Address the compensation to the owner's current state by the same operation
  identity as the step it reverses. It is a new business action, so it carries
  its own failure, retry bound, and terminal state.
- Give a forward path an inbound trigger, not only a transition. A post-pivot
  failure the owner never hears about leaves its state machine asserting an
  outcome that did not happen, so name the callback, event, or reconciliation
  read that delivers it and the durable marker that makes it apply once —
  [`reliable-messaging`](../../../../../docs/universal-disciplines/reliable-messaging/SKILL.md)
  owns that marker's mechanism.
- An unknown outcome authorizes reading the owner's state, not compensating:
  compensating a step that never applied is an unrelated second mutation.
  [`resilience-and-load`](../../../../../docs/universal-disciplines/distributed-system-design/references/resilience-and-load.md)
  owns resolving that ambiguity before creating new intent.

## Imitate

- A release refuses once the hold's own expiry has passed and escalates, rather
  than crediting back units the expiry already returned.
- A cancellation consults the pivot rather than the caller's intent: pre-pivot
  it compensates, post-pivot it opens a refund or return flow.
- "Reverse the payment" names the operation it reverses — void, refund, or
  chargeback — because post-capture there is no rollback, only a later
  transaction with its own failure mode.

## Validation Shape

- The reply is lost after the remote effect committed: read owner state by
  operation identity, and compensate only from an observed applied state.
- The compensation deadline passes while the flow is still compensating: the
  flow reaches a named terminal state with an owner instead of retrying against
  a window that can no longer accept it.
- Two cancellations arrive for one flow: the second observes the first's
  terminal state and applies nothing.
- The post-pivot failure notice is delivered twice: the flow transitions once,
  and the second delivery is a no-op rather than a second correction.
