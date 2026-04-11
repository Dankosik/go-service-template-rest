# Trace Context And Correlation

## When To Load This
Load this reference when the spec needs request IDs, trace propagation, baggage, log correlation, async message correlation, span links, cross-service propagation, retry linkage, or DLQ/redrive correlation.

## Operational Questions
- How does the operator move from an alerting metric to one representative trace and then to related logs?
- Which boundaries must preserve trace context: HTTP, gRPC, worker enqueue, message broker, retry queue, DLQ, redrive, or reconciliation?
- Is there a single parent-child relationship, or should spans use links for batch, fan-in, retries, or ambient HTTP plus message contexts?
- Which correlation fields are safe to propagate outside the trust boundary?
- Which identifiers belong only in logs/traces and must never become metric labels?

## Good Telemetry Examples
- HTTP services accept and propagate W3C `traceparent` and `tracestate`, generate a request ID when missing, and include trace/log correlation fields on structured logs.
- Async producers inject message creation context into message headers and record `correlation_id`, `message_id`, and `attempt` in logs/traces, not in metric labels.
- Batch consumers use span links from the batch process span to message creation contexts, with per-message attributes on links when values vary.
- Retry and DLQ transitions preserve the original correlation ID and message ID in logs/traces while metrics use bounded labels such as `reason="timeout"` and `queue="invoice-dlq"`.
- Baggage is allowlisted, documented, and stripped or rewritten before leaving trusted boundaries.

## Bad Telemetry Examples
- `requests_total{trace_id="..."}` because it creates one time series per trace and destroys metric usability.
- Propagating `user_id`, raw tenant ID, account ID, email, token, or plan name in baggage without an allowlist and egress rule.
- Forcing a batch consumer span to choose one producer as parent when it processed messages from many producers.
- Regenerating a new correlation ID on every retry, making the operator unable to connect the original attempt, DLQ entry, and redrive.
- Putting trace context only in logs while failing to inject or extract it on outbound calls.

## Cardinality Traps
- Trace IDs, span IDs, request IDs, message IDs, correlation IDs, and retry job IDs as metric labels.
- Baggage keys whose value space is unbounded or user-controlled.
- Destination names derived from customer, tenant, or account identifiers.
- Link attributes copied from full message headers instead of a bounded allowlist.

## Selected And Rejected Options
- Select W3C Trace Context as the default cross-service propagation format for HTTP-compatible boundaries.
- Select baggage only for allowlisted, low-risk context that is needed downstream; document egress behavior.
- Select request ID as a log and response correlation aid when clients need a support handle; do not use it as a metric dimension.
- Select span links for batch, fan-in, DLQ/redrive, or message processing under another ambient context.
- Reject `trace_id` metric labels and "search the metrics by request ID" workflows. Use exemplars or backend trace links if the platform supports them.
- Reject unfiltered baggage propagation to third-party services.

## Exa Source Links
- W3C Trace Context Recommendation: https://www.w3.org/TR/trace-context/
- OpenTelemetry Context Propagation: https://opentelemetry.io/docs/concepts/context-propagation/
- OpenTelemetry Baggage: https://opentelemetry.io/docs/concepts/signals/baggage/
- OpenTelemetry Logs Overview and correlation model: https://opentelemetry.io/docs/reference/specification/logs/overview/
- OpenTelemetry Messaging Spans: https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/
