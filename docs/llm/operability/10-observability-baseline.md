# Observability baseline instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing logs, metrics, traces, correlation IDs, or instrumentation defaults
  - Defining API/client/DB/worker/job observability contracts
  - Reviewing telemetry quality, cardinality risk, or monitoring gaps
  - Defining OpenTelemetry bootstrap, propagation, or sampling behavior
  - Standardizing naming conventions and telemetry review criteria
- Do not load when: Task is documentation-only with no runtime telemetry behavior impact

## Purpose
- This document defines repository defaults for production observability in Go services.
- Goal: predictable incident diagnosis through correlated logs, metrics, and traces with bounded cost.
- Defaults in this document are mandatory unless an ADR explicitly approves an exception.

## Baseline assumptions
- OpenTelemetry is the default telemetry framework for metrics and traces.
- OTLP export to OpenTelemetry Collector is default transport.
- Structured logs use JSON in stdout/stderr; in Go, `log/slog` is the default logger.
- OTel Go logs signal is treated as non-default; log/trace correlation is implemented via `trace_id` and `span_id` in structured logs.
- Trace propagation default is W3C Trace Context + W3C Baggage.
- Every signal must include consistent service identity: `service.name`, `service.version`, `deployment.environment.name`.

## Required inputs before changing observability behavior
Resolve these first. If unknown, apply defaults and state assumptions.

- Service type: API, worker, job runner, or mixed
- Main SLOs and operational questions (latency, error rate, throughput, backlog)
- Trust boundary for inbound propagation (public edge, internal only, mixed)
- Data sensitivity constraints for logs/traces
- Retry and idempotency model (sync and async)
- Existing dashboards/alerts that may break on metric or label changes

## OpenTelemetry defaults and required instrumentation

### Composition-root bootstrap (mandatory)
- Initialize OTel in `cmd/<service>/main.go` before starting servers/workers.
- Configure and register globally:
  - `resource.Resource` with `service.name`, `service.version`, `deployment.environment.name`
  - `trace.TracerProvider`
  - `metric.MeterProvider`
  - text map propagator with `tracecontext,baggage`
- Configure exporters and endpoints via environment variables, not hardcoded constants:
  - `OTEL_SERVICE_NAME`
  - `OTEL_RESOURCE_ATTRIBUTES`
  - `OTEL_EXPORTER_OTLP_ENDPOINT`
  - `OTEL_EXPORTER_OTLP_PROTOCOL`
  - `OTEL_EXPORTER_OTLP_HEADERS`
  - `OTEL_EXPORTER_OTLP_TIMEOUT`
- Sampling defaults:
  - Production default: `OTEL_TRACES_SAMPLER=parentbased_traceidratio`
  - Production default ratio: `OTEL_TRACES_SAMPLER_ARG=0.10`
  - Local dev default may use always-on sampling
- Shutdown rule:
  - tracer and meter providers must be shutdown on graceful termination to flush telemetry

### Required default instrumentation map

| Boundary | Default instrumentation | Mandatory behavior |
|---|---|---|
| HTTP server | `otelhttp.NewHandler` | create server spans, emit HTTP metrics, extract inbound trace context |
| HTTP client | `otelhttp.NewTransport` | create client spans, emit client metrics, inject outbound trace context |
| gRPC server/client | `otelgrpc` stats handlers | create RPC spans/metrics and propagate metadata context |
| SQL (`database/sql`) | `otelsql` | emit DB spans/metrics, use request/job context, expose pool stats |
| Go runtime | OTel runtime instrumentation | emit runtime/process metrics for saturation diagnostics |
| Message producer/consumer | OTel messaging instrumentation or manual spans with semconv | inject/extract context in message headers and trace send/process operations |
| Background jobs | manual root spans per run + instrumented downstream calls | each job run must be traceable and correlated with logs/metrics |

## Mandatory signal contract by component

### 1) API handlers

Required structured log fields:
- `timestamp`, `level`, `message`
- `service.name`, `service.version`, `deployment.environment.name`
- `trace_id`, `span_id`, `request_id`
- `component="api"`, `operation` (handler/use-case name), `outcome`
- `http.method`, `http.route`, `http.status_code`, `duration_ms`
- `error.type` for failures (bounded taxonomy)

Required RED + saturation metrics:
- `http.server.request.duration` (Histogram, unit `s`)
- Request rate and errors derived from server request metrics with low-cardinality dimensions
- `http.server.active_requests` for in-flight saturation visibility

