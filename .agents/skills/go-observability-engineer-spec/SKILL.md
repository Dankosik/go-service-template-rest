---
name: go-observability-engineer-spec
description: "Design observability-first specifications for Go services. Use when planning or revising telemetry behavior and you need explicit log, metric, trace, and correlation rules, SLI/SLO and error-budget policy, debuggability contracts, async observability requirements, and telemetry-cost guardrails. Skip when the task is a local code fix, endpoint payload design, physical SQL schema/migration scripting, CI/container setup, or low-level instrumentation tuning."
---

# Go Observability Engineer Spec

## Purpose
Make diagnosability, alertability, and telemetry cost explicit before coding so changed runtime behavior is observable, operable, privacy-safe, and safe to roll out.

## Specialist Stance
- Treat observability as an operator decision contract. Every log, metric, span, correlation field, dashboard, or alert must answer a concrete operational question and support a named response.
- Prefer bounded, stable, privacy-safe telemetry over "log more", raw identifiers, trace IDs as metric labels, or dashboard sprawl.
- Separate logical outcomes from attempts, retries, fallbacks, and transport details so SLOs and alerts reflect user or workflow impact.
- Hand off API, data, security, reliability, and delivery design when observability is only a dependent concern.

## Scope
Use this skill to specify or review:
- logs, metrics, traces, and correlation contracts
- service/resource identity and cross-signal consistency
- SLI/SLO, error-budget, alert, dashboard, and runbook expectations
- async retry, DLQ, lag, backlog, and reconciliation observability
- runtime diagnostics, probes, pprof/expvar/debug access, shutdown telemetry, sampling, retention, cardinality, cost, and privacy controls

## Boundaries
Do not:
- recommend telemetry that cannot answer a concrete operator question
- turn observability into exhaustive data collection
- use unbounded identifiers such as request IDs, trace IDs, user IDs, raw tenant IDs, message IDs, raw paths, raw queries, or error strings as metric labels
- make logs the alerting source of truth when a bounded metric can represent the same operator decision
- leave ownership for alerts, dashboards, runbooks, debug endpoints, or cost controls unclear
- drift into implementation tuning, API redesign, or database schema design as the primary output

## Workflow
1. Frame the changed runtime paths: API handlers, clients, database/cache calls, producers, consumers, jobs, reconcilers, shutdown, and debug surfaces.
2. Identify the operator decisions for each path: detect user impact, route an alert, isolate a dependency, decide rollback/degrade/retry/redrive, prove recovery, or investigate a specific entity.
3. Choose the cheapest sufficient signal for each decision:
   - metrics for SLOs, trends, alerting, capacity, backlog, and bounded aggregation
   - traces for causality, cross-boundary timing, fan-out, retries, and async linkage
   - logs for high-cardinality forensic detail that should not become a metric label
   - correlation fields only when they preserve a debugging path without leaking sensitive data
4. Specify the signal contract: names, units, attributes, cardinality limits, event boundaries, sampling, retention, owner, runbook/dashboard links, and validation evidence.
5. Record selected and rejected options. Reject "log more", raw IDs in metrics, generic dashboards, public debug endpoints, and paging alerts with no operator action.
6. Call out assumptions, blockers, and reopen conditions when signal quality, convention stability, privacy, or cost tradeoffs are not yet proven.

## Reference Selection
Load references lazily. Load at most one reference by default unless the task clearly spans multiple independent decision pressures, such as async DLQ behavior plus SLO alert policy. Treat references as compact rubrics and example banks, not exhaustive checklists or documentation dumps.

Pick the reference whose symptom most directly changes the model's decision:

