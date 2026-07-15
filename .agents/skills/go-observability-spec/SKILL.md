---
name: go-observability-spec
description: "Use when telemetry behavior, correlation, SLI/SLO and error-budget signals, alerts, diagnostics, async visibility, privacy, cardinality, sampling, or cost must be decided before coding; Own observability policy and proof obligations; Skip when the primary decision is service resilience, performance budgets, delivery governance, or instrumentation implementation."
---

# Go Observability Spec

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Define signal contracts, correlation, SLI/SLO/error-budget, alerts, diagnostics, privacy/cardinality, and async visibility. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate resilience objectives to `go-reliability-spec` and privacy/access policy to `go-security-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
