# Control Flow And Temporal Coupling

## When To Load
Load this when a diff changes branching, nesting, guard clauses, shared sentinels, named returns, deferred cleanup, rollback, audit, or phase ordering.

The goal is readable control flow with explicit side-effect order. Guard clauses are often good, but only when they preserve cleanup and which error wins.

## Review Lens
- Prefer a straight-line happy path when error handling can return early without hiding side effects.
- Flag delayed interpretation through `status`, `action`, `mode`, shared `err`, or booleans decoded at the tail.
- Narrow mutable local lifetimes. Long-lived locals make readers carry branch state through the whole function.
- When flattening, verify defer order, cleanup order, audit order, rollback behavior, and primary-vs-cleanup error precedence.

## Real Finding Examples
Finding example: sentinel state creates a hidden phase machine.

```text
[high] [go-language-simplifier-review] internal/app/imports/run.go:103
Issue: The refactor stores `action`, `status`, and `notify` across multiple branches and decodes them in one tail switch.
Impact: The operation now reads like a manual phase machine, so a future branch can set an inconsistent combination that still compiles.
Suggested fix: Return from each branch after selecting the explicit outcome, or replace the flag cluster with a small typed result that cannot encode invalid combinations.
Reference: references/control-flow-and-temporal-coupling.md
```

Finding example: flattening changed which error wins.

```text
[high] [go-language-simplifier-review] internal/infra/postgres/uow.go:74
Issue: The new guard-clause cleanup returns the rollback error before preserving the operation error.
Impact: The shorter path can hide the failure that caused rollback, which weakens diagnosis and may change caller-visible error matching.
Suggested fix: Keep the primary operation error explicit and attach cleanup failure only if the existing contract allows it.
Reference: references/control-flow-and-temporal-coupling.md
```

## Non-Findings To Avoid
- Do not flag nesting that is required to keep a transaction, lock, or response lifecycle obvious.
- Do not require guard clauses when shared cleanup must run once at the end and the code names that contract clearly.
- Do not object to a short-lived local result when it is consumed immediately and does not encode hidden state.
- Do not turn a control-flow finding into a full architecture review unless package ownership is the real blocker.

## Bad And Good Simplifications
Bad: delayed interpretation by tail state.

```go
status := http.StatusOK
notify := false
if err := validate(input); err != nil {
	status = http.StatusBadRequest
} else if err := reserve(ctx, input); err != nil {
	status = http.StatusConflict
	notify = true
}
writeStatus(w, status, notify)
```

Good: branch outcomes stay visible.

```go
if err := validate(input); err != nil {
	writeValidationProblem(w, err)
	return
}
if err := reserve(ctx, input); err != nil {
	writeRetryableConflict(w, err)
	return
}
writeOK(w)
```

Bad: cleanup failure overwrites the primary failure without saying so.

```go
if err := apply(ctx); err != nil {
	if rollbackErr := rollback(ctx); rollbackErr != nil {
		return rollbackErr
	}
	return err
}
```

Good: the precedence is explicit.

```go
if err := apply(ctx); err != nil {
	if rollbackErr := rollback(ctx); rollbackErr != nil {
		return fmt.Errorf("apply: %w; rollback: %v", err, rollbackErr)
	}
	return err
}
```

## Escalation Guidance
- Escalate to `go-concurrency-review` when control-flow changes hide goroutine, channel, lock, wait, or shutdown ownership.
- Escalate to `go-reliability-review` when timeout, retry, backpressure, startup, or shutdown behavior is at stake.
- Escalate to `go-db-cache-review` when transaction, rollback, rows cleanup, or cache invalidation order is unclear.
- Escalate to `go-idiomatic-review` when the finding depends on named returns, defer behavior, nil behavior, or error wrapping semantics.

## Source Anchors
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): indent error flow, handle errors, goroutine lifetimes, and useful test failures.
- [Effective Go](https://go.dev/doc/effective_go): Go programs should be clear to other Go programmers and follow established conventions.
- Repository pattern: `go-language-simplifier-review/evals/evals.json` includes handler state and hidden precedence scenarios.
- Repository pattern: `go-db-cache-review/references/transaction-boundary-review.md` for transaction order handoff.
