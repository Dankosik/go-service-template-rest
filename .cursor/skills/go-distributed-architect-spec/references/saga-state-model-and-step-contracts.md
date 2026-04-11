# Saga State Model And Step Contracts

## Behavior Change Thesis
When loaded for saga identity, step-contract, or stuck-flow symptoms, this file makes the model define a resumable monotonic state machine with one-active-flow rules instead of tracking "current step" in memory or retrying forever without terminal states.

## When To Load
Load when a flow needs durable saga state, a one-active-flow rule, state transitions, step contracts, timeout policy, stuck-flow handling, or workflow-engine mapping.

## Decision Rubric
- Use no saga when one owner transaction or one idempotent command to an owner is enough.
- Use a hand-rolled state machine only when the flow is small and the team can own persistence, timers, retries, and operator tooling.
- Use a saga orchestration framework or workflow engine when long-running timers, restart recovery, or many recovery paths dominate.
- Allow choreographed state only when each participant's local state is sufficient and no central outcome state is needed.
- Require stable `saga_id`, business idempotency key, expected-state/version checks, timeout class, and operator-visible terminal states.

## Imitate
- State machine has `STARTED`, `INVENTORY_RESERVED`, `PAYMENT_AUTHORIZED`, `COMPLETED`, `COMPENSATING`, `COMPENSATED`, and `FAILED_MANUAL_REVIEW`, with version-checked transitions. Copy the monotonic states and terminal manual-review state.
- `saga_id` is stable, `business_key` is unique for active instances, and each participant command includes the same idempotency key and causation ID. Copy the active-flow dedup rule.
- Each step contract states trigger, local transaction scope, command or event type, timeout, retry class, dedup boundary, success transition, failure transition, compensation, and reconciliation owner. Copy the contract completeness.
- A Dapr or Temporal activity assumes at-least-once execution and uses application-level idempotency keys for side effects. Copy the framework skepticism.

## Reject
- The orchestrator only stores "current step" in memory and cannot resume after restart.
- The spec says "retry until success" with no timeout, poison-message behavior, or operator-visible terminal state.
- Step responses update state without checking expected prior state or flow version.
- Two checkout requests for the same cart create two independent active sagas with no dedup or merge rule.

## Agent Traps
- Writing a state list without transition guards or allowed prior states.
- Naming timeouts but not saying whether timeout triggers retry, compensate, reconcile, or manual review.
- Forgetting one-active-flow rules for duplicate starts on the same business key.
- Treating workflow-code deployment as safe without considering durable in-flight instances and deterministic replay/versioning constraints.

## Validation Shape
- Duplicate start request: lookup by business idempotency key and return the existing flow outcome or operation resource.
- Timeout after side effect: reconcile with the participant before compensating; do not assume the side effect failed just because the reply did not arrive.
- Step succeeds but state transition fails locally: retry transition from the persisted outbox/inbox record or mark the saga for reconciliation.
- Workflow code changes while instances are running: keep deterministic replay and versioning constraints explicit before rollout.
