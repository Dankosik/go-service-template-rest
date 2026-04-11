# Control Flow And Temporal Coupling

Behavior Change Thesis: When loaded for branch flattening or phase-order changes, this file makes the model protect explicit side-effect and error precedence instead of likely mistake of praising guard clauses, loops, or tail switches that hide temporal coupling.

## When To Load
Load this when a diff changes branching, nesting, guard clauses, shared sentinels, named returns, deferred cleanup, rollback, audit, notification, or phase ordering.

Use this when the review question is "does the order still read safely?" If the issue is mostly error identity or status mapping, use `error-path-simplification.md`; if it is cleanup code that protects Go semantics, use `go-semantic-stop-signs.md`.

## Decision Rubric
- Guard clauses are good only when they preserve side-effect order, cleanup order, and which error wins.
- Flag delayed interpretation through `status`, `action`, `mode`, shared `err`, shared result structs, or booleans decoded at the tail.
- Flag loops over operations when the order is part of the contract and the step names no longer expose the durable boundary.
- Prefer narrowing mutable local lifetimes; long-lived locals make readers carry branch state through the function.
- Do not flag nesting that is required to keep a transaction, lock, response lifecycle, or cleanup boundary obvious.

## Imitate
Finding shape to copy when sentinel state creates a hidden phase machine:

```text
[high] [go-language-simplifier-review] internal/app/imports/run.go:103
Issue: The refactor stores `action`, `status`, and `notify` across multiple branches and decodes them in one tail switch.
Impact: The operation now reads like a manual phase machine, so a future branch can set an inconsistent combination that still compiles.
Suggested fix: Return from each branch after selecting the explicit outcome, or replace the flag cluster with a small typed result that cannot encode invalid combinations.
Reference: references/control-flow-and-temporal-coupling.md
```

Copy the move: explain the inconsistent-combination risk created by long-lived branch state.

Finding shape to copy when flattening changes error precedence:

```text
[high] [go-language-simplifier-review] internal/infra/postgres/uow.go:74
Issue: The new guard-clause cleanup returns the rollback error before preserving the operation error.
Impact: The shorter path can hide the failure that caused rollback, which weakens diagnosis and may change caller-visible error matching.
Suggested fix: Keep the primary operation error explicit and attach cleanup failure only if the existing contract allows it.
Reference: references/control-flow-and-temporal-coupling.md
```

Copy the move: identify which error used to win and why callers or operators care.

## Reject
Reject delayed interpretation like this:

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

Prefer branch outcomes that stay visible:

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

Reject cleanup precedence that hides the primary failure:

```go
if err := apply(ctx); err != nil {
	if rollbackErr := rollback(ctx); rollbackErr != nil {
		return rollbackErr
	}
	return err
}
```

Prefer explicit precedence:

```go
if err := apply(ctx); err != nil {
	if rollbackErr := rollback(ctx); rollbackErr != nil {
		return fmt.Errorf("apply: %w; rollback: %v", err, rollbackErr)
	}
	return err
}
```

## Agent Traps
- Do not call every early return simpler; first prove shared cleanup still runs exactly when intended.
- Do not convert a lifecycle into a slice of anonymous functions when phase names carry operational meaning.
- Do not treat named returns, defer, and rollback precedence as style-only; they often define contracts.
- Do not escalate to architecture unless package ownership, not local order, is the blocker.

## Validation Shape
Ask for targeted tests around each branch whose order or error precedence changed: primary error preserved, cleanup failure represented according to contract, audit/notification emitted only after the durable boundary, and cancellation/rollback behavior unchanged.