Required tracing and propagation:
- Extract inbound `traceparent`/`tracestate`
- Generate `request_id` if absent; return `X-Request-ID` in responses
- Use low-cardinality span names based on route templates, never raw path
- Record errors on spans and set error status for failed operations

### 2) Outbound clients (HTTP/RPC)

Required structured log fields:
- `timestamp`, `level`, `message`
- `trace_id`, `span_id`, `request_id`
- `component="client"`, `target.service`, `operation`, `outcome`
- `retry_attempt`, `timeout_ms`, `duration_ms`, `error.type`

Required RED + saturation metrics:
- `http.client.request.duration` (Histogram, unit `s`) for HTTP
- equivalent RPC client duration metric for RPC transports
- `outbound_retries_total{target,reason}` with bounded `reason`
- optional but recommended connection/pool saturation metrics for critical dependencies

Required tracing and propagation:
- Inject current trace context into every outbound call
- Never replace request/job context with `context.Background()` in call paths
- Attach retry attempt information as span attributes/events with bounded values

### 3) Database access

Required structured log fields:
- `timestamp`, `level`, `message`
- `trace_id`, `span_id`, `request_id`
- `component="db"`, `db.system`, `db.operation`, `db.name`, `outcome`
- `duration_ms`, `error.type`

Required RED + saturation metrics:
- `db.client.operation.duration` (or library-equivalent DB operation duration histogram)
- DB pool saturation metrics (`open`, `in_use`, `idle`, `wait_count`, `wait_duration`)
- DB error rate by bounded error classes

Required tracing and propagation:
- Use request/job context for all queries and transactions
- Emit DB spans via `otelsql` or equivalent instrumentation
- Do not capture non-parameterized SQL text by default

### 4) Workers and message consumers/producers

Required structured log fields:
- `timestamp`, `level`, `message`
- `trace_id`, `span_id`, `correlation_id`, `message_id`, `request_id` when present
- `component="worker"`, `event_type`, `consumer_group`, `attempt`, `outcome`
- `queue_or_topic`, `partition_or_shard`, `error.type`, `duration_ms`

Required RED + backlog metrics:
- `messaging.process.duration` (Histogram, unit `s`)
- `messaging.client.sent.messages`
- `messaging.client.consumed.messages`
- `async_retries_total{reason}` and `async_dlq_total{reason}` with bounded `reason`
- backlog/lag/oldest-message-age metrics from broker/platform

Required tracing and propagation:
- Inject trace context into message headers/attributes on produce
- Extract context on consume and create `process` spans
- Use span links for batch processing or multi-parent causality
- Preserve stable `correlation_id` across retries and DLQ transitions

### 5) Scheduled jobs and reconcilers

Required structured log fields:
- `timestamp`, `level`, `message`
- `trace_id`, `span_id`, `run_id`
- `component="job"`, `job.name`, `trigger.type`, `schedule`, `outcome`
- `duration_ms`, `items_processed`, `error.type`

Required RED + scheduling metrics:
- `job_runs_total{job,outcome}`
- `job_run_duration_seconds{job}` (Histogram)
- `job_schedule_delay_seconds{job}` (Histogram)
- optional per-job backlog metrics for queue-driven jobs

Required tracing and propagation:
- Start one root span per job run
- Propagate context to all downstream DB/client calls
- For replay/reconciliation of event batches, use span links to source contexts

## Structured log schema defaults
- Use snake_case keys in logs for predictable querying.
- Mandatory common keys for every log record:
  - `timestamp`, `level`, `message`
  - `service.name`, `service.version`, `deployment.environment.name`
  - `trace_id`, `span_id`
  - `component`, `operation`, `outcome`
- Error logging rules:
  - include `error.type` and sanitized `error.message`
  - never log secrets, tokens, credentials, DSNs, raw authorization headers, or full PII payloads
- Correlation rules:
  - sync flows use `request_id`
  - async flows use `correlation_id` + `message_id` + `attempt`

## Trace propagation and correlation rules

### Defaults
- Use `tracecontext,baggage` propagators.
- Trace identity is primary cross-service correlation key.
- `request_id` and `correlation_id` are operational correlation IDs, not auth inputs.

### Decision rules
1. If operation crosses process/network boundary, propagate trace context.
2. If operation is async, propagate trace context in message headers/attributes and keep `correlation_id` stable across retries.
3. If trust boundary is public edge, accept trace context for correlation but apply baggage allowlist and drop sensitive keys.
4. If no inbound trace context exists, create a new root span and keep the generated trace ID in logs.
5. If only one signal can carry high-cardinality IDs, prefer logs/traces, never metrics labels.

