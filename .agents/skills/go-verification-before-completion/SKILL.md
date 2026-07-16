---
name: go-verification-before-completion
description: "Fresh proof: Use when a correctness, readiness, or completion claim needs current evidence of equal scope. Own claim-to-command matching, result inspection, proof scope, and explicit gaps; Skip when code or tests must change, root cause is unknown, test strategy is unresolved, or implementation is still underway."
---

# Go Verification Before Completion

Use [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for completion authority. Make evidence fresh: reconstruct every claim the answer intends to make from the accepted outcome, proposed response, changed scope, and named rollout/readiness assertions; map each claim to current proof of equal scope and reject stale or narrower evidence. Load [the reference selector](references/index.md) only when one concrete proof pressure can change the command or conclusion. Route unknown root cause to `go-systematic-debugging` and unset rollout criteria to `go-delivery-platform`. Return one evidence-permitted status per claim—`verified`, `partially verified`, or `not verified`—with commands and the next action; do not repair or invent proof.
