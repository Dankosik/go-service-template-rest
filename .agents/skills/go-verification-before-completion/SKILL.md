---
name: go-verification-before-completion
description: "Evidence boundary: Use for a verification-only outcome or when a claim's proving scope is non-obvious or disputed. Own claim-to-evidence mapping and gaps; Skip routine validation already selected by Direct Work or Implementation."
metadata:
  invocation: model
  kind: method
---

# Go Verification Before Completion

Apply the shared [Evidence
Contract](../../../docs/spec-first-workflow/shared/evidence-contract.md). Load
one matching [reference](references/index.md) only when the selected proof's
claim boundary is non-obvious. Return [Evidence Result
V1](../../../docs/spec-first-workflow/interfaces/evidence-result-v1.md).

Route an unknown cause to `go-systematic-debugging` and missing rollout
criteria to `go-delivery-platform`.
