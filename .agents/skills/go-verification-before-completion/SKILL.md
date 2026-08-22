---
name: go-verification-before-completion
description: "Evidence boundary: Use for a verification-only outcome or when a claim's proving scope is non-obvious or disputed. Own claim-to-evidence mapping and gaps; Skip routine validation already selected by Direct Work or Implementation."
metadata:
  invocation: model
  kind: method
---

# Go Verification Before Completion

An **evidence boundary** is the behavior a proof would fail on. A completion
claim cannot be wider than that boundary.

`claim -> observable -> command or procedure -> result -> exercised scope -> gap`

Apply the shared [Evidence
Contract](../../../docs/spec-first-workflow/shared/evidence-contract.md). For
every claim, name the observable whose absence or incorrectness would make the
selected proof fail. Record the exact command or procedure, relevant
preconditions, result, cached or fresh state, and scope actually exercised.

A passing command proves only the surfaces it observed. File presence, status,
an implementation summary, a skipped integration suite, a test pattern matching
zero tests, or an unrelated aggregate cannot carry the claim.

Complete when every claim is supported at its stated scope, weakened to the
evidence boundary, or returned with one named missing proof and owner. Return
[Evidence Result
V1](../../../docs/spec-first-workflow/interfaces/evidence-result-v1.md). Load a
matching [reference](references/index.md) only when the boundary is non-obvious.
