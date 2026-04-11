# Async, DLQ, Lag, And Reconciliation Observability

## When To Load This
Load this reference when the spec touches producers, consumers, queues, Kafka/RabbitMQ/SQS/Pub/Sub, retries, poison messages, DLQs, redrive, idempotency, backlog, lag, oldest age, batch processing, scheduled jobs, or reconciliation loops.

## Operational Questions
- Did the producer durably hand off the message, and can the operator prove it?
- Is the consumer behind, stuck, failing, duplicating, or dropping messages?
- Is the backlog old enough to violate a user or workflow promise?
- Are retries healthy recovery, a dependency symptom, or poison-message churn?
- Are DLQ entries visible, owned, age-bounded, and redrivable?
- Did reconciliation find drift, repair it, defer it, or emit a completion event?

## Good Telemetry Examples
- Producer metric: `messaging.client.sent.messages` by bounded destination/template, operation, and `error.type`.
- Consumer metrics: consumed messages, process duration, processing outcome, retry count by bounded reason, DLQ depth, DLQ oldest age, redrive attempts, idempotency decisions, and backlog/lag growth.
- Trace: send/create spans on the producer; receive/process/settle spans on the consumer; span links for batch receive or redrive.
- Log: poison-message transition includes `message_id`, `correlation_id`, attempt count, bounded failure reason, and DLQ destination in logs/traces only.
- Reconciler metrics: run duration, drift found, drift repaired, drift unresolved, oldest unresolved drift age, partner failure class, completion event emitted.

## Bad Telemetry Examples
- A single "consumer lag" graph with no oldest age, no processing duration, no retry/DLQ count, and no stuck-consumer detection.
- `message_id`, `invoice_id`, `account_id`, or raw failure message as metric labels.
- Treating every retry attempt as a user-visible failure without a separate logical completion SLI.
- Moving messages to DLQ without metrics for DLQ depth, oldest age, redrive attempts, and redrive outcomes.
- Batch processing spans that lose per-message links and leave no path from failed item to producer.

## Cardinality Traps
- Message ID, invoice ID, account ID, tenant ID, webhook ID, reconciliation run ID, retry job ID, and raw idempotency key.
- Dynamic topic, queue, subscription, or DLQ names derived from customers or accounts; use a low-cardinality destination template when available.
- Retry labels from raw exception strings; use bounded reasons such as `timeout`, `rate_limited`, `validation`, `conflict`, `poison`, or `unknown`.
- Partition labels multiplied by topic, consumer group, region, and instance. Use partition only when an operator can act at partition level.
- Per-partner labels are acceptable only when the partner set is controlled and operationally owned.

## Selected And Rejected Options
- Select separate attempt, logical completion, and freshness signals for async workflows.
- Select backlog depth plus oldest age because depth alone cannot tell whether the queued work violates freshness.
- Select bounded retry and DLQ reason taxonomies so alerts route to the right owner.
- Select span links for batch, fan-in, retry, and redrive paths.
- Select reconciliation drift age and unresolved count when correctness is eventually repaired.
- Reject "consumer lag only" as sufficient async observability.
- Reject DLQ as a black box. A DLQ without depth, age, owner, redrive, and reconciliation visibility is an unobserved failure store.

## Exa Source Links
- OpenTelemetry Messaging Spans: https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/
- OpenTelemetry Messaging Metrics: https://opentelemetry.io/docs/specs/semconv/messaging/messaging-metrics/
- W3C Trace Context Recommendation: https://www.w3.org/TR/trace-context/
- Google SRE Workbook, Implementing SLOs: https://sre.google/workbook/implementing-slos/
- Google SRE Workbook, Monitoring: https://sre.google/workbook/monitoring/
