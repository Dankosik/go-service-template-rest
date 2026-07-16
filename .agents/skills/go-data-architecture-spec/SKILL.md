---
name: go-data-architecture-spec
description: "Decide authoritative data architecture before coding: data ownership/models, tenant identity, schema evolution/backfills, retention/history/projections, or datastore fit. Own data truth and lifecycle; skip runtime query/transaction/cache behavior, business acceptance, distributed recovery, and implementation ownership."
---

# Go Data Architecture Spec

Load the [shared specialist contract](../specialist-contract.md). Decide the authoritative model, enforceable write-time invariants, identity keys, schema evolution, lifecycle, and physical design from accepted domain meaning. For each pressure that can change the decision, state its thesis and load its [reference](references/index.md); add another only for an independent pressure. Send business acceptance to `go-domain-invariant-spec`, runtime access/cache behavior to `go-db-cache-spec`, distributed recovery to `go-distributed-spec`, and implementation ownership to `go-implementation-ownership-spec`. Return the applicable contract outcome with forced consequences, focused proof, and any blocker.
