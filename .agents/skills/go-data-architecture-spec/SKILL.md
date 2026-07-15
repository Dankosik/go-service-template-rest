---
name: go-data-architecture-spec
description: "Use when authoritative data ownership, models, schema evolution, tenant isolation, migration, retention, history, projections, or datastore choice must be decided before coding; Own data truth and lifecycle architecture; Skip when the primary decision is runtime query/transaction/cache behavior, business policy, distributed recovery, or implementation."
---

# Go Data Architecture Spec

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Decide authoritative model, write-time invariants, keys, schema evolution, retention, and physical access paths; consume domain meaning. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate business acceptance to `go-domain-invariant-spec` and runtime access/cache behavior to `go-db-cache-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
