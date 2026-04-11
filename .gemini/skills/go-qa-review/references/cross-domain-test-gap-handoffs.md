# Cross-Domain Test Gap Handoffs

## Behavior Change Thesis
When loaded for a QA gap that depends on specialist semantics, this file makes the model separate the local executable proof gap from the domain/API/DB/concurrency/security/reliability/performance/design question instead of punting vaguely or over-owning another review lane.

## When To Load
Load this when a missing or weak test depends on domain, API, DB/cache, concurrency, security, reliability, performance, or broader design semantics that QA review should not own alone.

## Decision Rubric
- First name the QA evidence gap: missing scenario, weak assertion, nondeterministic proof, or wrong validation level.
- Hand off only the semantic question QA cannot safely decide.
- Do not hand off just because the file lives in another domain.
- Do not hide an obvious local test fix behind a specialist review.
- Avoid duplicate findings: keep the QA finding about executable proof and put the specialist question in `Handoffs`.

## Imitate

```text
[high] [go-qa-review] internal/infra/postgres/ping_history_repository_test.go:76
Issue:
The new repository error path is tested only for a generic query failure, but the changed behavior depends on preserving transaction rollback after a partially successful write. QA can see the missing rollback proof, while the exact transaction invariant should be reviewed by DB/cache.
Impact:
A partial write can persist after a later failure with unit tests still passing because they never exercise the transaction boundary.
Suggested fix:
Add a rollback scenario using the existing Postgres integration harness and hand off transaction-boundary semantics to `go-db-cache-review`.
Reference:
Validate with `make test-integration` or `go test -tags=integration ./test/... -run 'Rollback' -count=1`; handoff: `go-db-cache-review`.
```

Copy this shape: QA finding first, specialist handoff second, same concrete behavior.

```text
Handoffs:
- `go-concurrency-review`: confirm whether the new shutdown test's channel gate establishes the intended happens-before edge before QA treats the race validation as sufficient.
```

Copy this shape: make the handoff question precise enough that the specialist can answer it.

## Reject

```text
[medium] [go-qa-review] internal/infra/postgres/ping_history_repository_test.go:76
Issue:
This might need DB review.
Impact:
The DB logic could be wrong.
Suggested fix:
Ask the DB agent.
Reference:
Run integration tests.
```

Reject this because it does not state the missing proof or the question for the specialist.

## Agent Traps
- A handoff is not a substitute for a concrete QA finding when the evidence gap is obvious.
- Do not take ownership of threat modeling, transaction isolation, retry budgets, lock correctness, or hot-path benchmark interpretation.
- Do not create separate duplicate findings for QA and specialist lanes; one QA finding plus one handoff is cleaner.
- Do not hand off simple assertion or scenario naming problems.

## Validation Shape
Match validation to the local proof gap, then name the specialist skill for semantic confirmation: `go-db-cache-review` for transaction/cache consistency, `go-concurrency-review` for happens-before and lifecycle proof, `go-security-review` for abuse/threat semantics, `go-reliability-review` for retry/timeout/degradation semantics, `go-performance-review` for benchmark proof, and `go-design-review` for broader design drift.
