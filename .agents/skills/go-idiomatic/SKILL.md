---
name: go-idiomatic
description: "Go idiom: Use when changed Go may violate language or standard-library contracts for errors, context, nil/zero values, receivers, method sets, aliasing, resources, or exported APIs. Own Go-semantic review; Skip when behavior is correct but readability, whole-diff structure, or package ownership is primary."
---

# Go Idiomatic

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct semantic obligations from changed symbols, callers, exported APIs, context/resource lifetimes, nil/zero states, errors, receiver/method sets, and mutable-value aliasing, then inspect Go semantics rather than style. Complete when the shared finding envelope accounts for every obligation; name any outside boundary or proof blocker with its forced consequence and focused proof. Unset behavior returns to the named domain Decision owner.

Load the [review selector](references/index.md) for one violated contract by default. Hand placement to `go-implementation-ownership` and behavior-preserving readability to `go-language-simplifier`.
