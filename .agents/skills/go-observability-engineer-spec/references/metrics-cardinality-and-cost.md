# Metrics Cardinality And Cost

## Behavior Change Thesis
When loaded for metric naming, labels, histograms, dashboards, retention, or cost symptoms, this file makes the model choose bounded aggregations or logs/traces for detail instead of likely mistake high-cardinality metric labels from IDs, raw paths, trace context, or error strings.

## When To Load
Load this when a draft proposes or may imply metric labels from user IDs, tenant IDs, account IDs, raw paths, request IDs, trace IDs, message IDs, job IDs, raw error messages, timestamps, dynamic destinations, or "whatever helps debugging."

## Decision Rubric
- A metric label survives only if it is bounded, stable, queryable during an incident, and meaningful under aggregation.
- Prefer route templates, status classes, bounded `outcome`, bounded `error.type`, retry reason taxonomies, service class, region, and dependency catalog names.
- Reject labels whose values can be created by users, traffic, data shape, timestamps, exception text, or generated IDs.
- Keep SLO and paging metrics separate from forensic identifiers. Use exemplars, traces, or structured logs for representative request or entity pivots.
- Align histogram buckets with SLO or operational decision cut points; do not add high-resolution buckets because the backend permits them.
- Treat new labels on high-traffic paths as a cost and retention decision, not a harmless schema change.

## Imitate
- `http.server.request.duration` with route template, method, status code or status class, and bounded `error.type`.
  Copy the route-template and bounded-error shape, not raw URL or exception text.
- `outbound_client_request_duration_seconds{dependency="fraud", method="Decide", outcome="timeout"}` when dependency and method are fixed service-catalog values.
  Copy the catalog-backed dimensions.
- `worker_backlog_oldest_age_seconds{worker="invoice-consumer", queue="invoice"}` for freshness risk.
  Copy the "age plus ownership" decision, not just queue depth.
- `reconciliation_drift_items_total{result="repaired", partner="adyen"}` only when partner values are controlled and operationally owned.
  Copy the bounded partner exception rule.

## Reject
- `request_duration_seconds{path="/accounts/123/reconcile", user_id="u9", trace_id="..."}` because every label can explode cardinality or leak identifiers.
- `error_total{message="sql: connection refused: host 10.2.4.9:5432"}` because raw messages drift and can contain sensitive data.
- `job_last_run{timestamp="2026-04-10T12:00:00Z"}` because time belongs in a sample value or event, not a label.
- A custom metric per tenant or account when operators only act by tier, region, product, or service class.
- Copying span attributes into metric labels without a separate metric-label budget.

## Agent Traps
- Calling a label "low cardinality" because today's dataset is small. Judge the value space, not current volume.
- Forgetting label multiplication: route x status x dependency x region x version x instance can turn one extra label into many series.
- Treating partition labels as harmless. Use them only when operators act at partition level.
- Keeping build ID or version labels forever after a rollout. Bound and expire rollout dimensions.
- Designing dashboard variables from entity IDs, which quietly recreates the same cardinality problem.

## Validation Shape
- List every proposed metric label and mark its value source as fixed taxonomy, service catalog, deployment catalog, or rejected dynamic value.
- For high-traffic paths, require a cost/cardinality review or an accepted-risk note for any new label.
- Check that metrics meant for SLO or alerting can be aggregated without losing their meaning.

## Canonical Verification Pointer
Use the current OpenTelemetry and Prometheus metric naming/semantic-convention docs when a name or unit choice depends on standards status.
