---
name: go-api-contract
description: "Observable contract. Use when a REST change can alter what a deployed client distinguishes across success, error, replay, async recovery, or compatibility."
metadata:
  invocation: model
  kind: method
---

# Go API Contract

A public API is an **observable contract**: every distinction a deployed client
can detect is a clause.

`request acceptance -> success -> failure -> replay or async recovery -> compatibility -> proof`

Apply the [shared specialist contract](../../contracts/specialist-contract.md).
For each affected operation, build one `ObservableCell{surface, old, accepted,
client_consequence, owner, proof}` matrix from
`api/openapi/service.yaml`, the serving router, `internal/problem`, and affected
consumers. Each changed cell names the old behavior, accepted behavior, client
consequence, canonical owner, and proof.

Status, body shape or absence, error code and details, default and nullability,
pagination order and cursor, resource identity, retry outcome, and unknown
mutation outcome are observables. A green schema diff proves syntax, not
compatibility.

A Decision fills every missing cell and its migration behavior. A Review tries
to falsify every accepted cell through a client-visible example. Complete only
when every changed observable has one stable clause and consumer-runnable proof
or a named evidence gap.

The scope starts at the changed published operation and closes only after every
terminal, replay, and unknown-outcome state that a client can distinguish.
Load the [reference selector](references/index.md) only for its stated pressure.
