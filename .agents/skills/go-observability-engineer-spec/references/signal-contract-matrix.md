# Signal Contract Matrix

## When To Load This
Load this reference when an observability spec needs a component-by-component contract across logs, metrics, traces, correlation, SLOs, alerts, owners, and validation.

This is the first reference to load when the task asks for a "signal matrix", "telemetry contract", "observability section", "spec.md-ready guidance", or cross-signal coverage for APIs, clients, DB/cache, producers, consumers, jobs, reconcilers, or shutdown.

## Operational Questions
- Which operator decision does this signal support: page, rollback, degrade, retry, redrive, isolate a dependency, prove recovery, or investigate one entity?
- Is the signal measuring a logical outcome, an attempt, a dependency call, or background completion?
- Can the operator aggregate it safely by service, route, dependency, operation, status class, region, or environment?
- Which signal should page, which should explain cause, and which should provide forensic detail?
- What does the operator do when the signal changes?

## Contract Template
Use a compact matrix rather than disconnected lists:

| Component or path | Operator decision | Metrics | Traces | Logs | Correlation | Alert/dashboard/runbook | Controls |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `POST /v1/payouts` | Detect user-impacting create failures and decide rollback/degrade | `http.server.request.duration` by `http.route`, method, status, bounded `error.type`; business outcome counter by `outcome` | Server span named from route template; child spans for DB, fraud gRPC, idempotency write | One completion event with route template, outcome, bounded error type, retry handoff | `trace_id`, `span_id`, generated request ID | SLO burn dashboard links route-level latency/error panels | No raw path, tenant ID, request body, or request ID as metric labels |
| Fraud gRPC call | Decide if dependency timeout should trigger async fallback or dependency escalation | Client latency and outcome by dependency, method, timeout class | Client span with deadline and status; retry spans separate from logical request | Timeout/fallback handoff event with bounded reason | Same trace/request ID; retry job correlation ID | Dependency panel from payout alert | Bounded `error.type`; no raw exception text labels |
| Reconciler run | Decide whether drift is increasing and whether manual repair/redrive is needed | Drift found/repaired counters, run duration, oldest unresolved drift age | Run span plus partner fan-out spans | Run summary with sample IDs only if privacy-safe and not in metrics | Run ID in logs/traces only | Ticket or page based on error-budget and age thresholds | Account IDs stay in logs/traces with redaction policy, not metric labels |

## Good Telemetry Examples
- Metric: `http.server.request.duration` with route template, method, status code, and bounded `error.type` lets an operator detect route-level latency/error impact without raw paths.
- Metric: `worker.retry_attempts_total{worker="invoice-consumer", reason="optimistic_conflict"}` lets an operator distinguish benign concurrency from poison-message failure if `reason` is a bounded taxonomy.
- Trace: `POST /v1/payouts` -> `grpc fraud.v1.Decide` -> `postgres payouts.insert` shows the operator where time was spent.
- Log: `payout.create.completed` with `outcome="accepted_async_retry"` explains the 202 handoff without making retry attempts count as completed payouts.
- Dashboard: one SLO landing panel, one dependency panel, one saturation/backlog panel, each linked from an alert and runbook.

## Bad Telemetry Examples
- "Add logs everywhere" because it does not state the operator decision, owner, event boundary, or privacy controls.
- `requests_total{path="/v1/payouts/po_123", tenant_id="t_456", request_id="..."}` because all three labels can explode cardinality and leak identifiers.
- A dashboard with every exported metric but no alert path, runbook, or first-response decision.
- A single `success=false` field for an API that returns `202 Accepted` and later completes asynchronously, because it collapses logical completion and admission.
- Treating DB timeout attempts and final user-visible request failure as the same metric.

## Cardinality Traps
- Raw URL path, URL query, route parameters, account IDs, tenant IDs, user IDs, request IDs, trace IDs, message IDs, job IDs, and raw error strings.
- Dynamic destination names such as per-user topics or per-account queues when no low-cardinality template exists.
- Status or error labels built from full exception messages instead of a bounded `error.type` taxonomy.
- Per-dependency labels when the dependency name is user-controlled or discovered from input.

## Selected And Rejected Options
- Select metrics for alerting and SLO math when dimensions are bounded and aggregations are meaningful.
- Select traces for causality and fan-out because they preserve timing and dependency relationships without turning IDs into labels.
- Select logs for high-cardinality investigation fields, especially entity IDs or request details that operators need after an alert.
- Reject logs as the only SLO source when the code can emit the same classification as a metric at decision time.
- Reject dashboard sprawl. Add a dashboard only if an alert, on-call runbook, or routine operations decision links to it.

## Exa Source Links
- OpenTelemetry Semantic Conventions: https://opentelemetry.io/docs/concepts/semantic-conventions/
- OpenTelemetry HTTP metrics semantic conventions: https://opentelemetry.io/docs/specs/semconv/http/http-metrics/
- OpenTelemetry HTTP spans semantic conventions: https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/http/
- Google SRE Workbook, Monitoring: https://sre.google/workbook/monitoring/
- Google SRE Workbook, Alerting on SLOs: https://sre.google/workbook/alerting-on-slos/
