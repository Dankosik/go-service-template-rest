---
name: go-distributed-review
description: "Use when changed behavior crosses a durable process or service boundary through sagas, messages, outbox/inbox, replay, ordering, compensation, redrive, or reconciliation; Own distributed-flow correctness against accepted policy; Skip when the primary defect is local synchronization, service lifecycle resilience, or SQL/cache execution."
---

# Go Distributed Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review durable success boundaries, idempotency, ordering, replay/redrive, compensation, reconciliation, and message compatibility. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate changed flow, ordering, compensation, or recovery policy to `go-distributed-spec`. Return an evidence-backed finding with its smallest safe correction and focused proof, or no findings; hand missing policy to its named specification owner.
