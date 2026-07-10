# Specification Review

Apply the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract. This file supplies only spec-specific falsification lenses and verdict consequences; it does not define another workflow phase.

## Read When

- The user requests an independent spec review.
- The spec is high-impact, hard to reverse, ambiguous, cross-owner, or otherwise difficult for its author to falsify credibly.
- A repaired spec needs confirmation that a prior blocker is closed.

## Inputs

- The exact `spec.md` revision or content anchor.
- Accepted brief, relevant evidence, and named sources of truth.
- Prior findings for a follow-up review.

## Outputs

Ranked findings with:

- spec anchor and evidence;
- downstream impact;
- classification: blocker, bounded concern/proof obligation, or non-blocking observation;
- owner and smallest repair/reopen target.

Verdict:

- `PASS`: no material readiness gap found within the evidence boundary.
- `CONCERNS`: the spec is usable, with named bounded risks or proof obligations.
- `FAIL`: a missing or contradictory decision prevents honest design/planning.

## Review Questions

- Is scope and behavior unambiguous to callers/operators?
- Are invariants, source-of-truth ownership, compatibility, failure expectations, and proof feasible where relevant?
- Does a non-goal or proof obligation hide a decision the spec must make now?
- Could design or planning proceed without inventing product meaning?

Report only findings that can change readiness or required proof. Do not block on writing style, optional detail, or a domain that has no trigger in the spec.

## Stop Rule

For an internal checkpoint, return the advisory verdict to the owning root; it repairs and re-reviews the affected surface in the same root session without a user handoff. For an explicitly user-requested standalone review, return findings and stop read-only. Do not rerun unrelated lenses by default.
