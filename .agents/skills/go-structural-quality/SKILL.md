---
name: go-structural-quality
description: "Deletion test. Use when a Go diff may overbuild, split responsibility, or add parallel structure and needs an abstraction-cost or collapse decision."
metadata:
  invocation: model
  kind: method
---

# Go Structural Quality

Judge whole-diff structure with a **deletion test** and one realistic change
simulation.

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
For every added abstraction, layer, file, compatibility shim, or parallel path,
record its present responsibility, where its complexity returns if deleted,
and which owners and files the next realistic change would touch with and
without it.

An interface with one adapter is not justified merely by hiding the adapter; it
must reduce current complexity or protect a real dependency direction. A
one-use helper survives when it uniquely carries a protocol or ownership
constraint. Split responsibility, stale surfaces, and parallel execution paths
remain collapse candidates.

A Decision selects the least structure that owns the current responsibility. A
Review tries to delete or collapse each candidate and rejects it only when the
simulation exposes higher locality or deletion cost.

Complete when every added structure passes both tests, each responsibility has
one owner, and no superseded execution path remains.
