# Orchestration Vs Choreography

## Behavior Change Thesis
When loaded for a "just publish events" or coordination-style symptom, this file makes the model choose an owned durable process or a bounded terminal-event handoff instead of an unowned event chain with no timeout, compensation, or outcome authority.

## When To Load
Load when a spec must choose orchestration, choreography, a workflow engine, or a bounded hybrid, especially when event cycles or ownerless outcomes are plausible.

## Decision Rubric
- Choose orchestration when one business outcome spans multiple steps and someone must own timeout, retry, compensation, DLQ, and reconciliation policy.
- Choose choreography only when reactions are independent, the event graph is small, and no central process outcome must be operated.
- Treat a workflow engine as an orchestration implementation for durable timers, restart recovery, and visibility; it does not remove activity idempotency requirements.
- Use a hybrid only at a named boundary, such as an orchestrated saga emitting a terminal domain event for independent listeners.
- Classify messages as commands or facts. A command instructs an owner; a domain event reports an already-owned state transition.

## Imitate
- Orchestrated checkout: Checkout starts `CreateOrderSaga`, creates a pending order, sends `ReserveInventory`, sends `AuthorizePayment`, and emits a terminal event after the saga reaches a terminal state. Copy the single process owner.
- Choreographed notification: User emits `UserEmailChanged`; EmailPreferences and SearchIndex update independently. Copy the lack of central outcome dependency.
- Hybrid with a boundary: Payment orchestration completes capture and marks `PaymentCaptured`; analytics and email services consume that terminal event outside the payment saga. Copy the named handoff.

## Reject
- Every participant publishes "next" events and no service owns the business outcome, timeout, or compensation.
- A choreography event graph forms a cycle where Service A's reaction triggers Service B, which republishes an event that triggers Service A again.
- An orchestrator calls synchronous HTTP endpoints and treats in-memory call stack state as the durable saga state.
- A flow mixes commands and domain events without naming which messages are instructions and which are facts.

## Agent Traps
- Picking choreography because it feels decoupled while hiding the business owner of failure and timeout.
- Treating a workflow engine as a magic exactly-once boundary for non-idempotent activities.
- Calling a design "hybrid" without naming where orchestration ends and independent reaction begins.
- Using domain events to smuggle imperative commands without owner accountability.

## Validation Shape
- Orchestrator crash: the durable flow state resumes from the last committed step and either retries or times out deterministically.
- Participant reply lost: the orchestrator retries the command with the same idempotency key or reconciles by querying the participant's owner state.
- Choreography consumer failure: the producer's event remains a fact; consumer recovery uses outbox/inbox, replay, or reconciliation rather than requiring the producer to know every consumer.
- Event cycle: the spec adds causation IDs, terminal event boundaries, and explicit "do not react to self-caused event" rules.
