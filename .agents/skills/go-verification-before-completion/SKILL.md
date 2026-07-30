---
name: go-verification-before-completion
description: "Claim-scoped proof: Use when correctness/readiness/completion needs current evidence. Own claim-command matching, scope, and gaps; Skip changes, unknown cause, unresolved strategy, or active implementation."
---

# Go Verification Before Completion

Use [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for completion authority. Reconstruct every claim the answer intends to make from the accepted outcome, proposed response, changed scope, and named rollout/readiness assertions; map each claim to current proof of equal scope and reject stale or narrower evidence. Reuse successful proof while the relevant content, environment or precondition, claim scope, provenance, and risk surface remain unchanged; attach a commit/tree identity only when proof crosses a checkout or integration boundary. Load [the reference selector](references/index.md) only when one concrete proof pressure can change the command or conclusion. Route unknown root cause to `go-systematic-debugging` and unset rollout criteria to `go-delivery-platform`. Return one evidence-permitted status per claim—`verified`, `partially verified`, or `not verified`—with commands and the next action; do not repair or invent proof.
