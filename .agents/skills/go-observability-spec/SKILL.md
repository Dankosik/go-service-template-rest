---
name: go-observability-spec
description: "Use when telemetry behavior, correlation, SLI/SLO and error-budget signals, alerts, diagnostics, async visibility, privacy, cardinality, sampling, or cost must be decided before coding; Own observability policy and proof obligations; Skip when the primary decision is service resilience, performance budgets, delivery governance, or instrumentation implementation."
---

# Go Observability Spec

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Outcome And Boundary

Define an operator decision contract that makes changed runtime behavior diagnosable, alertable, privacy-safe, and cost-bounded. Own resource identity and semantic conventions, logs, metrics, traces, correlation, SLI/SLO and error-budget signals, alerts, dashboards/runbooks, runtime diagnostics, and async/DLQ/reconciliation visibility.

Consume accepted domain, API, security, reliability, and delivery policy. Do not invent user outcomes, retry/degradation behavior, access policy, rollout policy, implementation wiring, API shape, or data schema; when one is unset, stop at the observability consequence and name its owner.

## Owned Core

- Frame each changed handler, client, database/cache call, producer, consumer, job, reconciler, shutdown path, and debug surface by the operator question and action it must support: detect impact, route an alert, isolate a dependency, decide rollback/degrade/retry/redrive, prove recovery, or investigate an entity.
- Choose the cheapest sufficient signal: bounded metrics for SLOs, trends, alerts, capacity, and backlog; traces for causality, cross-boundary timing, fan-out, retries, and async lineage; allowlisted structured logs for forensic detail; safe correlation fields for pivots that do not leak data.
- Specify stable resource identity, instrumentation scope, semantic-convention status, names, units, event boundaries, attributes, owners, and cross-signal consistency. Mark unstable conventions rather than silently inventing incompatible names.
- Bound privacy, cardinality, sampling, retention, and cost. Never use request/trace/message/user IDs, raw tenant IDs, raw paths/queries, or error strings as metric labels; do not log bodies, headers, credentials, tokens, PII, or identifiers without an explicit allowlist and redaction policy; prefer a bounded metric over log-scrape alerting.
- Separate user or workflow outcomes from transport attempts, retries, fallbacks, batches, and redrives. Preserve trace context safely, use span links for non-single-parent lineage, and keep high-cardinality correlation in logs/traces rather than metrics.
- Define every SLI with `good_events`, `total_events`, exclusions, measurement source, and window. Tie SLOs and error budgets to multi-window/burn-rate or low-traffic-safe alert semantics, event floors, owner, operator action, dashboard, runbook, and release/degradation consequence.
- Cover async depth and oldest age, lag/freshness, attempts, deduplication, DLQ entry, redrive, logical completion, and reconciliation. Separately define startup/liveness/readiness, drain/shutdown and telemetry flush, crash diagnostics, and authenticated non-public pprof/expvar/debug access.
- For each signal contract record validation evidence and assumptions. When a real live fork exists, include at least one rejected option and why it is noisy, costly, unsafe, or misleading; reject generic “log more,” dashboard sprawl, public debug endpoints, and pages with no action.

## Symptom-Driven References

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Broad telemetry contract or component-by-component coverage | [signal-contract-matrix.md](references/signal-contract-matrix.md) | Build an operator-decision matrix instead of disconnected signal lists. |
| Service identity, resource attributes, semantic conventions, names, or instrumentation scope | [resource-identity-and-semantic-conventions.md](references/resource-identity-and-semantic-conventions.md) | Reuse stable resource and OpenTelemetry conventions and expose unstable status. |
| Metric labels, histograms, retention, cost, dashboards, IDs, paths, or error strings | [metrics-cardinality-and-cost.md](references/metrics-cardinality-and-cost.md) | Bound aggregation and move forensic detail out of metrics. |
| Structured events, redaction, PII/secrets, request/query data, or support pivots | [structured-logs-and-privacy.md](references/structured-logs-and-privacy.md) | Define allowlisted forensic logs and privacy controls. |
| W3C Trace Context, baggage, async linkage, retries, DLQ/redrive, or propagation | [trace-context-and-correlation.md](references/trace-context-and-correlation.md) | Preserve safe correlation and non-single-parent lineage. |
| SLIs/SLOs, error budgets, burn rates, event floors, alerts, dashboards, or runbooks | [sli-slo-error-budget-and-alerting.md](references/sli-slo-error-budget-and-alerting.md) | Define good/total events and proportional operator response. |
| Queues, retries, DLQs, backlog, lag, oldest age, idempotency, jobs, or reconcilers | [async-dlq-lag-and-reconciliation-observability.md](references/async-dlq-lag-and-reconciliation-observability.md) | Separate attempt, completion, freshness, redrive, and reconciliation visibility. |
| Health probes, shutdown, pprof/expvar, debug listeners, crash data, or telemetry flush | [runtime-diagnostics-and-debug-endpoints.md](references/runtime-diagnostics-and-debug-endpoints.md) | Separate orchestration and incident-debug decisions with access controls. |

When extending standards-sensitive examples, verify primary OpenTelemetry, W3C Trace Context, Prometheus, Go, Kubernetes, SRE, or cloud-provider sources; retain only the small freshness-sensitive pointer needed by the contract.

## Return And Stop

Return the triggered signal matrix; SLI/SLO, error-budget, alert, dashboard, and runbook policy; diagnostics/probe/shutdown contract; async visibility; cost/cardinality/sampling/retention/privacy controls; validation evidence; and only forced neighboring consequences.

Block on a critical path without success/failure signals; unbounded dimensions without an approved exception; undefined SLI event semantics; a page without floor, owner, action, runbook, or dashboard; retries/DLQ without age, retry, redrive, completion, and reconciliation visibility; unsafe debug access or telemetry data; missing shutdown flush; or a critical observability decision deferred to instrumentation without proof.
