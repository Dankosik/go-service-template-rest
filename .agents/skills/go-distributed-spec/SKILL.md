---
name: go-distributed-spec
description: "Use when a cross-service durable flow needs consistency, saga, orchestration or choreography, outbox/inbox, idempotency, compensation, redrive, or reconciliation decisions before coding; Own distributed recovery policy and invariant handoffs; Skip when the primary decision is local transactions, service resilience, system topology, or implementation."
---

# Go Distributed Spec

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. First prove durable cross-service coordination is needed; then decide process ownership and coordination, assume at-least-once delivery, and make recovery executable. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate business identity to `go-domain-invariant-spec` and local query/cache policy to `go-db-cache-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
