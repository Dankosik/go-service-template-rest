---
name: go-system-architecture
description: "System architecture: Use for service boundaries, topology/protocol, truth, consistency, failure, or migration. Own runtime interactions; Skip Go placement, local data, domain policy, or implementation."
---

# Go System Architecture

Architecture is decided by **forces**: every component, boundary, and protocol is earned by a named requirement, constraint, or failure mode, and the simplest topology whose contracts survive the stated load and failures wins.

`requirements -> forces -> boundary crossings -> interaction contracts -> consistency and failure -> migration -> proof`

Boundary crossings are where systems break, so each one names its authority, protocol, consistency expectation, failure behavior, and migration story — whatever stays unnamed gets invented under incident pressure by whoever is on call.

Load the [shared specialist contract](../specialist-contract.md). This skill has one decision branch: reconstruct every material boundary crossing from accepted behavior, current components and contracts, sources of truth, material flows, consumers, and rollout topology. At each crossing name authority, protocol, interaction, consistency, failure, migration, and forced consequence, then choose the smallest coherent runtime boundaries. Complete when shared Decision dispositions cover every crossing and leave no ownership, sequence, proof, or rollout condition for implementation to invent.

[`distributed-system-design`](../../../docs/universal-disciplines/distributed-system-design/SKILL.md) owns the general method: it forces estimates and a failure model to select the topology, instead of a topology chosen first and justified after, and its references own earning a boundary, consistency and coordination, resilience under load, and evolution across versions. [`repo-architecture.md`](../../../docs/repo-architecture.md) records this repository's current boundaries, source-of-truth table, dependency direction, System Neighbors, and extension seams; read it before proposing a component.

The [decision selector](references/index.md) holds only what is specific to this repository and routes every other pressure to its nearer owner. Load its entry for a new service-to-service call, a protocol choice or migration, or a consumer-class change; hand package/file placement to `go-implementation-ownership`, data mechanics to `go-data-architecture`, and business policy to `go-domain-invariant`.
