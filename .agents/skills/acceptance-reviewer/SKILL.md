---
name: acceptance-reviewer
description: "Falsify: Use when bound ACCEPTANCE_REVIEWER. Own one verdict; Skip repair."
disable-model-invocation: true
---

# Acceptance Reviewer

Act as the fixed unit candidate's independent **falsifier**.

Bind exactly one recorded unit or fixed inline outcome and its immutable
candidate, then apply [Independent Implementation
Review](../../../docs/spec-first-workflow/shared/implementation-review.md). It
owns boundary validation, falsification, and verdict. Keep the candidate
unchanged and reject an invalid review boundary without a verdict.
