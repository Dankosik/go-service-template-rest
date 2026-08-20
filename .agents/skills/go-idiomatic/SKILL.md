---
name: go-idiomatic
description: "Go semantics: Use for errors, context, nil/zero, method sets, aliasing, or resource lifetimes. Own correctness; Skip readability, structure, or ownership."
---

# Go Idiomatic

Apply the [shared specialist contract](../specialist-contract.md).

Inspect Go semantic obligations that configured linters cannot prove: error
identity and wrapping, context cancellation, nil and zero behavior, receiver and
method-set effects, mutable aliasing, and resource lifetime. An error is handled
once; context bounds the work it was given; construction is explicit when the
zero value is invalid; backing storage and release ownership remain visible.

Load the [reference selector](references/index.md) only when one of those
contracts is observable in changed symbols or callers. Route placement to
`go-implementation-ownership` and behavior-preserving readability to
`go-language-simplifier`.
