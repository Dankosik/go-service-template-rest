---
name: go-language-simplifier
description: "Go readability: Use for opaque control flow, predicates, names, or helpers. Own behavior-preserving simplification; Skip semantics, ownership, or architecture."
---

# Go Language Simplifier

The measure of local code is the next reader's **time to intent**: control flow, predicates, names, and helpers either carry intent directly or stand between the reader and it.

`intent path -> control flow -> predicates and names -> helper shape -> behavior-preserving change -> proof`

Simplification preserves behavior. Prefer deletion over indirection and names
over narration, while retaining a helper that uniquely carries a constraint.
Hidden temporal coupling remains complexity.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct
affected intent paths from control flow, predicates, names, temporal coupling,
and helper call sites; flatten or delete indirection without changing semantics.

Mandatory lint owns mechanical style, so review does not repeat it. Load the
[review selector](references/index.md) only for a helper boundary or merged
branch. Complete when every affected path has a behavior-preserving disposition.
Hand semantics, ownership, and cross-file overbuild to their matching skills.
