---
name: go-api-contract
description: "API contract: Use for client-visible REST, errors, pagination, idempotency, async behavior, or compatibility. Own semantics/proof; Skip transport, security, and code."
---

# Go API Contract

An API is a **promise**: every client-visible behavior, once observable, will be built on — whether it was intended or not.

`resource model -> operation semantics -> errors -> limits and pagination -> idempotency and async -> compatibility -> proof`

Errors, pagination, idempotency, and `202` recovery are clauses like success
bodies. Additive evolution stays in place; narrowing an accepted observable
needs a version or negotiated migration.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct
client-visible representations, validation, errors, compatibility, idempotency,
and async recovery from accepted behavior, runtime/generated contracts, and
affected consumers. Complete when every observable clause has a Decision
disposition, forced consequence, and consumer-runnable proof.

Decide against `api/openapi/service.yaml`, its router, and `internal/problem`.
Load the [decision selector](references/index.md) for errors, compatibility,
retryable mutation idempotency, or async acceptance. Hand composition, topology,
data, trust, durable async execution, and external-provider behavior to their
matching owners.
