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
Load references lazily. Read only the files relevant to the current spec question:

- `references/signal-contract-matrix.md` for component-by-component signal matrices across logs, metrics, traces, correlation, owners, and validation.
- `references/trace-context-and-correlation.md` for W3C Trace Context, baggage, request IDs, async correlation IDs, span links, and cross-boundary propagation.
- `references/metrics-cardinality-and-cost.md` for metric names, label budgets, histograms, cardinality traps, cost controls, retention, and "logs/traces instead of metrics" decisions.
- `references/structured-logs-and-privacy.md` for structured log event design, redaction, PII/secrets handling, sanitized DB/query data, and log-to-trace correlation.
- `references/sli-slo-error-budget-and-alerting.md` for SLI ratios, SLO windows, error budget policy, multi-window burn-rate alerting, low-traffic event floors, and alert ownership.
- `references/async-dlq-lag-and-reconciliation-observability.md` for producer/consumer/retry/DLQ/redrive, lag/backlog/oldest-age, idempotency, and reconciliation telemetry.
- `references/runtime-diagnostics-and-debug-endpoints.md` for `/livez`, `/readyz`, `/startupz`, pprof, expvar, admin listeners, shutdown drain/flush, and incident-only diagnostics.

When extending examples or source guidance, research primary sources first. Prefer OpenTelemetry docs and semantic conventions, W3C Trace Context, Google SRE book/workbook chapters, Go and Kubernetes docs, Prometheus docs, and official cloud observability docs. Keep source links in the relevant reference file.

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
