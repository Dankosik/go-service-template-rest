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
- Dispositioned accepted risks/downstream proof obligations and prior design findings.

## Outputs

Ranked anchored findings and one verdict:

- `PASS`: the sufficient material evidence boundary has no hidden design work, current-phase defect, unowned question, or uncovered affected lens.
- `CONCERNS`: a bounded risk or downstream proof obligation still needs explicit owner disposition and fresh review; it does not permit planning.
- `FAIL`: specification or design must change first.

## Review Questions

- Are source of truth, contracts, runtime sequence, failure behavior, data/consistency, rollout, and proof explicit where relevant?
- Are package/file ownership, dependency direction, generated/manual authority, cleanup, and test ownership clear?
- Are viable alternatives genuinely closed, or has implementation been left a live fork?
- Can planning name task owners, files, tests, and evidence without deciding design?
- Can the reviewer inventory every input-bearing design surface on the current implementation completion path and materialize one representative of each materially distinct contract, record, configuration, migration input, or proof setup from approved sources without choosing semantics? When byte- or signature-sensitive behavior applies, can the reviewer also reproduce each required golden vector?
- Does the design add abstraction, dependency, or machinery without a present requirement?

Do not block on local naming or task order after the owning design decision is clear.

## Stop Rule

For an internal checkpoint, return findings to the owning root; it routes repairs and re-reviews under the shared convergence contract in the same root session without a user handoff. Use focused re-review for local repairs and full affected-surface review when behavior, contracts, ownership, shared assumptions, or proof changes. A focused re-review may close the named finding, but a phase-level `PASS` requires rechecking implementation-input closure across every materially distinct input-bearing surface on the current completion path. For an explicitly user-requested standalone review, return findings and stop read-only.
