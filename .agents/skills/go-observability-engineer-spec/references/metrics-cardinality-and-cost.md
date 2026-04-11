# Metrics Cardinality And Cost

## When To Load This
Load this reference when choosing metric names, labels, histogram buckets, aggregation levels, retention, sampling, dashboards, alert inputs, cost controls, or when a draft proposes user IDs, tenant IDs, paths, trace IDs, request IDs, message IDs, or error strings as metric labels.

## Operational Questions
- Which aggregation must the operator run during an incident?
- Can `sum()` or `avg()` across all label dimensions still mean something?
- Is every label bounded, stable, documented, and useful for alerting or diagnosis?
- Could an attacker or traffic pattern create new label values without limit?
- What is the cost and retention impact of adding this metric or label?
- Should this detail live in logs/traces instead of metrics?

## Good Telemetry Examples
- `http.server.request.duration` histogram with route template, method, status code, and low-cardinality `error.type`.
- `outbound_client_request_duration_seconds{dependency="fraud", method="Decide", outcome="timeout"}` when dependency and method are fixed service catalog values.
- `worker_backlog_oldest_age_seconds{worker="invoice-consumer", queue="invoice"}` because it answers whether queued work is violating freshness.
- `reconciliation_drift_items_total{result="repaired", partner="adyen"}` when partner names are bounded and operator-owned.
- Histogram buckets aligned with SLO cut points, not arbitrary high-resolution buckets that nobody queries.

## Bad Telemetry Examples
- `request_duration_seconds{path="/accounts/123/reconcile", user_id="u9", trace_id="..."}`.
- `error_total{message="sql: connection refused: host 10.2.4.9:5432"}` instead of `error.type="db_unavailable"`.
- `job_last_run{timestamp="2026-04-10T12:00:00Z"}`.
- A custom metric per tenant or per account when the operator only needs tier, region, or service class.
- Copying all span attributes into metric labels because the backend can technically ingest them.

## Cardinality Traps
- High-cardinality labels multiply together; one "mostly fine" label can become expensive when combined with route, status, dependency, region, version, and instance.
- Raw paths are not route templates. Use framework route templates such as `/users/{id}` only when available.
- Error strings drift over time and often contain IDs, addresses, SQL details, or user input.
- Partition labels can be useful for Kafka-style debugging but multiply with topic, consumer group, and region. Use them only where operators need partition-level action.
- Per-version labels are useful during rollout diagnosis, but stale versions and build IDs can create churn. Bound and expire them.

## Selected And Rejected Options
- Select low-cardinality route templates over raw paths.
- Select bounded taxonomies such as `outcome`, `status_class`, `error.type`, `retry_reason`, and `service_class` over raw exception strings.
- Select metrics for page and SLO inputs only when they can be aggregated predictably.
- Select logs/traces for request IDs, trace IDs, entity IDs, message IDs, and support handles.
- Select backend cost checks or metrics-management review when adding new labels to high-traffic paths.
- Reject one metric per customer, dynamic metric names, raw-label extraction from logs, and labels that exist only for dashboard curiosity.

## Exa Source Links
- OpenTelemetry HTTP metrics semantic conventions: https://opentelemetry.io/docs/specs/semconv/http/http-metrics/
- OpenTelemetry metrics semantic conventions: https://opentelemetry.io/docs/specs/semconv/general/metrics
- Prometheus metric and label naming: https://prometheus.io/docs/practices/naming/
- Google Cloud log-based metric label constraints and cost warning: https://cloud.google.com/logging/docs/logs-based-metrics/labels
- Google Cloud Metrics Management cardinality and cost guidance: https://cloud.google.com/monitoring/docs/metrics-management
- Google Cloud Managed Service for Prometheus cost controls: https://docs.cloud.google.com/stackdriver/docs/managed-prometheus/cost-controls
