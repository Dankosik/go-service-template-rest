# Implementation Review

Use when shared [Review](../shared/review.md) routes one fixed implementation
unit or inline outcome. This adapter owns only implementation falsifiers and
verdicts.

Try to disprove the postcondition and important constraints on the real path,
retained scope, dependencies, and claim-scoped proof. Consume the accepted
sources, fixed candidate identity when crossing a boundary, current proof, and
irreproducible external evidence. Reuse valid receipts and run only a missing or
adversarial falsifier.

`PASS` returns the candidate for acceptance. `FAIL` returns anchored
candidate-caused findings. `NEEDS_PARENT` names proof or action outside reviewer
authority. An unresolved or multi-unit boundary returns
`REVIEW_HANDOFF_INVALID` through [Review Result
V1](../interfaces/review-result-v1.md).
