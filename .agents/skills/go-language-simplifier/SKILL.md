---
name: go-language-simplifier
description: "Go readability: Use when correct local Go is obscured by control flow, predicates, naming, or helpers. Own behavior-preserving simplification; Skip Go semantics, package ownership, or whole-diff overbuild."
---

# Go Language Simplifier

The measure of local code is the next reader's **time to intent**: control flow, predicates, names, and helpers either carry intent directly or stand between the reader and it.

`intent path -> control flow -> predicates and names -> helper shape -> behavior-preserving change -> proof`

Simplification preserves observable behavior exactly — a semantics change is a different skill's finding, whichever direction it improves. Deletion beats abstraction wherever the indirection costs a reader more than it saves one, and a name that states the policy beats a comment that apologizes for its absence; neither rule survives contact with a helper that is the only place a constraint is recorded. Temporal coupling that the syntax hides is complexity even when every individual line is simple.

Load the [shared specialist contract](../specialist-contract.md). This skill has one review branch: reconstruct affected intent paths from the changed control flow, predicates, names, temporal coupling, and helper call sites under accepted behavior; make intent direct by flattening or deleting indirection without changing semantics. Complete when the shared finding envelope accounts for every path with a behavior-preserving correction.

Mandatory lint owns mechanical simplification here — nesting, if-else chains, naked returns, repeated literals, unused parameters and interfaces, and unwrapped or compared errors all fail the build on their own — so a finding that repeats a gate is spent for nothing. Load the [review selector](references/index.md) only when a helper boundary or a merged branch is the pressure. Hand semantic defects to `go-idiomatic`, ownership to `go-implementation-ownership`, and explicit cross-file overbuild to `go-structural-quality`.
