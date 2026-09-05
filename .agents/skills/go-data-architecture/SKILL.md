---
name: go-data-architecture
description: "Data authority. Use when identity, durable schema, backfill, projection, derived surfaces, retention, or datastore fit changes where a datum is true over its lifecycle."
metadata:
  invocation: model
  kind: method
---

# Go Data Architecture

Every datum answers to one **authority**: the single store and writer whose acceptance makes it true, from which every copy, projection, and cache admits being derived.

`authority -> identity and tenancy -> write invariant -> evolution and backfill -> derived surfaces -> retention -> proof`

A derived surface that cannot name its source, lag, and repair path is a second authority waiting to disagree, and retention is a designed lifecycle stage with an owner rather than a response to disk pressure.

Load the [shared specialist contract](../../contracts/specialist-contract.md).
From every changed writer or durable schema through each reader, projection,
export, cache, repair, and retention path, build `AuthorityRecord{datum,
identity, writer, invariant, derived_surfaces, lag, repair, evolution,
retention, proof}`. A derived surface that cannot name its source, lag, and
repair path is a competing authority. Complete when every affected datum has
one authority and every derived surface and lifecycle stage has a disposition.

[`postgres-schema-design`](../../../docs/universal-disciplines/postgres-schema-design/SKILL.md)
owns relational modeling and expand-and-contract mechanics. Load the [decision
selector](references/index.md) only when this repository's current migration,
publication, derived-surface, or retention policy changes the decision.
