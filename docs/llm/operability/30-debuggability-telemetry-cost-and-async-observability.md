# Debuggability, telemetry cost control, and async observability instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing health/readiness/startup/debug endpoints, pprof exposure, or crash diagnostics
  - Defining telemetry budgets, sampling, retention, histogram strategy, or cardinality controls
  - Defining observability for queues, retries, DLQ, lag/backlog, batch processing, or reconciliation jobs
  - Reviewing incident diagnosability vs telemetry cost trade-offs
  - Standardizing safe debug instrumentation and incident-mode observability escalation
- Do not load when: Task is documentation-only and does not affect runtime diagnostics, telemetry cost, or async observability behavior

## Purpose
- This document defines repository defaults for production diagnostics, safe debuggability, telemetry cost control, and async observability.
- Goal: preserve fast incident diagnosis under real failures without allowing uncontrolled telemetry volume/cardinality growth.
- Defaults in this document are mandatory unless an ADR explicitly approves an exception.

## Baseline assumptions
- OpenTelemetry is the default framework for traces and metrics, exported through OTLP.
- Structured logs are JSON and include correlation fields (`trace_id`, `span_id`, request/message correlation keys).
- Services run as long-lived processes under an orchestrator (for example Kubernetes-like probes and graceful termination).
- Diagnostics are exposed via explicit operational contracts, never through ad-hoc debug handlers.
- Async components follow at-least-once processing assumptions and must be idempotency-aware.

## Required inputs before changing diagnostics or telemetry behavior
Resolve these first. If unknown, apply defaults and document assumptions.

- Service class: `api`, `worker`, `async_consumer`, `job_runner`, or mixed
- Runtime topology: app listener(s), admin/debug listener(s), ingress/public exposure boundaries
- Broker topology: queue/log broker type, retry policy, DLQ policy, lag metrics source
- Critical incident questions: what must be diagnosable in first 5, 15, and 60 minutes
- Data sensitivity policy: PII/secrets restrictions for logs, traces, metrics, baggage
- Traffic profile: low/medium/high throughput and expected cost envelope
- Current retention/SLA constraints for metrics, logs, traces

## Production diagnostics baseline

### Health endpoint contract (mandatory)
- `GET /livez` answers only: "should process be restarted?".
- `GET /readyz` answers only: "should this instance receive new traffic now?".
- `GET /startupz` answers only: "has startup completed?".
- Machine contract uses status code only:
  - `200` success
  - `503` failure for readiness/startup conditions
- Human diagnostics can be exposed via `?verbose=1`, but verbose payload is not a machine contract.
- Built-in probes must not require HTTP auth parameters by default; keep probe endpoints minimal and perimeter-protected instead.

### Endpoint behavior defaults
- `/livez` must not depend on external dependencies (DB/cache/queue reachability).
- `/readyz` may include critical dependency checks with strict per-check timeouts.
- `/readyz` must fail during shutdown draining before connection stop.
- `/startupz` must stay failed until initialization gates are complete.
- All probe handlers must be idempotent, low-latency, and side-effect free.
- Readiness handlers must stay cheap even under burst probing while instance is not ready.

### Probe and startup defaults
- Startup protection is mandatory for slow-start services.
- Default startup budget rule:
  - `startup_budget_seconds = startup_probe.period_seconds * startup_probe.failure_threshold`
  - startup budget must cover worst-case startup path
- Do not "solve" slow start by only inflating liveness delay.
- If startup probe is enabled, liveness/readiness remain disabled until startup passes.
- Do not default to exec-probes for complex checks; process-spawn overhead can become a hidden saturation source.

### Shutdown and draining contract
- On `SIGTERM`/`SIGINT`, service must execute this order:
  1. set draining flag
  2. fail readiness immediately
  3. stop accepting new traffic/work
  4. gracefully finish in-flight work
  5. flush telemetry providers
  6. exit before orchestrator hard kill
- Default shutdown timeout:
  - reserve 80-90% of orchestrator grace period for app shutdown
  - keep remaining budget for platform hooks/scheduling jitter
