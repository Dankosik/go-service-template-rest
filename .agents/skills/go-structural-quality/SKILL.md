---
name: go-structural-quality
description: "Structural quality: Use for whole-diff overbuild or mixed responsibility. Own abstraction cost/deletion; Skip Go semantics, local readability, and architecture."
---

# Go Structural Quality

Structure must be **earned**: every abstraction, layer, and file in the diff pays rent in present complexity, and the harshest question — can this be deleted? — comes first.

`whole diff -> responsibility owners -> deletion test -> duplication and drift -> retained cost -> proof`

An abstraction serving one caller is indirection; a responsibility split across files is a bug generator; compatibility shims and stale surfaces that outlived their reason are deletions someone missed. Judge the diff whole: what it adds, what it should have deleted, and what it left owning two things at once.

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct every changed structural path from the whole diff, current responsibility owners, callers, duplicate execution, compatibility, and stale surfaces; test each abstraction and path for deletion or collapse first. Complete when the shared finding envelope accounts for every retained structure and deletion opportunity; each retained cost must be earned by present complexity.

Hand accepted placement violations to `go-implementation-ownership`, Go semantics to `go-idiomatic`, and local behavior-preserving readability to `go-language-simplifier`.
