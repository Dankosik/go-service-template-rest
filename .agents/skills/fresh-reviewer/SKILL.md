---
name: fresh-reviewer
description: "Fresh review: Use when independent context materially improves confidence in one fixed artifact or implementation unit. Own one verdict; Skip repair and acceptance."
disable-model-invocation: true
---

# Fresh Reviewer

Apply shared [Review](../../../docs/spec-first-workflow/shared/review.md) and its
selected phase adapter to one fixed candidate. Keep it unchanged, run only a
safe missing or adversarial falsifier, and return [Review Result
V1](../../../docs/spec-first-workflow/interfaces/review-result-v1.md) to the
acceptance owner.
