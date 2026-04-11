# Sync Async Workflow Ownership

## Behavior Change Thesis
When loaded for request-path vs async workflow decisions, this file makes the model name the process owner, pivot, durable state, and completion contract instead of choosing a queue, saga, cron, or workflow engine as the architecture.

## When To Load
Load when a task asks whether work belongs on the request path, in a queue, in a saga, behind a process manager, through choreography, or in a workflow engine.

## Decision Rubric
- Keep hard acceptance invariants inside one local transaction boundary when possible.
- Use synchronous calls only when the caller needs an immediate answer or command finality and the deadline, retry, idempotency, and fail-closed behavior are explicit.
- Use events for independent reactions that cannot decide whether the source fact exists.
- Shape events as domain facts with enough context for consumers; reject table, CRUD, or generic change events that make consumers infer intent from private schema deltas.
- Use commands or queues for owned work distribution where one owner accepts responsibility.
- Use orchestration when one owner must track timers, retries, approvals, cancellation, ambiguous outcomes, or operator repair.
- Use choreography only when independent reactions do not need a central status, cancellation, or repair owner.
- Consider a workflow engine or durable-execution platform when internal state machine/process manager limits are material: long timers, human tasks, replay/debug needs, fleet-wide operations, cross-owner orchestration, or ad hoc retry/state persistence becoming the architecture.

## Imitate

### Refund With Partner Reversal
Context: refunds require partner reversal calls, 24-hour retry timers, finance approval over a threshold, manual repair for ambiguous partner outcomes, and an auditable requester-visible status.

Choose: make `payments` the process owner and model refunds as a durable state machine, process manager, or workflow-engine-backed orchestration depending on platform fit. Return a durable operation/status resource after starting the flow.

Copy: this chooses durable ownership and client-visible progress before selecting implementation machinery.

### Independent Order Reactions
Context: order placement should notify analytics, send marketing email, and update a recommendation index. None of those consumers can decide whether the order exists.

Choose: publish an event from the owner transaction using an outbox or equivalent atomic linkage. Let consumers handle duplicates, retries, poison messages, and replay under their own ownership.

Copy: this avoids over-orchestrating unrelated side effects.

### Checkout Acceptance Before Pivot
Context: checkout must ensure a credit or inventory invariant before accepting the order.

Choose: keep the hard acceptance invariant in one local transaction where possible. If the invariant spans owners, use reservation or pending semantics only when eventual consistency and compensation are accepted.

Copy: this avoids remote calls after a non-compensable pivot without recovery.

## Reject
- "Put it in Kafka because we have Kafka." Bad because transport does not define ownership, state, retry, or repair.
- "Cron plus status flags is enough." Bad when timers, approvals, ambiguous outcomes, and manual repair need one durable owner.
- "Use pure choreography for a user-visible long-running process." Bad when cancellation, status, repair, and retry authority must be centralized.
- "Hold the HTTP request open until every side effect completes." Bad when only acceptance needs to be synchronous.
- "Add distributed locks to preserve a hard cross-owner invariant." Bad because it usually signals the ownership boundary is wrong.

## Agent Traps
- Do not call every multi-step flow a saga. Name the pivot and whether each step is compensable or forward-recoverable.
- Do not publish events without atomic linkage when the DB state and emitted fact must agree.
- Do not leave DLQ ownership implicit. A dead letter is a business and operations queue, not a trash can.
- Do not treat a workflow engine as a queue with timers; call out determinism, versioning, worker ownership, and replay constraints when durable execution is the recommendation.
- Do not introduce or reject a workflow engine by habit; check in-flight migration, replay/debug, operator repair, rollback, and operational ownership consequences.
