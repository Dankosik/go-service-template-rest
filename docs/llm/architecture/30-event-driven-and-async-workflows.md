# Event-driven and async workflow instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing event-driven service interactions
  - Choosing between events/commands and pub/sub/queues
  - Defining outbox/inbox, delivery semantics, retries, DLQ, and deduplication
  - Designing producer/consumer contracts, schema evolution, ordering, and replay behavior
  - Reviewing observability and idempotency for background workflows
- Do not load when: The task is a local synchronous code change with no async workflow, message contract, or broker behavior impact

## Purpose
- This document defines repository defaults for event-driven integrations, broker-based processing, and background workflows.
- The goal is predictable consistency, failure handling, and debuggability under at-least-once delivery.
- The defaults below are mandatory unless an ADR explicitly approves an exception.

## Async decision rules
Use this order. Do not choose broker/technology first.

### 1) Decide if async is justified
Use async when any point is true:
- The operation is not in the immediate user response path
- Expected duration is variable or often longer than interactive timeout budgets
- The use case is fan-out to multiple independent consumers
- The workflow can tolerate eventual consistency
- You need buffering and backpressure isolation between producer and consumer throughput

Do not use async when any point is true:
- The caller needs immediate success/failure semantics to continue user flow
- The invariant must be checked and committed atomically in one request
- The design is effectively synchronous RPC hidden behind a broker
- There is no clear owner for retry policy, DLQ handling, or reconciliation
- Observability for lag/retry/DLQ/idempotency is missing

### 2) Choose interaction intent: event vs command
- Event default:
  - A fact that already happened
  - Producer does not target a specific consumer
  - Multiple consumers may react independently
  - Naming default: past tense, versioned type (for example `order.created.v1`)
- Command default:
  - A request to perform specific work
  - One logical owner should process it
  - Naming default: imperative, versioned type (for example `payment.capture.v1`)
- Never model commands as broadcast domain events.
- Never use events as hidden request-response RPC.

### 3) Choose topology: pub/sub vs queue
- Pub/sub default:
  - Domain events with independent consumers
  - Low producer coupling to downstream ownership
  - One producer event may drive multiple bounded-context reactions
- Queue default:
  - Work distribution where one consumer group owns processing
  - Task execution, background jobs, and bounded worker pools
- If one use case needs both:
  - Publish one domain event
  - Let each consumer own its own queueing/retry strategy locally

## Consistency defaults: outbox/inbox and delivery semantics

### Outbox (producer side)
- For DB state changes that must emit events, transactional outbox is mandatory.
- Write business state and outbox row in the same local DB transaction.
- Publish from outbox asynchronously via worker/CDC.
- Worker claim pattern should prevent duplicate concurrent claims (for SQL stores, `FOR UPDATE SKIP LOCKED` style claim is the default).
- Direct dual write (`DB commit` + `broker publish` without atomic linkage) is a review blocker.

### Inbox (consumer side)
- Consumer deduplication store is mandatory for at-least-once flows with side effects.
- Default dedup key:
  - CloudEvents: `source + id`
  - Non-CloudEvents: `producer_service + message_id`
- Default storage contract:
  - Unique key on `(consumer_group, dedup_key)`
  - Insert-first dedup check (`ON CONFLICT DO NOTHING` or equivalent)
- Dedup retention default: 7 days minimum, or longer than maximum replay/redrive window (whichever is greater).

### Delivery semantics defaults
- System default is at-least-once delivery.
- Exactly-once end-to-end is not an assumed property.
- At-most-once is allowed only by explicit ADR with accepted data-loss risk.
- Ack/offset commit rule:
  - Commit only after local side effects and dedup/inbox state are committed
  - Never ack before durable state transition

## Retry, DLQ, and poison message policy

### Error classification
- Every handler MUST classify errors as:
  - `retryable_transient`
  - `non_retryable`
  - `poison_payload` (schema/validation/domain invariant impossible to satisfy)
- Classification must be deterministic and observable.

### Retry defaults
- Default retry strategy: exponential backoff with jitter.
- Default processing attempts: 8 total (initial + 7 retries).
- Default backoff: 1s base, factor 2, cap 5m, full jitter.
- Never use infinite retries.
- Never retry validation/auth/business-conflict errors as transient.

### DLQ defaults
- After max attempts, message goes to DLQ with full failure context.
- Non-retryable and poison messages go directly to DLQ.
- DLQ record must preserve:
  - `message_id`, `correlation_id`, `event_type`, `attempt`, `error_class`, `error_reason`
  - Linkable trace context
- DLQ retention default: 14 days minimum.
- Redrive policy:
  - Redrive only after root-cause fix
  - Redrive must be rate-limited and observable
  - Bulk replays without throttling are prohibited

## Producer and consumer design rules

### Producer contract
- Producer MUST publish versioned schemas and validate payload before publish.
- Envelope default for cross-service events:
  - CloudEvents-compatible attributes (`id`, `source`, `type`, `specversion`, `time`)
- Producer MUST include:
  - `message_id`
  - `correlation_id` (stable across workflow)
  - Trace propagation fields (`traceparent`, `tracestate`) in headers/attributes
- Partition/routing key MUST be explicit and documented per event type.
- If per-entity ordering is required, default partition key is aggregate/entity ID.

### Consumer contract
- Consumer MUST be idempotent by construction.
- Processing order in handler:
  - Validate envelope/schema
  - Dedup check (inbox)
  - Execute side effects inside local transaction boundary
  - Commit transaction
  - Ack/commit broker position
- External side effects (HTTP/RPC) MUST use idempotency keys derived from message identity.
- Consumer concurrency MUST be bounded; unbounded goroutine fan-out is prohibited.

