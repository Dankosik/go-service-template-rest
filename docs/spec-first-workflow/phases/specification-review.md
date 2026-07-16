# Specification Review

Apply the shared [Review
Independence](../shared/subagents-and-handoff.md#review-independence) contract.
That shared contract owns the generic review envelope and convergence; this file
supplies only Specification-specific affected-surface reconstruction,
falsification lenses, finding fields, and verdict consequences.

## Read When

- The user requests an independent spec review.
- Structured or orchestrated work has a completed specification.
- The spec is high-impact, hard to reverse, ambiguous, cross-owner, or otherwise
  difficult for its author to falsify credibly.
- A repaired spec needs confirmation that a prior blocker is closed.

## Inputs

- The exact `spec.md` revision or content anchor.
- Accepted brief, relevant evidence, named sources of truth, and repository or
  consumer surfaces the accepted outcome can affect.
- Prior findings for a follow-up review.

## Outputs

Add these Specification-specific fields to the shared review envelope:

- spec anchor and evidence;
- downstream impact;
- smallest Specification repair or upstream reopen target.

Verdict consequences:

- `PASS`: within the stated evidence boundary, no readiness gap, current-phase
  defect, unowned question, or uncovered affected lens remains.
- `CONCERNS`: a bounded risk or proof obligation still needs explicit owner
  disposition and fresh review; the spec cannot leave Specification on this
  verdict.
- `FAIL`: a missing or contradictory decision prevents honest design/planning.

## Review Method

Review the fixed candidate through one kernel: reconstruct -> falsify -> disposition -> verdict.

Reconstruct the affected behavior surface independently from the accepted
brief, relevant research, current runtime/generated contracts, and repository
or consumer surfaces the accepted outcome can affect.
Do not treat omission from the spec as evidence that a lens is not triggered.
Falsify it with the smallest decision-changing questions below, then apply the
shared covered/delegated/not-triggered disposition to every materially affected
lens and issue one evidence-bounded verdict on that exact revision.

Try to falsify the spec with the smallest decision-changing questions:

- Do material rules expose the context, trigger, preconditions, observable
  outcome, and applicable failure/absence/replay/recovery semantics needed to
  prevent downstream invention?
- Does a branching policy, lifecycle, interpretation-sensitive rule, or quality
  target use the smallest representation needed to remove ambiguity?
- Are factual claims grounded, normative choices explicitly accepted,
  inferences and assumptions labeled, and missing or conflicting evidence
  explicit?
- Are invariants, authority/source-of-truth ownership, compatibility, failure
  expectations, and proof feasible where relevant?
- Does a non-goal, risk acceptance, or proof obligation hide a decision the spec
  must make now?
- Could two reasonable downstream implementations satisfy the wording yet
  differ on the user/operator-visible outcome, scope, deliberately unchanged
  behavior, or another product decision needed by design/planning?

Report only findings that can change readiness or required proof. Do not block
on writing style, optional detail, or a lens independently shown not to be
triggered. The review stays read-only; repairs belong to the owning author.

## Stop Rule

For an internal checkpoint, return the complete review result to the owning root
under the shared Review Independence contract. For an explicitly requested
standalone review, return the complete result to the user and stop read-only.
