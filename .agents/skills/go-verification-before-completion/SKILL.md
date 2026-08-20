---
name: go-verification-before-completion
description: "Verification: Use for correctness, readiness, or completion claims. Own matching evidence and gaps; Skip changes, diagnosis, and strategy."
---

# Go Verification Before Completion

Apply the shared [Evidence
Contract](../../../docs/spec-first-workflow/shared/evidence-contract.md) to every
intended correctness, readiness, or completion claim.

Return one record per claim with matching current evidence, `verified`,
`partially_verified`, or `not_verified`, and the remaining gap or next owner.
Load the [reference selector](references/index.md) only when a concrete proof
surface changes the evidence mapping.

Do not repair the implementation. Route an unknown cause to
`go-systematic-debugging` and missing rollout criteria to
`go-delivery-platform`.
