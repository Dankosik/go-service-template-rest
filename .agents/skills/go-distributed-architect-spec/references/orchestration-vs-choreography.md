# Orchestration Vs Choreography

## When To Load
Load this when a spec needs to choose saga coordination style, decide if a workflow engine is warranted, prevent event cycles, or explain why a "just publish events" design is not enough.

## Option Comparisons
- Orchestration: one durable owner tracks the process, sends commands, handles replies, and owns timeout, retry, compensation, DLQ, and reconciliation policy. Choose it for multi-step or business-critical outcomes.
- Choreography: participants react to domain events and publish their own outcomes. Choose it when reactions are independent, the graph is small, and no central process outcome must be operated.
- Durable workflow engine: an orchestration implementation option when process state, timers, restart recovery, and long-running visibility are core needs. It is not a substitute for idempotent activities.
- Bounded hybrid: use only with a named handoff, for example an orchestrated checkout saga emits a terminal `OrderApproved` event that independent fulfillment listeners consume.

## Good Flow Examples
- Orchestrated checkout: Checkout starts `CreateOrderSaga`, creates a pending order, sends `ReserveInventory`, sends `AuthorizePayment`, and emits a terminal domain event after the saga reaches a terminal state.
- Choreographed notification: User emits `UserEmailChanged`; EmailPreferences and SearchIndex update independently. Neither consumer controls the other's outcome.
- Hybrid with a boundary: Payment orchestration completes capture and marks `PaymentCaptured`; unrelated analytics and email services consume that terminal event outside the payment saga.

## Bad Flow Examples
- Every participant publishes "next" events and no service owns the business outcome, timeout, or compensation.
- A choreography event graph forms a cycle where Service A's reaction triggers Service B, which republishes an event that triggers Service A again.
- An orchestrator calls synchronous HTTP endpoints and treats in-memory call stack state as the durable saga state.
- A flow mixes commands and domain events without naming which messages are instructions and which are facts.

## Failure-Mode Examples
- Orchestrator crash: the durable flow state resumes from the last committed step and either retries or times out deterministically.
- Participant reply lost: the orchestrator retries the command with the same idempotency key or reconciles by querying the participant's owner state.
- Choreography consumer failure: the producer's event remains a fact; consumer recovery uses outbox/inbox, replay, or reconciliation rather than requiring the producer to know every consumer.
- Event cycle: the spec adds causation IDs, terminal event boundaries, and explicit "do not react to self-caused event" rules.

## Exa Source Links
- [Microservices.io Saga pattern](https://microservices.io/patterns/data/saga.html)
- [Microservices.io orchestration-based saga article](https://microservices.io/post/sagas/2019/12/12/developing-sagas-part-4.html)
- [Dapr Workflow patterns](https://docs.dapr.io/developing-applications/building-blocks/workflow/workflow-patterns/)
- [Temporal Saga pattern article](https://temporal.io/blog/saga-pattern-made-easy)
