# Structured Logs And Privacy

## When To Load This
Load this reference when designing structured log events, privacy controls, redaction, log-to-trace correlation, audit/error events, DB/query telemetry, request/response body handling, or support-investigation fields.

## Operational Questions
- Which specific incident or support question requires a log field?
- Can the same decision be represented as a bounded metric instead of log scraping?
- Does the log event include enough trace/resource context to pivot from a metric alert?
- Which fields are sensitive, regulated, user-controlled, or externally propagated?
- Where is redaction enforced: source, middleware, collector, backend, or all of them?
- How long should the log be retained, and who is allowed to query it?

## Good Telemetry Examples
- `payout.create.completed` log with `trace_id`, `span_id`, `request_id`, `route_template="/v1/payouts"`, `outcome="accepted_async_retry"`, `error.type="fraud_timeout"`, and no request body.
- `invoice.consumer.deduped` log with `invoice_id` only if the privacy policy allows logs to carry it, while the metric uses `decision="duplicate"` without the ID.
- `db.query.text` records parameterized query text or sanitized literal placeholders, and `db.query.summary` remains low-cardinality and free of dynamic values.
- Security-relevant denials use structured event names and bounded reason codes, with secrets and tokens redacted before export.
- Collector-side redaction or filtering is a backup control, not the only place sensitive application data is handled.

## Bad Telemetry Examples
- Logging request/response bodies "for debugging" without data classification, sampling, retention, and access controls.
- Logging `Authorization`, cookies, tokens, API keys, session IDs, passwords, raw query strings, or full URLs with credentials.
- Building alerts from ad hoc log text such as "if this string appears, page".
- Logging SQL statements with literal email, card, token, or account values.
- Adding `tenant_id` and `user_id` to every log when the operator only needs them for a narrow support workflow.

## Cardinality Traps
- Entity IDs in logs are often acceptable for forensics, but never copy them into metric labels or dashboard variables by default.
- Dynamic log event names such as `payment_failed_for_user_123` make query and retention controls harder.
- Raw exception messages and stack traces can include paths, hostnames, SQL literals, request bodies, or tokens.
- Baggage and trace context can be logged accidentally by generic header logging.
- Query string keys can be stable while values are sensitive; preserve keys only when useful and redact values.

## Selected And Rejected Options
- Select structured logs for high-cardinality forensic detail after an alert fires.
- Select bounded metrics for alerting and SLI classification, with the same decision recorded in logs for later investigation.
- Select allowlists for log fields and baggage fields when the service crosses trust boundaries.
- Select source redaction first, collector redaction second, and backend access control as defense in depth.
- Reject raw body logging, full header logging, raw URL logging, and "temporary" PII logs without expiry.
- Reject using logs as the only place where SLO-impacting classification is computed when the service can emit a metric at decision time.

## Exa Source Links
- OpenTelemetry Logs Overview and correlation model: https://opentelemetry.io/docs/reference/specification/logs/overview/
- OpenTelemetry Handling Sensitive Data: https://opentelemetry.io/docs/security/handling-sensitive-data/
- OpenTelemetry Collector processors list, including redaction/filter/transform processors: https://opentelemetry.io/docs/collector/components/processor/
- OpenTelemetry Database spans and `db.query.text` sanitization: https://opentelemetry.io/docs/specs/semconv/database/database-spans/
- OpenTelemetry HTTP spans and URL scrubbing guidance: https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/http/