- `http.Server.Shutdown(ctx)` is mandatory for HTTP listeners.
- Long-lived/hijacked connections must have explicit shutdown logic.
- If pre-stop hooks are used, treat them as part of the same termination grace budget; pre-stop must be bounded and deterministic.

### Admin/debug endpoints and safe instrumentation
- Diagnostics endpoints (`/debug/pprof/*`, optional `/debug/vars`) must run on a separate admin listener.
- Default admin listen address: `127.0.0.1:9090` unless explicitly configured otherwise.
- Admin/debug listener must not be exposed through public ingress by default.
- Never register pprof implicitly on a public default mux.
- Safe debug instrumentation defaults:
  - heavy diagnostics are opt-in and time-bounded
  - each activation has owner, reason, start time, and auto-expire deadline
  - activation/deactivation events are audit-logged
  - debug outputs must follow redaction policy
- Mandatory kill-switches:
  - `ENABLE_ADMIN_ENDPOINTS`
  - `ENABLE_PPROF`
  - `ENABLE_EXPVAR`
  - `DEBUG_INSTRUMENTATION_TTL`

### Crash diagnostics defaults
- Production default: `GOTRACEBACK=single`.
- Incident escalation default: temporary `GOTRACEBACK=all` with rollback plan.
- Runtime traceback tuning must be documented (`GOTRACEBACK` and `runtime/debug.SetTraceback` policy).
- Panic/fatal crash logs must include:
  - `service.name`, `service.version`, `deployment.environment.name`
  - process identity (`pid`, host/pod identifier)
  - timestamp and panic/fatal classification
- Controlled crash snapshots may use `runtime/debug.PrintStack` in guarded recovery paths.
- If runtime supports crash-output duplication, configure crash artifact destination explicitly.
- Crash artifacts must follow retention and privacy policies (no secret leakage).

### Diagnostics review blockers
- One shared `/health` endpoint used for both liveness and readiness.
- Liveness failure coupled to temporary dependency outages.
- pprof/expvar reachable on public application listener.
- Shutdown path that exits without waiting for graceful completion.
- No documented crash collection and triage path.

## Telemetry cost control baseline

### Global budget defaults
- Treat telemetry budget as an explicit SLO-adjacent contract.
- Cardinality budget defaults:
  - each metric label must have bounded vocabulary
  - target cardinality per label is about `<= 10` values
  - cardinality above `100` values for any label requires explicit approval
- Dimension budget defaults:
  - prefer `<= 4` labels for custom operational metrics
  - any new high-impact label requires cardinality estimate in PR
- SDK guardrail defaults:
  - configure attribute count/value-length limits via OTel SDK environment limits
  - monitor dropped/truncated attribute counters and treat sustained growth as telemetry-quality incident

### Metrics and histogram strategy
- Use semantic-convention metrics where available; do not invent near-duplicate metrics.
- Histograms are mandatory for latency/freshness; summaries are not default for cross-instance aggregation.
- Histogram defaults:
  - use fixed boundaries and keep them stable
  - target 10-15 buckets for custom histograms
  - include bucket boundaries at SLO thresholds
- Histogram cost rule: each bucket adds time-series cost (`_bucket` plus `_sum` and `_count`); never expand buckets without cost rationale.
- HTTP server duration default boundaries:
  - `0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10` seconds
- Never use high-cardinality IDs in metric labels (`request_id`, `trace_id`, `message_id`, `correlation_id`, `user_id`, raw URL path).
- `http.route` must be route-template cardinality (for example `/orders/{id}`), never raw path.
- Do not promote unbounded header-derived values to metric attributes without strict allowlist/normalization.

### Sampling defaults
- Trace sampling defaults:
  - local/dev: `always_on`
  - production default: `parentbased_traceidratio`
  - production starting ratio: `0.10`
  - high-traffic edge services may start at `0.01` with explicit incident drill validation
- Tail sampling is optional and only allowed with:
  - collector routing guarantees for full trace co-location
  - memory/bytes/rate limits configured
  - explicit policies for error and high-latency retention
- Custom sampler logic in app code is non-default; if used, it must preserve upstream trace state and stay CPU-cheap.
- Log sampling defaults:
  - `debug` disabled in production by default
  - repeated identical non-actionable logs are rate-limited
  - repeated identical errors are fingerprinted and sampled after initial burst

