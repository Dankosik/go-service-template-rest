# Inbox TD-INBOX-004 cancellation recovery

status: done

Completion: the shared PostgreSQL transaction wrapper attributes one
watcher-owned cancellation without rewriting an earlier server error, settles
the protocol operation before one non-cancelled rollback, and the real inbox
lock-wait carrier proves no claim/effect leak and a reusable backend. The
accepted outbox-production-closure T5 receipt remains intact; this is its
narrow recovery, not a replacement inbox capability.

Blocked stop: leave this unit unchecked and record the failed TD-INBOX-004
oracle, narrower evidence, and its reopen owner. Reopen Go Ownership if a
different transaction owner must clean up; Technical Design if the shared
watcher cannot retain the one-operation mark through settlement; Test Design
if the fixed fake or real-PostgreSQL oracle cannot distinguish the result.

Global constraints: preserve the stateless inbox callback, caller-owned
transaction, unique-index arbitration, handler budget, and no-duplicate-effect
contract. No per-inbox cancellation path, pool option, Session/stage edit,
generic helper, query or rollback retry, schema/profile/transport change, or
second integration carrier is authorized. PostgreSQL integration and race
resources are serialized.

## Obligation reconciliation

| Accepted obligation | Recovery disposition |
| --- | --- |
| TD-INBOX-004 caller cancellation versus an unrelated `57014`, one cleanup, no claim/effect, backend reuse, and joined resources | INBOX-TD-INBOX-004-R1 owns the shared `Pool.InTx` watcher/settlement/rollback path, wrapper-local proof, and the existing real PostgreSQL inbox carrier. |
| Accepted inbox capability and outbox-production-closure T5 receipt | Retained unchanged; this unit repairs only the later-falsified shared wrapper boundary. |
| PR-GOSEC-01 aggregate failure | Remains blocked until this unit is accepted, then requires one fresh complete recorded PR-GOSEC proof, one serialized `make check-full`, and fresh implementation review before HTTP T5 resumes. |

## Acceptance units

- A-INBOX-TD-INBOX-004-R1: INBOX-TD-INBOX-004-R1 — one serial Local singleton because the shared wrapper and one real PostgreSQL lock-wait carrier jointly establish cancellation attribution and cleanup, and both use exclusive PostgreSQL/race resources.

- [x] INBOX-TD-INBOX-004-R1: A canceled duplicate-claim waiter is attributed to its handler without leaking transaction state or rewriting an earlier server PostgreSQL error
  - Source: `spec.md` R1/R3 and unchanged cancellation invariant; `design/overview.md` Inbox claim and transaction failure boundary plus Go responsibility map; `test-plan.md` TD-INBOX-004; accepted `../outbox-production-closure/tasks.md` T5 receipt.
  - Owner/surface/resources: only `internal/infra/postgres/{postgres.go,transaction.go,statement_timeout_test.go,transaction_test.go}` and `test/postgres_inbox_integration_test.go`. `Pool.InTx` owns the connection-scoped watcher mark, settlement, one non-cancelled rollback, and primary/cleanup error preservation; the inbox remains a stateless caller-transaction consumer. The existing real PostgreSQL fixture, unique-index lock, tagged waiter backend, cancellation event, and race run are exclusive resources.
  - Depends on: none.
  - Proof: TD-INBOX-004 wrapper-local oracle: `go test -vet=off ./internal/infra/postgres -run '^Test(InTx|RunInTx|ContextWatcher)' -count=1`; a tracked fake injects unmarked `57014` before parent cancellation and proves server error primary, exactly one post-settlement non-cancelled rollback, and a retained distinct cleanup error. TD-INBOX-004 real carrier: `REQUIRE_DOCKER=1 go test -tags=integration ./test -run '^TestPostgresInboxConcurrentClaimAndEffect$' -count=1` and `REQUIRE_DOCKER=1 go test -vet=off -race -tags=integration ./test -run '^TestPostgresInboxConcurrentClaimAndEffect$' -count=1`; it observes the waiter lock, proves commit skips and rollback applies exactly once, and for cancellation proves `errors.Is(err, context.Canceled)` while retaining watcher-marked `57014`, no waiter claim/effect, idle/reusable backend, no lingering waiter lock, and all goroutines/transactions joined.
  - Acceptance: bounded self-review, `git diff --check`, and one fresh independent fixed-unit Implementation Review after every named command passes. This receipt invalidates the blocked PR-GOSEC aggregate: do not rerun it before acceptance; afterward dispatch exactly one fresh PR-GOSEC-01 replacement for its complete recorded focused proof, one serialized recoverable `make check-full`, and fresh review. HTTP T5 remains frozen until that PR-GOSEC receipt accepts.
  - Reopen if: the connection-scoped watcher cannot settle the in-flight protocol operation before wrapper-owned cleanup, or one rollback cannot retain deterministic caller/server/cleanup identity — Technical Design; a different wrapper owns cleanup — Go Ownership; the fixed fake or real-PG lock-wait carrier cannot establish its oracle — Test Design.
  - Accepted: INBOX-TD-INBOX-004-R1; evidence: TD-INBOX-004 wrapper-local, real PostgreSQL, and race commands PASS; `git diff --check` PASS; fresh independent implementation review PASS; candidate: current bounded diff.

## Readiness review

Fresh independent Task Review / Readiness **PASS**: INBOX-TD-INBOX-004-R1 is
an executable serial singleton from the current watcher and transaction-wrapper
sources, the review-cleared inbox design/test contract, fixed writable paths,
and exclusive PostgreSQL/race resources. Its wrapper and real-PG oracles,
implementation-review boundary, and blocked PR-GOSEC replacement handoff leave
no behavior, ownership, mechanism, or proof choice to Implementation. This
verdict authorizes only INBOX-TD-INBOX-004-R1.
