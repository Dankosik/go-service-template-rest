---
name: go-verification-before-completion
description: "Claim-scoped proof: Use when correctness/readiness/completion needs current evidence. Own claim-command matching, scope, and gaps; Skip changes, unknown cause, unresolved strategy, or active implementation."
---

# Go Verification Before Completion

A completion claim is only as true as its **freshest matching evidence**: every claim maps to a command run against the current state at the claim's own scope, and anything staler, narrower, or inferred stays unverified.

`intended claims -> claim-to-command map -> scope match -> fresh or validly reused evidence -> per-claim status -> gaps`

Use [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for completion authority.

Reconstruct every claim the answer intends to make from the accepted outcome, proposed response, changed scope, and named rollout/readiness assertions; map each claim to current proof of equal scope and reject stale or narrower evidence. Reuse successful proof while the relevant content, environment or precondition, claim scope, provenance, and risk surface remain unchanged; attach a commit/tree identity only when proof crosses a checkout or integration boundary. A report from a worker, prior session, tool, or pasted log names the proof target rather than supplying it — rerun the claim-scoped command against this tree. Load [the reference selector](references/index.md) only when one concrete proof pressure can change the command or conclusion.

Return one evidence-permitted status per claim — `verified`, `partially verified`, or `not verified` — with commands and the next action; the closing summary carries the weakest status among the claims it covers. Verification reports evidence and gaps, and repair belongs to the owning skill. Route unknown root cause to `go-systematic-debugging` and unset rollout criteria to `go-delivery-platform`.
