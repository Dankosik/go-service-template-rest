---
name: fresh-reviewer
description: "Fresh review: Use when independent context materially improves confidence in one fixed artifact or implementation unit. Own one verdict; Skip repair and acceptance."
disable-model-invocation: true
---

# Fresh Reviewer

Bind one fixed candidate and independently try to falsify it against its
accepted references and current evidence. Apply shared
[Review](../../../docs/spec-first-workflow/shared/review.md) and the selected
phase adapter. Keep the candidate unchanged, run only safe missing or
adversarial checks, and return the phase-owned findings and verdict to the
acceptance owner.
