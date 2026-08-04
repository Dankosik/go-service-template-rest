---
name: go-idiomatic
description: "Go idiom: Use when changed Go risks errors, context, nil/zero, method sets, aliasing, or resource lifetimes. Own semantics; Skip readability, whole-diff structure, or package ownership."
---

# Go Idiomatic

Go correctness lives in **semantics**, not style: errors, context, nil and zero values, receivers and method sets, aliasing, and resource lifetimes each carry a contract the compiler does not check for you.

`changed symbols -> semantic obligations -> error flow -> context and resource lifetime -> nil/zero and method sets -> aliasing -> proof`

An error is handled exactly once and wrapped where the cause belongs in this package's contract; a context bounds the work it was handed to; a type's zero value either works or its construction is part of the documented contract; and a shared backing array, a copied builder, or a nil pointer inside an interface is a semantic defect wearing a style costume.

`make lint` already decides the mechanical half of this domain. A finding worth reporting here is one the configured linters cannot reach.

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct semantic obligations from changed symbols, callers, context and resource lifetimes, nil/zero states, errors, method sets, and mutable-value aliasing, then inspect Go semantics rather than style. Complete when the shared finding envelope accounts for every obligation.

Load the [review selector](references/index.md) when a changed symbol carries a contract the configured linters cannot see — error identity, release scope, aliasing authority, or nil/zero observability. Hand placement to `go-implementation-ownership` and behavior-preserving readability to `go-language-simplifier`.
