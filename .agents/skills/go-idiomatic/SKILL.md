---
name: go-idiomatic
description: "Semantic ownership: Use when errors, context, nil/zero, method sets, aliasing, or resource lifetimes change what a Go caller observes. Own language-level correctness; Skip readability, structure, or placement."
metadata:
  invocation: model
  kind: method
---

# Go Idiomatic

Go correctness follows **semantic ownership**: who owns error identity, work
lifetime, mutable backing storage, release, and the admissible zero state.

Apply the [shared specialist contract](../../contracts/specialist-contract.md).
For every changed boundary, build one ownership story from producer through
caller-visible observation. Name who may inspect the error, cancel the work,
mutate aliased state, release the resource, construct a non-zero value, and
reach each method through the stored type.

Configured linters prove mechanical shape, not that `%w` exposes the right
cause, `Close` occurs in the owning scope, a clone has sufficient depth, or a
typed nil is absent to its caller. A Decision assigns each obligation to one
owner and observable contract. A Review tries the plausible wrong default at
the actual caller boundary.

Complete when every changed semantic boundary has one owner, preserved
caller-visible behavior, and focused proof or a named gap. Load the [reference
selector](references/index.md) only when its stated pressure is observable in
changed symbols or callers.
