---
name: go-data-architecture
description: "Data architecture: Use for authority, tenant identity, schema evolution, backfills, retention, projections, or datastore fit. Own truth/lifecycle; Skip runtime DB/cache, domain, recovery, or code placement."
---

# Go Data Architecture

Every datum answers to one **authority**: the single store and writer whose acceptance makes it true, from which every copy, projection, and cache admits being derived.

`authority -> identity and tenancy -> write invariant -> evolution and backfill -> derived surfaces -> retention -> proof`

A derived surface that cannot name its source, lag, and repair path is a second authority waiting to disagree, and retention is a designed lifecycle stage with an owner rather than a response to disk pressure.

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: reconstruct every affected datum, identity, transition, derived surface, and lifecycle from accepted domain meaning, current schemas and stores, writers, readers, and retention paths; trace each to one authority with an enforceable write invariant, evolution path, and physical design. Complete when shared Decision dispositions cover every data authority, forced consequence, and focused proof.

[`postgres-schema-design`](../../../docs/universal-disciplines/postgres-schema-design/SKILL.md) owns the modeling itself — invariants into relations, keys and identity, tenant scoping, types, constraints, and the expand-and-contract migration sequence — so load it whenever relations, keys, or constraints must carry the invariant. The [decision selector](references/index.md) holds only what this repository decides differently from that discipline; load it when a migration touches a table that already holds rows, or a change publishes state downstream, adds a derived surface, or sets retention. Hand business acceptance to `go-domain-invariant`, runtime access/cache behavior to `go-db-cache`, durable recovery to `go-distributed`, concurrency mechanism choice to [`concurrency-control`](../../../docs/universal-disciplines/concurrency-control/SKILL.md), and placement to `go-implementation-ownership`.
