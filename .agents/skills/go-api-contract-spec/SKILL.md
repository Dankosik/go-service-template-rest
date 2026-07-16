---
name: go-api-contract-spec
description: "Define or revise client-visible REST contracts before coding: resources, representations, HTTP semantics, errors, pagination, idempotency, async behavior, and compatibility. Use when API behavior is the primary decision; not router topology, system/data architecture, security policy, or implementation."
---

# Go Api Contract Spec

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Define resource representations, validation, error behavior, compatibility, idempotency, and `202` recovery. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Hand router composition to `go-chi-spec`, topology to `go-system-architecture-spec`, data truth to `go-data-architecture-spec`, and trust policy to `go-security-spec`. Return the owned decision or shared-contract outcome with every forced consequence and focused proof; hand off any other owner’s policy.
