---
name: go-observability
description: "Observability: Use when logs, metrics, traces, SLI/SLOs, alerts, diagnostics, privacy, cardinality, sampling, or cost needs a decision, or when changed telemetry needs conformance review. Own telemetry policy, operator evidence, and review; Skip when reliability behavior, performance behavior, or delivery governance is primary."
---

# Go Observability

Load the [shared specialist contract](../specialist-contract.md). Keep signal identity, correlation, privacy, cardinality, diagnostics, SLI/SLO math, and alert actionability coherent.

## Choose The Branch

- **Decision** — select when telemetry policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when the signal contract, forced consequences, operator proof, cost/privacy bounds, and blockers are explicit.
- **Review** — select when changed telemetry must conform to accepted policy. Load the [review selector](references/review/index.md) for the changed signal. Complete when every affected signal is dispositioned as a finding or no finding with the smallest correction and focused proof; missing policy stays in the decision branch.

Hand resilience behavior to `go-reliability` and performance budgets to `go-performance`.
