# Saga State Model And Step Contracts

## When To Load
Load this when a flow needs a durable state machine, one-active-flow rule, state transition table, step contract, timeout policy, or workflow-engine mapping.

## Option Comparisons
- No saga: use when a single owner transaction or an idempotent command to one owner is enough.
- Hand-rolled state machine: useful when the flow is small and the team can own persistence, timers, retries, and operator tooling.
- Saga orchestration framework or workflow engine: useful when the flow is long-running, uses durable timers, needs restart recovery, or has many compensating/forward-recovery paths.
- Choreographed state: acceptable only when each participant can define its own local state and no central outcome state is needed.

## Good Flow Examples
- State machine has `STARTED`, `INVENTORY_RESERVED`, `PAYMENT_AUTHORIZED`, `COMPLETED`, `COMPENSATING`, `COMPENSATED`, and `FAILED_MANUAL_REVIEW` states, with version-checked transitions.
- `saga_id` is stable, `business_key` is unique for active instances, and each participant command includes the same idempotency key and causation ID.
- Each step contract states: trigger, local transaction scope, command/event type, timeout, retry class, dedup boundary, success transition, failure transition, compensation, and reconciliation owner.
- A Dapr or Temporal activity assumes at-least-once execution and uses application-level idempotency keys for side effects.

## Bad Flow Examples
- The orchestrator only stores "current step" in memory and cannot resume after restart.
- The spec says "retry until success" with no timeout, poison-message behavior, or operator-visible terminal state.
- Step responses update state without checking expected prior state or flow version.
- Two checkout requests for the same cart create two independent active sagas with no dedup or merge rule.

## Failure-Mode Examples
- Duplicate start request: lookup by business idempotency key and return the existing flow outcome or operation resource.
- Timeout after side effect: reconcile with the participant before compensating; do not assume the side effect failed just because the reply did not arrive.
- Step succeeds but state transition fails locally: retry transition from the persisted outbox/inbox record or mark the saga for reconciliation.
- Workflow code changes while instances are running: keep deterministic replay and versioning constraints explicit before rollout.

## Exa Source Links
- [Microservices.io saga consistency overview](https://microservices.io/post/microservices/2019/07/09/developing-sagas-part-1.html)
- [Microservices.io orchestration-based saga article](https://microservices.io/post/sagas/2019/12/12/developing-sagas-part-4.html)
- [Dapr Workflow features and concepts](https://docs.dapr.io/developing-applications/building-blocks/workflow/workflow-features-concepts)
- [Temporal Saga pattern article](https://temporal.io/blog/saga-pattern-made-easy)
