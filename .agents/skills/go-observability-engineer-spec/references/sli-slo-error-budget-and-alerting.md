# SLI/SLO, Error Budget, And Alerting

## When To Load This
Load this reference when the spec needs user-impacting SLIs, SLO windows, error budgets, burn-rate alerts, alert severity, low-traffic event floors, runbook/dashboard ownership, or release/degradation policy tied to budget state.

## Operational Questions
- What user or workflow promise is being measured?
- What are `good_events`, `total_events`, and exclusions?
- Is the SLI measured at admission, logical completion, delivery, freshness, or downstream workflow completion?
- Which budget burn should page a human, create a ticket, gate release, or trigger degradation?
- What event floor prevents low-traffic noise?
- Which dashboard and runbook does the alert open, and who owns it?

## Good Telemetry Examples
- Availability SLI: `good_events = POST /v1/payouts completed synchronously or accepted for durable async retry when the product promise allows async completion`; `total_events = valid payout create requests`; exclude caller validation failures.
- Async completion SLI: `good_events = invoice messages processed and completion event emitted before freshness target`; `total_events = valid invoice messages delivered to the consumer`.
- Latency SLI: `good_events = successful GET /v1/payouts/{id} responses under 200 ms`; track failed-request latency separately so fast 500s do not look healthy.
- Alert: page only when both short and long burn windows breach the selected threshold and the event floor is met; create tickets for slow burn or non-urgent budget consumption.
- Runbook: starts from SLO panel, then dependency panel, then traces/log queries for representative failing route or workflow.

## Bad Telemetry Examples
- "Pager on any 5xx > 0 in 5 minutes" for a low-QPS service.
- Average latency SLO with no percentile or good/total threshold.
- Counting `202 Accepted` as final success when the product promise is durable downstream completion.
- One SLO for all endpoints when admin, polling, SSE, and create flows have different user promises.
- Page with no runbook, no owner, no dashboard, and no defined operator action.

## Cardinality Traps
- Per-tenant, per-user, per-account, or per-message SLO labels by default.
- Route labels from raw paths instead of route templates.
- SLO metrics split by every dependency, turning user-impacting SLOs into implementation-detail alerts.
- Alert labels that include raw error messages or exception strings.
- Separate SLOs for every status code instead of a bounded good/bad classification.

## Selected And Rejected Options
- Select good/total ratio SLIs because they produce clear error-budget math and comparable tooling.
- Select a 28-day or organization-standard rolling SLO window unless the product promise, traffic pattern, or release policy requires a different window.
- Select separate SLIs for admission, final completion, stream continuity, and freshness when those promises differ.
- Select multi-window burn-rate alerts for paging when the service has enough traffic and a meaningful error budget.
- Select event floors or ticket-only alerts for low-traffic services to avoid paging on a single event without context.
- Select release/degradation policy tied to budget state only when the team can enforce it.
- Reject raw threshold alerts that are not tied to SLO impact, operator action, or proportional response.

## Exa Source Links
- Google SRE Workbook, Implementing SLOs: https://sre.google/workbook/implementing-slos/
- Google SRE Workbook, Alerting on SLOs: https://sre.google/workbook/alerting-on-slos/
- Google SRE Workbook, Monitoring: https://sre.google/workbook/monitoring/
- Google SRE Book, Practical Alerting: https://sre.google/sre-book/practical-alerting/
- OpenTelemetry HTTP metrics semantic conventions: https://opentelemetry.io/docs/specs/semconv/http/http-metrics/
