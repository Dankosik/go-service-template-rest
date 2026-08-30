# Implementation Review

Use when shared [Review](../shared/review.md) routes one fixed implementation
unit, inline outcome, or integrated candidate. This adapter owns only
implementation falsifiers and verdicts.

## Unit

Reject the handoff before detailed review when the unit contains more than one
independently acceptable outcome.

Try to disprove the postcondition and important constraints on the real path,
retained scope, dependencies, and claim-scoped proof. Consume the accepted
sources, fixed candidate identity when crossing a boundary, current proof, and
irreproducible external evidence. Reuse valid receipts and run only a missing or
adversarial falsifier.

A delegated execution lane, generated output, partial package change, or
intermediate handoff is reviewed here only when it is itself a distinct
acceptance unit.

Classify each surviving finding using [Review Result
V1](../interfaces/review-result-v1.md). A blocking finding must identify the
exact accepted claim it falsifies, the candidate anchor, a reproducible or
mechanically checkable failure, and the smallest repair boundary. Style
preferences, alternative architecture, naming improvements, speculative future
risks, and unrelated cleanup are `FOLLOW_UP` unless they falsify the current
Outcome, Boundary, constraint, or Accept-when claim. A `FOLLOW_UP` cannot fail
the current task.

Unjustified structure is not an alternative-architecture preference. An added
abstraction, layer, configuration surface, dependency, compatibility path, or
retained implementation without a current accepted responsibility or constraint
is a `TASK_DEFECT`; anchor it to the Outcome or constraint left unchanged after
deletion or collapse.

`PASS` returns the candidate for acceptance. `FAIL` returns anchored
candidate-caused findings. `NEEDS_PARENT` names proof or action outside reviewer
authority. An unresolved boundary, or a review that still must accept more than
one unit, returns `REVIEW_HANDOFF_INVALID` through [Review Result
V1](../interfaces/review-result-v1.md).

## Delta recheck

When Review selects a bounded delta recheck, falsify only the anchored findings
and their invalidated proof. Do not reopen unaffected reasoning.

## Integrated candidate

Use after two or more accepted units share one integrated candidate. Falsify
only whole-spec coverage, cross-unit compatibility, assembly of the candidate,
and the ledger's global Completion. Do not reopen accepted unit-local findings
unless the integrated candidate invalidated them. Each surviving finding names
the smallest affected unit as repair owner and uses `INTEGRATION_DEFECT` when
the failure is in a seam or assembly rather than an accepted unit-local output.
`PASS` returns the integrated candidate for ledger completion.
