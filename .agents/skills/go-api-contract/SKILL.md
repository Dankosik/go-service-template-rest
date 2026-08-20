---
name: go-api-contract
description: "API contract: Use for client-visible REST, errors, pagination, idempotency, async behavior, or compatibility. Own semantics/proof; Skip transport, security, and code."
metadata:
  invocation: model
  kind: method
---

# Go API Contract

An API is a promise: success bodies, errors, pagination, idempotency, and
accepted async recovery are equally observable clauses. Additive evolution
stays in place; narrowing an accepted observable needs a version or negotiated
migration.

Apply the [shared specialist contract](../../contracts/specialist-contract.md). Inspect
representations, validation, errors, compatibility, idempotency, and recovery
against `api/openapi/service.yaml`, its router, `internal/problem`, and affected
consumers.

Load the [reference selector](references/index.md) only for errors,
compatibility, retryable mutation idempotency, or async acceptance. Route
transport composition, topology, data, trust, durable execution, and provider
behavior to their matching owners.
