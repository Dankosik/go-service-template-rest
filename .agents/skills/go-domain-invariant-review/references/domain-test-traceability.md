# Domain Test Traceability Review Examples

## When To Load
Load this when a review changes invariant enforcement, transition guards, acceptance/rejection semantics, duplicate handling, failure ordering, or side-effect safety and the proving tests are missing, weak, renamed away from the business rule, or no longer assert the risky outcome.

Use approved specs, task artifacts, domain docs, and existing test names/assertions as authority. This file helps decide when missing proof is itself a domain review finding. Hand off broad test strategy, fixture structure, and assertion design depth to `go-qa-review`.

## Review Lens
Domain traceability is about whether a critical business rule can regress unnoticed. The finding should point to the changed production line or the changed test line, name the local rule, and explain what business regression can slip through. Avoid coverage-count advice.

## Bad Finding Example
```text
[low] internal/orders/service_test.go:20
Add more tests for edge cases.
```

Why it fails: it is generic QA advice without the invariant, failure mode, or business impact.

## Good Finding Example
```text
[medium] [go-domain-invariant-review] internal/orders/service.go:41
Issue:
Approved behavior says rejected completion must not charge the customer, but the changed tests only assert `ErrOrderNotAuthorized` and never assert that `payments.Capture` was not called on the rejected path.
Impact:
A future guard-order regression can pass the suite while still charging customers for orders the domain rejects.
Suggested fix:
Add a focused negative test with a fake payment recorder that calls `Complete` on a non-authorized order and asserts the domain error plus zero capture attempts. Keep broader fixture and assertion-style review with `go-qa-review`.
Reference:
Local order completion spec and changed test expectations; invariant-test guidance is calibration only.
```

## Non-Findings To Avoid
- Do not flag missing tests for behavior that did not change and is not adjacent to the diff's domain risk.
- Do not ask for line coverage, table-driven style, or fixture refactors as domain findings.
- Do not require integration tests when a focused unit test would prove the invariant.
- Do not duplicate `go-qa-review`; stay on the business rule that lacks proof.

## Smallest Safe Correction
Prefer the smallest proof that would fail on the risky regression:
- one negative test for forbidden transition or rejected command;
- one assertion that irreversible side effects did not fire on rejection;
- one duplicate/replay test proving the second application is no-op or rejection by contract;
- one stale-version test proving newer state is not overwritten;
- one test name or assertion update that restores traceability to the approved rule.

## Escalation Cases
Escalate when:
- the proof gap reflects missing or contradictory acceptance criteria;
- the required test level depends on DB/cache transactions, external side effects, or async retry policy;
- the test cannot be written without new domain seams or design changes;
- the validation obligation belongs to API contract, security, reliability, or data/cache behavior more than domain correctness;
- broad test strategy or flaky infrastructure is the blocker.

## Source Links From Exa
- [Martin Fowler: Test Invariant](https://martinfowler.com/bliki/TestInvariant.html)
- [Martin Fowler: Domain Model](https://martinfowler.com/eaaCatalog/domainModel.html)
- [Microsoft Learn: Use tactical DDD to design microservices](https://learn.microsoft.com/en-us/azure/architecture/microservices/model/tactical-domain-driven-design)
- [Microservices.io: Idempotent Consumer](https://microservices.io/patterns/communication-style/idempotent-consumer.html)
