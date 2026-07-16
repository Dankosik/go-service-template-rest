---
name: go-structural-quality
description: "Structural quality: Use when the user requests a harsh whole-diff review or a change is structurally overbuilt across files. Own abstraction cost, spaghetti growth, mixed responsibility, speculative flexibility, and missed deletion; Skip when Go semantics, local readability, or accepted architecture conformance is primary."
---

# Go Structural Quality

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: inspect the whole diff for abstraction cost, cross-file sprawl, mixed responsibilities, duplicate executable paths, speculative flexibility, and missed deletion. It is complete when every material structural pressure is dispositioned as a finding or no finding with the smallest deletion-first correction and focused proof.

Hand accepted placement violations to `go-implementation-ownership`, Go semantics to `go-idiomatic`, and local behavior-preserving readability to `go-language-simplifier`.
