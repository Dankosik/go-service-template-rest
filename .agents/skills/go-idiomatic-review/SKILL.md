---
name: go-idiomatic-review
description: "Use when changed Go may violate language or standard-library contracts for errors, context API or lifetime, nil and zero values, receivers, method sets, aliasing, resources, or exported APIs; Own Go-semantic correctness; Skip when behavior is correct but readability, whole-diff structure, or package ownership is the primary issue."
---

# Go Idiomatic Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review Go-semantic correctness across context lifetime, nil/zero values, errors/resources, receiver/method sets, and mutable-value aliasing. Use [the reference selector](references/index.md) to load one reference for the violated contract; add one only for an independent pressure. Escalate placement to `go-implementation-ownership-spec` and changed behavior to its domain spec. Return no findings or each evidence-backed finding with its forced consequence and focused proof; hand unresolved policy to its named specification owner.