## Schema evolution, ordering assumptions, and replay safety

### Schema evolution defaults
- Backward-compatible additive evolution is default.
- Breaking changes require new event type/version and migration window.
- Producers SHOULD dual-publish old+new versions during migration when consumers are not all upgraded.
- Consumers MUST be tolerant readers (ignore unknown fields unless contract forbids).
- Protobuf-specific hard rules:
  - Never reuse tag numbers
  - Never change field wire type incompatibly

### Ordering assumptions
- Never assume global ordering across topics/partitions/queues.
- Ordering is valid only within the broker’s documented boundary (for example partition or single-active consumer lane).
- If strict ordering is business-critical:
  - Document ordering key and boundary in the contract
  - Accept throughput trade-off explicitly
  - Add invariant tests for out-of-order handling
- Handlers MUST tolerate duplicates and out-of-order delivery unless strict ordering boundary is explicitly guaranteed and tested.

### Replay safety defaults
- Every consumer MUST be replay-safe.
- Replay-safe means:
  - Reprocessing the same message does not create duplicate side effects
  - Historical reprocessing can rebuild projections consistently
  - Handler behavior is deterministic for the same input and version
- Replay mode for consumers SHOULD support:
  - Controlled throughput
  - Clear checkpointing strategy
  - Side-effect guards for non-replayable integrations

## Observability and idempotency requirements for async processing

### Trace and log correlation
- W3C Trace Context propagation through message headers/attributes is mandatory.
- Producers and consumers MUST emit spans for send/process operations.
- Batch processing MUST use span links for multiple upstream messages.
- Structured logs for async handlers MUST include:
  - `message_id`, `correlation_id`, `event_type`, `consumer_group`, `attempt`, `outcome`, `error_class`
  - `trace_id`, `span_id`

### Metrics minimum contract
- Application-level metrics MUST include:
  - `async_messages_received_total`
  - `async_messages_processed_total{outcome=...}`
  - `async_processing_duration_seconds` (histogram)
  - `async_retries_total{reason=...}`
  - `async_dlq_total{reason=...}`
  - `async_dedup_total{decision=processed|duplicate_ignored|conflict}`
- Broker/platform metrics MUST include:
  - Queue/topic depth/backlog
  - Consumer lag
  - Oldest message age
  - DLQ depth and age
- Cardinality rule:
  - Never put `message_id`, `correlation_id`, `trace_id`, `user_id` in metric labels
  - Keep label sets bounded and reviewable

### Idempotency minimum contract
- Idempotency key scope MUST include tenant/account and operation identity where relevant.
- Same idempotency key + same payload MUST return equivalent outcome.
- Same idempotency key + different payload MUST be treated as conflict and surfaced for investigation.
- Idempotency decisions MUST be observable in logs and metrics.

## Anti-patterns
Treat each item as a review blocker unless an ADR explicitly accepts the risk.

- Dual writes without outbox or equivalent atomic linkage
- Ack/offset commit before durable side effects
- Infinite retries or retries without jitter/caps
- No DLQ ownership or no DLQ redrive procedure
- Commands broadcast as events without single logical owner
- “Async by default” to hide unclear service boundaries or synchronous coupling
- Schema-breaking changes without versioned event rollout
- Assuming global ordering across partitions/queues
- Replay without idempotency and side-effect isolation
- Missing lag/backlog/DLQ observability
- High-cardinality metric labels in async telemetry

## MUST / SHOULD / NEVER

### MUST
- MUST justify async usage before selecting broker/tooling.
- MUST define each message as event or command with explicit ownership semantics.
- MUST use outbox for state-change-driven event publication.
- MUST implement consumer idempotency via inbox/dedup storage.
- MUST classify errors into retryable/non-retryable/poison and apply bounded retries.
- MUST send exhausted/non-retryable messages to DLQ with diagnostic context.
- MUST version async schemas and document migration strategy.
- MUST document ordering boundary assumptions and replay safety for each consumer.
- MUST implement trace propagation, structured logs, and required async metrics.

### SHOULD
- SHOULD prefer orchestration for complex multi-step business workflows requiring strict operational control.
- SHOULD keep async handlers focused on one responsibility and one local transaction boundary.
- SHOULD use CloudEvents-compatible envelopes for cross-service events.
- SHOULD use reconciliation jobs for critical eventual-consistency projections.
- SHOULD include contract tests for producer/consumer schema compatibility.

### NEVER
- NEVER claim exactly-once end-to-end as a default property.
- NEVER use broker as hidden synchronous RPC transport for core request paths.
- NEVER rely on distributed locks as primary cross-service consistency control.
- NEVER deploy async workflows without lag, retry, DLQ, and dedup observability.
- NEVER perform unrestricted DLQ replay or backfill in production.

## Review checklist
Before approving event-driven or async workflow changes, verify:

- Async vs sync choice is explicit and justified by workflow constraints
- Event vs command semantics are explicit and consistent with ownership
- Pub/sub vs queue topology is chosen intentionally and documented
- Outbox is present for DB state changes that emit events
- Consumer inbox/dedup strategy exists with durable uniqueness constraints
- Ack/offset commit happens only after durable side effects
- Retry policy is bounded, jittered, and error-class aware
- DLQ policy, ownership, and redrive procedure are defined
- Message schema evolution path is backward-compatible or versioned with migration
- Ordering assumptions are explicit and tested at the correct boundary
- Replay procedure is safe, controlled, and idempotent
- Trace context propagation, log correlation fields, and async metrics are implemented
- No blocked anti-patterns are introduced without approved ADR exception
