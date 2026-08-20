---
name: go-verification-before-completion
description: "Verification: Use for correctness, readiness, or completion claims. Own matching evidence and gaps; Skip changes, diagnosis, and strategy."
---

# Go Verification Before Completion

A completion claim is only as true as its **freshest matching evidence**: every claim maps to a command run against the current state at the claim's own scope, and anything staler, narrower, or inferred stays unverified.

`intended claims -> claim-to-command map -> scope match -> fresh or validly reused evidence -> per-claim status -> gaps`

The [Direct Work stop rule](../../../docs/spec-first-workflow/direct-work.md#stop-rule)
owns direct completion claims. Load [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md)
only when its structured or conditional boundary applies.

Reconstruct every claim the answer intends to make from the accepted outcome, proposed response, changed scope, and named rollout/readiness assertions; map each claim to current proof of equal scope and reject stale or narrower evidence. Reuse or rerun proof under the Implementation [Evidence Contract](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md#evidence-contract); rerun when equal scope, identity, provenance, preconditions, or risk surface cannot be established. Load [the reference selector](references/index.md) only when one concrete proof pressure can change the command or conclusion.

Return one evidence-permitted status per claim — `verified`, `partially verified`, or `not verified` — with commands and the next action; the closing summary carries the weakest status among the claims it covers. Verification reports evidence and gaps, and repair belongs to the owning skill. Route unknown root cause to `go-systematic-debugging` and unset rollout criteria to `go-delivery-platform`.
