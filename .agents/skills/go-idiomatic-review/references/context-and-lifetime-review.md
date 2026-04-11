# Context And Lifetime Review

## When To Load It
Load this reference when a Go review touches `context.Context` parameters, request-scoped work, cancellation, deadlines, stored contexts, nil contexts, derived contexts, `context.Background`, API additions such as `FooContext`, or cancellation-related errors.

## Exa Source Links
- [Go Concurrency Patterns: Context](https://go.dev/blog/context)
- [Contexts and structs](https://go.dev/blog/context-and-structs)
- [context package](https://pkg.go.dev/context)
- [Go Code Review Comments: Contexts](https://go.dev/wiki/CodeReviewComments)
- [Keeping Your Modules Compatible](https://go.dev/blog/module-compatibility)
- [Go 1.20 release notes: context.WithCancelCause](https://go.dev/doc/go1.20)

## Review Cues
- A request path swaps caller `ctx` for `context.Background()` or `context.TODO()`.
- A struct stores `context.Context` and then uses it across multiple operations.
- A derived context is created but the cancel function is not called.
- A function accepts context after other arguments, or omits it where cancellation/deadline matters.
- A stable exported API needs context support without breaking callers.
- Code flattens `context.Canceled` or `context.DeadlineExceeded` into opaque errors.

## Bad Review Examples
Bad review:

```text
Add context everywhere.
```

Why it is bad: context matters for request-scoped cancellation, deadlines, and values. The review should name the affected call path rather than blanket-requiring it.

Bad review:

```text
Never use context.Background().
```

Why it is bad: background context is valid at process roots and non-request-specific work. The defect is using it inside a caller-owned request flow.

Bad review:

```text
Don't store context in structs because it is non-idiomatic.
```

Why it is bad: the merge risk is lifetime confusion and lost per-call cancellation, not taste.

## Good Review Examples
Good finding:

```text
[high] [go-idiomatic-review] internal/payments/client.go:74
Issue: Charge ignores the caller's ctx and creates the HTTP request with context.Background().
Impact: A canceled request can still call the downstream payment service and hold resources after the caller has given up.
Suggested fix: Use http.NewRequestWithContext(ctx, ...) and keep ctx flowing through the outbound call.
Reference: https://go.dev/blog/context
```

Good finding:

```text
[high] [go-idiomatic-review] internal/worker/worker.go:29
Issue: Worker stores the constructor ctx and uses it for every Fetch call.
Impact: Callers cannot set per-call deadlines or cancel one operation without canceling the whole worker lifetime.
Suggested fix: Remove ctx from Worker and accept ctx on Fetch and Process methods that perform request-scoped work.
Reference: https://go.dev/blog/context-and-structs
```

Good finding:

```text
[medium] [go-idiomatic-review] internal/cache/refresh.go:44
Issue: context.WithTimeout creates a timer but the returned cancel function is never called.
Impact: The timer and child context can live until timeout even when the operation finishes early.
Suggested fix: Call cancel with defer immediately after the error check or construction point.
Reference: https://pkg.go.dev/context
```

## Real Merge-Risk Impact
- Lost cancellation can keep outbound work running after the caller returns.
- Stored contexts obscure lifetime and make APIs hard to reason about.
- Missing cancel calls can leak timers and delay resource release.
- Breaking exported signatures to add context can force users to update immediately.
- Flattened cancellation errors can cause callers to retry user-canceled operations.

## Smallest Safe Correction
- Thread the incoming `ctx` through request-scoped calls.
- Accept `ctx context.Context` as the first parameter for operations that observe cancellation, deadlines, or request values.
- Use `defer cancel()` for `WithCancel`, `WithTimeout`, `WithDeadline`, and cause variants when the current function owns cancellation.
- Add a new `FooContext(ctx, ...)` API for stable exported packages instead of breaking an existing `Foo(...)` signature.
- Preserve cancellation causes or sentinel values when callers use them for policy.

## Validation Ideas
- Add tests that cancel the parent context before or during the operation and assert the downstream call stops.
- Add a timeout test using a fake clock or fast timeout only when deterministic.
- Test exported backward-compatible additions with both old and new entry points.
- Run focused package tests with `go test ./path/to/pkg`.

## Handoffs
- Hand off goroutine lifetime, channel cancellation, and worker shutdown depth to concurrency review.
- Hand off retry and deadline budgets to reliability review.
- Hand off request identity, auth values, and tenant data in context to security or API lanes.
- Hand off exported API evolution decisions to design/API review if the smallest fix changes public shape.
