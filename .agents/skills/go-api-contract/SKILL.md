---
name: go-api-contract
description: "API contract: Use for client-visible REST resources, HTTP/errors, pagination, idempotency, async behavior, or compatibility. Own semantics/proof; Skip transport topology, security, or implementation."
---

# Go API Contract

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: reconstruct every client-visible representation, validation rule, error, compatibility rule, idempotency rule, and `202` recovery behavior as a contract clause from accepted behavior, current runtime/generated contracts, and affected consumers. The contract is complete when every observable clause reaches a shared Decision disposition with its forced consequence and focused proof.

When a concrete API pressure can change the decision, load its [decision selector](references/index.md), then one matching reference by default. Hand router composition to `go-chi`, topology to `go-system-architecture`, data truth to `go-data-architecture`, and trust policy to `go-security`.
