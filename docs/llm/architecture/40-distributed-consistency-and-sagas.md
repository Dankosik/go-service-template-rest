# Distributed consistency and saga instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing cross-service multi-step business workflows
  - Defining data consistency guarantees between services
  - Choosing saga style (orchestration vs choreography) and compensation model
  - Defining idempotency keys, deduplication, reconciliation, and read models
  - Reviewing race-condition risks and hidden cross-service invariants
- Do not load when: The change is fully local inside one service and one datastore transaction boundary

## Purpose
- This document defines repository defaults for distributed data consistency in service-to-service workflows.
- The goal is predictable eventual consistency with explicit failure handling and no hidden atomicity assumptions.
- The defaults below are mandatory unless an ADR explicitly approves an exception.

## Consistency model and transaction boundaries

### Local transaction boundary (hard rule)
- One local transaction boundary equals one service-owned datastore.
- Cross-service business flow is a sequence of local transactions, not one global ACID transaction.
- Never assume atomic commit across multiple services/datastores.
- Every cross-service step MUST persist local state before emitting next-step intent (event/command) through outbox or equivalent atomic linkage.

### Eventual consistency contract (required per flow)
For every multi-step flow, define all points before implementation:
- Source of truth owner for each critical entity.
- Which invariants are:
  - `local_hard_invariant` (must hold at commit time in one service)
  - `cross_service_process_invariant` (can converge with delay)
- Acceptable inconsistency window (`max_staleness`) for user-visible state.
- Failure outcome policy (retry, compensate, manual intervention).
- Reconciliation ownership and schedule.

### Decision rule: keep atomic vs split flow
Keep behavior inside one service/module when any point is true:
- The invariant must be checked and committed atomically for most requests.
- Compensation is legally or financially unacceptable.
- User flow cannot tolerate intermediate states.

Use cross-service saga when all points are true:
- Ownership boundaries are clear per entity.
- Eventual consistency is acceptable for at least one step.
- Compensating action or forward-recovery path is explicitly defined.

## Workflow design protocol for LLMs
Follow this order. Do not skip steps.

### 1) Build invariant register first
- List each invariant as one line: `name`, `owner_service`, `type`, `enforcement_point`.
- Reject designs with unnamed or ownerless invariants.
- Never hide invariant enforcement in consumers without declaring ownership.

### 2) Model explicit state machine
- Define workflow states and transitions (`pending`, `in_progress`, `completed`, `compensating`, `failed`, `cancelled`).
- Persist saga/workflow state durably.
- Transitions MUST be monotonic and version-checked (optimistic concurrency).
- Disallow implicit transitions from log side effects alone.

### 3) Define step contract for every step
Each step MUST specify:
- Trigger (`command` or `event`)
- Local transaction scope
- Idempotency key source and dedup boundary
- Timeout and retry class
- Success event and next step
- Compensation action (or explicit `no_compensation` with reason)

### 4) Mark pivot transaction
- Steps before pivot MUST be compensable.
- Pivot is the point after which rollback is no longer primary strategy.
- Steps after pivot MUST be retryable, idempotent, and eventually complete via forward recovery.

### 5) Define race controls explicitly
- One active workflow instance per business key (`tenant + aggregate_id + flow_type`) by unique constraint or equivalent.
- Use compare-and-swap/version checks on step transitions.
- Use deterministic partition/order key when ordering by aggregate is required.
- Never make write decisions from stale projection/read-model data.

### 6) Define reconciliation and observability before rollout
- Reconciliation job spec is mandatory for critical eventual-consistency flows.
- Emit metrics/logs for lag, retries, compensation, and invariant violations.
- No rollout without failure-path visibility.

## Orchestration vs choreography

### Default selection
- Default: orchestration for business-critical or multi-step workflows.
- Use choreography for simple domain reactions without centralized process state.

### Choose orchestration when any point is true
- More than two services participate in one business outcome.
- You need strict control over timeouts/retries/compensation order.
- You need auditable workflow state and operator-friendly debugging.
- You need one owner for operational policy and SLO.

### Choose choreography only when all points are true
- Flow is simple and loosely coupled (no central business transaction outcome).
- Each consumer can act independently with no global step ordering requirement.
- Event cycles and duplicate reactions are explicitly prevented.
- Operational ownership for retry/DLQ/reconciliation is clear per consumer.

### Hard boundaries
- Do not mix orchestration and choreography in one flow without explicit boundary and owner handoff.
- Do not use event bus as hidden synchronous RPC.

## Compensation policy

### Compensation design rules
- Compensation MUST be a semantic inverse action, not an ad-hoc delete/update.
- Compensation MUST be idempotent.
- Compensation MUST include precondition checks to avoid over-compensating already corrected state.
- Compensation ordering defaults to reverse order of completed compensable steps.

### When compensation is not possible
- Mark step as non-compensable explicitly.
- Require pivot placement before this step.
- Define forward-recovery path (retry until success, manual queue, or alternative completion).
- Add operator runbook entry for unresolved cases.

### Timeout and stuck-flow policy
- Every step MUST have explicit timeout.
- Expired step transitions to `failed` or `compensating` deterministically.
- Never leave workflow in implicit "unknown" state without watchdog/escalation.

## Idempotency and deduplication defaults

