---
name: go-system-architecture
description: "System architecture: Use when service boundaries, topology, source of truth, sync/async flow, consistency, failure, or migration must be decided. Own runtime and component authority; Skip when package placement, local data mechanics, domain policy, or implementation review is primary."
---

# Go System Architecture

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: reconstruct every material boundary crossing from accepted behavior, current components and contracts, sources of truth, material flows, consumers, and rollout topology. At each crossing name authority, protocol, interaction, consistency, failure, migration, and forced consequence, then choose the smallest coherent runtime boundaries. Complete when shared Decision dispositions cover every crossing and leave no ownership, sequence, proof, or rollout condition for implementation to invent.

Load the [decision selector](references/index.md) for one concrete architecture pressure by default. Hand package/file placement to `go-implementation-ownership`, data mechanics to `go-data-architecture`, and business policy to `go-domain-invariant`.
