---
name: go-language-simplifier
description: "Go readability: Use when behaviorally correct, locally owned Go is hard to understand or change because of control flow, predicates, naming, or helper shape. Own behavior-preserving simplification review; Skip when Go semantics, package ownership, or explicit whole-diff structural overbuild is primary."
---

# Go Language Simplifier

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct affected intent paths from the changed control flow, predicates, names, temporal coupling, and helper call sites under accepted behavior; make intent direct by flattening or deleting indirection without changing semantics. Complete when the shared finding envelope accounts for every path, naming any outside boundary or proof blocker with the smallest behavior-preserving correction and focused proof. Missing behavior policy returns to the named domain Decision owner.

Load the [review selector](references/index.md) for one concrete pressure by default. Hand semantic defects to `go-idiomatic`, ownership to `go-implementation-ownership`, and explicit cross-file overbuild to `go-structural-quality`.
