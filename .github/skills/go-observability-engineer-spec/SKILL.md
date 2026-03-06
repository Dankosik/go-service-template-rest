---
name: go-observability-engineer-spec
description: "Design observability-first specifications for Go services. Use when planning or revising telemetry behavior and you need explicit log, metric, trace, and correlation rules, SLI/SLO and error-budget policy, debuggability contracts, async observability requirements, and telemetry-cost guardrails. Skip when the task is a local code fix, endpoint payload design, physical SQL schema/migration scripting, CI/container setup, or low-level instrumentation tuning."
---

# Go Observability Engineer Spec

## Purpose
Make diagnosability, alertability, and telemetry cost explicit before coding so that changed runtime behavior is observable, operable, and safe to roll out.

## Scope
Use this skill to define or review telemetry behavior: logs, metrics, traces, correlation, debuggability, async observability, SLI/SLO expectations, and telemetry-cost guardrails.

## Boundaries
Do not:
- recommend telemetry that cannot answer a concrete operational question
- turn observability into exhaustive data collection without cardinality, cost, and actionability controls
- drift into generic implementation tuning or unrelated API/data redesign as the primary output
- leave ownership for alerting, dashboards, or failure diagnosis unclear

## Escalate When
Escalate if critical paths lack observable success/failure signals, correlation cannot be preserved across boundaries, SLO/alert expectations are undefined, or telemetry cost and signal quality cannot be balanced responsibly.

## Core Defaults
- Treat observability coverage for changed critical paths as blocking by default.
- Use evidence-first design: every signal should answer an operational question for a real consumer.
- Prefer bounded, low-cardinality telemetry with stable semantics over ad hoc detail collection.
- Prefer the cheapest signal that answers the question: metrics for trends and alerts, traces for causality, logs for high-cardinality diagnosis.
- Treat missing critical observability facts as explicit assumptions or blockers.

## Expertise

### Telemetry Signal Contract
- Require OTel bootstrap in the composition root with resource identity, tracer provider, meter provider, and propagators.
- Make service identity fields mandatory across signals, including `service.name`, `service.version`, and deployment environment.
- Require baseline signal coverage for:
  - API handlers
  - outbound clients
  - DB access
  - workers, producers, and consumers
  - scheduled jobs and reconcilers
- Require RED metrics plus saturation or backlog signals where applicable.
- Use structured logs with stable common keys and a bounded `error.type` taxonomy.
- Keep span names low-cardinality and record errors consistently.

### Correlation And Propagation
- Default to W3C Trace Context plus Baggage.
- Require request ID generation and propagation for sync flows.
- Require stable `correlation_id`, `message_id`, and `attempt` for async flows.
- Preserve correlation continuity across retries and DLQ transitions.
- Use span links for batch or fan-in processing instead of forcing misleading single-parent lineage.

### SLI, SLO, Error Budget, And Alerting
- Define each SLI as an explicit ratio with `good_events`, `total_events`, and exclusions.
- Default to a 28-day SLO window unless there is a strong reason not to.
- Use service-class and criticality-aware targets for APIs, workers, and async consumers.
- Define budget states such as `green`, `yellow`, `orange`, and `red`, along with release/degradation implications.
- Use multi-window burn-rate rules with event-floor guards for low-traffic services.
- Every paging alert should have an owner, a runbook, and a dashboard.

### Debuggability And Runtime Diagnostics
- Keep probe semantics separate:
  - `/livez` for restart decisions
  - `/readyz` for traffic admission
  - `/startupz` for startup completion
- Make graceful shutdown observable: readiness fail, drain period, telemetry flush, bounded exit.
- Isolate admin and debug endpoints on a separate listener; do not expose them publicly by default.
- Keep pprof, expvar, or similar debug controls behind kill switches and time-bound incident activation.
- Make crash diagnostics explicit, including tracebacks, crash metadata, retention, and privacy constraints.

### Telemetry Cost And Cardinality
- Treat unbounded metric labels as a blocker.
- Prohibit request IDs, trace IDs, user IDs, raw paths, message IDs, and similar unbounded identifiers in metric labels.
- Use fixed histogram strategies with meaningful SLO cut points.
- Define trace and log sampling defaults by environment, plus incident burst mode with auto-expire behavior.
- Make retention and cost impact explicit when telemetry volume changes.
- Set attribute limits and observe dropped or truncated attributes.

### Async Observability
- Require trace coverage for producer send, consumer processing, retries, DLQ transitions, lag growth, and reconciliation runs.
- Require metrics for outcomes, retries, DLQ, lag, backlog, oldest age, and idempotency decisions.
- Keep retry reason taxonomy bounded and operationally meaningful.
- Require DLQ depth/age visibility and redrive observability.
- Require drift and repair telemetry for reconciliation.

### Privacy, Security, And Cross-Domain Alignment
- Require redaction and sanitization policy for logs, traces, URLs, query data, and DB statements.
- Prohibit secrets, tokens, credentials, and PII leakage in telemetry by default.
- Treat baggage as allowlisted data only.
- Keep telemetry dimensions tenant-safe and do not use tenant or user IDs as default metric labels.
- When API, distributed, data/cache, or release behavior depends on observability, make those interfaces explicit.

## Decision Quality Bar
For every major observability recommendation, include:
- the operational question being answered
- at least two viable options
- the selected option and at least one explicit rejection reason
- signal contract deltas across logs, metrics, traces, and correlation
- cardinality and cost impact with controls
- SLI/SLO/burn-rate/alerting impact
- runtime verification obligations
- assumptions, blockers, and reopen conditions

## Deliverable Shape
When writing the observability spec or review, cover:
- signal contract matrix by component
- SLI/SLO, error budget, and burn-rate policy
- alert routing, dashboards, and runbooks
- diagnostics and debuggability contract
- telemetry cost, cardinality, sampling, and retention controls
- async correlation, retry, DLQ, lag, and reconciliation observability

## Escalate Or Reject
- missing telemetry contract for a changed critical runtime path
- high-cardinality metric dimensions without a justified exception
- SLI/SLO targets without clear `good/total` semantics or exclusions
- burn-rate paging without event floors or without actionable runbook/dashboard linkage
- async retries or DLQ configured but not observable
- public exposure of debug endpoints or missing shutdown telemetry-flush contract
- telemetry changes that risk leaking secrets or PII
- critical observability decisions deferred to implementation
