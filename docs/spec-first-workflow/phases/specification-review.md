# Specification Review

Use when shared [Review](../shared/review.md) routes one fixed specification or
the user explicitly requests that standalone review. This adapter owns only the
Specification falsifiers and threshold.

## Inputs

Read the fixed candidate/diff, accepted brief, decision-changing Research,
named authorities, current runtime/generated/consumer surfaces, and still-valid
prior findings.

## Method

Independently reconstruct the affected actors and behavior surface; candidate
omission is not evidence that a lens is untriggered. Try to falsify:

- coverage of a materially affected or deliberately unchanged surface;
- factual support, authority, invariant, compatibility/failure expectation, or
  feasible proof expectation;
- a non-goal, risk acceptance, or proof item that hides a decision needed now;
- wording that lets two reasonable implementations differ in observable
  trigger, outcome, transition/effect, truth/finality, rejection/replay/recovery,
  compatibility, or success meaning.

Use the shared [Review Findings](../shared/review-findings.md). Anchor a
divergence to the competing outcomes and missing Specification-owned rule.

## Threshold And Reopen

`PASS` requires no surviving material divergence. `CONCERNS` may carry only
a bounded proof/risk obligation over already accepted behavior. Missing or
contradictory behavior, policy, authority, compatibility, truth semantics, or
success meaning is `FAIL` and reopens Specification or its named upstream
owner.
