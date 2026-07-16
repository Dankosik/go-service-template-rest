# Trace Context And Correlation

## Behavior Change Thesis
When loaded for request IDs, propagation, baggage, async correlation, retries, DLQ, redrive, batch, or fan-in symptoms, this file makes the model preserve safe trace/correlation continuity and use span links where lineage is not single-parent instead of likely mistake request-ID-only tracing, unfiltered baggage, or IDs as metric labels.

## When To Load
Load this when the spec needs W3C Trace Context, baggage, request IDs, log correlation, cross-service propagation, async message correlation, span links, retry linkage, DLQ/redrive correlation, or a support handle that crosses boundaries.

## Decision Rubric
- Default HTTP-compatible propagation to W3C `traceparent` and `tracestate`; document any repo or platform exception.
- Use request IDs as client/support handles and log correlation fields, not metric labels.
- Use baggage only from an allowlist. Strip or rewrite it before crossing trust boundaries when the receiving side should not see the value.
- Use parent-child spans for direct causality; use span links for batch, fan-in, retry/redrive, or processing under another ambient context.
- When linked contexts are already known at span start, specify creation-time links; adding them later can miss sampler decisions.
- Preserve original correlation/message identity through retry and DLQ transitions in logs/traces while metrics use bounded reason and destination labels.
- Do not require one ID to solve every problem. Trace ID, request ID, message ID, correlation ID, and idempotency key can have different privacy and lifetime rules.

## Imitate
- HTTP service accepts and propagates `traceparent` and `tracestate`, generates a request ID when missing, and includes trace/log correlation fields on structured logs.
  Copy the boundary propagation plus support-handle split.
- Async producer injects creation context into message headers and records `correlation_id`, `message_id`, and `attempt` in logs/traces only.
  Copy the "headers/logs/traces, not metric labels" placement.
- Batch consumer creates a process span with links to message creation contexts instead of choosing one producer as parent.
  Copy the link-based lineage model.
- DLQ/redrive keeps original correlation ID and message ID in logs/traces while metrics use `reason="timeout"` and `queue="invoice-dlq"`.
  Copy the stable retry ancestry.

## Reject
- `requests_total{trace_id="..."}` or `consumer_failures_total{message_id="..."}`.
- Propagating `user_id`, raw tenant ID, account ID, email, token, or plan name in baggage without allowlist and egress policy.
- Regenerating a new correlation ID on every retry so the operator cannot connect original attempt, DLQ entry, and redrive.
- Putting trace context only in logs while failing to inject or extract it on outbound calls.
- Forcing a batch process to pretend one message is the parent of all others.

## Agent Traps
- Confusing correlation with causality. A shared correlation ID does not make a parent-child span relationship true.
- Copying all message headers into span attributes or link attributes.
- Treating third-party propagation as safe because values are "just metadata."
- Letting trace context appear in alerts or metric labels rather than using trace links, exemplars, or logs.
- Forgetting async retry/redrive ancestry when the happy path trace looks correct.

## Validation Shape
- For every boundary, name inject/extract behavior and the fields permitted to cross it.
- For every async retry, DLQ, redrive, or batch flow, state whether lineage is parent-child or linked and why.
- Verify at least one alert-to-trace-to-log pivot without raw IDs in metric labels.

## Canonical Verification Pointer
Use the W3C Trace Context recommendation and current OpenTelemetry context/baggage guidance when propagation semantics or standards status affects the spec.
