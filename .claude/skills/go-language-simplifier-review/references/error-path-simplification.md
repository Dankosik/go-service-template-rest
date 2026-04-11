# Error-Path Simplification

Behavior Change Thesis: When loaded for deduplicated or normalized error handling, this file makes the model protect inspectability, status mapping, cancellation, and cleanup precedence instead of likely mistake of accepting generic error helpers that remove visible repetition.

## When To Load
Load this when a diff deduplicates, wraps, unwraps, normalizes, logs, maps, joins, or reorders error handling, especially when the stated goal is to reduce repetition.

Use this when the simplification risk is the error contract itself. If the risk is mainly temporal ordering around rollback or audit, use `control-flow-and-temporal-coupling.md`.

## Decision Rubric
- Keep distinct failure classes distinct when callers, operators, retries, transports, or tests must reason about them differently.
- Flag helpers that collapse validation, conflict, retryable, not-found, timeout, cancellation, and internal errors into one generic bucket.
- Preserve `errors.Is`, `errors.As`, and Go 1.26+ `errors.AsType` behavior when inspectability is part of the contract and the module supports it. Prefer `AsType` for concrete error-type checks in Go 1.26+ modules, but keep `errors.As` when the target is a non-error interface because `AsType` is constrained to error types.
- Do not demand `%w` for every wrapped error; wrapping can expose internals when callers should not inspect them.
- Keep which error wins explicit when cleanup, audit, rollback, notification, or logging can also fail.

## Imitate
Finding shape to copy when a generic helper destroys error identity:

```text
[high] [go-language-simplifier-review] internal/app/users/service.go:118
Issue: `newServiceError("save user", err)` formats the database and conflict paths with `%v`, so callers can no longer match `ErrEmailTaken` with `errors.Is`.
Impact: The helper removes visible repetition by weakening the caller-visible error contract and can route conflicts as internal failures at the transport boundary.
Suggested fix: Preserve the original wrapping contract with `%w` for inspectable errors, or keep the conflict branch local when it needs different mapping.
Reference: references/error-path-simplification.md
```

Copy the move: name the error identity that callers or transports rely on.

Finding shape to copy when dedupe merges cancellation with internal errors:

```text
[medium] [go-language-simplifier-review] internal/infra/http/export.go:91
Issue: The shared `writeError` path maps `context.Canceled`, deadline, validation, and internal failures through the same 500 response helper.
Impact: The response path is shorter, but operators and callers lose the distinction between client cancellation, timeout, and server failure.
Suggested fix: Keep cancellation and deadline checks explicit before the generic problem writer, or split policy-named helpers by failure class.
Reference: references/error-path-simplification.md
```

Copy the move: name the failure classes that should remain distinct and why.

## Reject
Reject generic formatting when it hides inspectable causes:

```go
func serviceErr(op string, err error) error {
	return fmt.Errorf("%s failed: %v", op, err)
}
```

Preserve inspection when callers own that contract:

```go
func serviceErr(op string, err error) error {
	return fmt.Errorf("%s failed: %w", op, err)
}
```

Reject a shared mapper that collapses all failure classes:

```go
func writeError(w http.ResponseWriter, err error) {
	http.Error(w, "request failed", http.StatusInternalServerError)
}
```

Prefer stable classes before fallback:

```go
func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, users.ErrNotFound):
		http.Error(w, "user not found", http.StatusNotFound)
	case errors.Is(err, users.ErrConflict):
		http.Error(w, "user conflict", http.StatusConflict)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
```

## Agent Traps
- Do not treat repeated `if err != nil { return ... }` as debt unless there is stable shared policy or diagnosis harm.
- Do not require wrapping when the original boundary intentionally hid internals.
- Do not approve an `errors.As` to `errors.AsType` cleanup unless the target type itself implements `error`.
- Do not let a "logging cleanup" reorder logging before the context needed for diagnosis exists.
- Do not make simplification review the final authority on subtle Go error trees; hand off deep wrapping or nil behavior questions to `go-idiomatic-review`.

## Validation Shape
Ask for proof that inspectable errors still satisfy `errors.Is`, `errors.As`, or Go 1.26+ `errors.AsType` when the module supports it, cancellation/deadline paths still map distinctly, and transport status mapping remains stable. For cleanup precedence, also verify the primary error remains visible according to the existing contract.
