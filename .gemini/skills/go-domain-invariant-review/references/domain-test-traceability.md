# Domain Test Traceability

## Behavior Change Thesis
When loaded for symptom "a changed business rule lacks proof that would fail on the risky regression", this file makes the model report a specific domain proof gap instead of likely mistake "ask for more tests, higher coverage, or table-driven style generally."

## When To Load
Load this when a review changes invariant enforcement, transition guards, acceptance/rejection semantics, duplicate handling, failure ordering, or side-effect safety and the proving tests are missing, weak, renamed away from the business rule, or no longer assert the risky outcome.

## Decision Rubric
- Missing proof is a domain finding only when a changed production or test line lets a named business regression pass unnoticed.
- Point to the changed production line when possible; point to the changed test line when the defect is weakened proof.
- Ask for the smallest test or assertion that would fail on the risky regression.
- Hand off broad test strategy, fixture design, assertion style, and flake depth to `go-qa-review`.

## Imitate
```text
[medium] [go-domain-invariant-review] internal/orders/service.go:41
Issue:
Approved behavior says rejected completion must not charge the customer, but the changed tests only assert `ErrOrderNotAuthorized` and never assert that `payments.Capture` was not called on the rejected path.
Impact:
A future guard-order regression can pass the suite while still charging customers for orders the domain rejects.
Suggested fix:
Add a focused negative test with a fake payment recorder that calls `Complete` on a non-authorized order and asserts the domain error plus zero capture attempts. Keep broader fixture and assertion-style review with `go-qa-review`.
Reference:
Local order completion spec and changed test expectations.
```

Copy the shape: rule, missing failure signal, regression that could pass, smallest proving assertion.

## Reject
```text
[low] internal/orders/service_test.go:20
Add more tests for edge cases.
```

Failure: this is generic QA advice without the invariant, failure mode, or business impact.

## Agent Traps
- Do not flag missing tests for behavior that did not change and is not adjacent to the diff's domain risk.
- Do not ask for line coverage, table-driven style, or fixture refactors as domain findings.
- Do not require integration tests when a focused unit test would prove the invariant.
- Do not duplicate `go-qa-review`; stay on the business rule that lacks proof.

## Validation Shape
Prefer one targeted proof: forbidden transition rejects, rejected command triggers no side effect, duplicate/replay is no-op or rejection by contract, stale input does not overwrite newer state, or a renamed test still names/asserts the approved rule.
