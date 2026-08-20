---
name: go-system-architecture
description: "System architecture: Use for boundaries, topology, protocol, truth, consistency, failure, or migration. Own runtime interactions; Skip Go placement."
metadata:
  invocation: model
  kind: method
---

# Go System Architecture

Architecture is decided by **forces**: every component, boundary, and protocol is earned by a named requirement, constraint, or failure mode, and the simplest topology whose contracts survive the stated load and failures wins.

`requirements -> forces -> boundary crossings -> interaction contracts -> consistency and failure -> migration -> proof`

Boundary crossings are where systems break, so each one names its authority, protocol, consistency expectation, failure behavior, and migration story — whatever stays unnamed gets invented under incident pressure by whoever is on call.

Load the [shared specialist contract](../../contracts/specialist-contract.md). Reconstruct
every material crossing from accepted behavior, current components and
contracts, sources of truth, flows, consumers, and rollout topology. At each
crossing name authority, protocol, interaction, consistency, failure, migration,
and forced consequence; then choose the smallest coherent runtime boundaries.

[Distributed system design](../../../docs/universal-disciplines/distributed-system-design/SKILL.md)
owns the general force, estimate, and failure-model method.

Load the [decision selector](references/index.md) for the affected current-state
leaf or a new service call, protocol, migration, or consumer-class change.
Complete when every crossing is
dispositioned and implementation has no boundary, ownership, sequence, proof,
or rollout condition left to invent. Hand placement, data mechanics, and
business policy to their matching owners.
