# Pivot, Compensation, And Forward Recovery

## Behavior Change Thesis
When loaded for rollback, cancellation, timeout, or irreversible-step symptoms, this file makes the model identify the pivot and choose semantic compensation or forward recovery instead of promising generic rollback or assuming a timeout means the side effect did not happen.

## When To Load
Load when a spec must identify the pivot transaction, define compensating actions, decide when compensation is impossible, or choose forward recovery and manual intervention paths.

## Decision Rubric
- Treat pre-pivot steps as compensable only when the compensation is semantic, idempotent, and guarded by current owner state.
- Name the pivot: the point after which the saga commits to forward completion instead of rollback.
- Use post-pivot forward recovery when a side effect cannot be safely undone, such as shipped goods or captured external payment.
- Use manual intervention only as a named waiting or terminal state with owner, alert, and operator-safe repair action.
- Reject the saga boundary when no valid compensation or forward recovery exists and partial execution is not acceptable.

## Imitate
- Before payment capture, inventory reservation can be released. After payment capture, shipment failures trigger retry, alternate fulfillment, or refund workflow by policy. Copy the pre-pivot vs post-pivot split.
- Compensation action is `ReleaseReservationIfPresent(order_id, reservation_id)`, not blind deletion of inventory records. Copy the state-guarded semantic undo.
- Payment authorization is idempotent by `payment_attempt_id`; capture is the pivot and has a documented refund or dispute recovery path. Copy the pivot plus forward path.
- Register compensation before or at the same durable point as the side effect it may need to undo. Copy the compensation-registration ordering.

## Reject
- "Rollback payment" without defining whether the operation was authorization, capture, transfer, refund, or void.
- Compensation deletes rows owned by another service.
- Post-pivot failure says "try later" with no retry budget, owner, terminal status, or manual repair runbook.
- Cancellation assumes the step did not run because the caller timed out.

## Agent Traps
- Using database rollback language for external side effects.
- Treating user cancellation as a universal rollback, regardless of pivot state.
- Compensating by mutating another service's storage instead of sending an owner command.
- Forgetting that compensation itself can fail and needs state, retries, and ownership.

## Validation Shape
- Step times out after the remote side effect succeeded: first reconcile by idempotency key, then choose compensate or forward-recover from observed owner state.
- Compensation fails: persist `COMPENSATION_FAILED`, alert the owner, and keep the compensation idempotent and retryable.
- User cancellation during a saga: apply a cancellation policy based on pivot state, not a blanket rollback.
- Downstream no longer accepts compensation because a later independent event occurred: switch to forward recovery or manual review with an auditable reason.