## Metrics strategy: technical vs business

### Technical metrics (mandatory)
- Every component must emit RED metrics.
- Every component must emit at least one saturation signal:
  - API/client: in-flight requests or connection usage
  - DB: pool wait/in-use/open
  - workers: backlog/lag/oldest age
  - jobs: schedule delay and running count

### Business metrics (mandatory when behavior changes business outcomes)
- Track domain outcomes explicitly, for example `orders_created_total`, `payments_captured_total`.
- Business metrics do not replace RED metrics.
- Each business metric must have:
  - owner team
  - exact semantic definition
  - bounded label set
  - dashboard panel or alert consumer

### Naming conventions
- For built-in OTel instrumentation, keep semantic-convention names as-is.
- For custom repository metrics, use Prometheus-compatible snake_case with unit/type suffixes:
  - counters: `_total`
  - duration histograms: `_seconds`
  - byte sizes: `_bytes`
  - ratios: `_ratio`
- Metric names must be stable; never encode variable data in metric names.
- Labels/attributes use lowercase snake_case and bounded vocabularies.
- `http.route` and equivalent route fields must use templates, not raw IDs/paths.

## Cardinality discipline (review-blocking)

### Hard rules
- Never use these as metric labels/attributes:
  - `request_id`, `trace_id`, `span_id`, `message_id`, `correlation_id`, `user_id`, `email`, raw URL path, full SQL text
- `error.type` must be a bounded taxonomy, not raw error strings.
- Dimensions must be bounded and documented before merge.

### Label decision rules
1. Can the value set be enumerated and kept bounded in production?
2. Is this dimension required for an actionable alert or SLO investigation?
3. Can the same investigation be solved via logs/traces instead?
4. If cardinality risk exists, reject metric dimension and keep data in logs/traces only.

## Anti-patterns
Treat each item as a review blocker unless an ADR explicitly accepts risk.

- Log spam in hot paths without actionability or sampling controls
- Missing `trace_id`/`span_id`/`request_id` on request-scoped logs
- Logging full request/response bodies by default
- Logging secrets, tokens, DSNs, or sensitive personal data
- High-cardinality labels (`user_id`, `request_id`, raw path, UUID-like values)
- Dynamic metric names assembled from runtime values
- Useless metrics with no owner, no consumer, and no alert/dashboard usage
- Spans without meaningful operation names or with raw URL/SQL payload as names
- Async retries that generate new correlation IDs and break chain visibility
- No backlog/lag observability for queue-based systems

## Review criteria (merge gate)

### Logs
- Structured JSON logs are used for all new operational events.
- Mandatory common fields are present and consistent.
- Request/job/message scoped logs include correlation fields.
- Sensitive data policy is enforced.

### Metrics
- RED coverage is present for changed components.
- Saturation signal is present for changed components.
- New labels have bounded cardinality and clear operational value.
- Metric naming follows repository conventions.

### Traces
- Inbound and outbound propagation is implemented for all network/message boundaries.
- API/client/DB/worker/job operations are instrumented with OTel defaults.
- Span naming follows low-cardinality templates.
- Error recording/status behavior is implemented consistently.

### Operability
- Dashboard and alert impact is documented for telemetry changes.
- Sampling and exporter changes are configurable and rollback-safe.
- Telemetry shutdown/flush behavior is present on graceful termination.

## MUST / SHOULD / NEVER

### MUST
- MUST bootstrap OTel providers and propagators in composition root.
- MUST instrument API, clients, DB, workers, and jobs by default.
- MUST include correlation fields in structured logs.
- MUST emit RED metrics plus saturation signals for each component class.
- MUST keep metric dimensions bounded and reviewable.
- MUST propagate trace context across sync and async boundaries.

### SHOULD
- SHOULD keep custom metric names minimal and semantically stable.
- SHOULD use bounded error taxonomies for `error.type`.
- SHOULD keep business metrics separate from technical health metrics.
- SHOULD use span links for batch and fan-in async processing.

### NEVER
- NEVER treat observability as optional for new production code paths.
- NEVER add high-cardinality identifiers to metrics labels.
- NEVER rely on logs alone without trace or metric correlation.
- NEVER merge telemetry changes that break existing incident response without migration notes.
