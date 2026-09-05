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

When named checks are still running on the fixed candidate, inspect the code
and supported traces without duplicating those checks. Pending execution alone
is not a defect or unavailable proof. Receive the results before a final PASS;
report discovered defects promptly. Failed or unavailable required proof keeps
its existing failure or NEEDS_PARENT path. If repair changes the candidate,
apply shared Review's delta and freshness rules.

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

For a structure `TASK_DEFECT`, name the current constraint violated and the
concrete unnecessary responsibility, dependency, duplication, or retained path.
Apply the accepted simplicity and dependency constraints, including
[Engineering](../../../AGENTS.md#engineering). A shorter equivalent design alone
does not establish a violation. An alternative organization that preserves
current responsibilities and constraints is `FOLLOW_UP`.

`PASS` returns the candidate for acceptance. `FAIL` returns anchored
candidate-caused findings. `NEEDS_PARENT` names proof or action outside reviewer
authority. An unresolved boundary, or a review that still must accept more than
one unit, returns `REVIEW_HANDOFF_INVALID` through [Review Result
V1](../interfaces/review-result-v1.md).

## Delta recheck

When Review selects a bounded delta recheck, falsify only the anchored findings
and their invalidated proof. Do not reopen unaffected reasoning.

## Integrated candidate

Use when shared Review selects an integrated candidate. Falsify
only whole-spec coverage, cross-unit compatibility, assembly of the candidate,
and the ledger's global Completion. Do not reopen accepted unit-local findings
unless the integrated candidate invalidated them. Each surviving finding names
the smallest affected existing unit's Lead as repair owner when its accepted
boundary covers the repair. Use `INTEGRATION_DEFECT` when the failure is in a
seam or assembly rather than an accepted unit-local output. Reopen Planning only
when no existing unit can own the repair without changing an accepted Outcome,
Boundary, or proof criterion. Preserve unaffected unit acceptance; the repaired
interaction still requires its claim-matched proof and applicable review.
`PASS` returns the integrated candidate for ledger completion.
