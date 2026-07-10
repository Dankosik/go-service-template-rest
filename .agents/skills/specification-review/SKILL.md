---
name: specification-review
description: "Read-only review method for a fixed spec revision, with anchored findings and PASS, CONCERNS, or FAIL."
---

# Specification Review

Use only for a ready spec revision or an explicitly read-only user review. Follow [specification review](../../../docs/spec-first-workflow/phases/specification-review.md).

Try to falsify scope, behavior, invariants, source-of-truth ownership, compatibility, proof feasibility, and downstream clarity only where the spec triggers them. Report anchored blockers, bounded concerns/proof obligations, and non-blocking observations. Do not edit the spec or block on prose polish.

Return `PASS`, `CONCERNS`, or `FAIL`, the evidence boundary, and the smallest repair/reopen owner. For an internal checkpoint, return to the owning root for same-session repair and fresh review without a user handoff. An explicitly requested standalone review returns findings only and stops read-only.
