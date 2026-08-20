# Specification Review

Use when shared [Review](../shared/review.md) routes one fixed specification or
the user explicitly requests that standalone review. This adapter owns only the
Specification falsifiers and threshold.

## Lenses

- coverage of a materially affected or deliberately unchanged surface;
- factual support, authority, invariant, compatibility/failure expectation, or
  feasible proof expectation;
- an assumption, owner boundary, irreversible consequence, or rollout
  compatibility claim without an evidence-backed disposition;
- a non-goal, risk acceptance, or proof item that hides a decision needed now;
- wording that lets two reasonable implementations differ in observable
  trigger, outcome, transition/effect, truth/finality, rejection/replay/recovery,
  compatibility, or success meaning.

`PASS` requires no surviving material divergence. `CONCERNS` may carry only
a bounded proof/risk obligation over already accepted behavior. Missing or
contradictory behavior, policy, authority, compatibility, truth semantics, or
success meaning is `FAIL` and reopens Specification or its named upstream
owner.
