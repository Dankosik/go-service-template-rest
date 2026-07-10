# Task Review / Readiness

Apply the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract. This file supplies only ledger-specific falsification lenses and verdict consequences; it does not define another workflow phase.

## Read When

- The user requests independent plan/readiness review.
- Structured or orchestrated work has a completed implementation ledger.
- Implementation is high-impact, broad, delegated, hard to reverse, or otherwise difficult for the planner to falsify.
- A repaired ledger needs confirmation that a prior blocker is closed.

## Inputs

- Exact `tasks.md` revision.
- Ready spec and required design/test/rollout artifacts.
- Repository source/command evidence needed to check ownership and proof feasibility.

## Outputs

Ranked anchored findings and one verdict:

- `PASS`: implementation can start.
- `CONCERNS`: implementation can start with named bounded risks/proof obligations.
- `FAIL`: the ledger or an upstream decision must be repaired first.

## Review Questions

- Does every task trace to an accepted decision and own an outcome rather than an activity?
- Are dependencies, owner/surface, generated-source order, cleanup, and proof clear?
- Can each proof actually establish the task claim, including failure and negative paths where relevant?
- Is the completion condition successful and observable, not merely “record blocker” or “run commands”?
- Would implementation have to choose behavior, design, test strategy, or rollout policy?
- Is the ledger smaller and clearer than the work it coordinates?

Do not require fields or phase files that cannot change execution or evidence.

## Stop Rule

For an internal checkpoint, return findings to the owning root; it repairs planning defects and re-reviews affected dependency/proof consequences in the same root session without a user handoff. Missing decisions reopen their upstream owner. For an explicitly user-requested standalone review, return findings and stop read-only.
