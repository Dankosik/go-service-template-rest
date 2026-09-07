---
name: go-verification-before-completion
description: "Evidence boundaries for claims. Use for verification-only work or when existing evidence may not prove the requested scope."
metadata:
  invocation: model
  kind: method
---

# Go Verification Before Completion

An **evidence boundary** is the behavior a proof would fail on. An empirical
claim cannot be wider than that boundary. Local development acceptance is a
separate sufficiency decision owned by the Evidence Contract, not a claim that
every production path has been observed.

`claim -> observable -> command or procedure -> result -> exercised scope -> gap`

An Implemented task handoff claims code production, not verified behavior, and
does not trigger this method. For a ledger, use this method at final validation
after all planned code is assembled; [Implementation](../../../docs/spec-first-workflow/phases/implementation.md#feedback-during-coding)
owns bounded feedback during coding.

Apply the shared [Evidence
Contract](../../../docs/spec-first-workflow/shared/evidence-contract.md). Select
only its local criterion and genuinely explicit additions. This skill and its
references do not create new gates or authorize test infrastructure. For a
verification claim, name the observable whose absence or incorrectness would make the
selected proof fail. Record the exact command or procedure, relevant
preconditions, result, cached or fresh state, and scope actually exercised.

A passing command proves only the surfaces it observed. File presence, status,
an implementation summary, a skipped integration suite, a test pattern matching
zero tests, or an unrelated aggregate cannot carry the claim.

Complete when required claims are supported at their stated scope or returned
with the exact missing required proof and owner. Stop ordinary local work at its
accepted build/unit boundary; disclose material optional gaps without blocking
completion or repairing their environment. Never weaken an explicitly requested
runtime or CI result into local success. Return
[Evidence Result
V1](../../../docs/spec-first-workflow/interfaces/evidence-result-v1.md). Load a
matching [reference](references/index.md) only when the boundary is non-obvious.
