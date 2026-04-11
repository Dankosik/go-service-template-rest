# Cross-Domain Test Gap Handoffs Examples

## When To Load
Load this when a missing or weak test depends on domain, API, DB/cache, concurrency, security, reliability, performance, or broader design semantics that QA review should not own alone.

## Review Lens
QA review owns proof quality. When the question is "what behavior must be true?", "is this security negative sufficient?", "does this transaction/cache path preserve consistency?", or "does this race proof establish a happens-before edge?", name the QA evidence gap and hand off the specialist semantics. Do not hide a local missing test behind a handoff.

## Bad Finding Example
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

Why it fails: it neither names the missing proof nor gives the DB reviewer a concrete question to inspect.

## Good Finding Example
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

## Non-Findings To Avoid
- Do not hand off just because a test lives in another domain; first identify the missing proof obligation.
- Do not let a handoff replace a clear local QA finding when the missing scenario is obvious and bounded.
- Do not take primary ownership of threat model, DB isolation, retry policy, or lock correctness when those semantics decide the expected behavior.
- Do not create duplicate findings across domains; state the QA gap and the specialist question separately.
- Do not demand specialist review when the test correction is a straightforward assertion or named scenario.

## Smallest Safe Correction
Separate proof mechanics from domain semantics:
- write the QA finding around the missing or weak executable evidence;
- add a `Handoffs` entry naming the specialist skill and exact question;
- suggest the smallest local test that would prove the accepted behavior;
- use specialist review only for the behavior definition or risk interpretation that QA cannot safely decide alone.

## Validation Command Examples
```bash
make test-integration
go test -tags=integration ./test/... -run 'PingHistoryRepository' -count=1
go test -race ./internal/infra/http -run '^TestServerRunAndShutdown$' -count=50
make openapi-check
go test ./internal/infra/http -run '^TestRouterRejectsRequestBodyTooLarge$' -count=1
```

## Source Links From Exa
- [testing package docs](https://pkg.go.dev/testing)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)
- [testing/synctest package docs](https://pkg.go.dev/testing/synctest)

## Repo-Local Convention Links
- `docs/build-test-and-development-commands.md`
- `test/README.md`
- `.agents/skills/go-db-cache-review/SKILL.md`
- `.agents/skills/go-concurrency-review/SKILL.md`
- `.agents/skills/go-security-review/SKILL.md`
- `.agents/skills/go-reliability-review/SKILL.md`
- `.agents/skills/go-performance-review/SKILL.md`
