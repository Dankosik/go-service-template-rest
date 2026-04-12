---
name: go-observability-review
description: "Review Go code changes for observability correctness: structured logs, metrics, traces, correlation, SLI/SLO and alert semantics, dashboard/runbook evidence, runtime diagnostics, telemetry privacy, cardinality, sampling, and cost guardrails."
---

# Go Observability Review

## Purpose
Review changed runtime paths so telemetry remains operator-useful, privacy-safe, bounded-cardinality, and tied to real response decisions.

## Specialist Stance
- Review the operator question before the signal.
- Prefer the cheapest sufficient signal: metrics for bounded aggregation and alerting, traces for causality, logs for forensic detail.
- Treat high-cardinality labels, raw identifiers, unactionable alerts, and public debug surfaces as review risks.
- Hand off API, data, reliability, security, performance, or delivery depth when observability is only the consequence.

## Scope
- Structured logs, metric instruments/labels, traces/spans, correlation fields, and OpenTelemetry setup changes.
- SLI/SLO, alert, dashboard, runbook, runtime diagnostic, health/debug, sampling, retention, and telemetry-cost behavior.
- Async retry, DLQ, lag, redrive, reconciliation, shutdown, and degraded-mode signals.
- Telemetry privacy: secrets, tokens, PII, tenant/user identifiers, request bodies, headers, queries, and error text.

## Boundaries
Do not:
- ask to "log more" without a concrete operator decision,
- make logs the alerting source of truth when a bounded metric fits,
- approve raw request, user, tenant, trace, path, query, or error-string values as metric labels without a documented bounded exception,
- redesign API, database, reliability, security, or delivery policy as the primary review.

## Review Checklist
- Each new or changed signal answers a named operator question.
- Metrics use stable names, units, and bounded labels; forensic detail moves to logs/traces instead of labels.
- Logs are structured, allowlisted, redacted, and correlated without leaking secrets or sensitive identifiers.
- Spans preserve useful causality, retry, async, and dependency context without unsafe baggage propagation.
- Alerts have event floors, owner, runbook/dashboard path, and actionable response.
- Runtime diagnostics and debug endpoints have explicit access and exposure policy.
- Shutdown and telemetry flush behavior remain bounded when touched.

## Finding Quality Bar
Each finding should include:
- exact `file:line`,
- the violated observability expectation,
- the operator failure mode, privacy leak, cardinality/cost risk, or diagnosability gap,
- the smallest useful correction,
- validation evidence or command that should prove the fix,
- any handoff needed for non-observability ownership.

Severity is merge-risk based:
- `critical`: telemetry change can expose secrets/PII or break critical incident detection with high impact.
- `high`: changed critical path loses essential success/failure visibility, pages unactionably, or creates unbounded cardinality.
- `medium`: bounded but meaningful signal, runbook, privacy, or diagnostic weakness.
- `low`: local telemetry clarity or cost-hardening issue.

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-observability-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

## Escalate When
- safe correction changes the signal contract or SLO/alert policy (`go-observability-engineer-spec`),
- the issue is primarily sensitive-data exposure or debug-surface access (`go-security-spec` or `go-security-review`),
- failure behavior must change to be observable (`go-reliability-spec`),
- delivery gates or release dashboards own the proof (`go-devops-spec` or `go-devops-review`),
- performance measurement, benchmarks, or profiling own the proof (`go-performance-review`).
