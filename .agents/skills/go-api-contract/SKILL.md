---
name: go-api-contract
description: "API contract: Use when client-visible REST resources, representations, HTTP semantics, errors, pagination, idempotency, async behavior, or compatibility must be decided. Own the API decision and its proof; Skip when router topology, system/data architecture, security policy, or implementation is primary."
---

# Go API Contract

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: define resource representations, validation, error behavior, compatibility, idempotency, and `202` recovery. It is complete when every client-visible choice, forced consequence, focused proof, and blocker is explicit.

When a concrete API pressure can change the decision, load its [decision selector](references/index.md), then one matching reference by default. Hand router composition to `go-chi`, topology to `go-system-architecture`, data truth to `go-data-architecture`, and trust policy to `go-security`.
