# Error-Path Simplification

## When To Load
Load this when a diff deduplicates, wraps, unwraps, normalizes, logs, maps, joins, or reorders error handling, especially when the stated goal is to reduce repetition.

Simpler error code must preserve the error contract readers and callers rely on: error identity, type inspection, cancellation, status mapping, cleanup precedence, and operator diagnosis.

## Review Lens
- Keep distinct failure classes distinct when callers, operators, retries, or transports must reason about them differently.
- Flag helpers that collapse validation, conflict, retryable, not-found, timeout, cancellation, and internal errors into one generic bucket.
- Preserve `errors.Is` and `errors.As` behavior when that is the contract.
- Keep which error wins explicit when cleanup, audit, rollback, or notification can also fail.

## Real Finding Examples
Finding example: a generic helper destroyed error identity.

```text
[high] [go-language-simplifier-review] internal/app/users/service.go:118
Issue: `newServiceError("save user", err)` formats the database and conflict paths with `%v`, so callers can no longer match `ErrEmailTaken` with `errors.Is`.
Impact: The helper removes visible repetition by weakening the caller-visible error contract and can route conflicts as internal failures at the transport boundary.
Suggested fix: Preserve the original wrapping contract with `%w` for inspectable errors, or keep the conflict branch local when it needs different mapping.
Reference: references/error-path-simplification.md
```

Finding example: dedupe merged cancellation with internal errors.

```text
[medium] [go-language-simplifier-review] internal/infra/http/export.go:91
Issue: The shared `writeError` path maps `context.Canceled`, deadline, validation, and internal failures through the same 500 response helper.
Impact: The response path is shorter, but operators and callers lose the distinction between client cancellation, timeout, and server failure.
Suggested fix: Keep cancellation and deadline checks explicit before the generic problem writer, or split policy-named helpers by failure class.
Reference: references/error-path-simplification.md
```

## Non-Findings To Avoid
- Do not flag a shared error boundary when it intentionally centralizes one stable status-mapping policy and preserves inspectability.
- Do not demand `%w` for every wrapped error; wrapping can expose internals as a contract when callers should not inspect them.
- Do not reject a typed or sentinel error helper that keeps errors inspectable and improves context.
- Do not treat repeated `if err != nil { return ... }` as simplification debt unless it hides a stable shared policy or makes diagnosis worse.

## Bad And Good Simplifications
Bad: generic formatting hides the cause.

```go
func serviceErr(op string, err error) error {
	return fmt.Errorf("%s failed: %v", op, err)
}
```

Good: preserve inspection when callers own that contract.

```go
func serviceErr(op string, err error) error {
	return fmt.Errorf("%s failed: %w", op, err)
}
```

Bad: a shared mapper collapses all failure classes.

```go
func writeError(w http.ResponseWriter, err error) {
	http.Error(w, "request failed", http.StatusInternalServerError)
}
```

Good: keep stable classes explicit before falling back.

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

## Escalation Guidance
- Escalate to `go-idiomatic-review` for deep error wrapping, typed errors, nil behavior, context handling, or standard-library contract questions.
- Escalate to `api-contract-designer-spec` when the simplification changes client-visible error shape or status semantics.
- Escalate to `go-reliability-review` when cancellation, deadline, retry, or degradation semantics are being collapsed.
- Escalate to `go-qa-review` when tests assert error strings while the contract is identity, type, or status mapping.

## Source Anchors
- [errors package](https://pkg.go.dev/errors): `errors.Is` and `errors.As` inspect wrapped error trees.
- [Error handling and Go](https://go.dev/doc/articles/error_handling.html): error values can carry context and structured information for callers.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): handle errors, do not panic for normal errors, and keep error flow readable.
- Repository pattern: `go-coder/references/errors-context-and-boundary-mapping.md`.
