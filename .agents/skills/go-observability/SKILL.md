---
name: go-observability
description: "Observability: Use when logs, metrics, traces, SLI/SLOs, alerts, diagnostics, privacy, cardinality, sampling, or cost needs a decision, or when changed telemetry needs conformance review. Own telemetry policy, operator evidence, and review; Skip when reliability behavior, performance behavior, or delivery governance is primary."
---

# Go Observability

Load the [shared specialist contract](../specialist-contract.md). Reconstruct operator questions and signals from accepted operability outcomes, changed runtime paths, existing telemetry, dashboards, alerts, and SLI/SLO surfaces; require each signal to support a named decision within correlation, privacy, cardinality, diagnostic, and cost bounds.

## Choose The Branch

- **Decision** — select when telemetry policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover every operator question and signal with cost/privacy bounds explicit.
- **Review** — select when changed telemetry must conform to accepted policy. Load the [review selector](references/review/index.md) for the changed signal. Account for every affected signal through the shared finding envelope, naming any outside boundary or proof blocker with the smallest correction and focused proof. Missing policy ends this run with a named Telemetry Decision handoff; conformance Review begins separately after acceptance.

Hand resilience behavior to `go-reliability` and performance budgets to `go-performance`.
