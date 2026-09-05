# Lifetime And Release

## Load When
Load when a Go review touches detached or background work, `context.WithoutCancel`, a stored context, resources opened inside a loop, `scanner.Err`, partial `Read`, or a `defer` whose scope outlives the resource.

## Decide
- `containedctx` gates stored contexts, and the repository's accepted exceptions are narrow: the struct owns that lifetime and cancels it, as the `internal/background` supervisor does, or an external interface requires a `Context()` method. A struct that owns neither is the finding.
- `context.WithoutCancel` keeps request values while dropping cancellation, and it also drops `Done`, `Err`, and `Deadline`. Detached work that must stay bounded needs a new deadline of its own.
- Release and completion are separate obligations. A `bufio.Scanner` stopping is not proof it finished: check `scanner.Err()` unless truncated input is the accepted contract.
- `Read` may return `n > 0` together with a terminal error. Consume those bytes before handling the error, or the final chunk is silently lost.
- `defer` releases at function scope. Inside a loop that opens a resource per iteration, close within the iteration or move the body into a helper whose scope is one resource.
- A close error matters where close is the flush: writers, files, protocols. On a read-only response body it is usually noise that must not mask the earlier read or status error.
- When the operation and completion or cleanup fail independently, preserve
  both with `errors.Join`; a later `Close` must not replace the primary failure.

## Inspect
A page loop with `defer resp.Body.Close()` — the defect is not the unchecked error `errcheck` reports, it is that every body stays open until the enclosing function returns.

## Reject
- "`rows.Close` is present, so the resource is handled" — closing proves release, not that iteration ended without an error.
- "Add a context parameter here" — without a blocking call, a deadline, or a request value, that adds a parameter and no cancellation boundary.

## Reopen
- Prove which caller-visible work continues, leaks, or truncates before reporting.
- Name the resource and the scope that owns its release, not just the missing call.
- Leave goroutine shutdown depth, retry budgets, and SQL transaction correctness to their own lanes.

## Prove
- Fail a fake reader, scanner, or iterator after it yields partial data and assert the function returns an error rather than a short success.
- Count closes with a fake body when loop lifetime is the defect.
- Fail both the operation and its completion step and assert both causes remain
  inspectable.
- Cancel the parent and assert detached work still ends on its own bound.
