---
name: go-api-contract
description: "API contract: Use for client-visible REST resources, HTTP/errors, pagination, idempotency, async behavior, or compatibility. Own semantics/proof; Skip transport topology, security, or implementation."
---

# Go API Contract

An API is a **promise**: every client-visible behavior, once observable, will be built on — whether it was intended or not.

`resource model -> operation semantics -> errors -> limits and pagination -> idempotency and async -> compatibility -> proof`

Errors, pagination, idempotency keys, and `202` recovery are contract clauses with the same weight as success bodies; an undocumented observable is still a promise. Compatibility is decided per clause: additive evolution stays in place, and anything narrowing an accepted observable forces a version or a negotiated migration rather than an edit.

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: reconstruct every client-visible representation, validation rule, error, compatibility rule, idempotency rule, and `202` recovery behavior as a contract clause from accepted behavior, current runtime/generated contracts, and affected consumers. The contract is complete when every observable clause reaches a shared Decision disposition with its forced consequence and a proof a consumer could run.

Decide clauses against the current `api/openapi/service.yaml`, the router serving it, and the `internal/problem` catalog it answers from. Load the [decision selector](references/index.md) when a change touches the error contract, a published operation's compatibility, a retryable mutation's idempotency, or async acceptance. Hand router composition to `go-chi`, topology to `go-system-architecture`, data truth to `go-data-architecture`, trust policy to `go-security`, and accepted async work's execution to [`durable-background-jobs`](../../../docs/universal-disciplines/durable-background-jobs/SKILL.md). Load [`external-api-integration`](../../../docs/universal-disciplines/external-api-integration/SKILL.md) when this service calls a provider it does not control: it forces request identity and outcome classification across an unreliable boundary, instead of clauses written as if every response arrives.
