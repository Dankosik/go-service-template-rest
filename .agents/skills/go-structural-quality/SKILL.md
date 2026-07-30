---
name: go-structural-quality
description: "Structural quality: Use for harsh whole-diff review or cross-file overbuild. Own abstraction cost, mixed responsibility, and missed deletion; Skip Go semantics, local readability, or architecture."
---

# Go Structural Quality

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct every changed structural path from the whole diff, current responsibility owners, callers, duplicate execution, compatibility, and stale surfaces; test each abstraction and path for deletion or collapse first. Complete when the shared finding envelope accounts for every retained structure and deletion opportunity, naming any outside boundary or proof blocker with focused proof; each retained cost must be earned by present complexity. Missing placement or behavior policy returns to its named Decision owner.

Hand accepted placement violations to `go-implementation-ownership`, Go semantics to `go-idiomatic`, and local behavior-preserving readability to `go-language-simplifier`.
