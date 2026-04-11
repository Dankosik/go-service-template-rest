# Async, DLQ, Lag, And Reconciliation Observability

## Behavior Change Thesis
When loaded for producer, consumer, retry, DLQ, redrive, backlog, lag, scheduled job, or reconciliation symptoms, this file makes the model separate attempt, durable handoff, logical completion, freshness, DLQ, redrive, and drift visibility instead of likely mistake "consumer lag" or retry counts as the whole async observability story.

## When To Load
Load this when the spec touches producers, consumers, queues, Kafka/RabbitMQ/SQS/Pub/Sub, retries, poison messages, DLQs, redrive, idempotency, backlog, lag, oldest age, batch processing, scheduled jobs, or reconciliation loops.

## Decision Rubric
- Specify producer durable-handoff proof separately from consumer processing proof.
- Specify consumer throughput, processing duration, bounded outcome, retry reason, idempotency decision, backlog depth, and oldest age. Depth without age is not enough for freshness promises.
- Treat retry attempts, DLQ moves, redrive attempts, and final logical completion as different events.
- Keep message ID, entity ID, idempotency key, retry job ID, and reconciliation run ID in logs/traces only when policy allows them; metric labels use bounded reason, destination, worker, and result values.
- Require DLQ depth, oldest age, owner, redrive attempts, redrive outcome, and reconciliation or manual repair path.
- For reconcilers, measure drift found, repaired, unresolved, deferred, oldest unresolved age, partner failure class, and completion event emission.

## Imitate
- Producer metric: `messaging.client.sent.messages` by bounded destination/template, operation, and `error.type`, paired with send/create span.
  Copy the durable handoff signal.
- Consumer metrics: consumed messages, process duration, processing outcome, retry count by bounded reason, idempotency decision, backlog depth, and backlog oldest age.
  Copy the "health plus freshness plus duplicate handling" set.
- DLQ log: poison-message transition includes `message_id`, `correlation_id`, attempt count, bounded failure reason, and DLQ destination in logs/traces only.
  Copy forensic fields without turning them into labels.
- Reconciler metrics: run duration, drift found, drift repaired, drift unresolved, oldest unresolved drift age, partner failure class, and completion event emitted.
  Copy the eventual-correctness proof.

## Reject
- A single consumer-lag graph with no oldest age, no processing duration, no retry/DLQ count, and no stuck-consumer detection.
- `message_id`, `invoice_id`, `account_id`, idempotency key, or raw failure message as metric labels.
- Treating every retry attempt as a user-visible failure without a separate logical completion or freshness SLI.
- Moving messages to DLQ without depth, oldest age, owner, redrive, and outcome visibility.
- Batch processing spans that lose per-message links and leave no path from failed item to producer.

## Agent Traps
- Counting retries as final failures and hiding eventual success.
- Treating DLQ as an operational solution while leaving it as an unobserved storage bucket.
- Using partition labels without naming an operator action at partition level.
- Assuming per-partner labels are safe when the partner set is user-created or not operationally owned.
- Forgetting reconciler "no drift found" completion events, which are often needed to prove recovery.

## Validation Shape
- Verify each async workflow has attempt, durable handoff, logical completion, and freshness signals or a recorded reason a category is not applicable.
- Verify DLQ contracts include depth, oldest age, owner, redrive attempts, redrive outcomes, and reconciliation/manual repair path.
- Verify trace/correlation path from producer to consumer to retry/DLQ/redrive without IDs in metric labels.

## Canonical Verification Pointer
Use current OpenTelemetry messaging semantic conventions when message span or metric names depend on standards status.
