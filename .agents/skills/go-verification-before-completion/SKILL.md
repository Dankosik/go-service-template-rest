---
name: go-verification-before-completion
description: "Use when a correctness, readiness, or completion claim needs fresh evidence; Own claim-to-command matching, result inspection, proof scope, and explicit gaps; Skip when code or tests must be changed, root cause is unknown, test strategy is unresolved, or implementation is still underway."
---

# Go Verification Before Completion

Use [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for completion authority. Load [the reference selector](references/index.md) only when its proof pressure changes the command or conclusion. Map each claim to fresh proof of equal scope and reject stale or narrower evidence. Return `verified`, `partially verified`, or `not verified` with commands and the next action; do not repair or invent proof.
