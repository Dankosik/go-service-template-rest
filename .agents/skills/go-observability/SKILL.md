---
name: go-observability
description: "Observability: Use for logs, metrics, traces, SLOs, alerts, privacy, or cardinality. Own operator evidence; Skip reliability, performance, and delivery."
---

# Go Observability

Telemetry is **operator evidence**: every signal answers a named operational
question or is avoidable cost.

`operator question -> signal -> SLI/SLO -> alert -> correlation -> cardinality, privacy, and cost -> proof`

Load the [shared specialist contract](../specialist-contract.md). Reconstruct
operator questions and signals from accepted outcomes, changed runtime paths,
existing telemetry, dashboards, alerts, and SLI/SLOs. Alert on user-visible
symptoms; keep causes as correlated diagnosis surfaces. Treat label cardinality
as a budget and every emitted field as a disclosure surface.

Load the [selector](references/index.md) for a new or widened metric label,
instrument, resource attribute, cross-service correlation, log event, payload
logging, readiness signal, or debug surface. Its reference carries the accepted
policy and enforcing repository owner.

For a **Decision**, cover every operator question and signal with privacy,
cardinality, diagnostic, and cost bounds. For **Review**, account for every
affected signal in the shared finding envelope.

Use [production diagnosis](../../../docs/universal-disciplines/production-diagnosis/SKILL.md)
for cross-service localization, [reliable messaging](../../../docs/universal-disciplines/reliable-messaging/SKILL.md)
for producer, consumer, DLQ, redrive, and replay signals, and [durable
jobs](../../../docs/universal-disciplines/durable-background-jobs/SKILL.md) for
queue age, leases, checkpoints, and drain. Hand resilience behavior to
`go-reliability` and performance budgets to `go-performance`.
