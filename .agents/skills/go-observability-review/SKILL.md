---
name: go-observability-review
description: "Use when changed Go or operational artifacts affect logs, metrics, traces, correlation, SLI/SLO signals, alerts, diagnostics, telemetry privacy, cardinality, sampling, or cost; Own observability conformance and evidence; Skip when the primary issue is unset telemetry policy, service reliability, performance behavior, or delivery governance."
---

# Go Observability Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Target, Boundary, And Invariants

Review changed runtime paths and operational artifacts so telemetry answers real operator decisions, remains privacy-safe and bounded-cost, and represents accepted outcomes rather than instrumentation accidents. Do not redesign API, data, reliability, security, performance, or delivery policy when observability is only the consequence.

1. Every changed signal names its operator question and uses the cheapest sufficient form: bounded metrics for aggregation/alerting, traces for causality, and structured logs for forensic detail.
2. Logs use stable events and allowlisted, redacted fields; secrets, tokens, PII, bodies, headers, queries, identifiers, and error text follow explicit privacy, access, and retention rules.
3. Metrics use stable names, units, and bounded taxonomies; raw paths, IDs, timestamps, trace context, user input, and error strings do not become labels without an accepted bounded exception.
4. Traces and correlation preserve safe cross-service, retry, batch, async, and dependency lineage without unsafe baggage propagation; logs, metrics, and spans share coherent resource and operation identity.
5. Signals distinguish attempts, durable handoff, admission, retries, final logical outcomes, freshness, DLQ/redrive, and reconciliation where those are separate promises.
6. SLIs, SLOs, alerts, dashboards, and runbooks measure the accepted user or workflow outcome and provide event floors, ownership, investigation entry points, and actionable response.
7. Diagnostics, probes, and debug endpoints preserve their accepted exposure and access policy; shutdown records bounded drain and exporter flush success/failure when touched.
8. Sampling, retention, label growth, and telemetry volume have explicit proof and cost consequences; high-cardinality detail moves to privacy-safe logs/traces instead of aggregate labels.

## Symptom-Driven References

| Symptom | Load |
| --- | --- |
| Several runtime paths or signal types lack an operator-decision contract. | [signal-contract-matrix.md](../go-observability-spec/references/signal-contract-matrix.md) |
| Structured logs, redaction, sensitive fields, raw payloads, or log-based alerting change. | [structured-logs-and-privacy.md](../go-observability-spec/references/structured-logs-and-privacy.md) |
| Trace propagation, baggage, request IDs, async lineage, retry, batch, or redrive correlation changes. | [trace-context-and-correlation.md](../go-observability-spec/references/trace-context-and-correlation.md) |
| Metric labels, histograms, raw identifiers, cardinality, retention, or telemetry cost changes. | [metrics-cardinality-and-cost.md](../go-observability-spec/references/metrics-cardinality-and-cost.md) |
| Resource attributes, semantic conventions, names, units, or cross-signal identity drift. | [resource-identity-and-semantic-conventions.md](../go-observability-spec/references/resource-identity-and-semantic-conventions.md) |
| SLI/SLO math, logical outcomes, error budgets, alert noise, dashboards, or runbooks change. | [sli-slo-error-budget-and-alerting.md](../go-observability-spec/references/sli-slo-error-budget-and-alerting.md) |
| Probes, pprof/expvar, debug exposure, shutdown diagnostics, or exporter flush changes. | [runtime-diagnostics-and-debug-endpoints.md](../go-observability-spec/references/runtime-diagnostics-and-debug-endpoints.md) |
| Producers, consumers, retries, DLQ/redrive, backlog age, jobs, or reconciliation change. | [async-dlq-lag-and-reconciliation-observability.md](../go-observability-spec/references/async-dlq-lag-and-reconciliation-observability.md) |

## Findings And Escalation

Inspect signal creation and export, correlation boundaries, outcome classification, dashboards/alerts/runbooks, diagnostics exposure, async paths, shutdown/flush, tests, and runtime evidence. Each finding adds the violated observability expectation, operator/privacy/cardinality/cost/diagnosability impact, and focused proof. `critical` includes secret/PII exposure or high-impact loss of critical detection; `high` includes missing critical-path outcome visibility, unactionable paging, or unbounded cardinality.

Escalate an unset or changed signal/SLO/alert contract to `go-observability-spec`; sensitive-data or debug access policy to `go-security-spec`; failure behavior to `go-reliability-spec`; release-gate evidence to `go-delivery-platform-review` or `go-delivery-platform-spec`; and benchmark/profile interpretation to `go-performance-review`. Stop instead of inventing policy or duplicating the primary owner.
