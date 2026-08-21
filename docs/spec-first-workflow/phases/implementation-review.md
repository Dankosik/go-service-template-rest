# Implementation Review

Use when shared [Review](../shared/review.md) routes one fixed implementation
unit or inline outcome. This adapter owns only implementation falsifiers and
verdicts.

Reject the handoff before detailed review when the unit contains more than one
independently acceptable outcome and the ledger records no exact inseparability
reason.

Try to disprove the postcondition and important constraints on the real path,
retained scope, dependencies, and claim-scoped proof. Consume the accepted
sources, fixed candidate identity when crossing a boundary, current proof, and
irreproducible external evidence. Reuse valid receipts and run only a missing or
adversarial falsifier.

Classify each surviving finding using [Review Result
V1](../interfaces/review-result-v1.md). A `FOLLOW_UP` cannot fail the current
task; when it falsifies the packet's Outcome, Boundary, or Accept-when claim, it
is a `TASK_DEFECT` instead.

`PASS` returns the candidate for acceptance. `FAIL` returns anchored
candidate-caused findings. `NEEDS_PARENT` names proof or action outside reviewer
authority. An unresolved or multi-unit boundary returns
`REVIEW_HANDOFF_INVALID` through [Review Result
V1](../interfaces/review-result-v1.md).
