# Structured Logs And Privacy

## Behavior Change Thesis
When loaded for structured log or privacy symptoms, this file makes the model design allowlisted forensic logs with redaction and metric pivots instead of likely mistake raw body/header logging, sensitive query capture, or log-scrape alerting.

## When To Load
Load this when the spec touches structured log events, support-investigation fields, redaction, PII/secrets, request or response bodies, raw headers, DB/query data, audit/error events, or log-to-trace correlation.

## Decision Rubric
- Every log field needs a concrete incident, audit, or support question. If the field cannot name its consumer, reject it.
- Use bounded structured event names, bounded reason codes, and stable resource/trace context.
- Keep alerting classification in metrics when the service can emit it at decision time; logs can carry the same classification for investigation.
- Allowlist sensitive or externally propagated fields. Redact at source first, then use collector/backend controls as defense in depth.
- Store entity identifiers in logs only when policy, retention, and access controls allow them; never promote those identifiers into metric labels.
- Log parameterized or sanitized query text only when it changes operator behavior; prefer query summary fields over raw SQL with literals.

## Imitate
- `payout.create.completed` with `trace_id`, `span_id`, `request_id`, route template, bounded `outcome`, bounded `error.type`, and no request body.
  Copy the completion-boundary event and bounded classification.
- `invoice.consumer.deduped` with `invoice_id` only when privacy policy permits logs to carry it, while the metric uses `decision="duplicate"` without the ID.
  Copy the split between forensic log detail and aggregate metric label.
- `db.query.text` as parameterized query text or sanitized literal placeholders, paired with low-cardinality `db.query.summary`.
  Copy sanitization and summary separation.

## Reject
- Logging request or response bodies "temporarily" without data classification, sampling, retention, access control, and expiry.
- Logging `Authorization`, cookies, tokens, API keys, session IDs, passwords, raw query strings, or URLs containing credentials.
- Paging on ad hoc log text such as "if this string appears, page" when bounded metrics can represent the state.
- Dynamic log event names such as `payment_failed_for_user_123`.
- Generic header logging that accidentally records baggage, trace context, cookies, or customer-controlled values.

## Agent Traps
- Treating logs as inherently safer than metrics. Logs often carry more sensitive data and broader access patterns.
- Adding `tenant_id` and `user_id` to every log because support might someday ask. Narrow the workflow and retention first.
- Assuming collector redaction is enough after the application already emitted secrets.
- Forgetting that exception strings and stack traces can include hostnames, SQL literals, file paths, request bodies, or tokens.
- Designing log fields that cannot be joined to traces or representative alerts.

## Validation Shape
- For each new log field, record consumer, sensitivity class, redaction point, retention/access rule, and whether it must stay out of metrics.
- Verify alerting and SLO-impacting classification exists as a bounded metric or record why logs are the only source.
- Verify one representative alert can pivot to trace and log query without exposing raw identifiers in the alert payload.

## Canonical Verification Pointer
Use current OpenTelemetry sensitive-data and log-correlation guidance when a field's safety or correlation semantics depends on collector/exporter behavior.
