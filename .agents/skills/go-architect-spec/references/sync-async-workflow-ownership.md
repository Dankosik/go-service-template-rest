# Sync Async Workflow Ownership

## When To Load
Load this when a task asks whether work belongs on the request path, in a queue, in a saga, behind a process manager, through choreography, or in a workflow engine.

Use it to name the process owner, pivot transaction, command/event distinction, state machine, retry and timer ownership, manual repair owner, and client-visible completion model. Do not choose a broker or workflow engine because it is available.

## Decision Examples

### Example 1: Merchant refunds with partner reversals
Context: Refunds require partner reversal calls, 24-hour retry timers, finance approval over $10k, manual repair for ambiguous partner outcomes, and an auditable requester-visible status. One engineer wants a workflow engine immediately; another wants cron jobs plus status flags in `payments`.

Selected option: Make `payments` the process owner and model refunds as a durable state machine or process manager first. Use orchestration because one owner must track timers, approvals, retries, ambiguous outcomes, cancellation, and manual repair. Return a durable status resource/operation after starting the flow instead of holding the request open until the partner reversal is final.

Rejected options:
- Plain cron plus ad hoc flags, because it hides process state and makes stuck detection and repair ownership ambiguous.
- Immediate workflow-engine adoption without criteria, because the architecture decision is durable workflow ownership, not tool selection.
- Pure choreography, because independent events do not provide one authoritative owner for timers, approvals, and repair.

Evidence that would change the decision:
- Workflow volume, human approvals, long timers, replay/debug needs, or cross-owner orchestration exceed what an internal state machine can operate safely.
- A platform workflow engine already exists with proven operational support and migration path.
- The flow becomes a simple fire-and-forget domain reaction with no central status, repair, or cancellation requirement.
- A hard invariant must remain inside one local transaction, pushing more of the flow back into a single owner.

Failure modes and rollback implications:
- Ambiguous partner outcomes can double-refund unless partner calls are idempotent and reconciled.
- A workflow engine migration can strand in-flight executions; define drain, replay, and operator repair before switching engines.
- Rolling back after the pivot may require forward recovery, not compensation; name the pivot and what rollback cannot undo.

### Example 2: Independent downstream reactions
Context: Order placement should notify analytics, send marketing email, and update a recommendation index. None of these reactions can decide whether the order exists.

Selected option: Publish events from the owning order transaction using an outbox or equivalent atomic linkage. Let independent consumers react asynchronously with idempotency, retry, and dead-letter ownership.

Rejected options:
- A central saga orchestrator for unrelated side effects.
- Synchronous calls from order creation to every downstream consumer.
- Publishing events without an outbox or equivalent when the order write and event emission must agree.

Evidence that would change the decision:
- A downstream step becomes part of the business decision to accept the order.
- A reaction requires ordered process state and user-visible repair owned by one team.
- A consumer cannot tolerate event lag and must be on a correctness-critical path.

Failure modes and rollback implications:
- Duplicate or reordered events can corrupt derived consumers unless handlers are idempotent and ordering assumptions are explicit.
- A poison message can block a queue; define DLQ ownership and replay rules.
- If event emission is broken, rollback may be a relay disable plus outbox replay, not a data rollback.

### Example 3: Synchronous decision before a pivot
Context: Checkout must ensure an order does not exceed a local credit or inventory invariant before the order is accepted.

Selected option: Keep hard acceptance invariants in one local transaction boundary when possible. If the invariant spans owners, use a saga only when eventual consistency and compensation are acceptable; otherwise revisit the ownership boundary.

Rejected options:
- Remote calls after a non-compensable pivot without reconciliation.
- Stretching the HTTP request until every downstream side effect completes.
- Distributed locks as the primary correctness mechanism.

Evidence that would change the decision:
- The operation can be expressed as a reservation or pending state with explicit timeout and compensation.
- Business accepts eventual consistency and exposes a pending status instead of immediate finality.
- The invariant is not actually hard and can become a derived read or notification concern.

Failure modes and rollback implications:
- Remote calls on the critical path consume deadline and failure budget; define timeout and fail-closed behavior.
- Compensation may not be possible after money movement, partner submission, or legal notification.
- If the ownership boundary is wrong, rollback should collapse the flow back into one owner rather than adding more distributed coordination.

## Source Links Gathered Through Exa
- Microservices.io, "Saga": https://microservices.io/patterns/data/saga
- Microservices.io, "Transactional outbox": https://microservices.io/patterns/data/transactional-outbox.html
- AWS Prescriptive Guidance, "Saga choreography pattern": https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/saga-choreography.html
- AWS Compute Blog, "Building a serverless distributed application using a saga orchestration pattern": https://aws.amazon.com/blogs/compute/building-a-serverless-distributed-application-using-a-saga-orchestration-pattern/
- AWS Prescriptive Guidance, "Transactional outbox pattern": https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html
- Azure Architecture Center, "CQRS pattern": https://learn.microsoft.com/en-us/azure/architecture/patterns/cqrs

