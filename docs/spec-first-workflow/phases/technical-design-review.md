# Technical Design Review

Apply the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract. This file supplies only design-specific falsification lenses and verdict consequences; it does not define another workflow phase.

## Read When

- The user requests independent design review.
- Structured or orchestrated work has triggered and completed technical design.
- Design is high-impact, hard to reverse, cross-owner, or difficult for the author to falsify.
- A repaired design needs confirmation that a prior blocker is closed.

## Inputs

- Ready spec and exact design revision.
- Relevant repository architecture and runtime/generated sources.
- Carried spec concerns and prior design findings.

## Outputs

Ranked anchored findings and one verdict:

- `PASS`: planning can proceed without hidden design work.
- `CONCERNS`: planning can proceed with named bounded risk/proof obligations.
- `FAIL`: specification or design must change first.

## Review Questions

- Are source of truth, contracts, runtime sequence, failure behavior, data/consistency, rollout, and proof explicit where relevant?
- Are package/file ownership, dependency direction, generated/manual authority, cleanup, and test ownership clear?
- Are viable alternatives genuinely closed, or has implementation been left a live fork?
- Can planning name task owners, files, tests, and evidence without deciding design?
- Does the design add abstraction, dependency, or machinery without a present requirement?

Do not block on local naming or task order after the owning design decision is clear.

## Stop Rule

For an internal checkpoint, return findings to the owning root; it routes and repairs blockers and re-reviews affected assumptions in the same root session without a user handoff. For an explicitly user-requested standalone review, return findings and stop read-only. Do not re-review the entire packet by default.
