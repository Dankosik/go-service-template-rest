---
name: go-system-architecture
description: "System architecture: Use when service boundaries, topology, source of truth, sync/async flow, consistency, failure, or migration must be decided. Own runtime and component authority; Skip when package placement, local data mechanics, domain policy, or implementation review is primary."
---

# Go System Architecture

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: choose the smallest coherent runtime boundary, authority topology, interaction model, consistency/failure behavior, and migration path. It is complete when every affected component owner, source of truth, sequence, forced consequence, proof, rollout condition, and blocker is explicit.

Load the [decision selector](references/index.md) for one concrete architecture pressure by default. Hand package/file placement to `go-implementation-ownership`, data mechanics to `go-data-architecture`, and business policy to `go-domain-invariant`.
