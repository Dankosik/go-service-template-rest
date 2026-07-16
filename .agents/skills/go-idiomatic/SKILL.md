---
name: go-idiomatic
description: "Go idiom: Use when changed Go may violate language or standard-library contracts for errors, context, nil/zero values, receivers, method sets, aliasing, resources, or exported APIs. Own Go-semantic review; Skip when behavior is correct but readability, whole-diff structure, or package ownership is primary."
---

# Go Idiomatic

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: inspect context lifetime, nil/zero values, errors/resources, receiver/method sets, exported APIs, and mutable-value aliasing. It is complete when every affected Go contract is dispositioned as a finding or no finding with forced consequence and focused proof, and unset behavior is handed to its domain skill.

Load the [review selector](references/index.md) for one violated contract by default. Hand placement to `go-implementation-ownership` and behavior-preserving readability to `go-language-simplifier`.
