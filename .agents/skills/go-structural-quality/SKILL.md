---
name: go-structural-quality
description: "Use when a Go diff adds layers, abstractions, compatibility shims, or parallel paths whose current necessity must be assessed."
metadata:
  invocation: model
  kind: method
---

# Go Structural Quality

Judge whole-diff structure with a **deletion test**: which current responsibility
or constraint would become harder to satisfy if the structure were removed?

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
Assess the present responsibility and deletion cost of added abstractions,
layers, compatibility shims, and parallel paths. For interacting owners,
competing designs, or a required Decision/Review handoff, compare which owners
and files one realistic next change would touch with and without the structure.
Record that comparison in the existing result. For one local case, a grounded
rationale is sufficient; adding a file alone requires no separate record or
change simulation. Required workflow result interfaces remain unchanged.

An interface with one adapter is not justified merely by hiding the adapter; it
must reduce current complexity or protect a real dependency direction. A
one-use helper survives when it uniquely carries a protocol or ownership
constraint. Split responsibility, stale surfaces, and parallel execution paths
remain collapse candidates.

A Decision selects the least structure that owns the current responsibility. A
Review tries to delete or collapse each candidate, retaining it when removal
would violate a current constraint or increase complexity. Use the change
simulation when the ownership or locality trade-off needs that comparison.

Complete when each added structure has a justified current purpose, each
responsibility has one owner, and no superseded execution path remains.