### External command idempotency
- Retry-unsafe external commands MUST require idempotency key.
- Default key scope: `tenant_or_account + operation + route_or_method + client_request_id`.
- Default idempotency retention TTL: 24 hours.
- Same key + same payload returns equivalent result.
- Same key + different payload returns conflict (`409` / `ABORTED`).

### Async consumer deduplication
- Consumer dedup store is mandatory for side-effecting handlers.
- Default dedup key:
  - CloudEvents: `source + id`
  - Non-CloudEvents: `producer_service + message_id`
- Storage default: unique constraint on `(consumer_group, dedup_key)`.
- Dedup retention default: minimum 7 days or longer than replay window.

### Commit ordering
- Persist side effects and dedup marker before ack/offset commit.
- Never acknowledge message delivery before durable state change.

## Reconciliation jobs and read models

### Read model rules
- Read model is for query optimization, not authoritative write validation.
- Every projection MUST expose freshness signal (`updated_at` or lag metric).
- If freshness exceeds budget, write path MUST query owner service or fail fast.

### Default staleness budgets
- Critical financial/inventory status: target <= 10s, hard cap <= 60s.
- Standard user-facing status: target <= 60s, hard cap <= 15m.
- Any stricter/looser SLO requires explicit ADR.

### Reconciliation defaults
- Critical flows: incremental reconciliation at least every 5 minutes.
- Non-critical flows: incremental reconciliation at least every 1 hour.
- Full-scope reconciliation: at least daily.
- Reconciliation MUST be idempotent, resumable, and watermark-based.
- Reconciliation should produce repair commands/events, not direct cross-service table writes.

## Why template avoids 2PC and dual writes

### 2PC default prohibition
- 2PC introduces blocking behavior under coordinator/participant failure and tightly couples availability across services.
- 2PC requires external transaction-manager discipline and operational complexity that does not match template defaults.
- Cross-service consistency default is local transactions + explicit process coordination.

### Dual write default prohibition
- Writing business state and publishing message separately creates unavoidable split-brain failure windows.
- Dual write paths are review blockers for business-critical flows.

### Approved replacement patterns
- Transactional outbox on producer side.
- Idempotent consumer with inbox/dedup store.
- Saga with explicit compensation/forward recovery.
- Reconciliation job to repair missed, duplicated, or out-of-order processing.

## Race-condition and hidden-invariant prevention rules
- Do not allow concurrent workflows mutating the same aggregate without serialization key.
- Do not issue parallel compensations for one saga instance.
- Do not model "check in service A, commit in service B" as atomic unless one owner enforces both.
- Do not gate write commands on stale read model snapshots.
- Do not rely on distributed locks as primary consistency control.
- If technical lock is unavoidable, require fencing token semantics and explicit failure-mode analysis.

## Anti-patterns
Treat each item as a review blocker unless an ADR explicitly accepts the risk.

- 2PC as default cross-service consistency mechanism
- Dual writes (`db commit` + direct publish/call) without outbox-equivalent atomic linkage
- Workflow defined only by events with no explicit state machine
- Missing compensation or forward-recovery path for failed steps
- No idempotency key policy for retry-unsafe external commands
- Ack/offset commit before durable side effects
- Using stale read model to enforce hard write invariants
- Hidden invariant ownership ("some consumer will fix it")
- Event choreography with uncontrolled cycles and no ownership boundaries
- Distributed locks used for correctness without fencing and recovery semantics

## MUST / SHOULD / NEVER

### MUST
- MUST enforce local transaction boundaries per service-owned datastore.
- MUST define invariant register and explicit owner for every critical business rule.
- MUST model multi-step flow as explicit saga/workflow state machine.
- MUST define compensation or forward-recovery semantics for each step.
- MUST implement idempotency for external commands and async consumers.
- MUST use outbox-equivalent atomic linkage for state change + message emission.
- MUST define reconciliation scope, cadence, and ownership for critical flows.
- MUST define and monitor read-model freshness SLOs.

### SHOULD
- SHOULD default to orchestration for complex business workflows.
- SHOULD keep one active workflow per business key via durable uniqueness constraints.
- SHOULD use optimistic concurrency for workflow and aggregate transitions.
- SHOULD keep read models rebuildable from source-of-truth history.
- SHOULD surface compensation/reconciliation outcomes in logs and metrics.

### NEVER
- NEVER assume global ACID semantics across services.
- NEVER use dual write as "temporary" shortcut for production paths.
- NEVER treat exactly-once end-to-end as an assumed property.
- NEVER hide cross-service invariants behind eventual read-model convergence.
- NEVER ship distributed consistency flows without operational reconciliation path.

## Review checklist
Before approving distributed-consistency or saga changes, verify:

- Local transaction boundaries are explicit and correct per step
- Invariant register exists with clear owner and enforcement point
- Orchestration vs choreography choice follows decision rules
- Step contracts include idempotency, timeout, retry class, and compensation/forward recovery
- Pivot transaction is identified and post-pivot steps are retryable/idempotent
- Outbox/inbox or equivalent dedup mechanisms are present
- Ack/offset commit ordering is durable-state-first
- Read model freshness budget is defined and observable
- Reconciliation jobs exist with cadence, watermarking, and repair strategy
- No blocked anti-patterns are introduced without approved ADR exception