### Log volume control defaults
- Request/message boundary logging defaults:
  - one completion log per request/message handling result
  - additional logs only for state transitions or actionable anomalies
- Do not log full payload bodies by default.
- Structured log field size must be bounded.
- Use explicit log levels; no "everything as error" policy.
- Incident mode can temporarily increase detail for scoped components only with TTL.
- For Loki-like backends, keep labels source-oriented and low-cardinality; request-specific IDs stay in log body or non-indexed metadata.

### Exemplars strategy defaults
- Exemplars are opt-in and only for key latency histograms used in incident triage.
- Do not enable exemplars globally by default.
- Exemplar enablement requires explicit in-memory budget and trace sampling compatibility check.

### Retention defaults
- Retention must be explicit, never implicit backend defaults.
- Default starting points:
  - metrics: `15d` hot retention
  - traces: `14d` hot retention
  - logs: `14d` operational retention
- Longer retention requires cost estimate and owner approval.
- Retention changes must include rollback-safe plan and dashboard/alert impact notes.
- Capacity estimation rule: `ingested_bytes_per_day * retention_days` must be documented for any retention increase.
- Multi-tenant isolation should be done at tenancy/project boundary, not by putting `tenant_id` into metric labels.

### Privacy-aware telemetry defaults
- Data minimization is mandatory across logs, traces, and metrics.
- Secrets and PII must not be emitted into telemetry signals.
- Pipeline-level redaction/sanitization is mandatory:
  - allowlist keys
  - blocked key/value patterns
  - URL sanitization
  - DB statement sanitization
- Baggage propagation is allowlist-based and minimal by default.
- URL/query and DB-statement telemetry must be sanitized before indexing/export.

## Async observability and correlation contract

### Required correlation fields for async flows
- Message transport headers/attributes must include:
  - `traceparent`
  - `tracestate` when present
  - stable `message_id`
  - stable `correlation_id` across retries and DLQ transitions
  - `attempt` or delivery count
  - enqueue/first-seen timestamp when available
- Logs for async handlers must include:
  - `trace_id`, `span_id`
  - `message_id`, `correlation_id`, `attempt`
  - `queue_or_topic`, `consumer_group`, `partition_or_shard` when applicable
  - `outcome`, `error.type`, `duration_ms`
- For CloudEvents transport, store tracing context in tracing extension attributes and keep original chain continuity across hops.

### Trace model across producer, consumer, retries, and DLQ
- Producer must create `send` span (usually `PRODUCER`) and inject context to message carrier.
- Consumer must create `process` span (`CONSUMER`) per handled message or batch.
- `receive` spans are recommended for pull/poll visibility but do not replace `process` spans.
- Batch processing must use span links to all upstream message contexts.
- Retry handling rules:
  - keep `correlation_id` stable
  - increment `attempt`
  - attach retry reason with low-cardinality taxonomy
- Retry policy default is capped exponential backoff with jitter and bounded max attempts.
- DLQ handling rules:
  - emit explicit transition telemetry for DLQ publish/move
  - preserve original correlation fields
  - record bounded root-cause category
- Ack/commit must happen after durable side effects complete; observability must reflect this lifecycle stage.

### Mandatory async metrics
- App-level async metrics:
  - `messaging.process.duration` (Histogram)
  - `messaging.client.operation.duration` (Histogram)
  - `messaging.client.sent.messages` (Counter)
  - `messaging.client.consumed.messages` (Counter)
  - `async_handler_outcome_total{outcome}`
  - `async_retry_scheduled_total{reason}`
  - `async_dlq_published_total{reason}`
  - `async_idempotency_decision_total{decision}`
- Broker/platform metrics:
  - queue depth/backlog
  - oldest message age
  - consumer lag
  - unacked/in-flight count
- Broker-specific minimums:
  - SQS: `ApproximateNumberOfMessagesVisible`, `ApproximateNumberOfMessagesNotVisible`, `ApproximateAgeOfOldestMessage`
  - Kafka: max consumer lag signal (for example `records-lag-max`) and fetch/consume throughput
  - RabbitMQ: `messages`, `messages_ready`, `messages_unacknowledged`
