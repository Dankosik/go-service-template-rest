---
name: go-observability
description: "Observability: Use for logs, metrics, traces, SLI/SLOs, alerts, diagnostics, privacy, cardinality, sampling, cost, or review. Own policy/operator evidence; Skip reliability, performance, or delivery."
---

# Go Observability

Telemetry is **operator evidence**: every signal exists to answer a named question an operator will ask under pressure, and a signal answering no question is pure cost.

`operator question -> signal choice -> SLI/SLO -> alert policy -> correlation -> cardinality, privacy, and cost -> proof`

Alert on symptoms users feel and let causes stay diagnosis surfaces, reached through correlation identifiers that survive hop boundaries. Cardinality is a budget spent deliberately — an unbounded label is a future outage of the telemetry itself — and every emitted field is a disclosure surface that passes the same privacy judgment as an API response.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct operator questions and signals from accepted operability outcomes, changed runtime paths, existing telemetry, dashboards, alerts, and SLI/SLO surfaces; require each signal to support a named decision within correlation, privacy, cardinality, diagnostic, and cost bounds.

## Choose The Branch

Both branches read the same [selector](references/index.md); each reference states the accepted policy together with the repository surface that enforces it.

- **Decision** — select when telemetry policy is absent or changing. Complete when shared Decision dispositions cover every operator question and signal with cost/privacy bounds explicit.
- **Review** — select when changed telemetry must conform to accepted policy. Account for every affected signal through the shared finding envelope, naming any outside boundary or proof blocker with the smallest correction and focused proof. Missing policy returns to the named Telemetry Decision owner.

This skill owns telemetry that is specific to Go and to this repository. It does not restate general practice: SLI and error-budget math, burn-rate alerting, and symptom-versus-cause alert design are standard SRE method and need no local rubric.

## Route Depth Elsewhere

- [`production-diagnosis`](../../../docs/universal-disciplines/production-diagnosis/SKILL.md) when signals must let an operator localize a live degradation across services. It forces each signal to serve a quantified symptom contract and per-hop attribution — including the negative space that clears a component and the ordering that separates a cause from its victims — instead of dashboards that display everything and localize nothing. It also owns the detection-gap question a postmortem asks: why did users notice before the alerts did?
- [`reliable-messaging`](../../../docs/universal-disciplines/reliable-messaging/SKILL.md) for producer, consumer, DLQ, redrive, and replay signals; its operations reference owns outbox and consumer depth/oldest-age, quarantine, and reconciliation drift.
- [`durable-background-jobs`](../../../docs/universal-disciplines/durable-background-jobs/SKILL.md) for queue age, stuck-work detection, lease and checkpoint progress, and drain observability.

Hand resilience behavior to `go-reliability` and performance budgets to `go-performance`.
