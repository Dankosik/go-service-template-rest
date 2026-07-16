---
name: go-data-architecture
description: "Data architecture: Use when authoritative data ownership, models, tenant identity, schema evolution, backfills, retention, history, projections, or datastore fit must be decided. Own data truth and lifecycle; Skip when runtime DB/cache behavior, business acceptance, distributed recovery, or implementation ownership is primary."
---

# Go Data Architecture

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: reconstruct every affected datum, identity, transition, derived surface, and lifecycle from accepted domain meaning, current schemas and stores, writers, readers, and retention paths; trace each to one authority with an enforceable write invariant, evolution path, and physical design. Complete when shared Decision dispositions cover every data authority, forced consequence, and focused proof.

For each pressure that can change the decision, load its [decision selector](references/index.md), then one matching reference by default. Hand business acceptance to `go-domain-invariant`, runtime access/cache behavior to `go-db-cache`, durable recovery to `go-distributed`, and placement to `go-implementation-ownership`.
