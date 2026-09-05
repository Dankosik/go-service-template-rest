---
name: go-observability
description: "Operator evidence. Use when an operational question needs a signal, SLI, SLO, or alert, or when an emitted field or label changes correlation, privacy, cardinality, or cost."
metadata:
  invocation: model
  kind: method
---

# Go Observability

Telemetry is **operator evidence**: every signal answers a named operational
question or is avoidable cost.

`operator question -> signal -> SLI/SLO -> alert -> correlation -> cardinality, privacy, and cost -> proof`

Load the [shared specialist contract](../../contracts/specialist-contract.md).
From every changed operator question through the page or diagnostic action it
enables, build `OperatorSignal{question, signal, SLI, alert, correlation,
labels, privacy, cost, owner, proof}` from accepted outcomes, changed runtime
paths, current telemetry, dashboards, and alerts. Alert on user-visible
symptoms; keep causes as correlated diagnosis surfaces. Treat label cardinality
as a budget and every emitted field as a disclosure surface.

Load the [selector](references/index.md) for a new or widened metric label,
instrument, resource attribute, cross-service correlation, log event, payload
logging, readiness signal, or debug surface. Its reference carries the accepted
policy and enforcing repository owner.

For a **Decision**, disposition every operator question and signal with privacy,
cardinality, diagnostic, and cost bounds. For **Review**, account for every
affected signal. A signal that cannot answer its named question or has no
bounded labels and readers is not complete.

Use [production diagnosis](../../../docs/universal-disciplines/production-diagnosis/SKILL.md)
for cross-service localization, [reliable messaging](../../../docs/universal-disciplines/reliable-messaging/SKILL.md)
for producer, consumer, DLQ, redrive, and replay signals, and [durable
jobs](../../../docs/universal-disciplines/durable-background-jobs/SKILL.md) for
queue age, leases, checkpoints, and drain.
