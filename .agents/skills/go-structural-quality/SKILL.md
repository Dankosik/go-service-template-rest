---
name: go-structural-quality
description: "Structural quality: Use for whole-diff overbuild or mixed responsibility. Own abstraction cost/deletion; Skip Go semantics, local readability, and architecture."
---

# Go Structural Quality

Apply the [shared specialist contract](../specialist-contract.md).

Judge the whole diff by present responsibility and deletion cost. Every
abstraction, layer, file, compatibility shim, and duplicate path must solve
current complexity; one-caller indirection, split responsibility, stale
surfaces, and parallel execution paths are deletion or collapse candidates.

Route accepted placement violations to `go-implementation-ownership`, Go
semantics to `go-idiomatic`, and local behavior-preserving readability to
`go-language-simplifier`.