| Reference | Load Symptom | Behavior Change |
| --- | --- | --- |
| `references/signal-contract-matrix.md` | Broad observability section, telemetry contract, or component-by-component coverage. | Makes the model write an operator-decision matrix per changed runtime path instead of disconnected lists of "add logs, metrics, and traces." |
| `references/resource-identity-and-semantic-conventions.md` | Service identity, resource attributes, semantic conventions, metric/span names, instrumentation scope, or cross-signal naming drift. | Makes the model reuse stable OpenTelemetry/resource conventions and mark unstable conventions explicitly instead of inventing custom names or inconsistent labels. |
| `references/metrics-cardinality-and-cost.md` | Metric names, labels, histograms, retention, cost, dashboards, or any draft label using IDs, paths, error strings, or request context. | Makes the model choose bounded aggregations or move detail to logs/traces instead of creating high-cardinality or costly metric labels. |
| `references/structured-logs-and-privacy.md` | Structured log event design, redaction, PII/secrets, request or DB/query data, log-to-trace pivots, or support fields. | Makes the model design allowlisted forensic logs with privacy controls instead of raw body/header logging or log-scrape alerting. |
| `references/trace-context-and-correlation.md` | Request IDs, W3C Trace Context, baggage, async correlation, span links, retry/DLQ/redrive correlation, or cross-service propagation. | Makes the model preserve safe trace/correlation continuity and use links where lineage is not single-parent instead of forcing IDs into metric labels or losing retry ancestry. |
| `references/sli-slo-error-budget-and-alerting.md` | SLIs, SLO windows, error budgets, burn-rate alerts, event floors, alert ownership, runbooks, dashboards, or release/degradation policy. | Makes the model define good/total events and proportional operator response instead of raw threshold pages or unactionable dashboards. |
| `references/async-dlq-lag-and-reconciliation-observability.md` | Producers, consumers, queues, retries, DLQs, redrive, backlog, lag, oldest age, idempotency, scheduled jobs, or reconcilers. | Makes the model separate attempt, logical completion, freshness, DLQ, redrive, and reconciliation visibility instead of treating "consumer lag" as sufficient. |
| `references/runtime-diagnostics-and-debug-endpoints.md` | Health probes, graceful shutdown, pprof, expvar, runtime diagnostics, admin/debug listener policy, crash diagnostics, or telemetry flush behavior. | Makes the model separate orchestration and incident-debug decisions with access controls instead of sharing probe semantics or exposing debug surfaces. |

When extending standards-sensitive examples, research primary sources first. Prefer OpenTelemetry docs and semantic conventions, W3C Trace Context, Google SRE book/workbook chapters, Go and Kubernetes docs, Prometheus docs, and official cloud observability docs. Do not keep source-link dumps; keep only tiny canonical verification pointers when freshness or standards status materially changes the guidance.

## Decision Quality Bar
For every material recommendation, include:
- operator question and decision
- selected signal and why it is the cheapest sufficient option
- at least one rejected option and why it is unsafe, noisy, costly, or misleading
- log, metric, trace, and correlation deltas where applicable
- cardinality, privacy, sampling, retention, and cost controls
- SLI/SLO, alerting, dashboard, runbook, or verification impact
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing an observability spec, cover:
- signal contract matrix by component or workflow stage
- SLI/SLO, error budget, burn-rate, and alert routing policy
- dashboard and runbook contract tied to alert decisions
- debug endpoint, health probe, shutdown, and runtime diagnostics contract
- telemetry cost, cardinality, sampling, retention, and privacy controls
- async correlation, retry, DLQ, lag, redrive, and reconciliation observability when relevant

## Escalate Or Reject
Escalate or reject the plan if it includes:
- a changed critical runtime path with no success/failure signal contract
- high-cardinality metric dimensions without an explicit bounded exception
- SLI/SLO targets without `good_events`, `total_events`, exclusions, and measurement source
- paging alerts without event floors, runbook/dashboard links, owner, or operator action
- async retries or DLQ behavior without lag, age, retry, redrive, and reconciliation visibility
- public debug endpoints, shared liveness/readiness semantics, or missing shutdown telemetry flush
- telemetry that risks leaking secrets, tokens, credentials, PII, or raw tenant/user identifiers
- critical observability decisions deferred to implementation without a recorded proof obligation
