---
name: go-data-architecture
description: "Data architecture: Use when authoritative data ownership, models, tenant identity, schema evolution, backfills, retention, history, projections, or datastore fit must be decided. Own data truth and lifecycle; Skip when runtime DB/cache behavior, business acceptance, distributed recovery, or implementation ownership is primary."
---

# Go Data Architecture

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: define the authoritative model, enforceable write-time invariants, identity keys, schema evolution, lifecycle, and physical design from accepted domain meaning. It is complete when the decision, forced consequences, focused proof, and blockers cover every affected data authority.

For each pressure that can change the decision, load its [decision selector](references/index.md), then one matching reference by default. Hand business acceptance to `go-domain-invariant`, runtime access/cache behavior to `go-db-cache`, durable recovery to `go-distributed`, and placement to `go-implementation-ownership`.
