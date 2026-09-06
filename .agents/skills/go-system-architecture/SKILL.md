---
name: go-system-architecture
description: "Runtime boundary design. Use when a change adds or changes a component crossing, protocol, source of truth, consistency expectation, failure model, or migration topology."
metadata:
  invocation: model
  kind: method
---

# Go System Architecture

Architecture is decided by **forces**: every component, boundary, and protocol is earned by a named requirement, constraint, or failure mode, and the simplest topology whose contracts survive the stated load and failures wins.

`requirements -> forces -> boundary crossings -> interaction contracts -> consistency and failure -> migration -> proof`

Boundary crossings are where systems break, so each one names its authority, protocol, consistency expectation, failure behavior, and migration story — whatever stays unnamed gets invented under incident pressure by whoever is on call.

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
From every new or changed runtime crossing through its migration and terminal
failure disposition, build `Crossing{from, to, authority, protocol,
interaction, consistency, failure, migration, forced_consequence, proof}` from
accepted behavior, current components, contracts, sources of truth, consumers,
and rollout topology. Choose the smallest coherent runtime boundaries.

[Distributed system design](../../../docs/universal-disciplines/distributed-system-design/SKILL.md)
owns the general force, estimate, and failure-model method.

Load the [decision selector](references/index.md) for the affected current-state
leaf or a new service call, protocol, migration, or consumer-class change.
Complete when every crossing is dispositioned and implementation has no
boundary, authority, sequence, proof, or rollout condition left to invent.
