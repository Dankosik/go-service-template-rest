---
name: test-design-session
description: "Define non-obvious risk scenarios and proof levels before planning; keep obvious proof directly in tasks.md."
---

# Test Design Session

Use [test design](../../../docs/spec-first-workflow/phases/test-design.md) when proof spans meaningful scenarios, failure modes, protected concerns, or test levels.

Create `test-plan.md` only when a scenario matrix adds value. Each material scenario names its source decision/risk, level, setup/action, expected observable, fail-before signal or rationale, command, residual gap, and reopen owner.

For structured or orchestrated work, use independent QA review and the test-design phase verdict before planning. Direct work uses them only when the user or risk requires it. This skill designs proof; it does not write tests or change approved behavior. The owning root repairs findings, re-reviews to the shared convergence condition, and may continue into planning and implementation in the same authorized request unless the user named test design or standalone QA review as the boundary.

Success means planning can map proof without inventing behavior or test strategy. Otherwise reopen the appropriate spec/design owner.
