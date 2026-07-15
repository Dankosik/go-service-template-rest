---
name: go-system-architecture-spec
description: "Use when a feature, refactor, or extraction needs service or component boundaries, topology, source-of-truth, sync/async flow, consistency, failure, migration, or operability decisions before coding; Own target-state system architecture; Skip when the primary decision is endpoint detail, physical data design, Go package placement, or post-code review."
---

# Go System Architecture Spec

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Decide component/source authority, sync/async boundaries, consistency, integration contracts, failure ownership, migration, and rollout topology. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate implementation placement to `go-implementation-ownership-spec` and local data truth to `go-data-architecture-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