- Reconciliation job metrics:
  - `reconcile_runs_total{job,outcome}`
  - `reconcile_run_duration_seconds{job}`
  - `reconcile_items_scanned_total{job}`
  - `reconcile_drift_found_total{job,drift_type}`
  - `reconcile_repair_applied_total{job,result}`
- DLQ monitoring minimums:
  - DLQ depth and oldest-message age
  - SQS DLQ status uses visible-depth signal, not send-count proxy
  - RabbitMQ DLX paths must track `x-death`/redelivery history and detect DLX cycles

### Correlation matrix (required)

| Flow stage | Trace requirement | Log requirement | Metric requirement |
|---|---|---|---|
| Producer send | `send` span + injected context | `message_id`, `correlation_id`, destination | sent counter + send latency |
| Consumer process | `process` span from extracted context | outcome + attempt + handler ID | process latency + outcome counter |
| Retry scheduling | new retry span/event linked to original | retry reason + next delay + attempt | retry counter by bounded reason |
| DLQ transition | span/event with original link | dlq reason + terminal attempt | DLQ counter + DLQ backlog/age |
| Queue lag growth | trace optional, logs include observed lag source | lag snapshot with component identity | lag/backlog/oldest-age gauges |
| Batch processing | one batch span with span links | batch size + processed/failed counts | batch duration + batch outcome counters |
| Reconciliation job | root run span + links to source traces when possible | run_id + drift summary + repair status | reconcile run duration + drift/repair counters |

### Async decision rules
1. If message crosses process boundary, propagate trace context via message headers/attributes.
2. If batch has multiple upstream contexts, use span links, not forced single-parent lineage.
3. If retry occurs, preserve correlation identity and represent attempt progression explicitly.
4. If DLQ event occurs, emit terminal outcome telemetry and retain searchable cause category.
5. If lag/backlog grows, correlate platform lag metrics with handler outcome and duration metrics before scaling decisions.
6. If trace sampling hides details, use logs keyed by correlation fields to reconstruct chain.
7. If periodic jobs or retry schedulers are synchronized, add deterministic jitter and observe schedule-delay metrics.
8. If processing is at-least-once, treat idempotency outcomes as first-class telemetry and investigation dimensions.

## Preventing telemetry explosion while preserving incident diagnosis

### Three-layer telemetry model (default)
- Layer 1: always-on low-cost signals
  - RED + saturation + backlog/lag metrics
  - bounded structured logs at operation boundaries
  - baseline trace sampling
- Layer 2: sampled diagnostics
  - additional span events and debug logs at controlled sample rates
  - scoped per component/operation with TTL
- Layer 3: incident burst mode
  - temporary targeted increases of trace/log detail for specific routes/topics/consumer groups
  - auto-expire and rollback to baseline configuration

### Escalation defaults
- Incident burst mode must define:
  - owner
  - scope (service/component/operation/topic)
  - max duration
  - explicit stop condition
- Default incident burst TTL: `60m`.
- Expired overrides must auto-revert; manual cleanup is not acceptable.

### Design rule: cheapest signal that answers the question
- Use metrics for aggregate trend and alert conditions.
- Use traces for causal path and latency decomposition.
- Use logs for high-cardinality diagnostics and payload-adjacent context.
- If one signal already answers the operational question, do not duplicate same detail in all three.

### Cost-protecting guardrails
- No new metric label without bounded cardinality justification.
- No new histogram without bucket count and SLO-boundary rationale.
- No new log field carrying unbounded user input without sanitization policy.
- No broad sampling increases without explicit TTL and rollback.
- Do not auto-map all resource attributes to metric labels; promote only intentionally selected dimensions.

### Anti-patterns to reject
- Treating telemetry as "collect everything forever".
- Adding request/message/user IDs into metric labels.
- Using raw URL path, headers, SQL text, or error strings as metric dimensions.
- Enabling global debug logs in production without sampling or rate limiting.
- Keeping incident sampling overrides enabled permanently.
- Emitting large payload dumps to logs/traces by default.
- Creating retry loops without retry metrics or reason taxonomy.
- DLQ enabled but not monitored for depth and age.
- Async traces missing correlation through retries or batch links.
- Replacing stable `correlation_id` on each retry attempt.
- RabbitMQ polling (`basic.get`-style) as production consumption mode.

