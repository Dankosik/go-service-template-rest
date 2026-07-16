---
name: go-api-contract
description: "API contract: Use when client-visible REST resources, representations, HTTP semantics, errors, pagination, idempotency, async behavior, or compatibility must be decided. Own the API decision and its proof; Skip when router topology, system/data architecture, security policy, or implementation is primary."
---

# Go API Contract

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: reconstruct every client-visible representation, validation rule, error, compatibility rule, idempotency rule, and `202` recovery behavior as a contract clause from accepted behavior, current runtime/generated contracts, and affected consumers. The contract is complete when every observable clause reaches a shared Decision disposition with its forced consequence and focused proof.

When a concrete API pressure can change the decision, load its [decision selector](references/index.md), then one matching reference by default. Hand router composition to `go-chi`, topology to `go-system-architecture`, data truth to `go-data-architecture`, and trust policy to `go-security`.
