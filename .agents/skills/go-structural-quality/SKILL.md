---
name: go-structural-quality
description: "Structural quality: Use when the user requests a harsh whole-diff review or a change is structurally overbuilt across files. Own abstraction cost, spaghetti growth, mixed responsibility, speculative flexibility, and missed deletion; Skip when Go semantics, local readability, or accepted architecture conformance is primary."
---

# Go Structural Quality

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct every changed structural path from the whole diff, current responsibility owners, callers, duplicate execution, compatibility, and stale surfaces; test each abstraction and path for deletion or collapse first. Complete when the shared finding envelope accounts for every retained structure and deletion opportunity, naming any outside boundary or proof blocker with focused proof; each retained cost must be earned by present complexity. Missing placement or behavior policy returns to its named Decision owner.

Hand accepted placement violations to `go-implementation-ownership`, Go semantics to `go-idiomatic`, and local behavior-preserving readability to `go-language-simplifier`.
