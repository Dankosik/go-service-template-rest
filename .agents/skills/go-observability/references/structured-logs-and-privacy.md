# Structured Logs And Privacy

## Load When
Load this when a change adds a log event or field, touches redaction or a sensitive value, or proposes logging request, response, or query content.

## Decide

**Correlation is published, not added.** `internal/observability/logctx` decorates the process logger so every record carries `request_id`, `trace_id`, and `span_id` from its context. It also handles the case that breaks naive decorators: under `WithGroup`, correlation is applied to the ungrouped base so the keys never nest into `db.request_id` and split a service's records across two queries. A call site adds none of this; it only has to pass the context, which sloglint `context: scope` enforces.

**The access log is fixed and its discriminator is `problem_code`.** `AccessLog` records `method`, `route`, `status`, `duration_ms`, and `problem_code`. That last field is why it exists: a 503 is load shedding, a saturated pool, or a draining instance, and status alone cannot tell them apart during the incident when the distinction is the whole question. Its cardinality is bounded by the problem catalog, which also makes it safe as a metric label and usable in SLI math when an SLO must exclude shed load from dependency failure.

**Probe routes are unlogged by default.** `/health/live` and `/health/ready` are matched on the routed template, not the raw path, and skipped unless `logHealthProbes` is set — orchestrators poll them for the life of the pod and every log backend bills the volume. An unmatched request that merely looks like a probe is still recorded.

**SQL text is already decided.** The pgx tracer runs with `otelpgx.WithDisableSQLStatementInAttributes()` and `WithDisableConnectionDetailsInAttributes()`. Proposing sanitized query capture is reopening that decision, not filling a gap; it needs the owner and the reason, not a redaction scheme.

**Judge an emitted field as a disclosure.** A log field reaches a backend with broader access and longer retention than an API response, so it passes the same judgment: who consumes it, what class of data it is, how long it lives. Exception strings and stack traces carry hostnames, SQL literals, file paths, and request content without anyone choosing to log them.

## Reject
- Adding `trace_id`, `request_id`, or `span_id` to a log call, because `logctx` already published them and the hand-written copy is the one that drifts.
- Paging on log text where the service can classify at decision time, because a bounded metric can carry the same classification and log-scrape alerting inherits the backend's indexing delay.
- A dynamic event name such as `payment_failed_for_user_123`, because the name becomes an unbounded dimension in every backend that indexes it.

## Prove
- For each new field: its consumer, its sensitivity class, and whether it must stay out of metric labels.
- Confirm the call site passes a context rather than re-adding correlation.
- Confirm any classification an alert or SLO depends on exists as a bounded metric, or record why logs are the only source.
