---
name: go-idiomatic-review
description: "Use when changed Go may violate language or standard-library contracts for errors, context API or lifetime, nil and zero values, receivers, method sets, aliasing, resources, or exported APIs; Own Go-semantic correctness; Skip when behavior is correct but readability, whole-diff structure, or package ownership is the primary issue."
---

# Go Idiomatic Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review context lifetime, nil/zero values, errors, resources, receiver/method sets, and mutable-value aliasing. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate placement to `go-implementation-ownership-spec` and changed behavior to its domain spec. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
