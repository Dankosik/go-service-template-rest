# Go errors and context instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Writing or reviewing functions that return errors
  - Designing API contracts or boundary behavior
  - Working with `context.Context`, cancellation, timeouts, I/O, HTTP, RPC, database calls, background jobs, or long-running work
- Do not load when: The task is simple, purely computational, and does not use errors or context beyond trivial defaults

## Error-handling rules

- Errors are part of the function contract. Design them deliberately.
- Return errors explicitly. Do not hide failures behind logs, booleans, or magic values.
- Add operation context to returned errors so failures are diagnosable.
- Use `fmt.Errorf("...: %w", err)` when callers may need to inspect the underlying cause.
- Use `errors.Is` and `errors.As` to inspect errors.
- Do not compare on `err.Error()` text.
- Do not parse error strings.
- Use sentinel errors sparingly and only for stable, meaningful conditions.
- Name sentinel errors as `ErrX`, for example `ErrNotFound`.
- Use typed errors when the caller needs structured fields, not just a yes/no category.
- If a custom error wraps another error, implement `Unwrap`.
- When multiple cleanup steps can fail, consider `errors.Join`.
- Never silently discard a returned error unless the ignore is deliberate, harmless, and obvious from the code.

## Error message style

- Error messages should usually start lowercase.
- Error messages should usually not end with punctuation.
- Messages should say what operation failed and include the resource or identifier when useful.
- Prefer messages like `open config "app.yaml": permission denied` over vague messages like `operation failed`.

## Wrapping policy

Wrap an error when both of these are true:
- More context helps the caller or operator understand the failure.
- Preserving the original cause is useful for programmatic handling.

Do not wrap mechanically at every layer if it adds noise without value. Each wrap should add meaningful context.

## Panic policy

- Do not use `panic` for normal error handling.
- `panic` is acceptable for programmer errors, impossible states, or package-level invariants that indicate a bug.
- Library code should almost never panic on ordinary bad input.

## Context rules

- Accept `ctx context.Context` as the first parameter when cancellation, deadlines, or request scope matter.
- Do not store context in structs.
- Do not pass nil context.
- Use `context.TODO()` only as a temporary placeholder when a real context is not yet available.
- Use context values only for request-scoped data that must cross API boundaries.
- Do not use context values as a substitute for ordinary function parameters or optional arguments.

## Derived context rules

- When you call `context.WithCancel`, `context.WithTimeout`, or `context.WithDeadline`, always call the returned cancel function.
- Call `cancel` as soon as the derived context is no longer needed.
- Ensure long-running work exits promptly when `ctx.Done()` is closed.
- Prefer context-driven cancellation over ad hoc stop channels unless there is a strong reason not to.

## Context-aware error behavior

- Preserve and propagate `context.Canceled` and `context.DeadlineExceeded` cleanly.
- Use `errors.Is(err, context.Canceled)` and `errors.Is(err, context.DeadlineExceeded)` where relevant.
- Do not convert cancellation into misleading business errors.
- When an operation is canceled, make that fact visible to the caller.

## API boundary guidance

- At process boundaries such as HTTP, gRPC, CLI, or message consumers, translate internal errors into stable external behavior.
- Do not leak sensitive internal details in user-facing errors.
- Keep enough internal context for logs and diagnostics.
- If an error category is part of the public contract, make that category stable and test it via `errors.Is` or a typed error.

## Common anti-patterns to avoid

- Comparing `err.Error()` strings
- Returning overly generic errors like `errors.New("failed")`
- Wrapping errors but then checking only direct equality
- Storing context in a struct field
- Forgetting to call `cancel`
- Swallowing context cancellation inside retry loops or worker code
- Logging an error and then returning the same error at every layer without adding value
- Replacing a rich underlying error with a vague message that loses the original cause

## What good output looks like

- The caller can understand what failed from the returned error.
- Programmatic checks work through wrapping.
- Context flows from entry points down to operations that may block.
- Cancellations and timeouts are explicit and well behaved.
- The code uses early returns and stays readable even with multiple error checks.

## Checklist

Before finalizing, verify that:
- Every meaningful returned error is checked or propagated.
- Wrapping uses `%w` when the cause should remain inspectable.
- Error messages are lowercase and action-oriented.
- `ctx` is the first parameter where needed.
- No context is stored in structs.
- Every derived context has a matching `cancel`.
- Cancellation and deadline errors remain recognizable with `errors.Is`.
