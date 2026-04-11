# Scenario Matrix Patterns

## When To Load
Load this when turning approved behavior into a scenario matrix, especially when the plan risks becoming happy-path-only or when fail, edge, abuse, retry, or concurrency rows need sharper observables.

## Source Grounding
- Use approved `spec.md`, design artifacts, and domain/API/data/reliability/security decisions as the source of scenario meaning.
- Use Go's `testing` docs for subtest and focused-run terminology, but keep this file at strategy level.
- Use repository command docs to keep matrix rows executable in local and CI paths.

## Selected/Rejected Level Examples
| Scenario class | Selected level | Rejected level | Why |
| --- | --- | --- | --- |
| Local rule with many parallel examples | Unit table/subtest strategy | Contract | Contract proof adds boundary noise when no public boundary changed. |
| Public validation error shape | Contract | Pure unit | The observable is status, problem body, headers, and documented field mapping. |
| DB uniqueness conflict or transaction rollback | Integration | Unit with fake repository only | The scenario depends on database-enforced behavior and real transaction semantics. |
| Cross-tenant object access denial | Contract or integration | Same-tenant happy path | The negative actor/scope mismatch is the behavior under test. |
| Parser robustness over unexpected input | Fuzz smoke with seed corpus | Enumeration-only unit list | Fuzzing explores inputs the author did not enumerate while preserving failing inputs as regressions. |
| Worker coordination and shared state | Targeted race-aware test plan | E2E smoke only | E2E smoke may miss the interleaving; the race detector needs the risky path executed. |

## Scenario Matrix Examples
Use the smallest matrix that still proves the changed behavior.

| Requirement | Happy path | Fail path | Edge path | Abuse or misuse | Retry/concurrency | Observable |
| --- | --- | --- | --- | --- | --- | --- |
| Request creates one resource | valid actor, valid payload | invalid field rejected | boundary value accepted/rejected as specified | oversized body or unknown field if relevant | same idempotency key replay | status, response body, `Location` or resource state |
| State transition | allowed state moves to next state | forbidden transition rejected | terminal state no-op or conflict as specified | stale version attempt | concurrent transition attempts | persisted state, conflict class, emitted event count |
| Cache-backed read | cache hit returns fresh value | origin failure follows fallback policy | stale or corrupt value handled | tenant key mismatch denied | miss coalescing under parallel reads | returned value, origin call count, cache write or bypass |
| Async processing | accepted work reaches terminal success | retryable dependency failure retried | poison message routed or escalated | duplicate message suppressed | replay after restart | durable state, retry count, DLQ/escalation signal |
| Security boundary | authorized actor succeeds | missing/expired credential denied | tenantless/internal actor follows explicit rule | object ID substitution by another actor | repeated misuse still fail-closed | 401/403 or concealment status, no side effect, audit signal if specified |

## Pass/Fail Observables
- Every row has preconditions, input/data shape, expected outcome, and pass/fail rule.
- Negative rows prove a meaningful rejection or denial, not merely "got an error".
- Edge rows name the boundary value or state that makes the case interesting.
- Retry/concurrency rows state duplicate-suppression, conflict, ordering, or race-sensitive observable.
- Abuse rows appear when trust boundaries, limits, ownership, or caller-controlled identifiers are involved.
- Rows that cannot name an observable should be escalated as untestable or underspecified behavior.

## Exa Source Links
- [testing package](https://pkg.go.dev/testing)
- [go command testing flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags)
- [Go Fuzzing](https://go.dev/doc/fuzz/)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)
- [Go security best practices](https://go.dev/doc/security/best-practices)

