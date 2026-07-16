---
name: go-language-simplifier
description: "Go readability: Use when behaviorally correct, locally owned Go is hard to understand or change because of control flow, predicates, naming, or helper shape. Own behavior-preserving simplification review; Skip when Go semantics, package ownership, or explicit whole-diff structural overbuild is primary."
---

# Go Language Simplifier

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: expose intent, flatten control flow, remove temporal coupling, clarify predicates and names, and delete helpers whose indirection costs more than it saves without changing behavior. It is complete when every affected readability pressure is dispositioned as a finding or no finding with the smallest behavior-preserving correction and focused proof.

Load the [review selector](references/index.md) for one concrete pressure by default. Hand semantic defects to `go-idiomatic`, ownership to `go-implementation-ownership`, and explicit cross-file overbuild to `go-structural-quality`.
