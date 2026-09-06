---
name: go-reliability
description: "Retry and deadline budgets. Use for any Go decision or review involving end-to-end deadlines, per-attempt timeouts, retry or backoff counts, overload, readiness, drain, shutdown, or rollout recovery."
metadata:
  invocation: model
  kind: method
---

# Go Reliability

Resilience is **budget** arithmetic: every dependency call and retry spends the
caller's remaining end-to-end budget.

`budget -> per-hop deadline -> failure disposition -> retry, degrade, or shed -> lifecycle -> rollout -> proof`

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
Trace the accepted parent budget through every wait, attempt, queue or pool
acquire, and terminal response or durable handoff. When comparing proposed
attempt counts, covering interacting waits, or handing off a Decision or
Review, record
`BudgetPath{parent_budget, waits, attempts, worst_case_spend, terminal_reserve,
attempt_admission, repeatability, failure_disposition, lifecycle, signal,
proof, gaps}` per path in the existing artifact or required result. A single
local path can retain its complete budget arithmetic and admission predicate
with matching proof without a separate record.

Reserve time for the terminal response or durable handoff before allocating
attempts and backoff. A retry policy satisfies:

`sum(attempt budgets) + sum(backoffs) + terminal reserve <= parent budget`

Use only accepted values or values read from their current owner. Keep every
unaccepted reserve, backoff bound, or useful-attempt minimum symbolic and return
it as an exact blocker; never choose numbers merely to make the inequality fit.
Proving that `N` attempts do not fit proves nothing about `N-1`: evaluate every
proposed count with the same complete inequality.

Start another attempt only when remaining budget covers the terminal reserve
plus a minimum useful attempt; cap its deadline at `remaining - reserve`. If the
reserve, backoff bound, or useful-attempt minimum is not accepted, return that
exact gap instead of inventing a fixed attempt count. A retry also needs jitter
and a safely repeatable effect; degradation and load shedding need named
behavior and signals.

Read the existing owner before adding policy: `internal/infra/http` owns request
budgets and shedding, `internal/health` owns cached readiness,
`cmd/service/internal/bootstrap` owns the clamped shutdown deadline, and
`internal/config/validate.go` enforces nested budgets. Load the [reference
selector](references/index.md) for a new outbound wait, pooled acquire,
concurrency or queue bound, readiness, drain, or teardown pressure.

For a **Decision**, disposition every dependency and lifecycle stage with
budget, failure, recovery, and rollout consequences. For **Review**, trace each
affected failure path. Reject a retry whose worst-case spend can exceed the
parent budget even when each individual attempt has a timeout. Complete when
every affected path has complete arithmetic and an admission predicate,
every path fits the parent bound, and accepted work has one visible terminal or
durable handoff disposition.
