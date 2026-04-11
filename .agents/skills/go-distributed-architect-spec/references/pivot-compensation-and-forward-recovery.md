# Pivot, Compensation, And Forward Recovery

## When To Load
Load this when a spec needs to identify the pivot transaction, define compensating actions, decide when compensation is impossible, or choose forward recovery and manual intervention paths.

## Option Comparisons
- Pre-pivot compensable step: choose when a later business failure may require undoing the step. The compensation must be semantic, idempotent, and guarded by current state.
- Pivot step: the point after which the saga should commit to forward completion instead of rollback. It should be explicit and easy to operate.
- Post-pivot forward recovery: choose when the side effect cannot be safely undone, for example a shipped package or captured external payment.
- Manual intervention: acceptable only as a named terminal or waiting state with alerting, owner, and operator-safe repair actions.
- Reject local transaction: choose when no valid compensation or forward recovery exists and partial execution is not acceptable.

## Good Flow Examples
- Before payment capture, inventory reservation can be released. After payment capture, shipment failures trigger forward recovery through retry, alternate fulfillment, or refund workflow by policy.
- Compensation action is `ReleaseReservationIfPresent(order_id, reservation_id)`, not blind deletion of inventory records.
- Payment authorization is idempotent by `payment_attempt_id`; capture is the pivot and has a documented refund or dispute recovery path.
- A compensation is registered before or at the same durable point as the side effect it may need to undo.

## Bad Flow Examples
- "Rollback payment" without defining whether the operation was authorization, capture, transfer, refund, or void.
- Compensation deletes rows owned by another service.
- Post-pivot failure says "try later" with no retry budget, owner, terminal status, or manual repair runbook.
- Cancellation assumes the step did not run because the caller timed out.

## Failure-Mode Examples
- Step times out after the remote side effect succeeded: first reconcile by idempotency key, then choose compensate or forward-recover from observed owner state.
- Compensation fails: persist `COMPENSATION_FAILED`, alert the owner, and keep the compensation idempotent and retryable.
- User cancellation during a saga: apply a cancellation policy based on pivot state, not a blanket rollback.
- Downstream no longer accepts compensation because a later independent event occurred: switch to forward recovery or manual review with an auditable reason.

## Exa Source Links
- [Microservices.io Saga pattern](https://microservices.io/patterns/data/saga.html)
- [Microservices.io saga consistency overview](https://microservices.io/post/microservices/2019/07/09/developing-sagas-part-1.html)
- [Temporal compensating actions article](https://temporal.io/blog/compensating-actions-part-of-a-complete-breakfast-with-sagas)
- [Temporal Saga pattern article](https://temporal.io/blog/saga-pattern-made-easy)
- [Dapr Workflow patterns](https://docs.dapr.io/developing-applications/building-blocks/workflow/workflow-patterns/)