## Review checklist (merge gate)

### Diagnostics
- Health endpoint semantics are split and implemented correctly.
- Startup behavior prevents false restarts on slow initialization.
- Shutdown performs readiness fail + graceful drain + telemetry flush.
- Admin/debug endpoints are isolated from public exposure.
- pprof/expvar exposure is explicitly controlled by config and network boundary.
- Crash diagnostic behavior is documented and testable in staging.
- Probe endpoints are machine-safe and do not rely on auth parameters for built-in probe mode.
- Shutdown timing accounts for pre-stop plus graceful-stop inside one total termination budget.

### Telemetry cost
- New metrics use bounded labels only and pass cardinality review.
- Histogram boundaries are fixed, justified, and include SLO cut-points.
- Trace sampling config is explicit for environment and service class.
- Log volume controls (level/sampling/rate-limit) are present for changed flows.
- Retention settings are explicit for metrics/logs/traces.
- Redaction/sanitization policy is enforced at pipeline level.
- OTel SDK attribute limits are configured and truncation/drop telemetry is monitored.
- Exemplar usage is scoped to key histograms with explicit memory budget.

### Async observability
- Trace context injection/extraction exists across message boundaries.
- Producer/consumer spans follow messaging semantics.
- Retries preserve correlation identity and increment attempts.
- DLQ transitions are observable with bounded reason taxonomy.
- Lag/backlog metrics are collected from broker/platform and linked to handler telemetry.
- Batch processing uses span links where multi-parent causality exists.
- Reconciliation jobs emit run-level traces, logs, and metrics.
- Retry policy uses bounded backoff+jitter and is observable.
- Broker-specific lag/depth metrics (SQS/Kafka/RabbitMQ) are covered.
- DLQ visibility includes depth, age, and attempt-history signals.

### Operability and incident response
- Baseline dashboards can answer first-response questions without ad-hoc code changes.
- Incident burst-mode instrumentation has TTL and rollback safety.
- Runbook links exist for lag spikes, retry storms, DLQ growth, and crash analysis.
- Changes include migration notes if dashboards/alerts/queries are impacted.

## MUST / SHOULD / NEVER

### MUST
- MUST separate liveness, readiness, and startup semantics.
- MUST isolate debug endpoints from public traffic.
- MUST provide deterministic graceful shutdown and draining sequence.
- MUST enforce bounded metric cardinality and stable histogram contracts.
- MUST configure explicit sampling, retention, and redaction policies.
- MUST set SDK-level telemetry attribute limits and monitor drop/truncate behavior.
- MUST correlate async traces/logs/metrics with stable message and workflow identifiers.
- MUST keep retries, DLQ transitions, lag, and reconciliation outcomes observable.
- MUST time-bound any incident-specific observability escalation.

### SHOULD
- SHOULD centralize telemetry helpers to prevent schema drift across services.
- SHOULD prefer low-cost aggregate metrics first, then sampled traces/logs for depth.
- SHOULD keep error taxonomy short and operationally meaningful.
- SHOULD use span links for batch and fan-in async paths.
- SHOULD test crash, shutdown, and debug endpoint behavior in pre-production drills.
- SHOULD keep broker-specific observability runbooks (SQS/Kafka/RabbitMQ) with concrete triage queries.

### NEVER
- NEVER expose pprof/expvar on public ingress by default.
- NEVER use unbounded identifiers as metric labels.
- NEVER log secrets, credentials, raw tokens, or unrestricted payloads.
- NEVER enable permanent high-detail incident mode.
- NEVER ship async code where retries/DLQ exist but are not observable.
- NEVER claim incident readiness if first-response correlation cannot be done within existing telemetry.
- NEVER use `tenant_id` as default metric-label dimension in multi-tenant systems.
- NEVER ACK/commit async work before durable side effects are completed.
