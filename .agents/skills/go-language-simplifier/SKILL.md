---
name: go-language-simplifier
description: "Indirection economics. Use when opaque Go control flow, predicates, names, helpers, or deduplication obscure intent and behavior must remain unchanged."
metadata:
  invocation: model
  kind: method
---

# Go Language Simplifier

Simplification is **indirection economics** inside a behavior-preserving
boundary: the reader must learn less without losing an observable distinction.

Apply the [shared specialist contract](../../contracts/specialist-contract.md).
For each proposed simplification, compare the caller-facing cost of names,
parameters, results, control flow, and hidden contracts before and after. Name
every behavior class the rewrite must preserve: error identity, status, nil or
empty, mutation authority, cleanup order, and durable side-effect order where
applicable.

Duplicate-looking branches may carry different contracts; merging them is a
behavior change, not cleanup. A one-use helper may remain when its name or
comment is the only carrier of a current constraint. Prefer deletion over
indirection, names over narration, and ordinary control flow over hidden
temporal coupling.

Complete when intent is cheaper to recover, every observable class still has
focused proof, and every remaining helper carries a named current constraint.
Load the [reference selector](references/index.md) only for a helper boundary or
merged branch.
